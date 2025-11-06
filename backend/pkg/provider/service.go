package provider

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"time"

	"dreamcreator/backend/pkg/logger"
	openaiclient "dreamcreator/backend/pkg/provider/openai"
	"dreamcreator/backend/pkg/proxy"
	"dreamcreator/backend/pkg/rate"
	"dreamcreator/backend/storage"
	"dreamcreator/backend/types"

	"go.uber.org/zap"
)

type Service struct {
	store   *storage.BoltStorage
	proxies *proxy.Manager
	limits  *rate.LimiterManager
}

func NewService(store *storage.BoltStorage, proxies *proxy.Manager) *Service {
	return &Service{
		store:   store,
		proxies: proxies,
		limits:  rate.NewLimiterManager(),
	}
}

// --- ID utils ---
func genID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// normalizeBase trims redundant trailing slashes while preserving a trailing
// "/v1" if present. The OpenAI-compatible client will append "/v1/..." when
// needed, so we avoid double slashes but do not forcibly strip version suffixes.
func normalizeBase(s string) string {
	v := strings.TrimSpace(s)
	// keep version suffix like /v1 to align with vendor docs; only trim redundant slashes
	for strings.HasSuffix(v, "//") {
		v = strings.TrimRight(v, "/")
	}
	if v != "" && strings.HasSuffix(v, "/") {
		v = strings.TrimRight(v, "/")
	}
	return v
}

// --- Provider ops ---

type CreateProviderInput struct {
	Type      types.ProviderType
	Policy    types.ProviderPolicy
	Platform  string
	Name      string
	BaseURL   string
	Region    string
	APIKey    string
	RateLimit types.RateLimit
	Enabled   bool
	// Reserved/extended
	AuthMethod       string
	APIVersion       string
	InferenceSummary bool
	APIUsage         map[string]any
	// Vertex AI
	ProjectID    string
	SAEmail      string
	SAPrivateKey string
}

type CreateProviderResult struct {
	ID     string `json:"id"`
	HasKey bool   `json:"has_key"`
}

// last4 no longer used (local client scenario); kept for potential logging or future use.
// func last4(s string) string { if len(s) >= 4 { return s[len(s)-4:] }; return s }

