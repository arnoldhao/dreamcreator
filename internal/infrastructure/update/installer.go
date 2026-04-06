package update

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var ErrPreparedUpdateNotFound = fmt.Errorf("prepared update not found")

type PlatformInstaller struct {
	stateDir       string
	planPath       string
	goos           string
	goarch         string
	executablePath func() (string, error)
	startDetached  func(name string, args []string) error
}

type stagedPlan struct {
	Platform     string `json:"platform"`
	Mode         string `json:"mode"`
	StageDir     string `json:"stageDir"`
	SourcePath   string `json:"sourcePath"`
	TargetPath   string `json:"targetPath"`
	RelaunchPath string `json:"relaunchPath"`
	InstallDir   string `json:"installDir,omitempty"`
}

func NewInstaller(statePath string) (*PlatformInstaller, error) {
	stateDir := strings.TrimSpace(filepath.Dir(statePath))
	if stateDir == "" || stateDir == "." {
		configDir, err := os.UserConfigDir()
		if err != nil {
			stateDir = filepath.Join(os.TempDir(), "dreamcreator")
		} else {
			stateDir = filepath.Join(configDir, "dreamcreator")
		}
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, err
	}
	return &PlatformInstaller{
		stateDir:       stateDir,
		planPath:       filepath.Join(stateDir, "update_install_plan.json"),
		goos:           runtime.GOOS,
		goarch:         runtime.GOARCH,
		executablePath: os.Executable,
		startDetached:  startDetachedCommand,
	}, nil
}

func (installer *PlatformInstaller) Install(ctx context.Context, artifactPath string) error {
	if installer == nil {
		return fmt.Errorf("installer not configured")
	}
	normalizedArtifact := strings.TrimSpace(artifactPath)
	if normalizedArtifact == "" {
		return fmt.Errorf("artifact path is empty")
	}
	if err := installer.cleanupStagedUpdate(); err != nil {
		return err
	}

	switch installer.goos {
	case "windows":
		return installer.prepareWindowsUpdate(normalizedArtifact)
	case "darwin":
		return installer.prepareMacUpdate(ctx, normalizedArtifact)
	default:
		return fmt.Errorf("update install is not supported on %s", installer.goos)
	}
}

func (installer *PlatformInstaller) RestartToApply(_ context.Context) error {
	if installer == nil {
		return fmt.Errorf("installer not configured")
	}
	plan, err := installer.loadPlan()
	if err != nil {
		return err
	}

	switch plan.Platform {
	case "windows":
		return installer.restartWindows(plan)
	case "darwin":
		return installer.restartDarwin(plan)
	default:
		return fmt.Errorf("unsupported update platform %q", plan.Platform)
	}
}

func (installer *PlatformInstaller) prepareWindowsUpdate(artifactPath string) error {
	currentExe, err := installer.currentExecutable()
	if err != nil {
		return err
	}

	stageDir, err := installer.newStageDir()
	if err != nil {
		return err
	}

	artifactName := filepath.Base(artifactPath)
	plan := stagedPlan{
		Platform:     "windows",
		StageDir:     stageDir,
		TargetPath:   currentExe,
		RelaunchPath: currentExe,
		InstallDir:   filepath.Dir(currentExe),
	}

	switch strings.ToLower(filepath.Ext(artifactName)) {
	case ".exe":
		stagedInstaller := filepath.Join(stageDir, artifactName)
		if err := copyFile(artifactPath, stagedInstaller); err != nil {
			return err
		}
		plan.Mode = "installer"
		plan.SourcePath = stagedInstaller
	case ".zip":
		execName := filepath.Base(currentExe)
		stagedExe, err := extractZipExecutable(artifactPath, filepath.Join(stageDir, "portable"), execName)
		if err != nil {
			return err
		}
		plan.Mode = "portable"
		plan.SourcePath = stagedExe
	default:
		return fmt.Errorf("unsupported windows update artifact %q", artifactName)
	}

	return installer.savePlan(plan)
}

func (installer *PlatformInstaller) prepareMacUpdate(ctx context.Context, artifactPath string) error {
	currentExe, err := installer.currentExecutable()
	if err != nil {
		return err
	}
	targetBundle, err := resolveAppBundle(currentExe)
	if err != nil {
		return err
	}

	stageDir, err := installer.newStageDir()
	if err != nil {
		return err
	}
	extractDir := filepath.Join(stageDir, "bundle")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return err
	}
	if err := extractMacArchive(ctx, artifactPath, extractDir); err != nil {
		return err
	}
	stagedBundle, err := findFirstAppBundle(extractDir)
	if err != nil {
		return err
	}
	_ = removeMacQuarantine(stagedBundle)

	return installer.savePlan(stagedPlan{
		Platform:     "darwin",
		Mode:         "bundle",
		StageDir:     stageDir,
		SourcePath:   stagedBundle,
		TargetPath:   targetBundle,
		RelaunchPath: targetBundle,
	})
}

