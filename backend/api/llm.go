package api

import (
	"context"
	"encoding/json"
	"strings"

	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/provider"
	"dreamcreator/backend/types"
	"go.uber.org/zap"
)

// LLMAPI 以与现有 API 一致的 Wails 绑定风格，暴露 Provider/LLM Profile 能力
type LLMAPI struct {
	ctx context.Context
	svc *provider.Service
}

func NewLLMAPI(svc *provider.Service) *LLMAPI {
	return &LLMAPI{svc: svc}
}

func (api *LLMAPI) Subscribe(ctx context.Context) {
	api.ctx = ctx
	logger.Info("LLMAPI.Subscribe OK")
}

// ---- Providers ----

type providerPublic struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Policy   string `json:"policy"`
	Platform string `json:"platform"`
	Name     string `json:"name"`
	BaseURL  string `json:"base_url"`
	Region   string `json:"region"`
	HasKey   bool   `json:"has_key"`
	APIKey   string `json:"api_key"`
	// Vertex key presence
	HasSAKey         bool            `json:"has_sa_key"`
	Models           []string        `json:"models"`
	RateLimit        types.RateLimit `json:"rate_limit"`
	Enabled          bool            `json:"enabled"`
	AuthMethod       string          `json:"auth_method"`
	APIVersion       string          `json:"api_version"`
	InferenceSummary bool            `json:"inference_summary"`
	APIUsage         map[string]any  `json:"api_usage"`
	CreatedAt        int64           `json:"created_at"`
	UpdatedAt        int64           `json:"updated_at"`
}

// toPublicProvider converts internal Provider to public view model.
func toPublicProvider(p *types.Provider) *providerPublic {
	if p == nil {
		return nil
	}
	return &providerPublic{
		ID:               p.ID,
		Type:             string(p.Type),
		Policy:           string(p.Policy),
		Platform:         p.Platform,
		Name:             p.Name,
		BaseURL:          p.BaseURL,
		Region:           p.Region,
		HasKey:           p.APIKey != "",
		APIKey:           p.APIKey,
		HasSAKey:         p.SAPrivateKey != "",
		Models:           p.Models,
		RateLimit:        p.RateLimit,
		Enabled:          p.Enabled,
		AuthMethod:       p.AuthMethod,
		APIVersion:       p.APIVersion,
		InferenceSummary: p.InferenceSummary,
		APIUsage:         p.APIUsage,
		CreatedAt:        p.CreatedAt.Unix(),
		UpdatedAt:        p.UpdatedAt.Unix(),
	}
}

func (api *LLMAPI) ListProviders() types.JSResp {
	list, err := api.svc.ListProviders(api.ctx)
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	outs := make([]*providerPublic, 0, len(list))
	for _, p := range list {
		outs = append(outs, toPublicProvider(p))
	}
	data, _ := json.Marshal(outs)
	return types.JSResp{Success: true, Data: string(data)}
}

// ListEnabledProviders returns only providers with Enabled = true.
func (api *LLMAPI) ListEnabledProviders() types.JSResp {
	list, err := api.svc.ListProviders(api.ctx)
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	outs := make([]*providerPublic, 0, len(list))
	for _, p := range list {
		if p != nil && p.Enabled {
			outs = append(outs, toPublicProvider(p))
		}
	}
	data, _ := json.Marshal(outs)
	return types.JSResp{Success: true, Data: string(data)}
}

type providerCreateReq struct {
	Type             string           `json:"type"`
	Policy           string           `json:"policy"`
	Platform         string           `json:"platform"`
	Name             string           `json:"name"`
	BaseURL          string           `json:"base_url"`
	Region           string           `json:"region"`
	APIKey           string           `json:"api_key"`
	RateLimit        *types.RateLimit `json:"rate_limit"`
	Enabled          *bool            `json:"enabled"`
	AuthMethod       string           `json:"auth_method"`
	APIVersion       string           `json:"api_version"`
	InferenceSummary *bool            `json:"inference_summary"`
	APIUsage         map[string]any   `json:"api_usage"`
	// Vertex
	ProjectID    string `json:"project_id"`
	SAEmail      string `json:"sa_email"`
	SAPrivateKey string `json:"sa_private_key"`
}

