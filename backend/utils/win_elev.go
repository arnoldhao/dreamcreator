//go:build windows

package utils

import (
    "golang.org/x/sys/windows"
)

// WindowsIsElevated returns true if the current process token is elevated (running as administrator).
func WindowsIsElevated() bool {
    token, err := windows.OpenCurrentProcessToken()
    if err != nil { return false }
    defer token.Close()
    return token.IsElevated()
}
