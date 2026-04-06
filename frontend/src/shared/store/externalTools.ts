export type ExternalToolStatus = "missing" | "installed" | "invalid";
export type ExternalToolKind = "bin" | "runtime";

export interface ExternalTool {
  name: string;
  kind?: ExternalToolKind | string;
  execPath?: string;
  version?: string;
  status?: ExternalToolStatus | string;
  sourceKind?: string;
  sourceRef?: string;
  manager?: string;
  installedAt?: string;
  updatedAt?: string;
}

export interface ExternalToolUpdateInfo {
  name: string;
  latestVersion?: string;
  recommendedVersion?: string;
  upstreamVersion?: string;
  releaseNotes?: string;
  releaseNotesUrl?: string;
  autoUpdate?: boolean;
  required?: boolean;
}

export interface InstallExternalToolRequest {
  name: string;
  version?: string;
}

export interface SetExternalToolPathRequest {
  name: string;
  execPath: string;
}

export interface VerifyExternalToolRequest {
  name: string;
}

export interface RemoveExternalToolRequest {
  name: string;
}

export interface OpenExternalToolDirectoryRequest {
  name: string;
}

export interface ExternalToolInstallState {
  name: string;
  stage?: string;
  progress?: number;
  message?: string;
  updatedAt?: string;
}

export interface GetExternalToolInstallStateRequest {
  name: string;
}
