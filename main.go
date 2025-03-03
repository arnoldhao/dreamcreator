package main

import (
	"CanMe/backend/api"
	"CanMe/backend/consts"
	"CanMe/backend/core/download"
	"CanMe/backend/core/events"
	"CanMe/backend/core/websockets"
	"CanMe/backend/services/languages"
	"CanMe/backend/services/llms"
	"CanMe/backend/services/ollama"
	"CanMe/backend/services/preferences"
	"CanMe/backend/services/subtitles"
	"CanMe/backend/services/systems"
	"CanMe/backend/services/trans"
	"CanMe/backend/storage"
	"CanMe/backend/storage/repository"
	"context"
	"embed"
	"fmt"
	"runtime"

	_ "CanMe/backend/pkg/extractors/bilibili"
	_ "CanMe/backend/pkg/extractors/youtube" // init youtube extractor

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
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

	// database
	persistentStorage, err := storage.NewPersistentStorage()
	if err != nil {
		panic(err)
	}
	storage.SetGlobalPersistentStorage(persistentStorage)
	repo := repository.NewDownloadRepository(persistentStorage.DBWithoutContext())

	// services
	events := events.NewEventBus()
	dserv := download.NewService(events, repo)

	// websocket
	websocketService := websockets.New()
	websocketService.Start()

	// api
	downloadAPI := api.NewDownloadAPI(dserv, events, websocketService)
	// instances
	languageService := languages.New()
	llmsService := llms.New()
	ollamaService := ollama.New()
	subtitlesService := subtitles.New()
	transService := trans.New()

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
		MaxWidth:                 4096,
		MaxHeight:                2160,
		DisableResize:            false,
		Fullscreen:               false,
		WindowStartState:         windowStartState,
		Frameless:                !isMacOS,
		AssetServer:              &assetserver.Options{Assets: assets},
		BackgroundColour:         &options.RGBA{R: 255, G: 255, B: 255, A: 0},
		Menu:                     appMenu,
		EnableDefaultContextMenu: true,
		Bind: []interface{}{
			preferencesService,
			languageService,
			llmsService,
			ollamaService,
			systemService,
			subtitlesService,
			websocketService,
			transService,
			downloadAPI,
		},
		Logger:             logger.NewDefaultLogger(),
		LogLevel:           logger.INFO,
		LogLevelProduction: logger.ERROR,
		OnStartup: func(ctx context.Context) {
			systemService.SetContext(ctx, consts.APP_VERSION)
			preferencesService.RegisterServices(ctx, websocketService)
			languageService.RegisterService(ctx)
			llmsService.RegisterServices(ctx)
			subtitlesService.SetContext(ctx)
			ollamaService.RegisterServices(ctx, websocketService)
			languageService.RegisterService(ctx)
			transService.Process(ctx, websocketService)
			websocketService.RegisterServices(ctx, transService, ollamaService, preferencesService)
			persistentStorage.AutoMigrate(ctx)
			dserv.Start(ctx)
			downloadAPI.Subscribe(ctx)
		},
		OnDomReady: func(ctx context.Context) {
			x, y := wailsRuntime.WindowGetPosition(ctx)
			wailsRuntime.WindowSetPosition(ctx, x, y)
			wailsRuntime.WindowShow(ctx)
		},
		OnShutdown: func(ctx context.Context) {},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			x, y := wailsRuntime.WindowGetPosition(ctx)
			preferencesService.SaveWindowPosition(x, y)
			return false
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               "",
			Theme:                             windows.SystemDefault,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   fmt.Sprintf("%s %s", consts.APP_NAME, consts.APP_VERSION),
				Message: "A modern lightweight cross-platform framework for developing desktop applications.\n\nCopyright 2024",
				Icon:    icon,
			},
			Appearance:           mac.DefaultAppearance,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
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
