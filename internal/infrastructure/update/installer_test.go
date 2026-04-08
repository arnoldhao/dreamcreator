package update

import (
	"path/filepath"
	"testing"
)

func TestResolveMacTargetBundleKeepsApplicationsInstall(t *testing.T) {
	t.Parallel()

	currentBundle := "/Applications/dreamcreator.app"
	targetBundle := resolveMacTargetBundle(currentBundle)
	if targetBundle != currentBundle {
		t.Fatalf("expected applications bundle to stay in place, got %q", targetBundle)
	}
}

func TestResolveMacTargetBundleMovesNonApplicationsInstall(t *testing.T) {
	t.Parallel()

	currentBundle := "/Users/test/Downloads/dreamcreator.app"
	targetBundle := resolveMacTargetBundle(currentBundle)
	if targetBundle != "/Applications/dreamcreator.app" {
		t.Fatalf("expected non-applications bundle to move to /Applications, got %q", targetBundle)
	}
}

func TestRestartDarwinUsesExplicitRelaunchAndFallbackPaths(t *testing.T) {
	t.Parallel()

	var (
		capturedName string
		capturedArgs []string
	)
	stateDir := t.TempDir()
	installer := &PlatformInstaller{
		stateDir: stateDir,
		planPath: filepath.Join(stateDir, "update_install_plan.json"),
		startDetached: func(name string, args []string) error {
			capturedName = name
			capturedArgs = append([]string(nil), args...)
			return nil
		},
	}

	plan := stagedPlan{
		SourcePath:   "/tmp/source.app",
		TargetPath:   "/Applications/dreamcreator.app",
		RelaunchPath: "/Applications/dreamcreator.app",
		FallbackPath: "/Users/test/bin/dreamcreator.app",
		StageDir:     "/tmp/stage",
	}
	if err := installer.restartDarwin(plan); err != nil {
		t.Fatalf("restartDarwin failed: %v", err)
	}

	if capturedName != "/bin/sh" {
		t.Fatalf("unexpected restart helper: %q", capturedName)
	}
	if len(capturedArgs) != 8 {
		t.Fatalf("unexpected helper args: %#v", capturedArgs)
	}
	if capturedArgs[4] != plan.RelaunchPath {
		t.Fatalf("expected relaunch path %q, got %q", plan.RelaunchPath, capturedArgs[4])
	}
	if capturedArgs[5] != plan.FallbackPath {
		t.Fatalf("expected fallback path %q, got %q", plan.FallbackPath, capturedArgs[5])
	}
}

func TestRestartDarwinDefaultsRelaunchAndFallbackToTarget(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	stateDir := t.TempDir()
	installer := &PlatformInstaller{
		stateDir: stateDir,
		planPath: filepath.Join(stateDir, "update_install_plan.json"),
		startDetached: func(_ string, args []string) error {
			capturedArgs = append([]string(nil), args...)
			return nil
		},
	}

	plan := stagedPlan{
		SourcePath: "/tmp/source.app",
		TargetPath: "/Applications/dreamcreator.app",
		StageDir:   "/tmp/stage",
	}
	if err := installer.restartDarwin(plan); err != nil {
		t.Fatalf("restartDarwin failed: %v", err)
	}

	if len(capturedArgs) != 8 {
		t.Fatalf("unexpected helper args: %#v", capturedArgs)
	}
	if capturedArgs[4] != plan.TargetPath {
		t.Fatalf("expected default relaunch path %q, got %q", plan.TargetPath, capturedArgs[4])
	}
	if capturedArgs[5] != plan.TargetPath {
		t.Fatalf("expected default fallback path %q, got %q", plan.TargetPath, capturedArgs[5])
	}
}
