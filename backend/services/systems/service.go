package systems

import (
	"CanMe/backend/consts"
	"CanMe/backend/services/preferences"
	"CanMe/backend/types"
	"CanMe/backend/utils/sliceUtil"
	"context"
	"runtime"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
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

	// maximize the window if screen size is lower than the minimum window size
	if screen, err := wailsRuntime.ScreenGetAll(ctx); err == nil && len(screen) > 0 {
		for _, sc := range screen {
			if sc.IsCurrent {
				if sc.Size.Width < consts.MIN_WINDOW_WIDTH || sc.Size.Height < consts.MIN_WINDOW_HEIGHT {
					wailsRuntime.WindowMaximise(ctx)
					break
				}
			}
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
	filepath, err := wailsRuntime.OpenDirectoryDialog(s.ctx, wailsRuntime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
	resp.Data = map[string]any{
		"path": filepath,
	}
	return
}

// OpenDirectory opens the specified directory in system file explorer
func (s *Service) OpenDirectory(path string) {
	url := "file:///" + path

	wailsRuntime.BrowserOpenURL(s.ctx, url)
}

// SelectFile open file dialog to select a file
func (s *Service) SelectFile(title string, extensions []string) (resp types.JSResp) {
	filters := sliceUtil.Map(extensions, func(i int) wailsRuntime.FileFilter {
		return wailsRuntime.FileFilter{
			Pattern: "*." + extensions[i],
		}
	})
	filepath, err := wailsRuntime.OpenFileDialog(s.ctx, wailsRuntime.OpenDialogOptions{
		Title:           title,
		ShowHiddenFiles: true,
		Filters:         filters,
	})
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
	resp.Data = map[string]any{
		"path": filepath,
	}
	return
}

// SaveFile open file dialog to save a file
func (s *Service) SaveFile(title string, defaultName string, extensions []string) (resp types.JSResp) {
	filters := sliceUtil.Map(extensions, func(i int) wailsRuntime.FileFilter {
		return wailsRuntime.FileFilter{
			Pattern: "*." + extensions[i],
		}
	})
	filepath, err := wailsRuntime.SaveFileDialog(s.ctx, wailsRuntime.SaveDialogOptions{
		Title:           title,
		ShowHiddenFiles: true,
		DefaultFilename: defaultName,
		Filters:         filters,
	})
	if err != nil {
		resp.Msg = err.Error()
		return
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

		dirty = false
		if f := wailsRuntime.WindowIsFullscreen(s.ctx); f != fullscreen {
			// full-screen switched
			fullscreen = f
			dirty = true
		}

		if w, h := wailsRuntime.WindowGetSize(s.ctx); w != width || h != height {
			// window size changed
			width, height = w, h
			dirty = true
		}

		if m := wailsRuntime.WindowIsMaximised(s.ctx); m != maximised {
			maximised = m
			dirty = true
		}

		if m := wailsRuntime.WindowIsMinimised(s.ctx); m != minimised {
			minimised = m
			dirty = true
		}

		if n := wailsRuntime.WindowIsNormal(s.ctx); n != normal {
			normal = n
			dirty = true
		}

		if dirty {
			wailsRuntime.EventsEmit(s.ctx, "window_changed", map[string]any{
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