func (installer *PlatformInstaller) restartWindows(plan stagedPlan) error {
	scriptPath := filepath.Join(installer.stateDir, "apply_update.ps1")
	if err := os.WriteFile(scriptPath, []byte(windowsApplyScript), 0o600); err != nil {
		return err
	}

	args := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		strconv.Itoa(os.Getpid()),
		plan.Mode,
		plan.SourcePath,
		plan.TargetPath,
		plan.InstallDir,
		plan.StageDir,
		installer.planPath,
	}
	return installer.startDetached("powershell.exe", args)
}

func (installer *PlatformInstaller) restartDarwin(plan stagedPlan) error {
	scriptPath := filepath.Join(installer.stateDir, "apply_update.sh")
	if err := os.WriteFile(scriptPath, []byte(darwinApplyScript), 0o700); err != nil {
		return err
	}

	args := []string{
		scriptPath,
		strconv.Itoa(os.Getpid()),
		plan.SourcePath,
		plan.TargetPath,
		plan.StageDir,
		installer.planPath,
	}
	return installer.startDetached("/bin/sh", args)
}

func (installer *PlatformInstaller) currentExecutable() (string, error) {
	if installer.executablePath == nil {
		return "", fmt.Errorf("executable path resolver not configured")
	}
	path, err := installer.executablePath()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("executable path is empty")
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil && strings.TrimSpace(resolved) != "" {
		path = resolved
	}
	return path, nil
}

func (installer *PlatformInstaller) newStageDir() (string, error) {
	stageRoot := filepath.Join(installer.stateDir, "update-stage")
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		return "", err
	}
	return os.MkdirTemp(stageRoot, "prepared-*")
}

func (installer *PlatformInstaller) cleanupStagedUpdate() error {
	plan, err := installer.loadPlan()
	if err != nil {
		if errors.Is(err, ErrPreparedUpdateNotFound) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(plan.StageDir) != "" {
		_ = os.RemoveAll(plan.StageDir)
	}
	_ = os.Remove(installer.planPath)
	return nil
}

func (installer *PlatformInstaller) loadPlan() (stagedPlan, error) {
	data, err := os.ReadFile(installer.planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return stagedPlan{}, ErrPreparedUpdateNotFound
		}
		return stagedPlan{}, err
	}
	var plan stagedPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return stagedPlan{}, err
	}
	if strings.TrimSpace(plan.SourcePath) == "" || strings.TrimSpace(plan.TargetPath) == "" {
		return stagedPlan{}, fmt.Errorf("prepared update plan is incomplete")
	}
	return plan, nil
}

func (installer *PlatformInstaller) savePlan(plan stagedPlan) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(installer.planPath, data, 0o600)
}

func resolveAppBundle(executablePath string) (string, error) {
	current := filepath.Dir(executablePath)
	for {
		if strings.HasSuffix(strings.ToLower(current), ".app") {
			return current, nil
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}
	return "", fmt.Errorf("mac app bundle not found for executable %q", executablePath)
}

func findFirstAppBundle(root string) (string, error) {
	var match string
	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".app") {
			match = path
			return fs.SkipDir
		}
		return nil
	})
	if walkErr != nil {
		return "", walkErr
	}
	if match == "" {
		return "", fmt.Errorf("no .app bundle found in %q", root)
	}
	return match, nil
}

func extractMacArchive(ctx context.Context, archivePath string, destDir string) error {
	if !strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return fmt.Errorf("unsupported mac update artifact %q", archivePath)
	}
	cmd := exec.CommandContext(ctx, "ditto", "-x", "-k", archivePath, destDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return err
		}
		return fmt.Errorf("extract mac archive: %s", message)
	}
	return nil
}

func removeMacQuarantine(path string) error {
	cmd := exec.Command("xattr", "-dr", "com.apple.quarantine", path)
	return cmd.Run()
}

