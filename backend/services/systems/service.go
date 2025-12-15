package systems

import (
	"context"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/services/preferences"
	"dreamcreator/backend/types"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type Service struct {
	ctx        context.Context
	appVersion string
}

var system *Service
var onceSystem sync.Once

func New() *Service {
	if system == nil {
		onceSystem.Do(func() {
			system = &Service{
				appVersion: "0.0.0",
			}
			go system.loopWindowEvent()
		})
	}
	return system
}

func (s *Service) SetContext(ctx context.Context, version string) {
	s.ctx = ctx
	s.appVersion = version

	// Maximise the current window if the screen size is lower than the minimum window size.
	app := application.Get()
	if app == nil {
		return
	}
	screens := app.Screen.GetAll()
	if len(screens) == 0 {
		return
	}
	// Prefer primary screen if available.
	screen := screens[0]
	for _, sc := range screens {
		if sc.IsPrimary {
			screen = sc
			break
		}
	}
	if screen.Size.Width < consts.MIN_WINDOW_WIDTH || screen.Size.Height < consts.MIN_WINDOW_HEIGHT {
		if win, ok := app.Window.Get("main"); ok && win != nil {
			win.Maximise()
		}
	}
}

func (s *Service) Info() (resp types.JSResp) {
	resp.Success = true
	resp.Data = struct {
		OS      string `json:"os"`
		Arch    string `json:"arch"`
		Version string `json:"version"`
	}{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		Version: s.appVersion,
	}
	return
}

// OpenDirectoryDialog open directory dialog
func (s *Service) OpenDirectoryDialog(title string) (resp types.JSResp) {
	app := application.Get()
	if app == nil {
		resp.Msg = "application not ready"
		return
	}
	dialog := application.OpenFileDialog().
		CanChooseDirectories(true).
		CanChooseFiles(false).
		ShowHiddenFiles(true).
		SetTitle(title)
	if win, ok := app.Window.Get("main"); ok && win != nil {
		dialog.AttachToWindow(win)
	}
	filepath, err := dialog.PromptForSingleSelection()
	if err != nil {
		resp = handleDialogError(err)
		return resp
	}
	resp.Success = true
	resp.Data = map[string]any{
		"path": filepath,
	}
	return
}

// OpenDirectory opens the specified directory in system file explorer
func (s *Service) OpenDirectory(path string) {
	// Delegate to OpenPath to avoid BrowserOpenURL scheme restrictions in newer Wails versions.
	// OpenPath already handles OS-specific open behaviour (open/xdg-open/Explorer).
	_ = s.OpenPath(path)
}

// OpenPath opens a file (or directory) with the system default application
func (s *Service) OpenPath(path string) (resp types.JSResp) {
	if path == "" {
		resp.Msg = "path is empty"
		return
	}
	if _, err := os.Stat(path); err != nil {
		resp.Msg = err.Error()
		return
	}
	go func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", path)
		case "windows":
			// Suppress console window on Windows when opening files
			cmd = winOpenCmd(path)
		default:
			cmd = exec.Command("xdg-open", path)
		}
		_ = cmd.Start()
	}()
	resp.Success = true
	return
}

// SelectFile open file dialog to select a file
func (s *Service) SelectFile(title string, extensions []string) (resp types.JSResp) {
	app := application.Get()
	if app == nil {
		resp.Msg = "application not ready"
		return
	}

	// Build Windows/macOS friendly filters: include DisplayName and an aggregated pattern
	var filters []application.FileFilter
	if len(extensions) > 0 {
		// sanitize, dedupe, and build patterns
		seen := map[string]bool{}
		pats := make([]string, 0, len(extensions))
		for _, e := range extensions {
			ext := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(e), "."))
			if ext == "" || seen[ext] {
				continue
			}
			seen[ext] = true
			pats = append(pats, "*."+ext)
		}
		if len(pats) > 0 {
			agg := strings.Join(pats, ";")
			filters = append(filters, application.FileFilter{
				DisplayName: "Supported Files (" + strings.Join(pats, ";") + ")",
				Pattern:     agg,
			})
			// individual filters
			for _, p := range pats {
				ext := strings.TrimPrefix(p, "*.")
				filters = append(filters, application.FileFilter{
					DisplayName: strings.ToUpper(ext) + " Files (*." + ext + ")",
					Pattern:     p,
				})
			}
		}
		// Always include All Files at the end
		filters = append(filters, application.FileFilter{DisplayName: "All Files (*.*)", Pattern: "*.*"})
	}

	dialog := application.OpenFileDialog().
		CanChooseFiles(true).
		ShowHiddenFiles(true)
	if title != "" {
		dialog.SetTitle(title)
	}
	for _, f := range filters {
		dialog.AddFilter(f.DisplayName, f.Pattern)
	}
	if win, ok := app.Window.Get("main"); ok && win != nil {
		dialog.AttachToWindow(win)
	}
	filepath, err := dialog.PromptForSingleSelection()
	if err != nil {
		resp = handleDialogError(err)
		return resp
	}
	resp.Success = true
	resp.Data = map[string]any{
		"path": filepath,
	}
	return
}

