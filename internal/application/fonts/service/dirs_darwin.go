//go:build darwin

package service

import "path/filepath"

func platformFontDirectories(home string) []string {
	return []string{
		"/System/Library/Fonts",
		"/Library/Fonts",
		filepath.Join(home, "Library", "Fonts"),
	}
}
