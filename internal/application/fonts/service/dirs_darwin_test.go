//go:build darwin

package service

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDarwinSystemFontAssetDirectories(t *testing.T) {
	base := t.TempDir()
	for _, dir := range []string{
		"com_apple_MobileAsset_Font8",
		"com_apple_MobileAsset_Font7",
	} {
		if err := os.Mkdir(filepath.Join(base, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.Mkdir(filepath.Join(base, "com_apple_MobileAsset_Other"), 0o755); err != nil {
		t.Fatalf("mkdir other asset dir: %v", err)
	}

	got := darwinSystemFontAssetDirectories(base)
	want := []string{
		filepath.Join(base, "com_apple_MobileAsset_Font7"),
		filepath.Join(base, "com_apple_MobileAsset_Font8"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected font asset directories %v, got %v", want, got)
	}
}
