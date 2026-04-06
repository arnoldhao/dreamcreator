//go:build !darwin && !windows && !linux

package service

func platformFontDirectories(_ string) []string {
	return []string{}
}