// SaveFile open file dialog to save a file
func (s *Service) SaveFile(title string, defaultName string, extensions []string) (resp types.JSResp) {
	app := application.Get()
	if app == nil {
		resp.Msg = "application not ready"
		return
	}

	// Reuse the same friendly filters as SelectFile
	var filters []application.FileFilter
	if len(extensions) > 0 {
		seen := map[string]bool{}
		pats := make([]string, 0, len(extensions))
		for _, e := range extensions {
			ext := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(e), "."))
			if ext == "" || seen[ext] {
				continue
			}
			seen[ext] = true
			pats = append(pats, "*."+ext)
		}
		if len(pats) > 0 {
			agg := strings.Join(pats, ";")
			filters = append(filters, application.FileFilter{
				DisplayName: "Supported Files (" + strings.Join(pats, ";") + ")",
				Pattern:     agg,
			})
			for _, p := range pats {
				ext := strings.TrimPrefix(p, "*.")
				filters = append(filters, application.FileFilter{
					DisplayName: strings.ToUpper(ext) + " Files (*." + ext + ")",
					Pattern:     p,
				})
			}
		}
		filters = append(filters, application.FileFilter{DisplayName: "All Files (*.*)", Pattern: "*.*"})
	}

	dialog := application.SaveFileDialog().
		SetFilename(defaultName).
		ShowHiddenFiles(true)
	for _, f := range filters {
		dialog.AddFilter(f.DisplayName, f.Pattern)
	}
	if win, ok := app.Window.Get("main"); ok && win != nil {
		dialog.AttachToWindow(win)
	}
	filepath, err := dialog.PromptForSingleSelection()
	if err != nil {
		resp = handleDialogError(err)
		return resp
	}
	resp.Success = true
	resp.Data = map[string]any{
		"path": filepath,
	}
	return
}

func (s *Service) loopWindowEvent() {
	var fullscreen, maximised, minimised, normal bool
	var width, height int
	var dirty bool
	for {
		time.Sleep(300 * time.Millisecond)
		if s.ctx == nil {
			continue
		}

		app := application.Get()
		if app == nil {
			continue
		}
		win, ok := app.Window.Get("main")
		if !ok || win == nil {
			continue
		}

		dirty = false
		if f := win.IsFullscreen(); f != fullscreen {
			// full-screen switched
			fullscreen = f
			dirty = true
		}

		if w, h := win.Size(); w != width || h != height {
			// window size changed
			width, height = w, h
			dirty = true
		}

		if m := win.IsMaximised(); m != maximised {
			maximised = m
			dirty = true
		}

		if m := win.IsMinimised(); m != minimised {
			minimised = m
			dirty = true
		}

		// Wails v3 no longer has a direct "IsNormal", so derive it.
		normal = !(fullscreen || maximised || minimised)

		if dirty {
			// Emit a custom event so the frontend can track window changes.
			win.EmitEvent("window_changed", map[string]any{
				"fullscreen": fullscreen,
				"width":      width,
				"height":     height,
				"maximised":  maximised,
				"minimised":  minimised,
				"normal":     normal,
			})

			if !fullscreen && !minimised {
				// save window size and position
				preferences.New().SaveWindowSize(width, height, maximised)
			}
		}
	}
}

// handleDialogError normalises platform-specific dialog errors so that
// user-cancelled operations are treated as benign.
func handleDialogError(err error) types.JSResp {
	resp := types.JSResp{}
	if err == nil {
		resp.Success = true
		return resp
	}
	msg := strings.ToLower(err.Error())
	// Preserve the v2 behaviour: treat "cancel"/"shellitem is nil" as a normal cancel.
	if strings.Contains(msg, "shellitem") ||
		strings.Contains(msg, "cancel") ||
		strings.Contains(msg, "canceled") ||
		strings.Contains(msg, "cancelled") {
		resp.Success = true
		resp.Data = map[string]any{"path": ""}
		return resp
	}
	resp.Msg = err.Error()
	return resp
}
