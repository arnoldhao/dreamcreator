package providers

import (
	"os"
	"path/filepath"
)

// persistentDepsRoot returns a per-user persistent directory to store dependency binaries.
// Example:
// - macOS: ~/Library/Application Support/dreamcreator/deps
// - Windows: %AppData%/dreamcreator/deps
// - Linux: ~/.config/dreamcreator/deps
func persistentDepsRoot() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		// fallback to temp dir if config dir is not available
		base = os.TempDir()
	}
	root := filepath.Join(base, "dreamcreator", "deps")
	_ = os.MkdirAll(root, 0o755)
	return root
}
