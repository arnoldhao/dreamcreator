package dto

type ProviderSkillSpec struct {
	ID          string `json:"id"`
	ProviderID  string `json:"providerId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
	SourceID    string `json:"sourceId,omitempty"`
	SourceName  string `json:"sourceName,omitempty"`
	SourceKind  string `json:"sourceKind,omitempty"`
	SourceType  string `json:"sourceType,omitempty"`
	SourcePath  string `json:"sourcePath,omitempty"`
}

type RegisterSkillRequest struct {
	Spec ProviderSkillSpec `json:"spec"`
}

type EnableSkillRequest struct {
	ID      string `json:"id"`
	Enabled bool   `json:"enabled"`
}

type DeleteSkillRequest struct {
	ID string `json:"id"`
}

type SearchSkillsRequest struct {
	Query         string `json:"query"`
	Limit         int    `json:"limit"`
	AssistantID   string `json:"assistantId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type SkillSearchResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Source      string `json:"source"`
}

type InspectSkillRequest struct {
	Skill         string `json:"skill"`
	AssistantID   string `json:"assistantId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type SkillDetailFile struct {
	Path        string `json:"path"`
	Size        int64  `json:"size,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

type SkillRuntimeInstallSpec struct {
	Kind    string   `json:"kind,omitempty"`
	ID      string   `json:"id,omitempty"`
	Label   string   `json:"label,omitempty"`
	Bins    []string `json:"bins,omitempty"`
	Formula string   `json:"formula,omitempty"`
	Tap     string   `json:"tap,omitempty"`
	Package string   `json:"package,omitempty"`
	Module  string   `json:"module,omitempty"`
}

type SkillRuntimeRequirements struct {
	PrimaryEnv string                    `json:"primaryEnv,omitempty"`
	Homepage   string                    `json:"homepage,omitempty"`
	OS         []string                  `json:"os,omitempty"`
	Bins       []string                  `json:"bins,omitempty"`
	AnyBins    []string                  `json:"anyBins,omitempty"`
	Env        []string                  `json:"env,omitempty"`
	Config     []string                  `json:"config,omitempty"`
	Install    []SkillRuntimeInstallSpec `json:"install,omitempty"`
	Nix        string                    `json:"nix,omitempty"`
}

type SkillDetail struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name"`
	Summary         string                    `json:"summary,omitempty"`
	URL             string                    `json:"url,omitempty"`
	Owner           string                    `json:"owner,omitempty"`
	CurrentVersion  string                    `json:"currentVersion,omitempty"`
	LatestVersion   string                    `json:"latestVersion,omitempty"`
	SelectedVersion string                    `json:"selectedVersion,omitempty"`
	Tags            []string                  `json:"tags,omitempty"`
	CreatedAt       int64                     `json:"createdAt,omitempty"`
	UpdatedAt       int64                     `json:"updatedAt,omitempty"`
	Changelog       string                    `json:"changelog,omitempty"`
	Files           []SkillDetailFile         `json:"files,omitempty"`
	SkillMarkdown   string                    `json:"skillMarkdown,omitempty"`
	Runtime         *SkillRuntimeRequirements `json:"runtimeRequirements,omitempty"`
}

type SyncSkillsRequest struct {
	ProviderID    string `json:"providerId"`
	AssistantID   string `json:"assistantId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type ResolveSkillsRequest struct {
	ProviderID string `json:"providerId"`
}

type InstallSkillRequest struct {
	Skill         string `json:"skill"`
	Version       string `json:"version,omitempty"`
	Force         bool   `json:"force,omitempty"`
	AssistantID   string `json:"assistantId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type UpdateSkillRequest struct {
	Skill         string `json:"skill"`
	Version       string `json:"version,omitempty"`
	Force         bool   `json:"force,omitempty"`
	AssistantID   string `json:"assistantId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type RemoveSkillRequest struct {
	Skill         string `json:"skill"`
	AssistantID   string `json:"assistantId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type SkillsStatusRequest struct {
	ProviderID    string `json:"providerId,omitempty"`
	AssistantID   string `json:"assistantId,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
}

type SkillsStatus struct {
	ClawhubReady  bool   `json:"clawhubReady"`
	Reason        string `json:"reason,omitempty"`
	WorkspaceRoot string `json:"workspaceRoot,omitempty"`
	CatalogCount  int    `json:"catalogCount"`
}

type SkillSpec struct {
	ID          string `json:"id"`
	ProviderID  string `json:"providerId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
}

type SkillPromptItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"path,omitempty"`
}

type ResolveSkillPromptRequest struct {
	ProviderID string `json:"providerId"`
}

type ResolveSkillPromptResponse struct {
	Items []SkillPromptItem `json:"items"`
}
