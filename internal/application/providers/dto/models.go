package dto

type Provider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Endpoint string `json:"endpoint"`
	Enabled  bool   `json:"enabled"`
	Builtin  bool   `json:"builtin"`
	Icon     string `json:"icon"`
}

type ProviderModel struct {
	ID                  string `json:"id"`
	ProviderID          string `json:"providerId"`
	Name                string `json:"name"`
	DisplayName         string `json:"displayName"`
	CapabilitiesJSON    string `json:"capabilitiesJson"`
	ContextWindowTokens *int   `json:"contextWindowTokens,omitempty"`
	MaxOutputTokens     *int   `json:"maxOutputTokens,omitempty"`
	SupportsTools       *bool  `json:"supportsTools,omitempty"`
	SupportsReasoning   *bool  `json:"supportsReasoning,omitempty"`
	SupportsVision      *bool  `json:"supportsVision,omitempty"`
	SupportsAudio       *bool  `json:"supportsAudio,omitempty"`
	SupportsVideo       *bool  `json:"supportsVideo,omitempty"`
	Enabled             bool   `json:"enabled"`
	ShowInUI            bool   `json:"showInUi"`
}

type ProviderWithModels struct {
	Provider Provider        `json:"provider"`
	Models   []ProviderModel `json:"models"`
}

type ProviderSecret struct {
	ProviderID string `json:"providerId"`
	APIKey     string `json:"apiKey"`
	OrgRef     string `json:"orgRef"`
}

type UpsertProviderRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Endpoint string `json:"endpoint"`
	Enabled  bool   `json:"enabled"`
}

type UpdateProviderModelRequest struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerId"`
	Enabled    bool   `json:"enabled"`
	ShowInUI   bool   `json:"showInUi"`
}

type ReplaceProviderModelsRequest struct {
	ProviderID string          `json:"providerId"`
	Models     []ProviderModel `json:"models"`
}

type SyncProviderModelsRequest struct {
	ProviderID string `json:"providerId"`
	APIKey     string `json:"apiKey"`
}

type UpsertProviderSecretRequest struct {
	ProviderID string `json:"providerId"`
	APIKey     string `json:"apiKey"`
	OrgRef     string `json:"orgRef"`
}