func (api *LLMAPI) CreateProvider(in providerCreateReq) types.JSResp {
	// default enabled to true if absent, to match UI expectations
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	var rl types.RateLimit
	if in.RateLimit != nil {
		rl = *in.RateLimit
	}
	res, err := api.svc.CreateProvider(api.ctx, provider.CreateProviderInput{
		Type:       types.ProviderType(in.Type),
		Policy:     types.ProviderPolicy(in.Policy),
		Platform:   in.Platform,
		Name:       in.Name,
		BaseURL:    in.BaseURL,
		Region:     in.Region,
		APIKey:     in.APIKey,
		RateLimit:  rl,
		Enabled:    enabled,
		AuthMethod: in.AuthMethod,
		APIVersion: in.APIVersion,
		InferenceSummary: func() bool {
			if in.InferenceSummary != nil {
				return *in.InferenceSummary
			}
			return false
		}(),
		APIUsage:     in.APIUsage,
		ProjectID:    in.ProjectID,
		SAEmail:      in.SAEmail,
		SAPrivateKey: in.SAPrivateKey,
	})
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(res)
	return types.JSResp{Success: true, Data: string(data)}
}

func (api *LLMAPI) UpdateProvider(id string, in providerCreateReq) types.JSResp {
	// Build pointer-based input so that omitted fields are not applied.
	var (
		tPtr   *types.ProviderType
		pPtr   *types.ProviderPolicy
		plat   *string
		name   *string
		base   *string
		region *string
		key    *string
		rlPtr  *types.RateLimit
		enPtr  *bool
		amPtr  *string
		avPtr  *string
		isPtr  *bool
		auPtr  *map[string]any
		proj   *string
		sa     *string
		sak    *string
	)
	if strings.TrimSpace(in.Type) != "" {
		v := types.ProviderType(in.Type)
		tPtr = &v
	}
	if strings.TrimSpace(in.Policy) != "" {
		v := types.ProviderPolicy(in.Policy)
		pPtr = &v
	}
	if in.Platform != "" {
		v := in.Platform
		plat = &v
	}
	if in.Name != "" {
		v := in.Name
		name = &v
	}
	if in.BaseURL != "" {
		v := in.BaseURL
		base = &v
	}
	if in.Region != "" {
		v := in.Region
		region = &v
	}
	if in.APIKey != "" {
		v := in.APIKey
		key = &v
	}
	if in.RateLimit != nil {
		rlPtr = in.RateLimit
	}
	if in.Enabled != nil {
		enPtr = in.Enabled
	}
	if strings.TrimSpace(in.AuthMethod) != "" {
		v := in.AuthMethod
		amPtr = &v
	}
	if strings.TrimSpace(in.APIVersion) != "" {
		v := in.APIVersion
		avPtr = &v
	}
	if in.InferenceSummary != nil {
		isPtr = in.InferenceSummary
	}
	if in.APIUsage != nil {
		m := in.APIUsage
		auPtr = &m
	}
	if in.ProjectID != "" {
		v := in.ProjectID
		proj = &v
	}
	if in.SAEmail != "" {
		v := in.SAEmail
		sa = &v
	}
	if in.SAPrivateKey != "" {
		v := in.SAPrivateKey
		sak = &v
	}

	res, err := api.svc.UpdateProvider(api.ctx, id, provider.UpdateProviderInput{
		Type:             tPtr,
		Policy:           pPtr,
		Platform:         plat,
		Name:             name,
		BaseURL:          base,
		Region:           region,
		APIKey:           key,
		RateLimit:        rlPtr,
		Enabled:          enPtr,
		AuthMethod:       amPtr,
		APIVersion:       avPtr,
		InferenceSummary: isPtr,
		APIUsage:         auPtr,
		ProjectID:        proj,
		SAEmail:          sa,
		SAPrivateKey:     sak,
	})
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(res)
	return types.JSResp{Success: true, Data: string(data)}
}

