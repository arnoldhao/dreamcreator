package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/domain/providers"
)

func TestRunImageTool_DataURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{"content": "analysis ok"},
				},
			},
		})
	}))
	defer server.Close()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{items: map[string]providers.Provider{
			"openai": {
				ID:       "openai",
				Enabled:  true,
				Endpoint: server.URL,
			},
		}},
		nil,
		imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"openai": {
				ProviderID: "openai",
				APIKey:     "test-key",
			},
		}},
	)

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image"))
	output, err := handler(context.Background(), `{"model":"openai/gpt-4o-mini","image":"`+dataURL+`","prompt":"describe"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if ok, _ := result["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, got %#v", result["ok"])
	}
	if resultText, _ := result["result"].(string); resultText != "analysis ok" {
		t.Fatalf("expected analysis text, got %q", resultText)
	}
	if model, _ := result["model"].(string); model != "openai/gpt-4o-mini" {
		t.Fatalf("expected model ref, got %q", model)
	}
}

func TestRunImageTool_SupportsMediaPrefix(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{"content": "media prefix ok"},
				},
			},
		})
	}))
	defer server.Close()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{items: map[string]providers.Provider{
			"openai": {
				ID:       "openai",
				Enabled:  true,
				Endpoint: server.URL,
			},
		}},
		nil,
		imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"openai": {
				ProviderID: "openai",
				APIKey:     "test-key",
			},
		}},
	)

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image"))
	output, err := handler(context.Background(), `{"model":"openai/gpt-4o-mini","image":"MEDIA: `+dataURL+`","prompt":"describe"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if resultText, _ := result["result"].(string); resultText != "media prefix ok" {
		t.Fatalf("expected media prefix result, got %q", resultText)
	}
}

func TestRunImageTool_RejectsUnsupportedScheme(t *testing.T) {
	t.Parallel()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{},
		nil,
		imageSecretRepoStub{},
	)
	output, err := handler(context.Background(), `{"model":"openai/gpt-4o-mini","image":"image:0"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if ok, _ := result["ok"].(bool); ok {
		t.Fatalf("expected ok=false, got true")
	}
	details, _ := result["details"].(map[string]any)
	if code, _ := details["error"].(string); code != "unsupported_image_reference" {
		t.Fatalf("expected unsupported_image_reference, got %q", code)
	}
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("expected content text in soft error")
	}
}

func TestRunImageTool_RejectsTooManyImages(t *testing.T) {
	t.Parallel()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{},
		nil,
		imageSecretRepoStub{},
	)

	output, err := handler(context.Background(), `{"model":"openai/gpt-4o-mini","images":["a.png","b.png"],"maxImages":1}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if ok, _ := result["ok"].(bool); ok {
		t.Fatalf("expected ok=false, got true")
	}
	details, _ := result["details"].(map[string]any)
	if code, _ := details["error"].(string); code != "too_many_images" {
		t.Fatalf("expected too_many_images, got %q", code)
	}
}

func TestRunImageTool_RespectsRemoteURLToggle(t *testing.T) {
	t.Parallel()

	handler := runImageTool(
		&settingsReaderStub{settings: settingsdto.Settings{
			Gateway: settingsdto.GatewaySettings{
				HTTP: settingsdto.GatewayHTTPSettings{
					Endpoints: settingsdto.GatewayHTTPEndpointsSettings{
						Responses: settingsdto.GatewayHTTPResponsesSettings{
							Enabled: true,
							Images: settingsdto.GatewayHTTPResponsesImagesSettings{
								AllowURL: false,
							},
						},
					},
				},
			},
		}},
		nil,
		imageProviderRepoStub{},
		nil,
		imageSecretRepoStub{},
	)
	_, err := handler(context.Background(), `{"model":"openai/gpt-4o-mini","image":"https://example.com/a.png"}`)
	if err == nil {
		t.Fatalf("expected remote URL blocked error")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunImageTool_BlocksLocalPathOutsideAllowedRoots(t *testing.T) {
	t.Parallel()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{},
		nil,
		imageSecretRepoStub{},
	)
	workspaceRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	payload, err := json.Marshal(map[string]any{
		"model": "openai/gpt-4o-mini",
		"image": filepath.Join(workspaceRoot, "go.mod"),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	_, err = handler(context.Background(), string(payload))
	if err == nil {
		t.Fatalf("expected local path blocked error")
	}
	if !strings.Contains(err.Error(), "outside allowed roots") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunImageTool_ModelOverrideUsesImplicitDefaultProvider(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{"content": "implicit provider ok"},
				},
			},
		})
	}))
	defer server.Close()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{items: map[string]providers.Provider{
			"openai": {
				ID:       "openai",
				Type:     providers.ProviderTypeOpenAI,
				Enabled:  true,
				Endpoint: server.URL,
			},
		}},
		nil,
		imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"openai": {
				ProviderID: "openai",
				APIKey:     "test-key",
			},
		}},
	)
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image"))
	output, err := handler(context.Background(), `{"model":"gpt-4o-mini","image":"`+dataURL+`","prompt":"describe"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if model, _ := result["model"].(string); model != "openai/gpt-4o-mini" {
		t.Fatalf("expected implicit openai model ref, got %q", model)
	}
}

func TestRunImageTool_UsesProviderDefaultsWhenModelMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{"content": "provider default ok"},
				},
			},
		})
	}))
	defer server.Close()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{items: map[string]providers.Provider{
			"openai": {
				ID:       "openai",
				Type:     providers.ProviderTypeOpenAI,
				Enabled:  true,
				Endpoint: server.URL,
			},
		}},
		imageModelRepoStub{items: map[string][]providers.Model{
			"openai": {
				{
					ID:             "openai-gpt-4o-mini",
					ProviderID:     "openai",
					Name:           "gpt-4o-mini",
					Enabled:        true,
					SupportsVision: ptrBool(true),
				},
			},
		}},
		imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"openai": {
				ProviderID: "openai",
				APIKey:     "test-key",
			},
		}},
	)

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image"))
	output, err := handler(context.Background(), `{"image":"`+dataURL+`","prompt":"describe"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if model, _ := result["model"].(string); model != "openai/gpt-4o-mini" {
		t.Fatalf("expected provider default model ref, got %q", model)
	}
}

func TestRunImageTool_UsesConfiguredPrimaryModelAsFallbackCandidate(t *testing.T) {
	t.Parallel()

	anthropicServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": "configured primary ok",
				},
			},
		})
	}))
	defer anthropicServer.Close()

	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("openai fallback should not be called when configured primary succeeds")
	}))
	defer openAIServer.Close()

	handler := runImageTool(
		&settingsReaderStub{settings: settingsdto.Settings{
			AgentModelProviderID: "anthropic",
			AgentModelName:       "claude-3-5-sonnet-latest",
		}},
		nil,
		imageProviderRepoStub{items: map[string]providers.Provider{
			"openai": {
				ID:       "openai",
				Type:     providers.ProviderTypeOpenAI,
				Enabled:  true,
				Endpoint: openAIServer.URL,
			},
			"anthropic": {
				ID:       "anthropic",
				Type:     providers.ProviderTypeAnthropic,
				Enabled:  true,
				Endpoint: anthropicServer.URL,
			},
		}},
		nil,
		imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"openai": {
				ProviderID: "openai",
				APIKey:     "openai-key",
			},
			"anthropic": {
				ProviderID: "anthropic",
				APIKey:     "anthropic-key",
			},
		}},
	)

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image"))
	output, err := handler(context.Background(), `{"image":"`+dataURL+`","prompt":"describe"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if model, _ := result["model"].(string); model != "anthropic/claude-3-5-sonnet-latest" {
		t.Fatalf("expected configured primary model, got %q", model)
	}
}

func TestRunImageTool_MultiImageDetailsUseObjectShape(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{"content": "multi image ok"},
				},
			},
		})
	}))
	defer server.Close()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{items: map[string]providers.Provider{
			"openai": {
				ID:       "openai",
				Enabled:  true,
				Endpoint: server.URL,
			},
		}},
		nil,
		imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"openai": {
				ProviderID: "openai",
				APIKey:     "test-key",
			},
		}},
	)
	dataURLA := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image-a"))
	dataURLB := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image-b"))
	output, err := handler(context.Background(), `{"model":"openai/gpt-4o-mini","images":["`+dataURLA+`","`+dataURLB+`"],"prompt":"describe"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	details, _ := result["details"].(map[string]any)
	rawImages, _ := details["images"].([]any)
	if len(rawImages) != 2 {
		t.Fatalf("expected 2 images in details, got %#v", details["images"])
	}
	first, _ := rawImages[0].(map[string]any)
	firstImage, _ := first["image"].(string)
	if strings.TrimSpace(firstImage) == "" {
		t.Fatalf("expected image details entry, got %#v", first)
	}
}

func TestRunImageTool_AnthropicProvider(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("unexpected api key header: %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatalf("expected anthropic-version header")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": "anthropic ok",
				},
			},
		})
	}))
	defer server.Close()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{items: map[string]providers.Provider{
			"anthropic": {
				ID:       "anthropic",
				Type:     providers.ProviderTypeAnthropic,
				Enabled:  true,
				Endpoint: server.URL,
			},
		}},
		nil,
		imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"anthropic": {
				ProviderID: "anthropic",
				APIKey:     "test-key",
			},
		}},
	)
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image"))
	output, err := handler(context.Background(), `{"model":"anthropic/claude-3-5-sonnet-latest","image":"`+dataURL+`","prompt":"describe"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if resultText, _ := result["result"].(string); resultText != "anthropic ok" {
		t.Fatalf("expected anthropic result text, got %q", resultText)
	}
}

func TestRunImageTool_MinimaxProvider(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/coding_plan/vlm" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"base_resp": map[string]any{
				"status_code": 0,
			},
			"content": "minimax ok",
		})
	}))
	defer server.Close()

	handler := runImageTool(
		nil,
		nil,
		imageProviderRepoStub{items: map[string]providers.Provider{
			"minimax": {
				ID:       "minimax",
				Type:     providers.ProviderType("minimax"),
				Enabled:  true,
				Endpoint: server.URL,
			},
		}},
		nil,
		imageSecretRepoStub{items: map[string]providers.ProviderSecret{
			"minimax": {
				ProviderID: "minimax",
				APIKey:     "test-key",
			},
		}},
	)
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-image"))
	output, err := handler(context.Background(), `{"model":"minimax/MiniMax-VL-01","image":"`+dataURL+`","prompt":"describe"}`)
	if err != nil {
		t.Fatalf("run image tool: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if resultText, _ := result["result"].(string); resultText != "minimax ok" {
		t.Fatalf("expected minimax result text, got %q", resultText)
	}
}

type imageProviderRepoStub struct {
	items map[string]providers.Provider
}

func (stub imageProviderRepoStub) List(context.Context) ([]providers.Provider, error) {
	result := make([]providers.Provider, 0, len(stub.items))
	for _, item := range stub.items {
		result = append(result, item)
	}
	return result, nil
}

func (stub imageProviderRepoStub) Get(_ context.Context, id string) (providers.Provider, error) {
	if item, ok := stub.items[id]; ok {
		return item, nil
	}
	return providers.Provider{}, providers.ErrProviderNotFound
}

func (stub imageProviderRepoStub) Save(context.Context, providers.Provider) error {
	return nil
}

func (stub imageProviderRepoStub) Delete(context.Context, string) error {
	return nil
}

type imageSecretRepoStub struct {
	items map[string]providers.ProviderSecret
}

func (stub imageSecretRepoStub) GetByProviderID(_ context.Context, providerID string) (providers.ProviderSecret, error) {
	if item, ok := stub.items[providerID]; ok {
		return item, nil
	}
	return providers.ProviderSecret{}, providers.ErrProviderSecretNotFound
}

func (stub imageSecretRepoStub) Save(context.Context, providers.ProviderSecret) error {
	return nil
}

func (stub imageSecretRepoStub) DeleteByProviderID(context.Context, string) error {
	return nil
}

type imageModelRepoStub struct {
	items map[string][]providers.Model
}

func (stub imageModelRepoStub) ListByProvider(_ context.Context, providerID string) ([]providers.Model, error) {
	result := append([]providers.Model(nil), stub.items[providerID]...)
	return result, nil
}

func (stub imageModelRepoStub) Get(_ context.Context, id string) (providers.Model, error) {
	for _, models := range stub.items {
		for _, modelItem := range models {
			if modelItem.ID == id {
				return modelItem, nil
			}
		}
	}
	return providers.Model{}, providers.ErrModelNotFound
}

func (stub imageModelRepoStub) Save(context.Context, providers.Model) error {
	return nil
}

func (stub imageModelRepoStub) ReplaceByProvider(context.Context, string, []providers.Model) error {
	return nil
}

func (stub imageModelRepoStub) Delete(context.Context, string) error {
	return nil
}

func TestParseImageToolModelRefRejectsMissingProvider(t *testing.T) {
	t.Parallel()

	_, _, err := parseImageToolModelRef("gpt-4o", "")
	if err == nil {
		t.Fatalf("expected missing provider error")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func ptrBool(value bool) *bool {
	return &value
}