func (s *Service) CreateProvider(ctx context.Context, in CreateProviderInput) (*CreateProviderResult, error) {
	// 允许通用型 Provider 先创建占位（BaseURL 可为空，后续再填写）
	nameTrim := strings.TrimSpace(in.Name)
	if nameTrim == "" || in.Type == "" {
		return nil, errors.New("name/type required")
	}
	// 全局去重：名称唯一（忽略大小写）；避免从 add-list 重复添加同一预设
	if recs, err := s.store.ListProviders(); err == nil {
		for _, r := range recs {
			if strings.EqualFold(strings.TrimSpace(r.Name), nameTrim) {
				return nil, errors.New("provider already exists")
			}
		}
	}
	id := genID("prov")
	policy := string(in.Policy)
	if policy == "" {
		policy = string(types.PolicyCustom)
	}
	rec := &storage.ProviderRecord{
		ID:       id,
		Type:     string(in.Type),
		Policy:   policy,
		Platform: strings.TrimSpace(in.Platform),
		Name:     nameTrim,
		BaseURL:  normalizeBase(in.BaseURL),
		// optional region for cloud providers
		Region:       strings.TrimSpace(in.Region),
		APIKey:       in.APIKey,
		ProjectID:    strings.TrimSpace(in.ProjectID),
		SAEmail:      strings.TrimSpace(in.SAEmail),
		SAPrivateKey: in.SAPrivateKey,
		Models:       nil,
		RateLimit:    storage.RateLimitRec{RPM: in.RateLimit.RPM, RPS: in.RateLimit.RPS, Burst: in.RateLimit.Burst, Concurrency: in.RateLimit.Concurrency},
		Enabled:      in.Enabled,
		AuthMethod: func() string {
			if strings.TrimSpace(in.AuthMethod) != "" {
				return strings.ToLower(strings.TrimSpace(in.AuthMethod))
			}
			return "api"
		}(),
		APIVersion:       strings.TrimSpace(in.APIVersion),
		InferenceSummary: in.InferenceSummary,
		APIUsage:         in.APIUsage,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if err := s.store.SaveProvider(rec); err != nil {
		return nil, err
	}
	// 初始化限速器
	s.limits.Configure(id, in.RateLimit.RPS, in.RateLimit.RPM, in.RateLimit.Burst, in.RateLimit.Concurrency)
	return &CreateProviderResult{ID: id, HasKey: in.APIKey != ""}, nil
}

// UpdateProviderInput uses pointer fields to distinguish "not provided" from
// explicit zero-values (false/0/empty). Only non-nil fields will be applied.
type UpdateProviderInput struct {
	Type             *types.ProviderType
	Policy           *types.ProviderPolicy
	Platform         *string
	Name             *string
	BaseURL          *string
	Region           *string
	APIKey           *string
	RateLimit        *types.RateLimit
	Enabled          *bool
	AuthMethod       *string
	APIVersion       *string
	InferenceSummary *bool
	APIUsage         *map[string]any
	// Vertex AI
	ProjectID    *string
	SAEmail      *string
	SAPrivateKey *string
}

func (s *Service) UpdateProvider(ctx context.Context, id string, in UpdateProviderInput) (*CreateProviderResult, error) {
	rec, err := s.store.GetProvider(id)
	if err != nil {
		return nil, err
	}
	// 名称可编辑性：仅 custom 允许
	if in.Name != nil {
		if strings.ToLower(rec.Policy) == string(types.PolicyCustom) || rec.Policy == "" {
			rec.Name = *in.Name
		}
	}
	if in.BaseURL != nil {
		rec.BaseURL = normalizeBase(*in.BaseURL)
	}
	if in.Type != nil {
		rec.Type = string(*in.Type)
	}
	if in.Policy != nil {
		rec.Policy = string(*in.Policy)
	}
	if in.Platform != nil {
		rec.Platform = strings.TrimSpace(*in.Platform)
	}
	if in.APIKey != nil {
		rec.APIKey = *in.APIKey
	}
	if in.Region != nil {
		rec.Region = strings.TrimSpace(*in.Region)
	}
	if in.ProjectID != nil {
		rec.ProjectID = strings.TrimSpace(*in.ProjectID)
	}
	if in.SAEmail != nil {
		rec.SAEmail = strings.TrimSpace(*in.SAEmail)
	}
	if in.SAPrivateKey != nil {
		rec.SAPrivateKey = *in.SAPrivateKey
	}
	if in.Enabled != nil {
		rec.Enabled = *in.Enabled
	}
	if in.AuthMethod != nil {
		rec.AuthMethod = strings.ToLower(strings.TrimSpace(*in.AuthMethod))
	}
	if in.APIVersion != nil {
		rec.APIVersion = strings.TrimSpace(*in.APIVersion)
	}
	if in.InferenceSummary != nil {
		rec.InferenceSummary = *in.InferenceSummary
	}
	if in.APIUsage != nil {
		rec.APIUsage = *in.APIUsage
	}
	if in.RateLimit != nil {
		rl := *in.RateLimit
		rec.RateLimit = storage.RateLimitRec{RPM: rl.RPM, RPS: rl.RPS, Burst: rl.Burst, Concurrency: rl.Concurrency}
	}
	// If this is a preset_hidden provider and currently disabled, make it enabled when
	// user starts configuring it (BaseURL/APIKey provided) unless caller explicitly set Enabled.
	if strings.ToLower(rec.Policy) == string(types.PolicyPresetHidden) && !rec.Enabled && in.Enabled == nil {
		if (in.BaseURL != nil && strings.TrimSpace(*in.BaseURL) != "") || (in.APIKey != nil && strings.TrimSpace(*in.APIKey) != "") {
			rec.Enabled = true
		}
	}
	rec.UpdatedAt = time.Now()
	if err := s.store.SaveProvider(rec); err != nil {
		return nil, err
	}
	// 重新配置限速
	s.limits.Configure(id, rec.RateLimit.RPS, rec.RateLimit.RPM, rec.RateLimit.Burst, rec.RateLimit.Concurrency)
	return &CreateProviderResult{ID: rec.ID, HasKey: rec.APIKey != ""}, nil
}

func (s *Service) DeleteProvider(ctx context.Context, id string) error {
	logger.Info("provider.Service.DeleteProvider start", zap.String("id", id))
	rec, err := s.store.GetProvider(id)
	if err != nil {
		return err
	}
	pol := strings.ToLower(strings.TrimSpace(rec.Policy))
	switch pol {
	case string(types.PolicyPresetShow):
		// 不允许删除固定展示的预设
		return errors.New("cannot delete preset_show provider")
	case string(types.PolicyPresetHidden):
		// 重置为初始化结构（种子默认）
		s.applySeedDefaults(rec)
		rec.UpdatedAt = time.Now()
		if err := s.store.SaveProvider(rec); err != nil {
			return err
		}
		s.limits.Remove(id)
		logger.Info("provider.Service.DeleteProvider reset preset_hidden", zap.String("id", id))
		return nil
	default:
		// custom 或未知：直接删除
		s.limits.Remove(id)
		if err := s.store.DeleteProvider(id); err != nil {
			logger.Warn("provider.Service.DeleteProvider failed", zap.String("id", id), zap.Error(err))
			return err
		}
		logger.Info("provider.Service.DeleteProvider success", zap.String("id", id))
		return nil
	}
}

func (s *Service) GetProvider(ctx context.Context, id string) (*types.Provider, error) {
	rec, err := s.store.GetProvider(id)
	if err != nil {
		return nil, err
	}
	return &types.Provider{
		ID:       rec.ID,
		Type:     types.ProviderType(rec.Type),
		Policy:   types.ProviderPolicy(rec.Policy),
		Platform: rec.Platform,
		Name:     rec.Name,
		BaseURL:  rec.BaseURL,
		Region:   rec.Region,
		APIKey:   rec.APIKey,

		ProjectID:        rec.ProjectID,
		SAEmail:          rec.SAEmail,
		SAPrivateKey:     rec.SAPrivateKey,
		Models:           rec.Models,
		RateLimit:        types.RateLimit{RPM: rec.RateLimit.RPM, RPS: rec.RateLimit.RPS, Burst: rec.RateLimit.Burst, Concurrency: rec.RateLimit.Concurrency},
		Enabled:          rec.Enabled,
		AuthMethod:       rec.AuthMethod,
		APIVersion:       rec.APIVersion,
		InferenceSummary: rec.InferenceSummary,
		APIUsage:         rec.APIUsage,
		CreatedAt:        rec.CreatedAt,
		UpdatedAt:        rec.UpdatedAt,
	}, nil
}

func (s *Service) ListProviders(ctx context.Context) ([]*types.Provider, error) {
	recs, err := s.store.ListProviders()
	if err != nil {
		return nil, err
	}
	out := make([]*types.Provider, 0, len(recs))
	for _, r := range recs {
		out = append(out, &types.Provider{
			ID:       r.ID,
			Type:     types.ProviderType(r.Type),
			Policy:   types.ProviderPolicy(r.Policy),
			Platform: r.Platform,
			Name:     r.Name,
			BaseURL:  r.BaseURL,
			Region:   r.Region,
			APIKey:   r.APIKey,

			ProjectID:        r.ProjectID,
			SAEmail:          r.SAEmail,
			SAPrivateKey:     r.SAPrivateKey,
			Models:           r.Models,
			RateLimit:        types.RateLimit{RPM: r.RateLimit.RPM, RPS: r.RateLimit.RPS, Burst: r.RateLimit.Burst, Concurrency: r.RateLimit.Concurrency},
			Enabled:          r.Enabled,
			AuthMethod:       r.AuthMethod,
			APIVersion:       r.APIVersion,
			InferenceSummary: r.InferenceSummary,
			APIUsage:         r.APIUsage,
			CreatedAt:        r.CreatedAt,
			UpdatedAt:        r.UpdatedAt,
		})
	}
	return out, nil
}

// (Removed legacy preset/migration code for simplicity)

// --- LLM Profile ops ---

func (s *Service) CreateLLMProfile(ctx context.Context, p *types.LLMProfile) (*types.LLMProfile, error) {
	if p == nil {
		return nil, errors.New("nil profile")
	}
	if p.ProviderID == "" || p.Model == "" {
		return nil, errors.New("provider_id/model required")
	}
	if p.ID == "" {
		p.ID = genID("llmprof")
	}
	rec := &storage.LLMProfileRecord{
		ID:           p.ID,
		ProviderID:   p.ProviderID,
		Model:        p.Model,
		Temperature:  p.Temperature,
		TopP:         p.TopP,
		JSONMode:     p.JSONMode,
		SysPromptTpl: p.SysPromptTpl,
		CostWeight:   p.CostWeight,
		MaxTokens:    p.MaxTokens,
		Metadata:     p.Metadata,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.store.SaveLLMProfile(rec); err != nil {
		return nil, err
	}
	return s.GetLLMProfile(ctx, rec.ID)
}

func (s *Service) GetLLMProfile(ctx context.Context, id string) (*types.LLMProfile, error) {
	rec, err := s.store.GetLLMProfile(id)
	if err != nil {
		return nil, err
	}
	return &types.LLMProfile{
		ID:           rec.ID,
		ProviderID:   rec.ProviderID,
		Model:        rec.Model,
		Temperature:  rec.Temperature,
		TopP:         rec.TopP,
		JSONMode:     rec.JSONMode,
		SysPromptTpl: rec.SysPromptTpl,
		CostWeight:   rec.CostWeight,
		MaxTokens:    rec.MaxTokens,
		Metadata:     rec.Metadata,
		CreatedAt:    rec.CreatedAt,
		UpdatedAt:    rec.UpdatedAt,
	}, nil
}

func (s *Service) ListLLMProfiles(ctx context.Context) ([]*types.LLMProfile, error) {
	recs, err := s.store.ListLLMProfiles()
	if err != nil {
		return nil, err
	}
	out := make([]*types.LLMProfile, 0, len(recs))
	for _, r := range recs {
		out = append(out, &types.LLMProfile{
			ID:           r.ID,
			ProviderID:   r.ProviderID,
			Model:        r.Model,
			Temperature:  r.Temperature,
			TopP:         r.TopP,
			JSONMode:     r.JSONMode,
			SysPromptTpl: r.SysPromptTpl,
			CostWeight:   r.CostWeight,
			MaxTokens:    r.MaxTokens,
			Metadata:     r.Metadata,
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
		})
	}
	return out, nil
}

func (s *Service) UpdateLLMProfile(ctx context.Context, p *types.LLMProfile) (*types.LLMProfile, error) {
	if p == nil || p.ID == "" {
		return nil, errors.New("id required")
	}
	rec, err := s.store.GetLLMProfile(p.ID)
	if err != nil {
		return nil, err
	}
	if p.ProviderID != "" {
		rec.ProviderID = p.ProviderID
	}
	if p.Model != "" {
		rec.Model = p.Model
	}
	rec.Temperature = p.Temperature
	rec.TopP = p.TopP
	rec.JSONMode = p.JSONMode
	rec.SysPromptTpl = p.SysPromptTpl
	rec.CostWeight = p.CostWeight
	rec.MaxTokens = p.MaxTokens
	if p.Metadata != nil {
		rec.Metadata = p.Metadata
	}
	rec.UpdatedAt = time.Now()
	if err := s.store.SaveLLMProfile(rec); err != nil {
		return nil, err
	}
	return s.GetLLMProfile(ctx, rec.ID)
}

func (s *Service) DeleteLLMProfile(ctx context.Context, id string) error {
	return s.store.DeleteLLMProfile(id)
}

// --- Models cache and test ---

func (s *Service) RefreshModels(ctx context.Context, providerID string) ([]string, error) {
	rec, err := s.store.GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	// MOCK 支持
	if os.Getenv("MOCK_LLM") == "true" || os.Getenv("MOCK_LLM") == "1" {
		models := []string{"gpt-4o-mini", "gpt-4o", "text-embedding-3-large"}
		rec.Models = models
		if err := s.store.SaveProvider(rec); err != nil {
			return nil, err
		}
		_ = s.store.SaveModelsCache(&storage.ModelsCacheRecord{ProviderID: providerID, Models: models})
		return models, nil
	}

	// 限速器：每次调用前申请许可
	limiter := s.limits.Get(providerID)
	if limiter != nil {
		if err := limiter.Acquire(ctx); err != nil {
			return nil, err
		}
		defer limiter.Release()
	}
	client := openaiclient.NewClient(rec.BaseURL, rec.APIKey, s.proxies.GetHTTPClient())
	models, err := client.ListModels(ctx)
	if err != nil {
		logger.Warn("list models failed", zap.Error(err))
		return nil, err
	}
	rec.Models = models
	if err := s.store.SaveProvider(rec); err != nil {
		return nil, err
	}
	_ = s.store.SaveModelsCache(&storage.ModelsCacheRecord{ProviderID: providerID, Models: models})
	return models, nil
}

func (s *Service) TestConnection(ctx context.Context, providerID string) (ok bool, models []string, errMsg string) {
	models, err := s.RefreshModels(ctx, providerID)
	if err != nil {
		return false, nil, err.Error()
	}
	return true, models, ""
}

// ResetLLMData clears provider, profiles and models cache data.
func (s *Service) ResetLLMData(ctx context.Context) error {
	return s.store.ResetLLMData()
}

// --- Defaults initialization (Presets via Policy) ---

type seedProvider struct {
	Name    string
	BaseURL string
	Policy  types.ProviderPolicy
	Enabled bool
}

func defaultSeeds() []seedProvider {
	// Preset seeds per UI: preset_show (sidebar list) + preset_hidden (add-list entries).
	// Keys are left empty for safety; users fill API key later.
	// Type uses openai_compat for unified handling.

	presetShow := []seedProvider{
		{Name: "Anthropic", BaseURL: "https://api.anthropic.com/v1", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "Azure OpenAI", BaseURL: "", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "DeepSeek", BaseURL: "https://api.deepseek.com/", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "GitHub Copilot", BaseURL: "", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "Google AI", BaseURL: "https://generativelanguage.googleapis.com/v1", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "Groq", BaseURL: "https://api.groq.com/openai/v1", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "Mistral", BaseURL: "https://api.mistral.ai/v1", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "Ollama", BaseURL: "http://127.0.0.1:11434", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "OpenAI", BaseURL: "https://api.openai.com/v1", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "OpenRouter", BaseURL: "https://openrouter.ai/api/v1", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "Perplexity", BaseURL: "https://api.perplexity.ai", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "Together", BaseURL: "https://api.together.xyz/v1", Policy: types.PolicyPresetShow, Enabled: true},
		{Name: "xAI", BaseURL: "https://api.xai.io/v1", Policy: types.PolicyPresetShow, Enabled: true},
	}

	// Note: Top two in add-list "OpenAI Compatible" and "Anthropic Compatible" are for custom creation, not presets.
	presetHidden := []seedProvider{
		{Name: "302.AI", BaseURL: "https://api.302.ai/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "AiHubMix", BaseURL: "https://aihubmix.com/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "阿里云", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Amazon Bedrock", BaseURL: "", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Cerebras", BaseURL: "https://api.cerebras.ai/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Deepbricks", BaseURL: "", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Fireworks", BaseURL: "https://api.fireworks.ai/inference/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "GitHub Models", BaseURL: "https://models.inference.ai.azure.com/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Hugging Face", BaseURL: "https://router.huggingface.co/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Hyperbolic", BaseURL: "https://api.hyperbolic.xyz/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Jina DeepSearch", BaseURL: "https://deepsearch.jina.ai/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Kimi", BaseURL: "https://api.moonshot.cn/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "LM Studio", BaseURL: "http://127.0.0.1:1234/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Poe", BaseURL: "https://api.poe.com/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "硅基流动", BaseURL: "https://api.siliconflow.cn/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Vercel", BaseURL: "https://ai-gateway.vercel.sh/v1", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Vertex AI", BaseURL: "", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "火山引擎", BaseURL: "https://ark.cn-beijing.volces.com/api/v3", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "Z.ai", BaseURL: "https://api.z.ai/api/paas/v4", Policy: types.PolicyPresetHidden, Enabled: false},
		{Name: "智谱", BaseURL: "https://open.bigmodel.cn/api/paas/v4", Policy: types.PolicyPresetHidden, Enabled: false},
	}

	return append(presetShow, presetHidden...)
}

// seedByName returns the seed definition for the given provider display name.
func seedByName(name string) (*seedProvider, bool) {
	n := strings.TrimSpace(name)
	if n == "" {
		return nil, false
	}
	seeds := defaultSeeds()
	for i := range seeds {
		sd := seeds[i]
		if strings.EqualFold(strings.TrimSpace(sd.Name), n) {
			return &sd, true
		}
	}
	return nil, false
}

func defaultRateLimitRec() storage.RateLimitRec {
	return storage.RateLimitRec{RPS: 2, RPM: 120, Burst: 4, Concurrency: 4}
}

// applySeedDefaults rewrites a preset provider record to its initial default
// configuration based on seeds (without changing ID/CreatedAt/Policy/Name).
func (s *Service) applySeedDefaults(rec *storage.ProviderRecord) {
	sd, ok := seedByName(rec.Name)
	rec.Type = string(types.ProviderOpenAICompat)
	if ok {
		rec.BaseURL = normalizeBase(sd.BaseURL)
	} else {
		rec.BaseURL = ""
	}
	// Clear auth/config/state and set defaults
	rec.APIKey = ""
	rec.Models = nil
	rec.RateLimit = defaultRateLimitRec()
	rec.Enabled = false
	rec.Region = ""
	rec.ProjectID = ""
	rec.SAEmail = ""
	rec.SAPrivateKey = ""
	rec.AuthMethod = "api"
	rec.APIVersion = ""
	rec.InferenceSummary = false
	rec.APIUsage = nil
}

// defaultBaseURLFor returns the seeded default BaseURL for a given provider name.
// (unused) defaultBaseURLFor was replaced by seedByName + applySeedDefaults for clarity.

// EnsureDefaultProviders seeds a minimal set of preset providers when empty.
// It uses a migration flag to ensure at-most-once initialization across runs.
func (s *Service) EnsureDefaultProviders(ctx context.Context) (int, error) {
	// 简化逻辑：仅当 providers 为空时进行初始化，无需迁移标记
	recs, err := s.store.ListProviders()
	if err != nil {
		return 0, err
	}
	if len(recs) > 0 {
		return 0, nil
	}
	// Seed defaults
	count := 0
	now := time.Now()
	for _, sd := range defaultSeeds() {
		id := genID("prov")
		rec := &storage.ProviderRecord{
			ID:        id,
			Type:      string(types.ProviderOpenAICompat),
			Policy:    string(sd.Policy),
			Name:      sd.Name,
			BaseURL:   strings.TrimRight(sd.BaseURL, "/"),
			APIKey:    "",
			Models:    nil,
			RateLimit: storage.RateLimitRec{RPS: 2, RPM: 120, Burst: 4, Concurrency: 4},
			Enabled: func() bool {
				if sd.Policy == types.PolicyPresetHidden {
					return false
				}
				return sd.Enabled
			}(),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := s.store.SaveProvider(rec); err != nil {
			return count, err
		}
		s.limits.Configure(id, rec.RateLimit.RPS, rec.RateLimit.RPM, rec.RateLimit.Burst, rec.RateLimit.Concurrency)
		count++
	}
	return count, nil
}
