//go:build windows

package utils

import (
    "strconv"
    "golang.org/x/sys/windows/registry"
)

// WindowsSupportsAcrylic reports whether the current Windows build supports
// DWM SystemBackdropType (Windows 11 22621+), which enables Acrylic/Mica.
// This mirrors Wails' internal check (IsWindowsVersionAtLeast 10.0.22621).
func WindowsSupportsAcrylic() bool {
    key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
    if err != nil {
        return false
    }
    defer key.Close()

    // Try CurrentBuild first, then fallback to CurrentBuildNumber
    var buildStr string
    if v, _, err := key.GetStringValue("CurrentBuild"); err == nil && v != "" {
        buildStr = v
    } else if v2, _, err2 := key.GetStringValue("CurrentBuildNumber"); err2 == nil && v2 != "" {
        buildStr = v2
    } else {
        return false
    }

    build, err := strconv.Atoi(buildStr)
    if err != nil {
        return false
    }
    return build >= 22621
}

