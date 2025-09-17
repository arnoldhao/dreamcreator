package providers

import (
    "os"
    "path/filepath"
)

// persistentDepsRoot returns a per-user persistent directory to store dependency binaries.
// Example:
// - macOS: ~/Library/Application Support/CanMe/deps
// - Windows: %AppData%/CanMe/deps
// - Linux: ~/.config/CanMe/deps
func persistentDepsRoot() string {
    base, err := os.UserConfigDir()
    if err != nil || base == "" {
        // fallback to temp dir if config dir is not available
        base = os.TempDir()
    }
    root := filepath.Join(base, "CanMe", "deps")
    _ = os.MkdirAll(root, 0o755)
    return root
}

