package main

import (
	"context"
	"dreamcreator/backend/api"
	"dreamcreator/backend/appmenu"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/core/downtasks"
	"dreamcreator/backend/core/imageproxies"
	"dreamcreator/backend/core/subtitles"
	"dreamcreator/backend/pkg/downinfo"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/provider"
	"dreamcreator/backend/pkg/proxy"
	"dreamcreator/backend/pkg/websockets"
	"dreamcreator/backend/services/preferences"
	"dreamcreator/backend/services/systems"
	"dreamcreator/backend/storage"
	"dreamcreator/backend/utils"
	"embed"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	wvEvents "github.com/wailsapp/wails/v3/pkg/events"
	"go.uber.org/zap"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed frontend/src/assets/images/icon.png
var icon []byte

func main() {
	var appQuitting atomic.Bool

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

	// window
	windowWidth, windowHeight, maximised := preferencesService.GetWindowSize()
	windowStartState := application.WindowStateNormal
	if maximised {
		windowStartState = application.WindowStateMaximised
	}

	isMacOS := runtime.GOOS == "darwin"

	// Build application options for Wails v3.
	app := application.New(application.Options{
		Name:        consts.AppDisplayName(),
		Description: consts.APP_DESC,
		Icon:        icon,
		// Let the app continue running even if all windows are closed.
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		Linux: application.LinuxOptions{
			ProgramName: consts.AppDisplayName(),
		},
		Windows: application.WindowsOptions{
			// Keep defaults; user data path & browser path will be chosen automatically.
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		// Bind services/APIs so the existing wailsjs bindings continue to work.
		Services: []application.Service{
			// Frameworks
			application.NewService(preferencesService),
			application.NewService(systemService),
			// Packages
			application.NewService(websocketService),
			// APIs
			application.NewService(dtAPI),
			application.NewService(pathsAPI),
			application.NewService(utilsAPI),
			application.NewService(subtitlesAPI),
			application.NewService(dependenciesAPI),
			application.NewService(cookiesAPI),
			application.NewService(llmAPI),
			// Bootstrap service to emulate v2 OnStartup behaviour.
			application.NewService(&bootstrapService{
				pref:         preferencesService,
				systems:      systemService,
				proxyManager: proxyManager,
				ws:           websocketService,
				dt:           dtService,
				ips:          ipsService,
				subs:         subtitlesService,
				dtAPI:        dtAPI,
				pathsAPI:     pathsAPI,
				utilsAPI:     utilsAPI,
				subsAPI:      subtitlesAPI,
				depAPI:       dependenciesAPI,
				cookiesAPI:   cookiesAPI,
				llmService:   llmService,
				llmAPI:       llmAPI,
			}),
		},
		// OnShutdown in v3 does not receive a context; replicate the old behaviour here.
		OnShutdown: func() {
			appQuitting.Store(true)
			if err := dtService.Close(); err != nil {
				logger.Error("Error closing downtasks service", zap.Error(err))
			}
			if boltStorage != nil {
				if err := boltStorage.Close(); err != nil {
					logger.Warn("Error closing bolt storage", zap.Error(err))
				}
			}
		},
	})

	// Application-level menu (must be created after the app).
	appMenu := app.NewMenu()
	if isMacOS {
		appMenu.AddRole(application.AppMenu)
		appMenu.AddRole(application.EditMenu)
		appMenu.AddRole(application.WindowMenu)
		appMenu.AddRole(application.HelpMenu)
	} else {
		appMenu.AddRole(application.FileMenu)
		appMenu.AddRole(application.EditMenu)
		appMenu.AddRole(application.WindowMenu)
		appMenu.AddRole(application.HelpMenu)
	}
	// NOTE: Don't call appMenu.Update() here on macOS - the native menu isn't fully
	// initialised until the app runloop is active. We'll set the menu once after
	// we add our custom items below.

	// Create the main window with options equivalent to the old v2 options.App.
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "main",
		Title:            consts.AppDisplayName(),
		Width:            windowWidth,
		Height:           windowHeight,
		MinWidth:         consts.MIN_WINDOW_WIDTH,
		MinHeight:        consts.MIN_WINDOW_HEIGHT,
		MaxWidth:         4096,
		MaxHeight:        2160,
		DisableResize:    false,
		StartState:       windowStartState,
		Frameless:        !isMacOS,
		BackgroundType:   application.BackgroundTypeTranslucent,
		BackgroundColour: application.NewRGBA(255, 255, 255, 0),
		// Use per-platform window options to approximate the previous behaviour.
		Windows: buildWindowsWindowOptions(),
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBarHiddenInset,
			// Use liquid glass-style translucency similar to the previous transparent+translucent combo.
			Backdrop:   application.MacBackdropTranslucent,
			Appearance: application.DefaultAppearance,
		},
		Linux: application.LinuxWindow{
			Icon:                icon,
			WebviewGpuPolicy:    application.WebviewGpuPolicyOnDemand,
			WindowIsTranslucent: true,
			Menu:                appMenu,
		},
	})

	// Predeclare the settings window so hooks can reference it.
	var settingsWindow *application.WebviewWindow

	// Create a dedicated Settings window (hidden by default; shown on demand).
	// NOTE: This window intentionally runs in a reduced "settings" mode on the frontend,
	// so it doesn't start background subsystems like WebSocket auto-reconnect.
	settingsWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:          "settings",
		Title:         "Settings",
		Width:         800,
		Height:        600,
		MinWidth:      800,
		MinHeight:     600,
		MaxWidth:      800,
		MaxHeight:     600,
		DisableResize: true,
		// Fixed-size Settings window: only allow close; grey out minimise/maximise traffic lights.
		MinimiseButtonState: application.ButtonDisabled,
		MaximiseButtonState: application.ButtonDisabled,
		CloseButtonState:    application.ButtonEnabled,
		StartState:          application.WindowStateNormal,
		Hidden:              true,
		// Route into the SPA "settings window" mode.
		URL: "/?window=settings",

		Frameless:        false,
		BackgroundType:   application.BackgroundTypeTranslucent,
		BackgroundColour: application.NewRGBA(255, 255, 255, 0),

		Windows: buildWindowsWindowOptions(),
		Mac: application.MacWindow{
			// Match the main window: no native title text; traffic lights blend into our custom top bar.
			TitleBar:   application.MacTitleBarHiddenInset,
			Backdrop:   application.MacBackdropTranslucent,
			Appearance: application.DefaultAppearance,
		},
		Linux: application.LinuxWindow{
			Icon:                icon,
			WebviewGpuPolicy:    application.WebviewGpuPolicyOnDemand,
			WindowIsTranslucent: true,
			Menu:                appMenu,
		},
	})

	// Persist window position on close (equivalent to v2 OnBeforeClose behaviour).
	//
	// IMPORTANT: Use a hook (not OnWindowEvent) so we can Cancel() reliably.
	mainWindow.RegisterHook(wvEvents.Common.WindowClosing, func(event *application.WindowEvent) {
		// Always persist last-known position.
		x, y := mainWindow.Position()
		preferencesService.SaveWindowPosition(x, y)

		// If the app is quitting, allow the window to be destroyed.
		if appQuitting.Load() {
			return
		}

		// macOS: closing the main window should hide it (keep app alive), so Dock click can reopen it.
		if isMacOS {
			event.Cancel()
			// Avoid redundant Hide() calls which can spam WindowHide events/logs.
			if mainWindow.IsVisible() {
				mainWindow.Hide()
			}
			return
		}

		// Windows/Linux: closing the main window quits the app; ensure any auxiliary windows close too.
		appQuitting.Store(true)
		if settingsWindow != nil {
			settingsWindow.Close()
		}
	})

	// Closing the Settings window should behave like "hide" (macOS Preferences style).
	//
	// IMPORTANT: use a hook (not OnWindowEvent) because OnWindowEvent listeners are invoked
	// asynchronously and cannot reliably Cancel(); Wails also registers an internal WindowClosing
	// listener that destroys the window. A hook runs first and can Cancel() to prevent destruction.
	settingsWindow.RegisterHook(wvEvents.Common.WindowClosing, func(event *application.WindowEvent) {
		if appQuitting.Load() {
			return
		}
		event.Cancel()
		// Avoid redundant Hide() calls which can spam WindowHide events/logs.
		if settingsWindow.IsVisible() {
			settingsWindow.Hide()
		}
	})

	openSettingsWindow := func() {
		settingsWindow.Show()
	}

	// Localised application menu controller (keeps main.go lean; listens to frontend locale events).
	appmenu.Install(app, appMenu, appmenu.Options{
		IsMacOS:      isMacOS,
		AppName:      consts.AppDisplayName(),
		OpenSettings: openSettingsWindow,
		Preferences:  preferencesService,
	})
	app.Menu.SetApplicationMenu(appMenu)

	// Ensure the main window is shown and focused once the application has started.
	app.Event.OnApplicationEvent(wvEvents.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		// At this point the native window implementation is initialised, so it's
		// safe to show/focus the main window.
		mainWindow.Show()
		mainWindow.Focus()
	})

	// macOS: Clicking the Dock icon when no window is visible should re-open the main window.
	if isMacOS {
		app.Event.OnApplicationEvent(wvEvents.Mac.ApplicationShouldHandleReopen, func(event *application.ApplicationEvent) {
			// If the user reopens, bring the main window back.
			mainWindow.Show()
		})
		// If the app is activated with no visible windows (e.g. Cmd+Tab or Dock),
		// show the main window so the app doesn't appear "stuck running".
		app.Event.OnApplicationEvent(wvEvents.Mac.ApplicationDidBecomeActive, func(event *application.ApplicationEvent) {
			if appQuitting.Load() {
				return
			}
			if !mainWindow.IsVisible() && (settingsWindow == nil || !settingsWindow.IsVisible()) {
				mainWindow.Show()
			}
		})
	}

	// Run the application.
	if err := app.Run(); err != nil {
		logger.Error("App run error", zap.Error(err))
	}
}

