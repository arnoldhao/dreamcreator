package main

import (
	"CanMe/backend/consts"
	"CanMe/backend/services/languages"
	"CanMe/backend/services/llms"
	"CanMe/backend/services/ollama"
	"CanMe/backend/services/preferences"
	"CanMe/backend/services/subtitles"
	"CanMe/backend/services/systems"
	"CanMe/backend/services/trans"
	"CanMe/backend/services/websockets"
	"CanMe/backend/storage"
	"context"
	"embed"
	"fmt"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// frameworks
	preferencesService := preferences.New()
	systemService := systems.New()

	// websocket
	websocketService := websockets.New()
	websocketService.Start()

	// instances
	languageService := languages.New()
	llmsService := llms.New()
	ollamaService := ollama.New()
	subtitlesService := subtitles.New()
	transService := trans.New()

	// database
	persistentStorage, err := storage.NewPersistentStorage()
	if err != nil {
		panic(err)
	}
	storage.SetGlobalPersistentStorage(persistentStorage)

	// window
	windowWidth, windowHeight, maximised := preferencesService.GetWindowSize()
	windowStartState := options.Normal
	if maximised {
		windowStartState = options.Maximised
	}

	// menu
	isMacOS := runtime.GOOS == "darwin"
	appMenu := menu.NewMenu()
	if isMacOS {
		appMenu.Append(menu.AppMenu())
		appMenu.Append(menu.EditMenu())
		appMenu.Append(menu.WindowMenu())
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:                    consts.APP_NAME,
		Width:                    windowWidth,
		Height:                   windowHeight,
		MinWidth:                 consts.MIN_WINDOW_WIDTH,
		MinHeight:                consts.MIN_WINDOW_HEIGHT,
		WindowStartState:         windowStartState,
		Frameless:                !isMacOS,
		Menu:                     appMenu,
		EnableDefaultContextMenu: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: options.NewRGBA(255, 255, 255, 0),
		StartHidden:      true,
		OnStartup: func(ctx context.Context) {
			systemService.SetContext(ctx, consts.APP_VERSION)
			languageService.RegisterService(ctx)
			llmsService.RegisterServices(ctx)
			subtitlesService.SetContext(ctx)
			ollamaService.RegisterServices(ctx, websocketService)
			languageService.RegisterService(ctx)
			transService.Process(ctx, websocketService)
			websocketService.RegisterServices(ctx, transService, ollamaService)
			persistentStorage.AutoMigrate(ctx)
		},
		OnDomReady: func(ctx context.Context) {
			x, y := wailsRuntime.WindowGetPosition(ctx)
			wailsRuntime.WindowSetPosition(ctx, x, y)
			wailsRuntime.WindowShow(ctx)
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			x, y := wailsRuntime.WindowGetPosition(ctx)
			preferencesService.SaveWindowPosition(x, y)
			return false
		},
		OnShutdown: func(ctx context.Context) {},
		Bind: []interface{}{
			preferencesService,
			languageService,
			llmsService,
			ollamaService,
			systemService,
			subtitlesService,
			websocketService,
			transService,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   fmt.Sprintf("%s %s", consts.APP_NAME, consts.APP_VERSION),
				Message: "A modern lightweight cross-platform framework for developing desktop applications.\n\nCopyright © 2024",
				Icon:    icon,
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: false,
		},
		Linux: &linux.Options{
			ProgramName:         consts.APP_NAME,
			Icon:                icon,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyOnDemand,
			WindowIsTranslucent: true,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
