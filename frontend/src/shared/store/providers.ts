export interface Provider {
  id: string;
  name: string;
  type: string;
  compatibility: string;
  endpoint: string;
  enabled: boolean;
  builtin: boolean;
  icon?: string;
}

export interface ProviderModel {
  id: string;
  providerId: string;
  name: string;
  displayName: string;
  capabilitiesJson: string;
  contextWindowTokens?: number;
  maxOutputTokens?: number;
  supportsTools?: boolean;
  supportsReasoning?: boolean;
  supportsVision?: boolean;
  supportsAudio?: boolean;
  supportsVideo?: boolean;
  enabled: boolean;
  showInUi: boolean;
}

export interface ProviderWithModels {
  provider: Provider;
  models: ProviderModel[];
}

export interface ProviderSecret {
  providerId: string;
  apiKey: string;
  orgRef: string;
}

export interface SyncProviderModelsRequest {
  providerId: string;
  apiKey: string;
}

export interface UpdateProviderModelRequest {
  id: string;
  providerId: string;
  enabled: boolean;
  showInUi: boolean;
}

export interface ReplaceProviderModelsRequest {
  providerId: string;
  models: ProviderModel[];
}

export interface UpsertProviderRequest {
  id?: string;
  name: string;
  type: string;
  compatibility?: string;
  endpoint: string;
  enabled: boolean;
}

export interface UpsertProviderSecretRequest {
  providerId: string;
  apiKey: string;
  orgRef: string;
}
