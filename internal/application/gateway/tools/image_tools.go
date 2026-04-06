package tools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	assistantdto "dreamcreator/internal/application/assistant/dto"
	assistantservice "dreamcreator/internal/application/assistant/service"
	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/domain/providers"
)

const (
	defaultImageToolPrompt       = "Describe the image."
	defaultImageToolMaxImages    = 20
	defaultImageToolMaxBytes     = 10 * 1024 * 1024
	defaultImageToolMaxRedirects = 3
	defaultImageToolTimeoutMs    = 10000
	defaultImageToolMaxTokens    = 4096
	maxImageToolResponseBytes    = 2 * 1024 * 1024
	defaultAnthropicAPIVersion   = "2023-06-01"
)

var (
	imageToolDataURLPattern    = regexp.MustCompile(`(?i)^data:([^;,]+);base64,([a-z0-9+/=\r\n]+)$`)
	imageToolMediaPrefix       = regexp.MustCompile(`(?i)^\s*media\s*:\s*`)
	imageToolSchemePattern     = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*:`)
	imageToolWindowsPathPrefix = regexp.MustCompile(`^[a-zA-Z]:[\\/]`)
)

type imageToolFetchPolicy struct {
	AllowURL     bool
	URLAllowlist []string
	AllowedMimes []string
	LocalRoots   []string
	MaxBytes     int
	MaxRedirects int
	Timeout      time.Duration
}

type imageToolLoadedImage struct {
	ResolvedImage string
	RewrittenFrom string
	MIMEType      string
	Base64        string
}

type imageToolModelCandidate struct {
	Ref        string
	ProviderID string
	ModelName  string
}

type imageToolUserError struct {
	Code    string
	Message string
	Details map[string]any
}

func (err *imageToolUserError) Error() string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Message)
}

func imageToolUserErrorResult(err *imageToolUserError) map[string]any {
	code := "invalid_image_request"
	message := "Invalid image request."
	details := map[string]any{}
	if err != nil {
		if value := strings.TrimSpace(err.Code); value != "" {
			code = value
		}
		if value := strings.TrimSpace(err.Message); value != "" {
			message = value
		}
		for key, value := range err.Details {
			details[key] = value
		}
	}
	details["error"] = code
	return map[string]any{
		"ok":      false,
		"error":   code,
		"message": message,
		"content": []map[string]any{
			{
				"type": "text",
				"text": message,
			},
		},
		"details": details,
	}
}

func runImageTool(
	settings SettingsReader,
	assistants *assistantservice.AssistantService,
	providerRepo providers.ProviderRepository,
	modelRepo providers.ModelRepository,
	secretRepo providers.SecretRepository,
) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if providerRepo == nil || secretRepo == nil {
			return "", errors.New("provider repositories unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		imageInputs := collectImageToolInputs(payload)
		if len(imageInputs) == 0 {
			return "", errors.New("image required")
		}
		maxImages := resolveImageToolMaxImages(payload)
		if len(imageInputs) > maxImages {
			return marshalResult(imageToolUserErrorResult(&imageToolUserError{
				Code:    "too_many_images",
				Message: fmt.Sprintf("Too many images: %d provided, maximum is %d. Please reduce the number of images.", len(imageInputs), maxImages),
				Details: map[string]any{
					"count": len(imageInputs),
					"max":   maxImages,
				},
			})), nil
		}

		policy := resolveImageToolFetchPolicy(ctx, settings, payload)
		loadedImages, err := loadImageToolInputs(ctx, imageInputs, policy)
		if err != nil {
			var userErr *imageToolUserError
			if errors.As(err, &userErr) {
				return marshalResult(imageToolUserErrorResult(userErr)), nil
			}
			return "", err
		}

		modelOverride := strings.TrimSpace(getStringArg(payload, "model"))
		configuredPrimaryRef := resolveImageToolConfiguredPrimaryRef(ctx, settings)
		candidates, maxTokens, err := resolveImageToolCandidates(
			ctx,
			assistants,
			modelOverride,
			configuredPrimaryRef,
			providerRepo,
			modelRepo,
			secretRepo,
		)
		if err != nil {
			return "", err
		}
		prompt := strings.TrimSpace(getStringArg(payload, "prompt"))
		if prompt == "" {
			prompt = defaultImageToolPrompt
		}

		text, modelRef, attempts, err := runImageToolWithFallback(
			ctx,
			candidates,
			maxTokens,
			prompt,
			loadedImages,
			providerRepo,
			modelRepo,
			secretRepo,
		)
		if err != nil {
			return "", err
		}
		result := map[string]any{
			"ok":       true,
			"model":    modelRef,
			"result":   text,
			"attempts": attempts,
			"content": []map[string]any{
				{
					"type": "text",
					"text": text,
				},
			},
		}
		details := map[string]any{
			"model":    modelRef,
			"attempts": attempts,
		}
		if len(loadedImages) == 1 {
			result["image"] = loadedImages[0].ResolvedImage
			details["image"] = loadedImages[0].ResolvedImage
			if strings.TrimSpace(loadedImages[0].RewrittenFrom) != "" {
				details["rewrittenFrom"] = loadedImages[0].RewrittenFrom
			}
		} else {
			images := make([]string, 0, len(loadedImages))
			detailImages := make([]map[string]any, 0, len(loadedImages))
			for _, item := range loadedImages {
				images = append(images, item.ResolvedImage)
				imageDetails := map[string]any{
					"image": item.ResolvedImage,
				}
				if strings.TrimSpace(item.RewrittenFrom) != "" {
					imageDetails["rewrittenFrom"] = item.RewrittenFrom
				}
				detailImages = append(detailImages, imageDetails)
			}
			result["images"] = images
			details["images"] = detailImages
		}
		result["details"] = details
		return marshalResult(result), nil
	}
}

func collectImageToolInputs(payload toolArgs) []string {
	candidates := make([]string, 0, 8)
	if raw, ok := payload["image"].(string); ok {
		candidates = append(candidates, raw)
	}
	if rawItems, ok := payload["images"].([]any); ok {
		for _, item := range rawItems {
			if value, ok := item.(string); ok {
				candidates = append(candidates, value)
			}
		}
	}
	if rawItems, ok := payload["images"].([]string); ok {
		candidates = append(candidates, rawItems...)
	}
	if len(candidates) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		normalized := trimmed
		if strings.HasPrefix(normalized, "@") {
			normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "@"))
		}
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func resolveImageToolMaxImages(payload toolArgs) int {
	value := defaultImageToolMaxImages
	if parsed, ok := getNumberArg(payload, "maxImages"); ok && parsed > 0 {
		value = int(parsed)
	}
	if value <= 0 {
		return defaultImageToolMaxImages
	}
	return value
}

func resolveImageToolFetchPolicy(ctx context.Context, settings SettingsReader, payload toolArgs) imageToolFetchPolicy {
	policy := imageToolFetchPolicy{
		AllowURL:     true,
		URLAllowlist: nil,
		AllowedMimes: nil,
		LocalRoots:   resolveDefaultMediaLocalRoots(),
		MaxBytes:     defaultImageToolMaxBytes,
		MaxRedirects: defaultImageToolMaxRedirects,
		Timeout:      time.Duration(defaultImageToolTimeoutMs) * time.Millisecond,
	}
	if settings != nil {
		if loaded, err := settings.GetSettings(ctx); err == nil {
			images := pickImageSettings(loaded)
			policy.AllowURL = images.AllowURL
			policy.URLAllowlist = normalizeStringSlice(images.URLAllowlist)
			policy.AllowedMimes = normalizeStringSlice(images.AllowedMimes)
			if images.MaxBytes > 0 {
				policy.MaxBytes = images.MaxBytes
			}
			if images.MaxRedirects > 0 {
				policy.MaxRedirects = images.MaxRedirects
			}
			if images.TimeoutMs > 0 {
				policy.Timeout = time.Duration(images.TimeoutMs) * time.Millisecond
			}
		}
	}
	if maxBytesMb, ok := getNumberArg(payload, "maxBytesMb"); ok && maxBytesMb > 0 {
		maxBytes := int(maxBytesMb * 1024 * 1024)
		if maxBytes > 0 {
			policy.MaxBytes = maxBytes
		}
	}
	if policy.MaxBytes <= 0 {
		policy.MaxBytes = defaultImageToolMaxBytes
	}
	if policy.MaxRedirects <= 0 {
		policy.MaxRedirects = defaultImageToolMaxRedirects
	}
	if policy.Timeout <= 0 {
		policy.Timeout = time.Duration(defaultImageToolTimeoutMs) * time.Millisecond
	}
	return policy
}

func resolveImageToolConfiguredPrimaryRef(ctx context.Context, settings SettingsReader) string {
	if settings == nil {
		return ""
	}
	loaded, err := settings.GetSettings(ctx)
	if err != nil {
		return ""
	}
	providerID := strings.TrimSpace(loaded.AgentModelProviderID)
	modelName := strings.TrimSpace(loaded.AgentModelName)
	if providerID == "" || modelName == "" {
		return ""
	}
	return providerID + "/" + modelName
}

func loadImageToolInputs(ctx context.Context, inputs []string, policy imageToolFetchPolicy) ([]imageToolLoadedImage, error) {
	loaded := make([]imageToolLoadedImage, 0, len(inputs))
	for _, rawInput := range inputs {
		item, err := loadSingleImageToolInput(ctx, rawInput, policy)
		if err != nil {
			return nil, err
		}
		loaded = append(loaded, item)
	}
	return loaded, nil
}

func loadSingleImageToolInput(ctx context.Context, rawInput string, policy imageToolFetchPolicy) (imageToolLoadedImage, error) {
	trimmed := strings.TrimSpace(rawInput)
	imageInput := trimmed
	if strings.HasPrefix(imageInput, "@") {
		imageInput = strings.TrimSpace(strings.TrimPrefix(imageInput, "@"))
	}
	imageInput = imageToolMediaPrefix.ReplaceAllString(imageInput, "")
	if imageInput == "" {
		return imageToolLoadedImage{}, errors.New("image required (empty string in array)")
	}
	lower := strings.ToLower(imageInput)
	isFileURL := strings.HasPrefix(lower, "file://")
	isHTTPURL := strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
	isDataURL := strings.HasPrefix(lower, "data:")
	if imageToolSchemePattern.MatchString(imageInput) && !imageToolWindowsPathPrefix.MatchString(imageInput) && !isFileURL && !isHTTPURL && !isDataURL {
		return imageToolLoadedImage{}, &imageToolUserError{
			Code:    "unsupported_image_reference",
			Message: fmt.Sprintf("Unsupported image reference: %s. Use a file path, a file:// URL, a data: URL, or an http(s) URL.", rawInput),
			Details: map[string]any{
				"image": rawInput,
			},
		}
	}

	if isDataURL {
		buffer, mimeType, err := decodeImageToolDataURL(imageInput)
		if err != nil {
			return imageToolLoadedImage{}, err
		}
		if err := validateImageToolMIME(mimeType, policy.AllowedMimes); err != nil {
			return imageToolLoadedImage{}, err
		}
		if policy.MaxBytes > 0 && len(buffer) > policy.MaxBytes {
			return imageToolLoadedImage{}, fmt.Errorf("image exceeds max bytes: %d > %d", len(buffer), policy.MaxBytes)
		}
		return imageToolLoadedImage{
			ResolvedImage: imageInput,
			MIMEType:      mimeType,
			Base64:        base64.StdEncoding.EncodeToString(buffer),
		}, nil
	}

	if isHTTPURL {
		if !policy.AllowURL {
			return imageToolLoadedImage{}, errors.New("remote image URLs are disabled")
		}
		payload, mimeType, err := downloadImageToolURL(ctx, imageInput, policy)
		if err != nil {
			return imageToolLoadedImage{}, err
		}
		return imageToolLoadedImage{
			ResolvedImage: imageInput,
			MIMEType:      mimeType,
			Base64:        base64.StdEncoding.EncodeToString(payload),
		}, nil
	}

	resolvedPath, err := resolveImageToolLocalPath(imageInput, policy.LocalRoots)
	if err != nil {
		return imageToolLoadedImage{}, err
	}
	payload, err := readImageToolLocalFile(resolvedPath, policy.MaxBytes)
	if err != nil {
		return imageToolLoadedImage{}, err
	}
	mimeType := resolveImageToolMIMEFromPayload(resolvedPath, payload)
	if err := validateImageToolMIME(mimeType, policy.AllowedMimes); err != nil {
		return imageToolLoadedImage{}, err
	}
	return imageToolLoadedImage{
		ResolvedImage: resolvedPath,
		MIMEType:      mimeType,
		Base64:        base64.StdEncoding.EncodeToString(payload),
	}, nil
}

func decodeImageToolDataURL(dataURL string) ([]byte, string, error) {
	matches := imageToolDataURLPattern.FindStringSubmatch(strings.TrimSpace(dataURL))
	if len(matches) != 3 {
		return nil, "", errors.New("invalid data URL (expected base64 image data)")
	}
	mimeType := strings.ToLower(strings.TrimSpace(matches[1]))
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, "", fmt.Errorf("unsupported data URL type: %s", mimeType)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(matches[2]))
	if err != nil {
		return nil, "", errors.New("invalid data URL payload")
	}
	if len(decoded) == 0 {
		return nil, "", errors.New("invalid data URL: empty payload")
	}
	return decoded, mimeType, nil
}

func downloadImageToolURL(ctx context.Context, rawURL string, policy imageToolFetchPolicy) ([]byte, string, error) {
	if !isImageToolURLAllowed(rawURL, policy.URLAllowlist) {
		return nil, "", errors.New("image URL is not allowed by allowlist")
	}
	client := &http.Client{
		Timeout: policy.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= policy.MaxRedirects {
				return errors.New("too many redirects")
			}
			if !isImageToolURLAllowed(req.URL.String(), policy.URLAllowlist) {
				return errors.New("redirect target is not allowed")
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = resp.Status
		}
		return nil, "", fmt.Errorf("image download failed: %s", message)
	}
	payload, err := readImageToolBodyLimited(resp.Body, policy.MaxBytes)
	if err != nil {
		return nil, "", err
	}
	mimeType := resolveImageToolMIME(resp.Header.Get("Content-Type"), payload)
	if err := validateImageToolMIME(mimeType, policy.AllowedMimes); err != nil {
		return nil, "", err
	}
	return payload, mimeType, nil
}

func resolveImageToolLocalPath(raw string, roots []string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("image path is required")
	}
	resolved, err := resolveInboundPath(raw, roots)
	if err != nil {
		return "", errors.New("local image path outside allowed roots")
	}
	return resolved, nil
}

func readImageToolLocalFile(path string, maxBytes int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readImageToolBodyLimited(file, maxBytes)
}

func readImageToolBodyLimited(reader io.Reader, maxBytes int) ([]byte, error) {
	if maxBytes <= 0 {
		maxBytes = defaultImageToolMaxBytes
	}
	limited := io.LimitReader(reader, int64(maxBytes)+1)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(payload) > maxBytes {
		return nil, fmt.Errorf("image exceeds max bytes: %d > %d", len(payload), maxBytes)
	}
	return payload, nil
}

func resolveImageToolMIME(headerValue string, payload []byte) string {
	if value := strings.TrimSpace(headerValue); value != "" {
		if parsed, _, err := mime.ParseMediaType(value); err == nil && strings.TrimSpace(parsed) != "" {
			return strings.ToLower(strings.TrimSpace(parsed))
		}
		return strings.ToLower(strings.TrimSpace(value))
	}
	if len(payload) > 0 {
		return strings.ToLower(strings.TrimSpace(http.DetectContentType(payload)))
	}
	return "image/png"
}

func resolveImageToolMIMEFromPayload(path string, payload []byte) string {
	if ext := strings.ToLower(strings.TrimSpace(filepath.Ext(path))); ext != "" {
		if detected := strings.TrimSpace(mime.TypeByExtension(ext)); detected != "" {
			if parsed, _, err := mime.ParseMediaType(detected); err == nil {
				if strings.TrimSpace(parsed) != "" {
					return strings.ToLower(strings.TrimSpace(parsed))
				}
			}
			return strings.ToLower(strings.TrimSpace(detected))
		}
	}
	return resolveImageToolMIME("", payload)
}

func validateImageToolMIME(mimeType string, allowed []string) error {
	normalized := strings.ToLower(strings.TrimSpace(mimeType))
	if !strings.HasPrefix(normalized, "image/") {
		return fmt.Errorf("unsupported media type: %s", normalized)
	}
	if len(allowed) == 0 {
		return nil
	}
	for _, candidate := range allowed {
		current := strings.ToLower(strings.TrimSpace(candidate))
		if current == "" {
			continue
		}
		if strings.HasSuffix(current, "/*") {
			prefix := strings.TrimSuffix(current, "*")
			if strings.HasPrefix(normalized, prefix) {
				return nil
			}
			continue
		}
		if current == normalized {
			return nil
		}
	}
	return fmt.Errorf("image mime %s is not allowed", normalized)
}

func isImageToolURLAllowed(raw string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return true
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	full := strings.ToLower(strings.TrimSpace(raw))
	for _, entry := range allowlist {
		candidate := strings.ToLower(strings.TrimSpace(entry))
		if candidate == "" {
			continue
		}
		if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
			if strings.HasPrefix(full, candidate) {
				return true
			}
			continue
		}
		if strings.HasPrefix(candidate, "*.") {
			suffix := strings.TrimPrefix(candidate, "*.")
			if host == suffix || strings.HasSuffix(host, "."+suffix) {
				return true
			}
			continue
		}
		if host == candidate {
			return true
		}
	}
	return false
}

func resolveImageToolCandidates(
	ctx context.Context,
	assistants *assistantservice.AssistantService,
	modelOverride string,
	configuredPrimaryRef string,
	providerRepo providers.ProviderRepository,
	modelRepo providers.ModelRepository,
	secretRepo providers.SecretRepository,
) ([]imageToolModelCandidate, int, error) {
	maxTokens := defaultImageToolMaxTokens
	assistantRefs := make([]string, 0, 4)
	defaultProvider := ""
	configuredRef := strings.TrimSpace(configuredPrimaryRef)
	if configuredRef != "" {
		providerID, _, err := parseImageToolModelRef(configuredRef, "")
		if err == nil {
			defaultProvider = providerID
		}
	}

	if assistants != nil {
		items, err := assistants.ListAssistants(ctx, true)
		if err == nil {
			selected, ok := pickDefaultAssistantForImageTool(items)
			if ok {
				if selected.Model.Image.MaxTokens > 0 {
					maxTokens = selected.Model.Image.MaxTokens
				}
				assistantRefs = append(assistantRefs, strings.TrimSpace(selected.Model.Image.Primary))
				assistantRefs = append(assistantRefs, normalizeStringSlice(selected.Model.Image.Fallbacks)...)
				if len(assistantRefs) == 0 {
					assistantRefs = append(assistantRefs, strings.TrimSpace(selected.Model.Agent.Primary))
					assistantRefs = append(assistantRefs, normalizeStringSlice(selected.Model.Agent.Fallbacks)...)
					if selected.Model.Agent.MaxTokens > 0 {
						maxTokens = selected.Model.Agent.MaxTokens
					}
				}
				for _, ref := range assistantRefs {
					providerID, _, parseErr := parseImageToolModelRef(ref, "")
					if parseErr == nil {
						defaultProvider = providerID
						break
					}
				}
			}
		}
	}
	if defaultProvider == "" {
		defaultProvider = resolveImageToolDefaultProviderFromRepos(ctx, providerRepo, secretRepo)
	}

	candidates := make([]imageToolModelCandidate, 0, len(assistantRefs)+1)
	seen := map[string]struct{}{}
	addCandidate := func(raw string, allowImplicitProvider bool) error {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		fallbackProvider := ""
		if allowImplicitProvider {
			fallbackProvider = defaultProvider
		}
		providerID, modelName, err := parseImageToolModelRef(raw, fallbackProvider)
		if err != nil {
			return err
		}
		ref := providerID + "/" + modelName
		if _, exists := seen[ref]; exists {
			return nil
		}
		seen[ref] = struct{}{}
		candidates = append(candidates, imageToolModelCandidate{
			Ref:        ref,
			ProviderID: providerID,
			ModelName:  modelName,
		})
		return nil
	}

	if modelOverride != "" {
		if err := addCandidate(modelOverride, true); err != nil {
			return nil, 0, err
		}
	}
	if configuredRef != "" {
		if err := addCandidate(configuredRef, false); err != nil {
			configuredRef = ""
		}
	}
	for _, ref := range assistantRefs {
		if err := addCandidate(ref, false); err != nil {
			continue
		}
	}
	if len(candidates) == 0 {
		for _, candidate := range resolveImageToolProviderCandidateDefaults(ctx, providerRepo, modelRepo, secretRepo) {
			ref := strings.TrimSpace(candidate.Ref)
			if ref == "" {
				continue
			}
			if _, exists := seen[ref]; exists {
				continue
			}
			seen[ref] = struct{}{}
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil, 0, errors.New("image model is not configured")
	}
	if maxTokens <= 0 {
		maxTokens = defaultImageToolMaxTokens
	}
	return candidates, maxTokens, nil
}

func resolveImageToolDefaultProviderFromRepos(
	ctx context.Context,
	providerRepo providers.ProviderRepository,
	secretRepo providers.SecretRepository,
) string {
	if providerRepo == nil || secretRepo == nil {
		return ""
	}
	list, err := providerRepo.List(ctx)
	if err != nil || len(list) == 0 {
		return ""
	}
	sort.SliceStable(list, func(i, j int) bool {
		left := imageToolProviderOrderKey(list[i])
		right := imageToolProviderOrderKey(list[j])
		if left == right {
			return strings.TrimSpace(list[i].ID) < strings.TrimSpace(list[j].ID)
		}
		return left < right
	})
	for _, provider := range list {
		providerID := strings.TrimSpace(provider.ID)
		if providerID == "" || !provider.Enabled {
			continue
		}
		if !imageToolProviderHasAPIKey(ctx, secretRepo, providerID) {
			continue
		}
		return providerID
	}
	return ""
}

func resolveImageToolProviderCandidateDefaults(
	ctx context.Context,
	providerRepo providers.ProviderRepository,
	modelRepo providers.ModelRepository,
	secretRepo providers.SecretRepository,
) []imageToolModelCandidate {
	if providerRepo == nil || secretRepo == nil || modelRepo == nil {
		return nil
	}
	providersList, err := providerRepo.List(ctx)
	if err != nil || len(providersList) == 0 {
		return nil
	}
	sort.SliceStable(providersList, func(i, j int) bool {
		left := imageToolProviderOrderKey(providersList[i])
		right := imageToolProviderOrderKey(providersList[j])
		if left == right {
			return strings.TrimSpace(providersList[i].ID) < strings.TrimSpace(providersList[j].ID)
		}
		return left < right
	})
	seen := map[string]struct{}{}
	result := make([]imageToolModelCandidate, 0, len(providersList))
	for _, provider := range providersList {
		providerID := strings.TrimSpace(provider.ID)
		if providerID == "" || !provider.Enabled {
			continue
		}
		if !imageToolProviderHasAPIKey(ctx, secretRepo, providerID) {
			continue
		}
		modelName := resolveImageToolDefaultModelName(ctx, modelRepo, providerID)
		if modelName == "" {
			continue
		}
		ref := providerID + "/" + modelName
		if _, exists := seen[ref]; exists {
			continue
		}
		seen[ref] = struct{}{}
		result = append(result, imageToolModelCandidate{
			Ref:        ref,
			ProviderID: providerID,
			ModelName:  modelName,
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func resolveImageToolDefaultModelName(ctx context.Context, modelRepo providers.ModelRepository, providerID string) string {
	if modelRepo == nil {
		return ""
	}
	models, err := modelRepo.ListByProvider(ctx, providerID)
	if err != nil || len(models) == 0 {
		return ""
	}
	for _, modelItem := range models {
		name := strings.TrimSpace(modelItem.Name)
		if name == "" || !modelItem.Enabled {
			continue
		}
		if modelItem.SupportsVision != nil && *modelItem.SupportsVision {
			return name
		}
	}
	for _, modelItem := range models {
		name := strings.TrimSpace(modelItem.Name)
		if name == "" || !modelItem.Enabled {
			continue
		}
		return name
	}
	return ""
}

func imageToolProviderHasAPIKey(ctx context.Context, secretRepo providers.SecretRepository, providerID string) bool {
	if secretRepo == nil || strings.TrimSpace(providerID) == "" {
		return false
	}
	secret, err := secretRepo.GetByProviderID(ctx, providerID)
	if err != nil {
		return false
	}
	return strings.TrimSpace(secret.APIKey) != ""
}

func imageToolProviderOrderKey(provider providers.Provider) int {
	switch imageToolProviderKind(provider) {
	case string(providers.ProviderTypeOpenAI):
		return 0
	case string(providers.ProviderTypeAnthropic):
		return 1
	case "minimax":
		return 2
	default:
		return 100
	}
}

func imageToolProviderKind(provider providers.Provider) string {
	kind := strings.ToLower(strings.TrimSpace(string(provider.Type)))
	if kind == "" {
		kind = strings.ToLower(strings.TrimSpace(provider.ID))
	}
	switch kind {
	case "minimax":
		return "minimax"
	case string(providers.ProviderTypeAnthropic):
		return string(providers.ProviderTypeAnthropic)
	case string(providers.ProviderTypeOpenAI):
		return string(providers.ProviderTypeOpenAI)
	default:
		return kind
	}
}

func pickDefaultAssistantForImageTool(items []assistantdto.Assistant) (assistantdto.Assistant, bool) {
	for _, item := range items {
		if item.IsDefault {
			return item, true
		}
	}
	for _, item := range items {
		if item.Enabled {
			return item, true
		}
	}
	if len(items) == 0 {
		return assistantdto.Assistant{}, false
	}
	return items[0], true
}

func parseImageToolModelRef(raw string, fallbackProvider string) (string, string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", "", errors.New("model is empty")
	}
	if strings.Contains(value, "/") {
		parts := strings.SplitN(value, "/", 2)
		providerID := strings.TrimSpace(parts[0])
		modelName := strings.TrimSpace(parts[1])
		if providerID == "" || modelName == "" {
			return "", "", errors.New("model ref must include provider prefix")
		}
		return providerID, modelName, nil
	}
	if strings.Contains(value, ":") {
		parts := strings.SplitN(value, ":", 2)
		providerID := strings.TrimSpace(parts[0])
		modelName := strings.TrimSpace(parts[1])
		if providerID == "" || modelName == "" {
			return "", "", errors.New("model ref must include provider prefix")
		}
		return providerID, modelName, nil
	}
	if strings.TrimSpace(fallbackProvider) == "" {
		return "", "", errors.New("model ref must include provider prefix")
	}
	return strings.TrimSpace(fallbackProvider), value, nil
}

func runImageToolWithFallback(
	ctx context.Context,
	candidates []imageToolModelCandidate,
	maxTokens int,
	prompt string,
	images []imageToolLoadedImage,
	providerRepo providers.ProviderRepository,
	modelRepo providers.ModelRepository,
	secretRepo providers.SecretRepository,
) (string, string, []map[string]string, error) {
	attempts := make([]map[string]string, 0, len(candidates))
	for _, candidate := range candidates {
		text, err := runImageToolModel(
			ctx,
			candidate,
			maxTokens,
			prompt,
			images,
			providerRepo,
			modelRepo,
			secretRepo,
		)
		if err == nil {
			return text, candidate.Ref, attempts, nil
		}
		attempts = append(attempts, map[string]string{
			"model":    candidate.Ref,
			"provider": candidate.ProviderID,
			"id":       candidate.ModelName,
			"error":    strings.TrimSpace(err.Error()),
		})
	}
	if len(attempts) == 0 {
		return "", "", nil, errors.New("image model invocation failed")
	}
	last := attempts[len(attempts)-1]
	return "", "", attempts, fmt.Errorf("image model invocation failed: %s", last["error"])
}

func runImageToolModel(
	ctx context.Context,
	candidate imageToolModelCandidate,
	maxTokens int,
	prompt string,
	images []imageToolLoadedImage,
	providerRepo providers.ProviderRepository,
	modelRepo providers.ModelRepository,
	secretRepo providers.SecretRepository,
) (string, error) {
	provider, err := providerRepo.Get(ctx, candidate.ProviderID)
	if err != nil {
		return "", err
	}
	if !provider.Enabled {
		return "", errors.New("provider is disabled")
	}
	modelName := candidate.ModelName
	effectiveMaxTokens := maxTokens
	if modelRepo != nil {
		models, listErr := modelRepo.ListByProvider(ctx, candidate.ProviderID)
		if listErr == nil {
			for _, modelItem := range models {
				name := strings.TrimSpace(modelItem.Name)
				if !strings.EqualFold(name, candidate.ModelName) {
					continue
				}
				if modelItem.SupportsVision != nil && !*modelItem.SupportsVision {
					return "", fmt.Errorf("model does not support images: %s/%s", candidate.ProviderID, name)
				}
				modelName = name
				if modelItem.MaxOutputTokens != nil && *modelItem.MaxOutputTokens > 0 {
					if effectiveMaxTokens <= 0 {
						effectiveMaxTokens = *modelItem.MaxOutputTokens
					} else if effectiveMaxTokens > *modelItem.MaxOutputTokens {
						effectiveMaxTokens = *modelItem.MaxOutputTokens
					}
				}
				break
			}
		}
	}
	secret, err := secretRepo.GetByProviderID(ctx, candidate.ProviderID)
	if err != nil {
		return "", err
	}
	apiKey := strings.TrimSpace(secret.APIKey)
	if apiKey == "" {
		return "", errors.New("provider api key missing")
	}
	return invokeImageModelCompletionForProvider(ctx, provider, apiKey, secret.OrgRef, modelName, effectiveMaxTokens, prompt, images)
}

func invokeImageModelCompletionForProvider(
	ctx context.Context,
	provider providers.Provider,
	apiKey string,
	orgRef string,
	modelName string,
	maxTokens int,
	prompt string,
	images []imageToolLoadedImage,
) (string, error) {
	switch imageToolProviderKind(provider) {
	case "minimax":
		return invokeMinimaxImageModelCompletion(ctx, provider.Endpoint, apiKey, prompt, images)
	case string(providers.ProviderTypeAnthropic):
		return invokeAnthropicImageModelCompletion(ctx, provider.Endpoint, apiKey, modelName, maxTokens, prompt, images)
	default:
		return invokeOpenAICompatibleImageModelCompletion(ctx, provider.Endpoint, apiKey, orgRef, modelName, maxTokens, prompt, images)
	}
}

func invokeOpenAICompatibleImageModelCompletion(
	ctx context.Context,
	baseURL string,
	apiKey string,
	orgRef string,
	modelName string,
	maxTokens int,
	prompt string,
	images []imageToolLoadedImage,
) (string, error) {
	endpoint := resolveImageToolChatURL(baseURL)
	if endpoint == "" {
		return "", errors.New("provider endpoint is required")
	}
	content := make([]map[string]any, 0, len(images)+1)
	content = append(content, map[string]any{
		"type": "text",
		"text": prompt,
	})
	for _, item := range images {
		content = append(content, map[string]any{
			"type": "image_url",
			"image_url": map[string]any{
				"url": "data:" + item.MIMEType + ";base64," + item.Base64,
			},
		})
	}
	requestBody := map[string]any{
		"model": modelName,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": content,
			},
		},
	}
	if maxTokens > 0 {
		requestBody["max_tokens"] = maxTokens
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	if trimmed := strings.TrimSpace(orgRef); trimmed != "" {
		request.Header.Set("OpenAI-Organization", trimmed)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxImageToolResponseBytes))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if message := parseImageToolErrorMessage(body); message != "" {
			return "", errors.New(message)
		}
		return "", fmt.Errorf("provider request failed: %s", response.Status)
	}
	text, err := parseImageToolResponseText(body)
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", errors.New("image model returned no text")
	}
	return text, nil
}

func invokeMinimaxImageModelCompletion(
	ctx context.Context,
	baseURL string,
	apiKey string,
	prompt string,
	images []imageToolLoadedImage,
) (string, error) {
	if len(images) == 0 {
		return "", errors.New("image required")
	}
	endpoint := resolveMinimaxVLMURL(baseURL)
	if endpoint == "" {
		return "", errors.New("provider endpoint is required")
	}
	first := images[0]
	imageDataURL := "data:" + first.MIMEType + ";base64," + first.Base64
	requestBody := map[string]any{
		"prompt":    prompt,
		"image_url": imageDataURL,
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	request.Header.Set("MM-API-Source", "DreamCreator")
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxImageToolResponseBytes))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if message := parseImageToolErrorMessage(body); message != "" {
			return "", errors.New(message)
		}
		return "", fmt.Errorf("provider request failed: %s", response.Status)
	}
	text, err := parseMinimaxImageToolResponseText(body)
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", errors.New("image model returned no text")
	}
	return text, nil
}

func invokeAnthropicImageModelCompletion(
	ctx context.Context,
	baseURL string,
	apiKey string,
	modelName string,
	maxTokens int,
	prompt string,
	images []imageToolLoadedImage,
) (string, error) {
	endpoint := resolveAnthropicMessagesURL(baseURL)
	if endpoint == "" {
		return "", errors.New("provider endpoint is required")
	}
	content := make([]map[string]any, 0, len(images)+1)
	content = append(content, map[string]any{
		"type": "text",
		"text": prompt,
	})
	for _, item := range images {
		content = append(content, map[string]any{
			"type": "image",
			"source": map[string]any{
				"type":       "base64",
				"media_type": item.MIMEType,
				"data":       item.Base64,
			},
		})
	}
	if maxTokens <= 0 {
		maxTokens = defaultImageToolMaxTokens
	}
	requestBody := map[string]any{
		"model":      modelName,
		"max_tokens": maxTokens,
		"messages": []map[string]any{
			{
				"role":    "user",
				"content": content,
			},
		},
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-api-key", strings.TrimSpace(apiKey))
	request.Header.Set("anthropic-version", defaultAnthropicAPIVersion)
	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxImageToolResponseBytes))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if message := parseImageToolErrorMessage(body); message != "" {
			return "", errors.New(message)
		}
		return "", fmt.Errorf("provider request failed: %s", response.Status)
	}
	text, err := parseAnthropicImageToolResponseText(body)
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", errors.New("image model returned no text")
	}
	return text, nil
}

func resolveImageToolChatURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if strings.TrimSpace(parsed.Path) == "" || parsed.Path == "/" {
		parsed.Path = "/chat/completions"
		return parsed.String()
	}
	if strings.HasSuffix(parsed.Path, "/chat/completions") {
		return parsed.String()
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/chat/completions"
	return parsed.String()
}

func resolveMinimaxVLMURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return ""
	}
	parsed.Path = "/v1/coding_plan/vlm"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func resolveAnthropicMessagesURL(baseURL string) string {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	pathValue := strings.TrimSpace(parsed.Path)
	switch {
	case pathValue == "" || pathValue == "/":
		parsed.Path = "/v1/messages"
	case strings.HasSuffix(pathValue, "/v1/messages"), strings.HasSuffix(pathValue, "/messages"):
		return parsed.String()
	case strings.HasSuffix(pathValue, "/v1"):
		parsed.Path = strings.TrimRight(pathValue, "/") + "/messages"
	default:
		parsed.Path = strings.TrimRight(pathValue, "/") + "/v1/messages"
	}
	return parsed.String()
}

func parseImageToolErrorMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return trimToMaxChars(trimmed, 280)
	}
	if errorValue, ok := payload["error"]; ok {
		switch typed := errorValue.(type) {
		case string:
			if value := strings.TrimSpace(typed); value != "" {
				return value
			}
		case map[string]any:
			if value := strings.TrimSpace(toCanvasString(typed["message"])); value != "" {
				return value
			}
		}
	}
	return trimToMaxChars(trimmed, 280)
}

func parseImageToolResponseText(body []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if choices, ok := payload["choices"].([]any); ok {
		for _, rawChoice := range choices {
			choice, ok := rawChoice.(map[string]any)
			if !ok {
				continue
			}
			if message, ok := choice["message"].(map[string]any); ok {
				if text := extractImageToolTextFromContent(message["content"]); text != "" {
					return text, nil
				}
			}
		}
	}
	if text := extractImageToolTextFromContent(payload["output_text"]); text != "" {
		return text, nil
	}
	return "", errors.New("image model returned no text")
}

func parseAnthropicImageToolResponseText(body []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if text := extractImageToolTextFromContent(payload["content"]); text != "" {
		return text, nil
	}
	return "", errors.New("image model returned no text")
}

func parseMinimaxImageToolResponseText(body []byte) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if baseResp, ok := payload["base_resp"].(map[string]any); ok {
		status := int(getNumberValue(baseResp["status_code"]))
		if status != 0 {
			message := strings.TrimSpace(toCanvasString(baseResp["status_msg"]))
			if message == "" {
				message = "MiniMax VLM API error"
			}
			return "", errors.New(message)
		}
	}
	if text := strings.TrimSpace(toCanvasString(payload["content"])); text != "" {
		return text, nil
	}
	return "", errors.New("image model returned no text")
}

func getNumberValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	default:
		return 0
	}
}

func extractImageToolTextFromContent(content any) string {
	switch typed := content.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := extractImageToolTextFromContent(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	case map[string]any:
		if text := strings.TrimSpace(toCanvasString(typed["text"])); text != "" {
			return text
		}
		if text := extractImageToolTextFromContent(typed["content"]); text != "" {
			return text
		}
		if text := extractImageToolTextFromContent(typed["output_text"]); text != "" {
			return text
		}
	}
	return ""
}

func pickImageSettings(settings settingsdto.Settings) settingsdto.GatewayHTTPResponsesImagesSettings {
	images := settings.Gateway.HTTP.Endpoints.Responses.Images
	if len(images.URLAllowlist) == 0 {
		images.URLAllowlist = []string{}
	}
	if len(images.AllowedMimes) == 0 {
		images.AllowedMimes = []string{}
	}
	if images.MaxBytes <= 0 {
		images.MaxBytes = defaultImageToolMaxBytes
	}
	if images.MaxRedirects <= 0 {
		images.MaxRedirects = defaultImageToolMaxRedirects
	}
	if images.TimeoutMs <= 0 {
		images.TimeoutMs = defaultImageToolTimeoutMs
	}
	return images
}
