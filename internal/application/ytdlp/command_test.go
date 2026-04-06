package ytdlp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/externaltools"
)

type toolResolverStub struct {
	paths map[externaltools.ToolName]string
}

func (stub toolResolverStub) ResolveExecPath(_ context.Context, name externaltools.ToolName) (string, error) {
	if execPath, ok := stub.paths[name]; ok {
		return execPath, nil
	}
	return "", fmt.Errorf("%s not found", name)
}

func TestBuildExplicitToolArgsUsesConfiguredFFmpegAndBunPaths(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ffmpegPath := filepath.Join(tempDir, "ffmpeg", "ffmpeg")
	bunPath := filepath.Join(tempDir, "bun", "bun")
	if err := os.MkdirAll(filepath.Dir(ffmpegPath), 0o755); err != nil {
		t.Fatalf("mkdir ffmpeg dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bunPath), 0o755); err != nil {
		t.Fatalf("mkdir bun dir: %v", err)
	}
	if err := os.WriteFile(ffmpegPath, []byte(""), 0o755); err != nil {
		t.Fatalf("write ffmpeg file: %v", err)
	}
	if err := os.WriteFile(bunPath, []byte(""), 0o755); err != nil {
		t.Fatalf("write bun file: %v", err)
	}

	args := BuildExplicitToolArgs(context.Background(), toolResolverStub{
		paths: map[externaltools.ToolName]string{
			externaltools.ToolFFmpeg: ffmpegPath,
			externaltools.ToolBun:    bunPath,
		},
	})

	if len(args) != 5 {
		t.Fatalf("expected 5 explicit tool args, got %d: %v", len(args), args)
	}
	if args[0] != "--ffmpeg-location" || args[1] != filepath.Dir(ffmpegPath) {
		t.Fatalf("unexpected ffmpeg args: %v", args[:2])
	}
	if args[2] != "--no-js-runtimes" || args[3] != "--js-runtimes" || args[4] != "bun:"+bunPath {
		t.Fatalf("unexpected js runtime args: %v", args[2:])
	}
}

func TestBuildCommandUsesExplicitToolArgsWithoutMutatingPATH(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")

	tempDir := t.TempDir()
	ffmpegPath := filepath.Join(tempDir, "ffmpeg", "ffmpeg")
	bunPath := filepath.Join(tempDir, "bun", "bun")
	if err := os.MkdirAll(filepath.Dir(ffmpegPath), 0o755); err != nil {
		t.Fatalf("mkdir ffmpeg dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bunPath), 0o755); err != nil {
		t.Fatalf("mkdir bun dir: %v", err)
	}
	if err := os.WriteFile(ffmpegPath, []byte(""), 0o755); err != nil {
		t.Fatalf("write ffmpeg file: %v", err)
	}
	if err := os.WriteFile(bunPath, []byte(""), 0o755); err != nil {
		t.Fatalf("write bun file: %v", err)
	}

	command, err := BuildCommand(context.Background(), CommandOptions{
		ExecPath: filepath.Join(tempDir, "yt-dlp"),
		Tools: toolResolverStub{
			paths: map[externaltools.ToolName]string{
				externaltools.ToolFFmpeg: ffmpegPath,
				externaltools.ToolBun:    bunPath,
			},
		},
		Request: dto.CreateYTDLPJobRequest{
			URL:            "https://example.com/watch?v=1",
			WriteThumbnail: true,
			SubtitleLangs:  []string{"en"},
		},
		OutputTemplate: filepath.Join(tempDir, "downloads", "%(title)s.%(ext)s"),
	})
	if err != nil {
		t.Fatalf("build command: %v", err)
	}
	defer command.Cancel()
	if command.Cleanup != nil {
		defer command.Cleanup()
	}

	argsJoined := strings.Join(command.Args, "\n")
	if !strings.Contains(argsJoined, "--ffmpeg-location\n"+filepath.Dir(ffmpegPath)) {
		t.Fatalf("expected explicit ffmpeg args, got %v", command.Args)
	}
	if !strings.Contains(argsJoined, "--no-js-runtimes") || !strings.Contains(argsJoined, "--js-runtimes\nbun:"+bunPath) {
		t.Fatalf("expected explicit bun runtime args, got %v", command.Args)
	}
	if strings.Contains(argsJoined, "--write-thumbnail") {
		t.Fatalf("expected primary command to omit thumbnail args, got %v", command.Args)
	}
	if strings.Contains(argsJoined, "--write-subs") || strings.Contains(argsJoined, "--write-auto-subs") {
		t.Fatalf("expected primary command to omit subtitle args, got %v", command.Args)
	}

	pathEntry := ""
	for _, entry := range command.Cmd.Env {
		if strings.HasPrefix(entry, "PATH=") {
			pathEntry = entry
			break
		}
	}
	if pathEntry != "PATH=/usr/bin" {
		t.Fatalf("expected PATH to remain unchanged, got %q", pathEntry)
	}
}

func TestBuildSubtitleCommandUsesSubtitleArgsWithoutMutatingPATH(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")

	tempDir := t.TempDir()
	ffmpegPath := filepath.Join(tempDir, "ffmpeg", "ffmpeg")
	bunPath := filepath.Join(tempDir, "bun", "bun")
	if err := os.MkdirAll(filepath.Dir(ffmpegPath), 0o755); err != nil {
		t.Fatalf("mkdir ffmpeg dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(bunPath), 0o755); err != nil {
		t.Fatalf("mkdir bun dir: %v", err)
	}
	if err := os.WriteFile(ffmpegPath, []byte(""), 0o755); err != nil {
		t.Fatalf("write ffmpeg file: %v", err)
	}
	if err := os.WriteFile(bunPath, []byte(""), 0o755); err != nil {
		t.Fatalf("write bun file: %v", err)
	}

	command, err := BuildSubtitleCommand(context.Background(), CommandOptions{
		ExecPath: filepath.Join(tempDir, "yt-dlp"),
		Tools: toolResolverStub{
			paths: map[externaltools.ToolName]string{
				externaltools.ToolFFmpeg: ffmpegPath,
				externaltools.ToolBun:    bunPath,
			},
		},
		Request: dto.CreateYTDLPJobRequest{
			URL:            "https://example.com/watch?v=1",
			SubtitleLangs:  []string{"en", "ja"},
			SubtitleAuto:   true,
			SubtitleFormat: "vtt",
		},
		OutputTemplate:   filepath.Join(tempDir, "downloads", "%(title)s.%(ext)s"),
		SubtitleTemplate: filepath.Join(tempDir, "downloads", "subs", "%(title)s.%(ext)s"),
	})
	if err != nil {
		t.Fatalf("build subtitle command: %v", err)
	}
	defer command.Cancel()

	argsJoined := strings.Join(command.Args, "\n")
	for _, expected := range []string{
		"--skip-download",
		"--write-auto-subs",
		"--sub-langs\nen,ja",
		"--sub-format\nvtt",
		"-o\nsubtitle:" + filepath.Join(tempDir, "downloads", "subs", "%(title)s.%(ext)s"),
		"--ffmpeg-location\n" + filepath.Dir(ffmpegPath),
		"--no-js-runtimes",
		"--js-runtimes\nbun:" + bunPath,
	} {
		if !strings.Contains(argsJoined, expected) {
			t.Fatalf("expected subtitle command args to contain %q, got %v", expected, command.Args)
		}
	}

	pathEntry := ""
	for _, entry := range command.Cmd.Env {
		if strings.HasPrefix(entry, "PATH=") {
			pathEntry = entry
			break
		}
	}
	if pathEntry != "PATH=/usr/bin" {
		t.Fatalf("expected PATH to remain unchanged, got %q", pathEntry)
	}
	if command.PrintFilePath != "" {
		t.Fatalf("expected subtitle command not to allocate print file, got %q", command.PrintFilePath)
	}
}
