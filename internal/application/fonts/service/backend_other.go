//go:build !windows

package service

import "context"

func augmentPlatformFontCatalog(_ context.Context, _ *fontCatalog) error {
	return nil
}

func exportPlatformFontFamily(_ context.Context, _ []string) (ExportedFontFamily, bool, error) {
	return ExportedFontFamily{}, false, nil
}
