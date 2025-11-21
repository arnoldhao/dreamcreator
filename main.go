package main

import (
    "context"
    "dreamcreator/backend/api"
    "dreamcreator/backend/consts"
    "dreamcreator/backend/core/downtasks"
    "dreamcreator/backend/core/imageproxies"
    "dreamcreator/backend/core/subtitles"
    "dreamcreator/backend/mcpserver"
    "dreamcreator/backend/pkg/provider"
    "dreamcreator/backend/pkg/downinfo"
    "dreamcreator/backend/pkg/events"
    "dreamcreator/backend/pkg/logger"
    "dreamcreator/backend/pkg/proxy"
    "dreamcreator/backend/pkg/websockets"
	"dreamcreator/backend/services/preferences"
	"dreamcreator/backend/services/systems"
	"dreamcreator/backend/storage"
	"dreamcreator/backend/utils"
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

//go:embed frontend/src/assets/images/icon.png
var icon []byte

func main() {
	// Frameworks
	preferencesService := preferences.New()
	systemService := systems.New()

	// log
	defer logger.GetLogger().Sync()
	logger.Info("dreamcreator Start!", zap.String("Version", consts.APP_VERSION), zap.Time("now", time.Now()))

	// bolt storage
	boltStorage, err := storage.NewBoltStorage()
	if err != nil {
		logger.Error("Error creating bolt storage", zap.Error(err))
	}

	// Packages
	// # Events
	eventBus := events.NewEventBus(events.DefaultEventBusOptions())
	// # Proxy
	proxyManager := proxy.NewManager(proxy.DefaultConfig(), eventBus)
	// # Download
	downloadClient := downinfo.NewClient(downinfo.DefaultConfig())
	preferencesService.SetPackageClients(proxyManager, downloadClient)

    // Services
    // # Downtasks
    dtService := downtasks.NewService(eventBus, proxyManager, downloadClient, preferencesService, boltStorage)
    // # Imagesproxies
    ipsService := imageproxies.NewService(proxyManager, boltStorage)
    // # LLM Service (providers)
    llmService := provider.NewService(boltStorage, proxyManager)
    // # Subtitles (inject provider service for AI translation)
    subtitlesService := subtitles.NewService(boltStorage, proxyManager, eventBus, llmService)

	// Packages
	// # Websocket
	websocketService := websockets.New()

	// API
	// # Downtasks API
	dtAPI := api.NewDowntasksAPI(dtService, subtitlesService, eventBus, websocketService)
	// # Paths API
	pathsAPI := api.NewPathsAPI(preferencesService, dtService)
	// # Utils API
	utilsAPI := api.NewUtilsAPI(ipsService)
    // # Subtitles API
    subtitlesAPI := api.NewSubtitlesAPI(subtitlesService, eventBus, websocketService)
    // # Dependencies API
    dependenciesAPI := api.NewDependenciesAPI(dtService)
    // # Cookies API (New)
    cookiesAPI := api.NewCookiesAPI(dtService)
    // # LLM API (Wails style)
    llmAPI := api.NewLLMAPI(llmService)

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

	// Windows options: enable Acrylic only when the platform supports it; otherwise run opaque.
	var winOpts *windows.Options
	if runtime.GOOS == "windows" {
		if utils.WindowsSupportsAcrylic() {
			winOpts = &windows.Options{
				WebviewIsTransparent:              true,
				WindowIsTranslucent:               true,
				BackdropType:                      windows.Acrylic,
				DisableWindowIcon:                 false,
				DisableFramelessWindowDecorations: false,
				WebviewUserDataPath:               "",
				Theme:                             windows.SystemDefault,
			}
		} else {
			winOpts = &windows.Options{
				WebviewIsTransparent:              false,
				WindowIsTranslucent:               false,
				BackdropType:                      windows.None,
				DisableWindowIcon:                 false,
				DisableFramelessWindowDecorations: false,
				WebviewUserDataPath:               "",
				Theme:                             windows.SystemDefault,
			}
		}
	} else {
		// Non-Windows builds ignore this, but keep a default object to satisfy the field.
		winOpts = &windows.Options{
			WebviewIsTransparent:              true,
			WindowIsTranslucent:               true,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
			WebviewUserDataPath:               "",
			Theme:                             windows.SystemDefault,
		}
	}

	err = wails.Run(&options.App{
		Title:                    consts.AppDisplayName(),
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
            llmAPI,
		},
		Logger: logger.NewWailsLogger(),
		OnStartup: func(ctx context.Context) {
			// Frameworks
			preferencesService.SetContext(ctx)
			systemService.SetContext(ctx, consts.APP_VERSION)
			// Packages
			proxyManager.SetContext(ctx)
			websocketService.SetContext(ctx)
			websocketService.Start()
			// Services
			dtService.SetContext(ctx)
			ipsService.SetContext(ctx)
			subtitlesService.SetContext(ctx)
			// APIs
			dtAPI.Subscribe(ctx)
			pathsAPI.Subscribe(ctx)
			utilsAPI.Subscribe(ctx)
			subtitlesAPI.Subscribe(ctx)
            dependenciesAPI.Subscribe(ctx)
            cookiesAPI.WailsInit(ctx)
            // LLM API
            // Seed default providers if DB is empty (one-off via migration flag)
            if n, err := llmService.EnsureDefaultProviders(ctx); err != nil {
                logger.Warn("seed default providers failed", zap.Error(err))
            } else if n > 0 {
                logger.Info("seeded default providers", zap.Int("count", n))
            }
            llmAPI.Subscribe(ctx)
			// MCP
			if err := mcpServer.Start(ctx); err != nil {
				logger.Error("Error starting MCP server", zap.Error(err))
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
				logger.Error("Error closing downtasks service", zap.Error(err))
			}
			// 关闭 Bolt 数据库，释放文件锁，便于开发环境重载
			if boltStorage != nil {
				if err := boltStorage.Close(); err != nil {
					logger.Warn("Error closing bolt storage", zap.Error(err))
				}
			}
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			x, y := wailsRuntime.WindowGetPosition(ctx)
			preferencesService.SaveWindowPosition(x, y)
			return false
		},
		Windows: winOpts,
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   fmt.Sprintf("%s %s", consts.AppDisplayName(), consts.APP_VERSION),
				Message: consts.APP_DESC,
				Icon:    icon,
			},
			Appearance: mac.DefaultAppearance,
			// Enable transparent webview + translucent window to support native vibrancy
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
		},
		Linux: &linux.Options{
			ProgramName:         consts.AppDisplayName(),
			Icon:                icon,
			WebviewGpuPolicy:    linux.WebviewGpuPolicyOnDemand,
			WindowIsTranslucent: true,
		},
	})

	if err != nil {
		logger.Error("App run error", zap.Error(err))
	}
}