func (api *LLMAPI) DeleteProvider(id string) types.JSResp {
	logger.Info("LLMAPI.DeleteProvider", zap.String("id", id))
	if err := api.svc.DeleteProvider(api.ctx, id); err != nil {
		logger.Warn("LLMAPI.DeleteProvider failed", zap.String("id", id), zap.Error(err))
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	logger.Info("LLMAPI.DeleteProvider success", zap.String("id", id))
	return types.JSResp{Success: true}
}

func (api *LLMAPI) TestProvider(id string) types.JSResp {
	ok, models, errMsg := api.svc.TestConnection(api.ctx, id)
	out := map[string]any{"ok": ok, "models": models, "error": errMsg}
	data, _ := json.Marshal(out)
	return types.JSResp{Success: true, Data: string(data)}
}

func (api *LLMAPI) RefreshModels(id string) types.JSResp {
	models, err := api.svc.RefreshModels(api.ctx, id)
	ok := err == nil
	if !ok {
		logger.Warn("refresh models failed", zap.Error(err))
	}
	out := map[string]any{"ok": ok, "models": models}
	if err != nil {
		out["error"] = err.Error()
	}
	data, _ := json.Marshal(out)
	return types.JSResp{Success: ok, Data: string(data), Msg: func() string {
		if err != nil {
			return err.Error()
		}
		return ""
	}()}
}

// ---- LLM Profiles (legacy) removed; use Global Profiles instead ----

// ---- Global Profiles (model-agnostic) ----

func (api *LLMAPI) ListGlobalProfiles() types.JSResp {
	list, err := api.svc.ListGlobalProfiles(api.ctx)
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(list)
	return types.JSResp{Success: true, Data: string(data)}
}

func (api *LLMAPI) CreateGlobalProfile(p types.GlobalProfile) types.JSResp {
	out, err := api.svc.CreateGlobalProfile(api.ctx, &p)
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(out)
	return types.JSResp{Success: true, Data: string(data)}
}

func (api *LLMAPI) UpdateGlobalProfile(id string, p types.GlobalProfile) types.JSResp {
	p.ID = id
	out, err := api.svc.UpdateGlobalProfile(api.ctx, &p)
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(out)
	return types.JSResp{Success: true, Data: string(data)}
}

func (api *LLMAPI) DeleteGlobalProfile(id string) types.JSResp {
	if err := api.svc.DeleteGlobalProfile(api.ctx, id); err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	return types.JSResp{Success: true}
}

// ---- Maintenance ----

// ResetLLMData clears providers, profiles and models cache data.
// Use with caution. It's recommended to guard usage on the caller side.
func (api *LLMAPI) ResetLLMData() types.JSResp {
	if err := api.svc.ResetLLMData(api.ctx); err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	// After clearing, seed default providers so UI has expected presets.
	n, err := api.svc.EnsureDefaultProviders(api.ctx)
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	data, _ := json.Marshal(map[string]any{"seeded": n})
	return types.JSResp{Success: true, Data: string(data)}
}

// ListAddableProviders exposes add-list candidates so the frontend stays simple:
// - special: two creation options (openai_compat, anthropic_compat)
// - presets: preset_hidden providers that are currently disabled (enabled=false)
func (api *LLMAPI) ListAddableProviders() types.JSResp {
	list, err := api.svc.ListProviders(api.ctx)
	if err != nil {
		return types.JSResp{Success: false, Msg: err.Error()}
	}
	presets := make([]*providerPublic, 0)
	for _, p := range list {
		if p == nil {
			continue
		}
		if strings.ToLower(string(p.Policy)) == string(types.PolicyPresetHidden) && !p.Enabled {
			presets = append(presets, toPublicProvider(p))
		}
	}
	resp := map[string]any{
		"special": []map[string]string{{"type": string(types.ProviderOpenAICompat), "label": "OpenAI Compatible"}, {"type": string(types.ProviderAnthropicCompat), "label": "Anthropic Compatible"}},
		"presets": presets,
	}
	data, _ := json.Marshal(resp)
	return types.JSResp{Success: true, Data: string(data)}
}

// Preset-related API removed. Frontend should manage providers via CRUD only.
