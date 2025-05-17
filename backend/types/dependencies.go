package types

type SoftwareInfo struct {
	Available     bool   `json:"available"`
	Path          string `json:"path"`
	ExecPath      string `json:"execPath"`
	Version       string `json:"version"`
	LatestVersion string `json:"latestVersion"`
	NeedUpdate    bool   `json:"needUpdate"`
}
