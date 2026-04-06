export type AssistantPromptMode = "full" | "minimal" | "none";
export type AssistantCallMode = "auto" | "custom";
export type AssistantLocaleMode = "auto" | "manual";

export interface AssistantSoul {
  coreTruths?: string[];
  boundaries?: string[];
  rules?: string[];
  vibe?: string;
  continuity?: string;
}

export interface AssistantIdentity {
  name?: string;
  creature?: string;
  emoji?: string;
  role?: string;
  soul?: AssistantSoul;
}

export interface AssistantAvatarAssetRef {
  path?: string;
  displayName?: string;
  source?: string;
  assetId?: string;
  meta?: Record<string, string>;
}

export type Assistant3DAvatar = AssistantAvatarAssetRef;
export type Assistant3DMotion = AssistantAvatarAssetRef;

export interface AssistantAvatar {
  avatar3d?: AssistantAvatarAssetRef;
  motion?: AssistantAvatarAssetRef;
}

export interface UserLocale {
  mode?: AssistantLocaleMode | string;
  value?: string;
  current?: string;
}

export interface UserExtraField {
  key?: string;
  value?: string;
}

export interface AssistantUser {
  name?: string;
  preferredAddress?: string;
  pronouns?: string;
  notes?: string;
  language?: UserLocale;
  timezone?: UserLocale;
  location?: UserLocale;
  extra?: UserExtraField[];
}

export interface ModelConfig {
  inherit?: boolean;
  primary?: string;
  fallbacks?: string[];
  stream?: boolean;
  temperature?: number;
  maxTokens?: number;
}

export interface AssistantModel {
  agent: ModelConfig;
  image: ModelConfig;
  embedding: ModelConfig;
}

export interface AssistantToolItem {
  id?: string;
  enabled: boolean;
}

export interface AssistantTools {
  items?: AssistantToolItem[];
}

export interface AssistantSkills {
  mode?: "on" | "off" | string;
  maxSkillsInPrompt?: number;
  maxPromptChars?: number;
}

export interface CallToolsConfig {
  mode?: AssistantCallMode | string;
  allowList?: string[];
  denyList?: string[];
}

export interface CallSkillsConfig {
  mode?: AssistantCallMode | string;
  allowList?: string[];
}

export interface AssistantCall {
  tools: CallToolsConfig;
  skills: CallSkillsConfig;
}

export interface AssistantMemory {
  enabled: boolean;
}

export interface AssistantReadiness {
  ready: boolean;
  missing?: string[];
}

export interface Assistant {
  id: string;
  builtin: boolean;
  deletable: boolean;
  identity: AssistantIdentity;
  avatar: AssistantAvatar;
  user: AssistantUser;
  model: AssistantModel;
  tools: AssistantTools;
  skills: AssistantSkills;
  call: AssistantCall;
  memory: AssistantMemory;
  readiness: AssistantReadiness;
  enabled: boolean;
  isDefault: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Assistant3DAvatarAsset {
  kind: "3davatar" | "vrma" | string;
  path: string;
  name: string;
  displayName?: string;
  updatedAt?: string;
  source?: string;
  assetId?: string;
}

export interface ImportAssistant3DAvatarFromPathRequest {
  kind: "3davatar" | "vrma" | string;
  path: string;
}

export interface ReadAssistant3DAvatarSourceRequest {
  kind: "3davatar" | "vrma" | string;
  path: string;
}

export interface ReadAssistant3DAvatarSourceResponse {
  contentBase64: string;
  mime: string;
  fileName: string;
  sizeBytes: number;
}

export interface DeleteAssistantAvatarAssetRequest {
  kind: "3davatar" | "vrma" | string;
  path: string;
}

export interface UpdateAssistantAvatarAssetRequest {
  kind: "3davatar" | "vrma" | string;
  path: string;
  displayName?: string;
}

export interface CreateAssistantRequest {
  identity: AssistantIdentity;
  avatar: AssistantAvatar;
  user: AssistantUser;
  model: AssistantModel;
  tools: AssistantTools;
  skills: AssistantSkills;
  call: AssistantCall;
  memory: AssistantMemory;
  enabled?: boolean;
  isDefault?: boolean;
}

export interface UpdateAssistantRequest {
  id: string;
  identity?: AssistantIdentity;
  avatar?: AssistantAvatar;
  user?: AssistantUser;
  model?: AssistantModel;
  tools?: AssistantTools;
  skills?: AssistantSkills;
  call?: AssistantCall;
  memory?: AssistantMemory;
  enabled?: boolean;
  isDefault?: boolean;
}

export interface DeleteAssistantRequest {
  id: string;
}

export interface SetDefaultAssistantRequest {
  id: string;
}

export interface AssistantMemorySummary {
  summary: string;
}

export interface AssistantProfileOptions {
  roles: string[];
  defaultRole?: string;
  vibes: string[];
  defaultVibe?: string;
}