// buildWindowsWindowOptions creates a WindowsWindow configuration equivalent
// to the old v2 windows.Options logic, including Acrylic fallback.
func buildWindowsWindowOptions() application.WindowsWindow {
	if runtime.GOOS != "windows" {
		// Non-Windows builds will ignore these fields.
		return application.WindowsWindow{
			BackdropType: application.Auto,
			Theme:        application.SystemDefault,
		}
	}

	if utils.WindowsSupportsAcrylic() {
		return application.WindowsWindow{
			BackdropType:                      application.Acrylic,
			DisableIcon:                       false,
			DisableFramelessWindowDecorations: false,
			Theme:                             application.SystemDefault,
		}
	}

	return application.WindowsWindow{
		BackdropType:                      application.None,
		DisableIcon:                       false,
		DisableFramelessWindowDecorations: false,
		Theme:                             application.SystemDefault,
	}
}

// bootstrapService emulates the old v2 OnStartup hook so that existing services
// and APIs continue to receive the Wails application context and can subscribe
// to events as before.
type bootstrapService struct {
	pref         *preferences.Service
	systems      *systems.Service
	proxyManager *proxy.Manager
	ws           *websockets.Service

	dt   *downtasks.Service
	ips  *imageproxies.Service
	subs *subtitles.Service

	dtAPI      *api.DowntasksAPI
	pathsAPI   *api.PathsAPI
	utilsAPI   *api.UtilsAPI
	subsAPI    *api.SubtitlesAPI
	depAPI     *api.DependenciesAPI
	cookiesAPI *api.CookiesAPI

	llmService *provider.Service
	llmAPI     *api.LLMAPI
}

// ServiceStartup is called by Wails v3 when the application starts. We use it
// to wire up all existing components, mirroring the previous OnStartup logic.
func (b *bootstrapService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	// Frameworks
	b.pref.SetContext(ctx)
	b.systems.SetContext(ctx, consts.APP_VERSION)

	// Packages
	b.proxyManager.SetContext(ctx)
	b.ws.SetContext(ctx)
	b.ws.Start()

	// Services
	b.dt.SetContext(ctx)
	b.ips.SetContext(ctx)
	b.subs.SetContext(ctx)

	// APIs
	b.dtAPI.Subscribe(ctx)
	b.pathsAPI.Subscribe(ctx)
	b.utilsAPI.Subscribe(ctx)
	b.subsAPI.Subscribe(ctx)
	b.depAPI.Subscribe(ctx)
	b.cookiesAPI.WailsInit(ctx)

	// LLM bootstrap (seed default providers once)
	if n, err := b.llmService.EnsureDefaultProviders(ctx); err != nil {
		logger.Warn("seed default providers failed", zap.Error(err))
	} else if n > 0 {
		logger.Info("seeded default providers", zap.Int("count", n))
	}
	b.llmAPI.Subscribe(ctx)

	return nil
}
