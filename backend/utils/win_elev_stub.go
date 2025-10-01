//go:build !windows

package utils

// WindowsIsElevated is a stub for non-Windows platforms.
func WindowsIsElevated() bool { return false }