func extractZipExecutable(archivePath, destDir, execName string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var candidate string
	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return "", err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		src, err := file.Open()
		if err != nil {
			return "", err
		}
		dst, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			_ = src.Close()
			return "", err
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return "", err
		}
		_ = dst.Close()
		_ = src.Close()
		if strings.EqualFold(filepath.Base(path), execName) {
			candidate = path
		}
	}

	if candidate == "" {
		return "", fmt.Errorf("executable %s not found in archive", execName)
	}
	return candidate, nil
}

func copyFile(src string, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

const windowsApplyScript = `param(
  [Parameter(Mandatory = $true)][int]$ParentPid,
  [Parameter(Mandatory = $true)][string]$Mode,
  [Parameter(Mandatory = $true)][string]$SourcePath,
  [Parameter(Mandatory = $true)][string]$TargetPath,
  [Parameter(Mandatory = $true)][string]$InstallDir,
  [Parameter(Mandatory = $true)][string]$StageDir,
  [Parameter(Mandatory = $true)][string]$PlanPath
)

$ErrorActionPreference = "Stop"

for ($i = 0; $i -lt 480; $i++) {
  $proc = Get-Process -Id $ParentPid -ErrorAction SilentlyContinue
  if (-not $proc) {
    break
  }
  Start-Sleep -Milliseconds 250
}

switch ($Mode) {
  "installer" {
    $result = Start-Process -FilePath $SourcePath -ArgumentList @("/S", "/D=" + $InstallDir) -Verb RunAs -Wait -PassThru
    if ($result.ExitCode -ne 0) {
      throw ("installer exited with code " + $result.ExitCode)
    }
  }
  "portable" {
    Copy-Item $SourcePath $TargetPath -Force
  }
  default {
    throw ("unsupported update mode: " + $Mode)
  }
}

Start-Process -FilePath $TargetPath -WorkingDirectory $InstallDir | Out-Null
Remove-Item $PlanPath -Force -ErrorAction SilentlyContinue
Remove-Item $StageDir -Recurse -Force -ErrorAction SilentlyContinue
`

const darwinApplyScript = `#!/bin/sh
set -eu

PARENT_PID="$1"
SOURCE_APP="$2"
TARGET_APP="$3"
STAGE_DIR="$4"
PLAN_PATH="$5"
BACKUP_APP="${TARGET_APP}.old"

while kill -0 "$PARENT_PID" 2>/dev/null; do
  sleep 0.25
done

install_direct() {
  mkdir -p "$(dirname "$TARGET_APP")"
  rm -rf "$BACKUP_APP"
  if [ -d "$TARGET_APP" ]; then
    mv "$TARGET_APP" "$BACKUP_APP"
  fi
  if /usr/bin/ditto "$SOURCE_APP" "$TARGET_APP"; then
    return 0
  fi
  rm -rf "$TARGET_APP"
  if [ -d "$BACKUP_APP" ]; then
    mv "$BACKUP_APP" "$TARGET_APP"
  fi
  return 1
}

install_privileged() {
  /usr/bin/osascript - "$SOURCE_APP" "$TARGET_APP" "$BACKUP_APP" <<'APPLESCRIPT'
on run argv
  set sourceApp to item 1 of argv
  set targetApp to item 2 of argv
  set backupApp to item 3 of argv
  set commandText to "set -e; rm -rf " & quoted form of backupApp & "; " & ¬
    "if [ -d " & quoted form of targetApp & " ]; then mv " & quoted form of targetApp & " " & quoted form of backupApp & "; fi; " & ¬
    "if /usr/bin/ditto " & quoted form of sourceApp & " " & quoted form of targetApp & "; then " & ¬
    "/usr/bin/xattr -dr com.apple.quarantine " & quoted form of targetApp & " >/dev/null 2>&1 || true; " & ¬
    "rm -rf " & quoted form of backupApp & "; " & ¬
    "else rm -rf " & quoted form of targetApp & "; if [ -d " & quoted form of backupApp & " ]; then mv " & quoted form of backupApp & " " & quoted form of targetApp & "; fi; exit 1; fi"
  do shell script commandText with administrator privileges
end run
APPLESCRIPT
}

if ! install_direct; then
  install_privileged
fi

/usr/bin/xattr -dr com.apple.quarantine "$TARGET_APP" >/dev/null 2>&1 || true
open "$TARGET_APP"
rm -rf "$BACKUP_APP"
rm -f "$PLAN_PATH"
rm -rf "$STAGE_DIR"
`

var _ interface {
	Install(context.Context, string) error
	RestartToApply(context.Context) error
} = (*PlatformInstaller)(nil)
