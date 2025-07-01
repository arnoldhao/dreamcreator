package main

import (
	"CanMe/backend/api"
	"CanMe/backend/consts"
	"CanMe/backend/core/downtasks"
	"CanMe/backend/core/imageproxies"
	"CanMe/backend/core/subtitles"
	"CanMe/backend/mcpserver"
	"CanMe/backend/pkg/browercookies"
	"CanMe/backend/pkg/downinfo"
	"CanMe/backend/pkg/events"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/pkg/proxy"
	"CanMe/backend/pkg/websockets"
	"CanMe/backend/services/preferences"
	"CanMe/backend/services/systems"
	"CanMe/backend/storage"
	"context"
	"embed"
	"fmt"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Frameworks
	preferencesService := preferences.New()
	systemService := systems.New()

	// log
	defer logger.GetLogger().Sync()
	logger.GetLogger().Info("CanMe Start!", zap.Time("now", time.Now()))

	// bolt storage
	boltStorage, err := storage.NewBoltStorage()
	if err != nil {
		logger.GetLogger().Error("Error creating bolt storage", zap.Error(err))
	}

	// Packages
	// # Events
	eventBus := events.NewEventBus(events.DefaultEventBusOptions())
	// # Proxy
	proxyManager := proxy.NewManager(proxy.DefaultConfig(), eventBus)
	// # Download
	downloadClient := downinfo.NewClient(downinfo.DefaultConfig())
	preferencesService.SetPackageClients(proxyManager, downloadClient)

	// # CookieManager (New)
	cookieManager := browercookies.NewCookieManager(boltStorage)

	// Services
	// # Downtasks
	dtService := downtasks.NewService(eventBus, proxyManager, downloadClient, preferencesService, boltStorage, cookieManager)
	// # Imagesproxies
	ipsService := imageproxies.NewService(proxyManager, boltStorage)
	// # Subtitles
	subtitlesService := subtitles.NewService(boltStorage)

	// Packages
	// # Websocket
	websocketService := websockets.New()
	websocketService.Start()

	// API
	// # Downtasks API
	dtAPI := api.NewDowntasksAPI(dtService, eventBus, websocketService)
	// # Paths API
	pathsAPI := api.NewPathsAPI(preferencesService, dtService)
	// # Utils API
	utilsAPI := api.NewUtilsAPI(ipsService)
	// # Subtitles API
	subtitlesAPI := api.NewSubtitlesAPI(subtitlesService)
	// # Dependencies API
	dependenciesAPI := api.NewDependenciesAPI(dtService)
	// # Cookies API (New)
	cookiesAPI := api.NewCookiesAPI(dtService)

	// MCP
	// # MCP Server
	mcpServer := mcpserver.NewService(dtService)

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
			// Framworks
			preferencesService,
			systemService,
			// Packages
			websocketService,
			// APIs
			dtAPI,
			pathsAPI,
			utilsAPI,
			subtitlesAPI,
			dependenciesAPI,
			cookiesAPI,
		},
		Logger: logger.NewWailsLogger(),
		OnStartup: func(ctx context.Context) {
			// Frameworks
			preferencesService.SetContext(ctx)
			systemService.SetContext(ctx, consts.APP_VERSION)
			// Packages
			proxyManager.SetContext(ctx)
			websocketService.SetContext(ctx)
			// Services
			dtService.SetContext(ctx)
			ipsService.SetContext(ctx)
			// APIs
			dtAPI.Subscribe(ctx)
			pathsAPI.Subscribe(ctx)
			utilsAPI.Subscribe(ctx)
			subtitlesAPI.Subscribe(ctx)
			dependenciesAPI.Subscribe(ctx)
			cookiesAPI.WailsInit(ctx)
			// MCP
			if err := mcpServer.Start(ctx); err != nil {
				logger.GetLogger().Error("Error starting MCP server", zap.Error(err))
			}
		},
		OnDomReady: func(ctx context.Context) {
			x, y := wailsRuntime.WindowGetPosition(ctx)
			wailsRuntime.WindowSetPosition(ctx, x, y)
			wailsRuntime.WindowShow(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			// 关闭下载任务服务
			if err := dtService.Close(); err != nil {
				logger.GetLogger().Error("Error closing downtasks service", zap.Error(err))
			}
		},
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
				Message: consts.APP_DESC,
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
