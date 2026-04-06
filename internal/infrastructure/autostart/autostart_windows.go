//go:build windows
// +build windows

package autostart

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

func setEnabled(appName string, _ string, execPath string, launchArg string, enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if !enabled {
		if err := key.DeleteValue(appName); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}

	value := fmt.Sprintf("\"%s\" %s", execPath, launchArg)
	return key.SetStringValue(appName, value)
}
