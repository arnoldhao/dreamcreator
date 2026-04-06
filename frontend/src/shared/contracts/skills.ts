export interface ProviderSkillSpec {
  id: string;
  providerId: string;
  name: string;
  description: string;
  version: string;
  enabled: boolean;
  sourceId?: string;
  sourceName?: string;
  sourceKind?: string;
  sourceType?: string;
  sourcePath?: string;
}

export interface ResolveSkillsRequest {
  providerId?: string;
}

export interface SkillsStatusRequest {
  providerId?: string;
  assistantId?: string;
  workspaceRoot?: string;
}

export interface SkillsStatus {
  clawhubReady: boolean;
  reason?: string;
  workspaceRoot?: string;
  catalogCount: number;
}

export interface DeleteSkillRequest {
  id: string;
}

export interface SearchSkillsRequest {
  query: string;
  limit?: number;
  assistantId?: string;
  workspaceRoot?: string;
}

export interface InstallSkillRequest {
  skill: string;
  version?: string;
  force?: boolean;
  assistantId?: string;
  workspaceRoot?: string;
}

export interface UpdateSkillRequest {
  skill: string;
  version?: string;
  force?: boolean;
  assistantId?: string;
  workspaceRoot?: string;
}

export interface SyncSkillsRequest {
  providerId?: string;
  assistantId?: string;
  workspaceRoot?: string;
}

export interface RemoveSkillRequest {
  skill: string;
  assistantId?: string;
  workspaceRoot?: string;
}

export interface SkillSearchResult {
  id: string;
  name: string;
  description: string;
  url: string;
  source: string;
}

export interface InspectSkillRequest {
  skill: string;
  assistantId?: string;
  workspaceRoot?: string;
}

export interface SkillDetailFile {
  path: string;
  size?: number;
  sha256?: string;
  contentType?: string;
}

export interface SkillRuntimeInstallSpec {
  kind?: string;
  id?: string;
  label?: string;
  bins?: string[];
  formula?: string;
  tap?: string;
  package?: string;
  module?: string;
}

export interface SkillRuntimeRequirements {
  primaryEnv?: string;
  homepage?: string;
  os?: string[];
  bins?: string[];
  anyBins?: string[];
  env?: string[];
  config?: string[];
  install?: SkillRuntimeInstallSpec[];
  nix?: string;
}

export interface SkillDetail {
  id: string;
  name: string;
  summary?: string;
  url?: string;
  owner?: string;
  currentVersion?: string;
  latestVersion?: string;
  selectedVersion?: string;
  tags?: string[];
  createdAt?: number;
  updatedAt?: number;
  changelog?: string;
  files?: SkillDetailFile[];
  skillMarkdown?: string;
  runtimeRequirements?: SkillRuntimeRequirements;
}
