package dto

type ExternalTool struct {
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	ExecPath    string `json:"execPath"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	SourceKind  string `json:"sourceKind,omitempty"`
	SourceRef   string `json:"sourceRef,omitempty"`
	Manager     string `json:"manager,omitempty"`
	InstalledAt string `json:"installedAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type ExternalToolUpdateInfo struct {
	Name               string `json:"name"`
	LatestVersion      string `json:"latestVersion"`
	RecommendedVersion string `json:"recommendedVersion,omitempty"`
	UpstreamVersion    string `json:"upstreamVersion,omitempty"`
	ReleaseNotes       string `json:"releaseNotes"`
	ReleaseNotesURL    string `json:"releaseNotesUrl"`
	AutoUpdate         bool   `json:"autoUpdate,omitempty"`
	Required           bool   `json:"required,omitempty"`
}

type ExternalToolInstallState struct {
	Name      string `json:"name"`
	Stage     string `json:"stage"`
	Progress  int    `json:"progress"`
	Message   string `json:"message"`
	UpdatedAt string `json:"updatedAt"`
}

type InstallExternalToolRequest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Manager string `json:"manager,omitempty"`
}

type SetExternalToolPathRequest struct {
	Name     string `json:"name"`
	ExecPath string `json:"execPath"`
}

type VerifyExternalToolRequest struct {
	Name string `json:"name"`
}

type RemoveExternalToolRequest struct {
	Name string `json:"name"`
}

type OpenExternalToolDirectoryRequest struct {
	Name string `json:"name"`
}

type GetExternalToolInstallStateRequest struct {
	Name string `json:"name"`
}
