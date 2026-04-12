package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	agentservice "dreamcreator/internal/application/agent/service"
	assistantservice "dreamcreator/internal/application/assistant/service"
	channelpairing "dreamcreator/internal/application/channels/pairing"
	telegrammenu "dreamcreator/internal/application/channels/telegram"
	connectorsservice "dreamcreator/internal/application/connectors/service"
	appevents "dreamcreator/internal/application/events"
	externaltoolsservice "dreamcreator/internal/application/externaltools/service"
	fontservice "dreamcreator/internal/application/fonts/service"
	gatewayapprovals "dreamcreator/internal/application/gateway/approvals"
	gatewayauth "dreamcreator/internal/application/gateway/auth"
	gatewayautomation "dreamcreator/internal/application/gateway/automation"
	gatewaychannels "dreamcreator/internal/application/gateway/channels"
	gatewaycommands "dreamcreator/internal/application/gateway/commands"
	gatewayconfig "dreamcreator/internal/application/gateway/config"
	gatewaycontrolplane "dreamcreator/internal/application/gateway/controlplane"
	gatewaycron "dreamcreator/internal/application/gateway/cron"
	gatewayevents "dreamcreator/internal/application/gateway/events"
	gatewayheartbeat "dreamcreator/internal/application/gateway/heartbeat"
	gatewaymodels "dreamcreator/internal/application/gateway/models"
	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
	gatewayobservability "dreamcreator/internal/application/gateway/observability"
	gatewaypairing "dreamcreator/internal/application/gateway/pairing"
	gatewayqueue "dreamcreator/internal/application/gateway/queue"
	gatewayruntime "dreamcreator/internal/application/gateway/runtime"
	gatewayruntimedto "dreamcreator/internal/application/gateway/runtime/dto"
	gatewaysandbox "dreamcreator/internal/application/gateway/sandbox"
	gatewaysubagent "dreamcreator/internal/application/gateway/subagent"
	gatewaytools "dreamcreator/internal/application/gateway/tools"
	gatewayusage "dreamcreator/internal/application/gateway/usage"
	gatewayvoice "dreamcreator/internal/application/gateway/voice"
	libraryservice "dreamcreator/internal/application/library/service"
	memoryservice "dreamcreator/internal/application/memory/service"
	appnotice "dreamcreator/internal/application/notice"
	providerservice "dreamcreator/internal/application/providers/service"
	sessionmanager "dreamcreator/internal/application/session"
	settingsdto "dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/application/settings/service"
	skillsservice "dreamcreator/internal/application/skills/service"
	softwareupdate "dreamcreator/internal/application/softwareupdate"
	subagentservice "dreamcreator/internal/application/subagent/service"
	apptelemetry "dreamcreator/internal/application/telemetry"
	threadservice "dreamcreator/internal/application/thread/service"
	toolsservice "dreamcreator/internal/application/tools/service"
	applicationupdate "dreamcreator/internal/application/update"
	workspaceservice "dreamcreator/internal/application/workspace/service"
	domainsession "dreamcreator/internal/domain/session"
	"dreamcreator/internal/domain/settings"
	"dreamcreator/internal/domain/workspace"
	"dreamcreator/internal/infrastructure/agentrepo"
	"dreamcreator/internal/infrastructure/approvalsrepo"
	"dreamcreator/internal/infrastructure/assistantrepo"
	"dreamcreator/internal/infrastructure/automationrepo"
	"dreamcreator/internal/infrastructure/autostart"
	"dreamcreator/internal/infrastructure/configrepo"
	"dreamcreator/internal/infrastructure/connectorsrepo"
	"dreamcreator/internal/infrastructure/diagnosticsrepo"
	"dreamcreator/internal/infrastructure/externaltoolsrepo"
	"dreamcreator/internal/infrastructure/gatewayeventsrepo"
	"dreamcreator/internal/infrastructure/gatewayqueuerepo"
	"dreamcreator/internal/infrastructure/heartbeatrepo"
	"dreamcreator/internal/infrastructure/libraryicons"
	"dreamcreator/internal/infrastructure/libraryrepo"
	"dreamcreator/internal/infrastructure/logging"
	"dreamcreator/internal/infrastructure/noderepo"
	"dreamcreator/internal/infrastructure/noticerepo"
	"dreamcreator/internal/infrastructure/persistence"
	"dreamcreator/internal/infrastructure/providersrepo"
	"dreamcreator/internal/infrastructure/providersync"
	"dreamcreator/internal/infrastructure/proxy"
	"dreamcreator/internal/infrastructure/secure"
	"dreamcreator/internal/infrastructure/sessionrepo"
	"dreamcreator/internal/infrastructure/settingsrepo"
	"dreamcreator/internal/infrastructure/skillsrepo"
	"dreamcreator/internal/infrastructure/subagentrepo"
	"dreamcreator/internal/infrastructure/telemetryrepo"
	"dreamcreator/internal/infrastructure/threadrepo"
	"dreamcreator/internal/infrastructure/toolpolicyrepo"
	infrastructureupdate "dreamcreator/internal/infrastructure/update"
	"dreamcreator/internal/infrastructure/usagerepo"
	"dreamcreator/internal/infrastructure/voicerepo"
	"dreamcreator/internal/infrastructure/workspacefs"
	"dreamcreator/internal/infrastructure/workspacerepo"
	"dreamcreator/internal/infrastructure/ws"
	telegramchannel "dreamcreator/internal/presentation/channels/telegram"
	webchannel "dreamcreator/internal/presentation/channels/web"
	openaihttp "dreamcreator/internal/presentation/gateway/http/openai"
	openresponseshttp "dreamcreator/internal/presentation/gateway/http/openresponses"
	gatewaytoolhttp "dreamcreator/internal/presentation/gateway/http/tools"
	gatewayws "dreamcreator/internal/presentation/gateway/ws"
	gatewaymethods "dreamcreator/internal/presentation/gateway/ws/methods"
	presentationhttp "dreamcreator/internal/presentation/http"
	"dreamcreator/internal/presentation/i18n"
	"dreamcreator/internal/presentation/wails"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
	"go.uber.org/zap"
)

var (
	// AppVersion can be overridden via APP_VERSION env or ldflags "-X dreamcreator/internal/app.AppVersion=1.2.3".
	AppVersion     = "dev"
	AppName        = "Dream Creator"
	AppDescription = "An AI assistant for content creators."
)

type providersUpdatedWindowNotifier struct {
	manager *wails.WindowManager
}

func (notifier providersUpdatedWindowNotifier) ProvidersUpdated() {
	if notifier.manager == nil {
		return
	}
	notifier.manager.EmitProvidersUpdated()
}

type settingsBroadcastAdapter struct {
	service *service.SettingsService

	mu      sync.RWMutex
	applier func(settingsdto.Settings)
}

func newSettingsBroadcastAdapter(settingsService *service.SettingsService) *settingsBroadcastAdapter {
	return &settingsBroadcastAdapter{service: settingsService}
}

func (adapter *settingsBroadcastAdapter) SetApplier(applier func(settingsdto.Settings)) {
	if adapter == nil {
		return
	}
	adapter.mu.Lock()
	adapter.applier = applier
	adapter.mu.Unlock()
}

func (adapter *settingsBroadcastAdapter) GetSettings(ctx context.Context) (settingsdto.Settings, error) {
	if adapter == nil || adapter.service == nil {
		return settingsdto.Settings{}, errors.New("settings service unavailable")
	}
	return adapter.service.GetSettings(ctx)
}

func (adapter *settingsBroadcastAdapter) UpdateSettings(ctx context.Context, request settingsdto.UpdateSettingsRequest) (settingsdto.Settings, error) {
	if adapter == nil || adapter.service == nil {
		return settingsdto.Settings{}, errors.New("settings service unavailable")
	}
	return adapter.service.UpdateSettings(ctx, request)
}

func (adapter *settingsBroadcastAdapter) ApplySettings(updated settingsdto.Settings) {
	if adapter == nil {
		return
	}
	adapter.mu.RLock()
	applier := adapter.applier
	adapter.mu.RUnlock()
	if applier != nil {
		applier(updated)
	}
}

func CreateApplication(assets fs.FS) (*application.App, error) {
	appVersion := resolveVersion(os.Getenv("APP_ENV"))
	startup := currentStartupContext(os.Args[1:])
	appIcon := loadAppIcon(assets)
	trayIcon := loadTrayIcon(assets)
	var windowManager *wails.WindowManager

	app := application.New(application.Options{
		Name:        AppName,
		Description: AppDescription,
		Icon:        appIcon,
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// Keep the app alive even if windows are closed; we hide on close instead.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.dreamapp.dreamcreator",
			ExitCode: 0,
			OnSecondInstanceLaunch: func(_ application.SecondInstanceData) {
				if windowManager == nil {
					return
				}
				windowManager.HandleSecondInstanceLaunch()
			},
		},
	})

	ctx := context.Background()

	database, err := openDatabase(ctx)
	if err != nil {
		return nil, err
	}

	app.OnShutdown(func() {
		if windowManager != nil {
			windowManager.PersistAllBounds()
		}
		_ = database.Close()
	})

	repo := settingsrepo.NewSQLiteSettingsRepository(database.Bun)
	themeProvider := NewAppThemeProvider(app)
	defaultLanguage := i18n.DetectSystemLanguage()
	settingsService := service.NewSettingsService(repo, themeProvider, settings.DefaultSettingsWithLanguage(defaultLanguage.String()))
	settingsNotifier := newSettingsBroadcastAdapter(settingsService)
	skillRepo := skillsrepo.NewSettingsRepository(repo)
	skillsService := skillsservice.NewSkillsService(skillRepo, settingsService)

	currentSettings, err := settingsService.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	logDir, err := logging.DefaultLogDir()
	if err != nil {
		return nil, err
	}

	proxyManager, err := proxy.NewManager(proxy.Config{
		Mode:     settings.ProxyMode(currentSettings.Proxy.Mode),
		Scheme:   settings.ProxyScheme(currentSettings.Proxy.Scheme),
		Host:     currentSettings.Proxy.Host,
		Port:     currentSettings.Proxy.Port,
		Username: currentSettings.Proxy.Username,
		Password: currentSettings.Proxy.Password,
		NoProxy:  currentSettings.Proxy.NoProxy,
		Timeout:  time.Duration(currentSettings.Proxy.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		return nil, err
	}

	telemetryConfig := resolveTelemetryConfig()
	telemetryService := apptelemetry.NewService(
		telemetryrepo.NewSQLiteStateRepository(database.Bun),
		wails.NewTelemetrySignalEmitter(app),
		settingsService,
		telemetryConfig.AppID,
		appVersion,
	)

	appLogger, err := logging.NewLogger(logging.Config{
		Directory:  logDir,
		Level:      settings.LogLevel(currentSettings.LogLevel),
		MaxSizeMB:  currentSettings.LogMaxSizeMB,
		MaxBackups: currentSettings.LogMaxBackups,
		MaxAgeDays: currentSettings.LogMaxAgeDays,
		Compress:   currentSettings.LogCompress,
	})
	if err != nil {
		return nil, err
	}

	zap.L().Info("application started",
		zap.String("logDir", logDir),
		zap.String("logLevel", currentSettings.LogLevel),
		zap.String("language", currentSettings.Language),
		zap.String("appearance", currentSettings.Appearance),
	)

	app.OnShutdown(func() {
		_ = appLogger.Sync()
	})

	autostartManager, err := autostart.NewManager(AppName)
	if err != nil {
		zap.L().Warn("autostart manager unavailable", zap.Error(err))
	}

	eventBus := appevents.NewInMemoryBus()
	serverCtx, serverCancel := context.WithCancel(ctx)
	realtimeServer := ws.NewServer("127.0.0.1:0", eventBus)
	if err := realtimeServer.Start(serverCtx); err != nil {
		serverCancel()
		return nil, err
	}
	app.OnShutdown(func() {
		serverCancel()
		_ = realtimeServer.Shutdown(context.Background())
	})

	gatewayAuth := gatewayauth.NewInMemoryService()
	gatewayAuth.AllowAnonymous(true)
	gatewayScopeGuard := gatewayauth.NewDefaultScopeGuard()
	gatewayRouter := gatewaycontrolplane.NewRouter(gatewayScopeGuard)
	pairingService := gatewaypairing.NewService()
	gatewaymethods.RegisterPairing(gatewayRouter, pairingService)
	gatewayEventStore := gatewayeventsrepo.NewSQLiteEventStore(database.Bun)
	gatewayEvents := gatewayevents.NewBroker(gatewayEventStore)
	skillsService.SetRealtimeNotifier(func(ctx context.Context, event skillsservice.SkillsRealtimeEvent) {
		if gatewayEvents == nil {
			return
		}
		action := strings.ToLower(strings.TrimSpace(event.Action))
		stage := strings.ToLower(strings.TrimSpace(event.Stage))
		if action == "" || stage == "" {
			return
		}
		envelope := appevents.NewGatewayEventEnvelope("skills.catalog", fmt.Sprintf("skills.%s.%s", action, stage))
		_, _ = gatewayEvents.Publish(ctx, envelope, event)
	})
	gatewayRouter.SetAuditHandler(func(ctx context.Context, session *gatewaycontrolplane.SessionContext, result gatewayauth.ScopeCheckResult) {
		payload := map[string]any{
			"method":         result.Method,
			"requiredScopes": result.RequiredScopes,
			"reason":         result.Reason,
		}
		if session != nil {
			payload["role"] = session.Role
			payload["scopes"] = session.Scopes
			payload["sessionId"] = session.ID
		}
		envelope := gatewayevents.Envelope{
			Type:      "gateway.auth.failed",
			Topic:     "gateway.auth",
			Timestamp: time.Now(),
		}
		_, _ = gatewayEvents.Publish(ctx, envelope, payload)
	})
	approvalStore := approvalsrepo.NewSQLiteApprovalStore(database.Bun)
	approvalService := gatewayapprovals.NewService(approvalStore)
	approvalService.SetEventPublisher(gatewayapprovals.NewGatewayEventPublisher(gatewayEvents))
	gatewaymethods.RegisterApprovals(gatewayRouter, approvalService)
	configRevisionStore := configrepo.NewSQLiteRevisionStore(database.Bun)
	telegramBotService := telegrammenu.NewBotService(settingsService, nil, proxyManager.HTTPClient())
	telegramBotService.SetApprovalResolver(approvalService)
	telegramBotService.SetSkillPromptResolver(skillsService)
	telegramPairingStore, err := channelpairing.NewStore("telegram")
	if err != nil {
		zap.L().Warn("telegram pairing store unavailable", zap.Error(err))
	} else {
		telegramBotService.SetPairingStore(telegramPairingStore)
	}
	gatewayEvents.Subscribe(gatewayevents.Filter{Type: "exec.approval.requested"}, func(record gatewayevents.Record) {
		if len(record.Payload) == 0 {
			return
		}
		payload := append([]byte(nil), record.Payload...)
		go func() {
			var request gatewayapprovals.Request
			if err := json.Unmarshal(payload, &request); err != nil {
				return
			}
			forwardCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = telegramBotService.ForwardExecApprovalRequested(forwardCtx, request)
		}()
	})
	gatewayEvents.Subscribe(gatewayevents.Filter{Type: "exec.approval.resolved"}, func(record gatewayevents.Record) {
		if len(record.Payload) == 0 {
			return
		}
		payload := append([]byte(nil), record.Payload...)
		go func() {
			var request gatewayapprovals.Request
			if err := json.Unmarshal(payload, &request); err != nil {
				return
			}
			forwardCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = telegramBotService.ForwardExecApprovalResolved(forwardCtx, request)
		}()
	})
	telegramMenuService := telegrammenu.NewMenuService(settingsService, skillsService, proxyManager.HTTPClient())
	configService := gatewayconfig.NewService(settingsService, configRevisionStore, telegramMenuService)
	configService.SetRuntimeSyncer(telegramBotService)
	gatewaymethods.RegisterConfig(gatewayRouter, configService)
	gatewayCommandService := gatewaycommands.NewService(settingsService)
	gatewaymethods.RegisterCommands(gatewayRouter, gatewayCommandService)
	gatewayRouter.Register("gateway.ping", []string{"gateway.ping"}, func(_ context.Context, _ *gatewaycontrolplane.SessionContext, _ []byte) (any, *gatewaycontrolplane.GatewayError) {
		return map[string]string{"status": "ok"}, nil
	})
	gatewayHub := gatewayws.NewHub()
	gatewayServer := gatewayws.NewServer(gatewayRouter, gatewayAuth, gatewayHub)
	gatewayServer.SetEnabledProvider(func() bool {
		current, err := settingsService.GetSettings(ctx)
		if err != nil {
			return false
		}
		return current.Gateway.ControlPlaneEnabled
	})
	gatewayEvents.SetPublisher(gatewayServer)
	realtimeServer.SetAccessGuard(func(r *http.Request) (bool, int, string) {
		if r == nil {
			return true, 0, ""
		}
		if strings.HasPrefix(r.URL.Path, "/gateway/ws") {
			current, err := settingsService.GetSettings(ctx)
			if err != nil {
				return false, http.StatusServiceUnavailable, "gateway settings unavailable"
			}
			if !current.Gateway.ControlPlaneEnabled {
				return false, http.StatusForbidden, "gateway control plane disabled"
			}
		}
		return true, 0, ""
	})
	realtimeServer.Handle("/gateway/ws", gatewayServer.Handler())
	nodeRegistryStore := noderepo.NewSQLiteNodeRegistryStore(database.Bun)
	nodeInvokeLogStore := noderepo.NewSQLiteNodeInvokeLogStore(database.Bun)
	nodeRegistry := gatewaynodes.NewRegistry(pairingService, nodeRegistryStore)
	nodeInvoker := gatewaynodes.NewDeviceInvoker(nil)
	nodeService := gatewaynodes.NewService(nodeRegistry, approvalService, nodeInvoker, nodeInvokeLogStore, gatewayEvents)
	gatewaymethods.RegisterNodes(gatewayRouter, nodeService)

	windowManager, err = wails.NewWindowManager(app, settingsService, appVersion, trayIcon, startup.launchedByAutoStart)
	if err != nil {
		return nil, err
	}
	settingsNotifier.SetApplier(func(updated settingsdto.Settings) {
		if windowManager != nil {
			windowManager.ApplySettings(updated)
		}
	})
	configService.SetSettingsApplier(windowManager)
	skillsService.SetSettingsUpdatedNotifier(func(updated settingsdto.Settings) {
		if windowManager != nil {
			windowManager.ApplySettings(updated)
		}
	})
	telegramBotService.SetModelSelectionNotifiers(
		func(updated settingsdto.Settings) {
			if windowManager != nil {
				windowManager.ApplySettings(updated)
			}
		},
		func() {
			if windowManager != nil {
				windowManager.EmitAssistantsUpdated()
			}
		},
	)

	accentCtx, accentCancel := context.WithCancel(ctx)
	app.OnShutdown(accentCancel)
	startAccentColorWatcher(accentCtx, settingsService, windowManager)

	updateCatalog := buildSoftwareUpdateService(proxyManager)
	updateService, err := buildUpdateService(proxyManager, eventBus, windowManager, updateCatalog, appVersion)
	if err != nil {
		return nil, err
	}

	settingsHandler := wails.NewSettingsHandler(settingsService, windowManager, appLogger, proxyManager, autostartManager)
	app.RegisterService(application.NewService(settingsHandler))
	noticeStore := noticerepo.NewSQLiteStore(database.Bun)
	noticeService := appnotice.NewService(noticeStore, eventBus)
	app.RegisterService(application.NewService(wails.NewNoticeHandler(noticeService)))
	providerRepo := providersrepo.NewSQLiteProviderRepository(database.Bun)
	modelRepo := providersrepo.NewSQLiteModelRepository(database.Bun)
	secretRepo := providersrepo.NewSQLiteProviderSecretRepository(database.Bun)
	usageRepo := usagerepo.NewSQLiteUsageLedgerRepository(database.Bun)
	usageService := gatewayusage.NewService(usageRepo)
	telegramBotService.SetModelRepositories(providerRepo, modelRepo)
	modelSyncer := providersync.NewEndpointModelsSyncer()
	logoCache := providersync.NewModelsDevLogoCache()
	providerService := providerservice.NewProvidersService(providerRepo, modelRepo, secretRepo, modelSyncer, logoCache)
	if err := providerService.EnsureDefaults(ctx); err != nil {
		return nil, err
	}
	providerHandler := wails.NewProviderHandler(providerService, providersUpdatedWindowNotifier{
		manager: windowManager,
	}, usageService, providersync.NewModelsDevSyncer(), telemetryService)
	app.RegisterService(application.NewService(providerHandler))
	app.RegisterService(application.NewService(wails.NewSkillsHandler(skillsService)))

	connectorsRepo := connectorsrepo.NewSQLiteConnectorRepository(database.Bun)
	connectorsService := connectorsservice.NewConnectorsService(connectorsRepo)
	if err := connectorsService.EnsureDefaults(ctx); err != nil {
		return nil, err
	}
	app.RegisterService(application.NewService(wails.NewConnectorsHandler(connectorsService, telemetryService)))

	externalToolsRepo := externaltoolsrepo.NewSQLiteExternalToolRepository(database.Bun)
	externalToolsService := externaltoolsservice.NewExternalToolsService(externalToolsRepo, updateCatalog, appVersion)
	if err := externalToolsService.EnsureDefaults(ctx); err != nil {
		return nil, err
	}
	skillsService.SetExternalTools(externalToolsService)
	app.RegisterService(application.NewService(wails.NewExternalToolsHandler(externalToolsService, windowManager, telemetryService)))

	libraryRepo := libraryrepo.NewSQLiteLibraryRepository(database.Bun)
	moduleConfigRepo := libraryrepo.NewSQLiteModuleConfigRepository(database.Bun)
	fileRepo := libraryrepo.NewSQLiteFileRepository(database.Bun)
	operationRepo := libraryrepo.NewSQLiteOperationRepository(database.Bun)
	operationChunkRepo := libraryrepo.NewSQLiteOperationChunkRepository(database.Bun)
	presetRepo := libraryrepo.NewSQLiteTranscodePresetRepository(database.Bun)
	historyRepo := libraryrepo.NewSQLiteHistoryRepository(database.Bun)
	workspaceStateRepo := libraryrepo.NewSQLiteWorkspaceStateRepository(database.Bun)
	fileEventRepo := libraryrepo.NewSQLiteFileEventRepository(database.Bun)
	subtitleDocumentRepo := libraryrepo.NewSQLiteSubtitleDocumentRepository(database.Bun)
	subtitleRevisionRepo := libraryrepo.NewSQLiteSubtitleRevisionRepository(database.Bun)
	subtitleReviewRepo := libraryrepo.NewSQLiteSubtitleReviewSessionRepository(database.Bun)
	faviconCache := libraryicons.NewFaviconCache()
	libraryService := libraryservice.NewLibraryService(
		libraryRepo,
		moduleConfigRepo,
		fileRepo,
		operationRepo,
		operationChunkRepo,
		historyRepo,
		workspaceStateRepo,
		fileEventRepo,
		subtitleDocumentRepo,
		subtitleRevisionRepo,
		subtitleReviewRepo,
		presetRepo,
		settingsService,
		faviconCache,
		externalToolsService,
		proxyManager,
		connectorsService,
		eventBus,
		telemetryService,
	)
	if err := libraryService.EnsureDefaultTranscodePresets(ctx); err != nil {
		return nil, err
	}
	app.RegisterService(application.NewService(wails.NewLibraryHandler(libraryService)))

	workspaceRepo := workspacerepo.NewSQLiteWorkspaceRepository(database.Bun)
	if err := ensureGlobalWorkspace(ctx, workspaceRepo); err != nil {
		return nil, err
	}
	threadRepo := threadrepo.NewSQLiteThreadRepository(database.Bun)
	messageRepo := threadrepo.NewSQLiteThreadMessageRepository(database.Bun)
	runRepo := threadrepo.NewSQLiteThreadRunRepository(database.Bun)
	runEventRepo := threadrepo.NewSQLiteThreadRunEventRepository(database.Bun)
	toolService := toolsservice.NewToolService()
	toolService.SetPolicy(gatewaytools.NewPolicyPipeline(settingsService))
	toolExecutor := gatewaytools.NewRegistryExecutor()
	toolService.SetExecutor(toolExecutor)
	policyAuditStore := toolpolicyrepo.NewSQLitePolicyAuditStore(database.Bun)
	sandboxService := gatewaysandbox.NewService(currentSettings.Gateway.SandboxEnabled, secure.NoopHealthChecker{})
	gatewayToolService := gatewaytools.NewService(toolService, approvalService, sandboxService, settingsNotifier, policyAuditStore, gatewayEvents)
	toolsInvokeHandler := gatewaytoolhttp.NewHandler(gatewayToolService)
	realtimeServer.Handle("/tools/invoke", toolsInvokeHandler)
	realtimeServer.Handle("/tools/invoke/", toolsInvokeHandler)
	app.RegisterService(application.NewService(wails.NewToolsHandler(toolService, gatewayToolService)))
	workspaceBaseDir, err := workspacefs.DefaultBaseDir()
	if err != nil {
		return nil, err
	}
	workspaceManager, err := workspacefs.NewManager(workspaceBaseDir)
	if err != nil {
		return nil, err
	}
	assistantRepo := assistantrepo.NewSQLiteAssistantRepository(database.Bun)
	workspaceService := workspaceservice.NewWorkspaceService(workspaceRepo, workspaceManager, assistantRepo)
	skillsService.SetWorkspaceResolver(workspaceService)
	app.RegisterService(application.NewService(wails.NewWorkspaceHandler(workspaceService)))
	assistantService := assistantservice.NewAssistantService(assistantRepo, workspaceService, settingsService)
	telegramBotService.SetAssistantService(assistantService)
	if err := assistantService.EnsureDefaults(ctx); err != nil {
		return nil, err
	}
	app.RegisterService(application.NewService(wails.NewAssistantHandler(assistantService)))
	assistantSnapshotResolver := assistantservice.NewAssistantSnapshotResolver(assistantRepo, threadRepo)
	sessionStore := sessionrepo.NewSQLiteGatewaySessionStore(database.Bun)
	sessionService := sessionmanager.NewService(sessionStore)
	threadService := threadservice.NewThreadService(
		threadRepo,
		messageRepo,
		runRepo,
		runEventRepo,
		sessionService,
		assistantRepo,
		modelRepo,
	)
	threadService.SetGatewayEventBroker(gatewayEvents)
	memoryService := memoryservice.NewMemoryService(
		database.Bun,
		settingsService,
		assistantRepo,
		threadRepo,
		messageRepo,
		providerRepo,
		modelRepo,
		secretRepo,
	)
	assistantService.SetMemorySummaryReader(memoryService)
	memoryService.SetPrincipalProfileRefresher(telegramBotService)
	memoryService.SetPrincipalAvatarNotifier(func(assistantID string, principalType string, principalID string) {
		windowManager.EmitMemoryPrincipalAvatarUpdated(map[string]any{
			"assistantId":   assistantID,
			"principalType": principalType,
			"principalId":   principalID,
		})
	})
	threadService.SetMemoryLifecycle(memoryService)
	telegramBotService.SetThreadService(threadService)
	app.RegisterService(application.NewService(wails.NewThreadHandler(threadService, eventBus, windowManager)))
	app.RegisterService(application.NewService(wails.NewMemoryHandler(memoryService)))
	agentRepo := agentrepo.NewSQLiteAgentRepository(database.Bun)
	agentService := agentservice.NewAgentService(agentRepo, threadRepo, runRepo, runEventRepo, assistantRepo)
	if err := agentService.EnsureDefaults(ctx); err != nil {
		return nil, err
	}
	app.RegisterService(application.NewService(wails.NewAgentHandler(agentService)))
	gatewayModelsService := gatewaymodels.NewService(providerRepo, modelRepo)
	gatewaymethods.RegisterModels(gatewayRouter, gatewayModelsService)
	gatewaymethods.RegisterAgents(gatewayRouter, agentService)
	purgeCtx, purgeCancel := context.WithCancel(ctx)
	app.OnShutdown(purgeCancel)
	startThreadPurgeWorker(purgeCtx, threadService)

	voiceConfigRepo := voicerepo.NewSQLiteVoiceConfigRepository(database.Bun)
	ttsJobRepo := voicerepo.NewSQLiteTTSJobRepository(database.Bun)
	voiceService := gatewayvoice.NewService(voiceConfigRepo, ttsJobRepo, usageService, settingsService, gatewayServer)
	gatewaymethods.RegisterUsage(gatewayRouter, usageService)
	gatewaymethods.RegisterVoice(gatewayRouter, voiceService)
	gatewaytools.RegisterBuiltinTools(ctx, toolService, toolExecutor, gatewaytools.BuiltinToolDeps{
		Settings:      settingsNotifier,
		Sessions:      sessionStore,
		Threads:       threadService,
		Agents:        agentService,
		Assistant:     assistantService,
		GatewayConfig: configService,
		Nodes:         nodeService,
		Voice:         voiceService,
		Library:       libraryService,
		Skills:        skillsService,
		Connectors:    connectorsService,
		ExternalTools: externalToolsService,
		Providers:     providerRepo,
		Models:        modelRepo,
		Secrets:       secretRepo,
		Memory:        memoryService,
	})
	sessionManager := sessionmanager.NewManager()
	queueStore := gatewayqueuerepo.NewSQLiteQueueStore(database.Bun)
	queueGlobal := currentSettings.Gateway.Queue.GlobalConcurrency
	queueCaps := gatewayqueue.GlobalCaps{}
	if queueGlobal > 0 {
		queueCaps.Steer = queueGlobal
		queueCaps.Followup = queueGlobal
		queueCaps.Collect = queueGlobal
	}
	queuePolicy := gatewayqueue.Policy{
		DefaultMode: string(domainsession.QueueModeFollowup),
		DefaultCap:  currentSettings.Gateway.Queue.SessionConcurrency,
		GlobalCaps:  queueCaps,
	}
	queueManager := gatewayqueue.NewManager(sessionManager, gatewayqueue.NewPolicyResolver(queuePolicy), queueStore, gatewayEvents)
	runEventSink := gatewayevents.NewRunEventSink(gatewayEvents)
	abortRegistry := gatewayruntime.NewAbortRegistry()
	controlRegistry := gatewayruntime.NewControlRegistry()
	runtimeService := gatewayruntime.NewService(
		providerRepo,
		modelRepo,
		secretRepo,
		threadRepo,
		messageRepo,
		runRepo,
		runEventRepo,
		sessionService,
		queueManager,
		gatewayToolService,
		assistantService,
		assistantSnapshotResolver,
		workspaceService,
		skillsService,
		settingsService,
		usageService,
		eventBus,
		runEventSink,
		abortRegistry,
		controlRegistry,
		threadService,
		memoryService,
		telemetryService,
	)
	libraryService.SetOneShotRuntime(runtimeService)
	libraryService.RecoverPendingJobs(ctx)
	threadService.SetTitleRuntime(runtimeService)
	gatewaymethods.RegisterRuntime(gatewayRouter, runtimeService)
	heartbeatStores := gatewayheartbeat.StoreOptions{}
	heartbeatStoreMode := strings.ToLower(strings.TrimSpace(os.Getenv("DREAMCREATOR_HEARTBEAT_STORE")))
	if heartbeatStoreMode != "memory" {
		heartbeatStores.EventStore = heartbeatrepo.NewSQLiteHeartbeatStore(database.Bun)
	}
	heartbeatStores.BusyCheck = func(ctx context.Context) (bool, string) {
		if runRepo != nil {
			type recentActiveCounter interface {
				CountActiveSince(ctx context.Context, since time.Time) (int, error)
			}
			if counter, ok := any(runRepo).(recentActiveCounter); ok {
				cutoff := time.Now().Add(-90 * time.Second)
				if count, err := counter.CountActiveSince(ctx, cutoff); err == nil && count > 0 {
					return true, "requests-in-flight"
				}
			} else if count, err := runRepo.CountActive(ctx); err == nil && count > 0 {
				return true, "requests-in-flight"
			}
		}
		return false, ""
	}
	heartbeatStores.WorkspaceResolver = workspaceService
	heartbeatService := gatewayheartbeat.NewService(
		settingsService,
		threadRepo,
		threadService,
		runtimeService,
		heartbeatStores,
		gatewayEvents,
		noticeService,
	)
	app.RegisterService(application.NewService(wails.NewHeartbeatHandler(heartbeatService)))
	gatewaymethods.RegisterHeartbeat(gatewayRouter, heartbeatService)
	telegramBotService.SetRuntime(runtimeService)
	if err := telegramBotService.Refresh(ctx); err != nil {
		zap.L().Warn("telegram runtime refresh failed", zap.Error(err))
	}
	app.OnShutdown(func() {
		telegramBotService.Close()
	})
	subagentRunStore := subagentrepo.NewSQLiteRunStore(database.Bun)
	gatewaySubagentService := gatewaysubagent.NewGatewayService(
		subagentservice.NewSpawner(),
		queueManager,
		subagentRunStore,
		gatewayEvents,
		runtimeService,
		abortRegistry,
		controlRegistry,
		threadService,
		settingsService,
		heartbeatService,
	)
	telegramBotService.SetSubagentGateway(gatewaySubagentService)
	gatewaymethods.RegisterSubagent(gatewayRouter, gatewaySubagentService)
	gatewaytools.RegisterSubagentTools(ctx, toolService, toolExecutor, gatewaytools.SubagentToolDeps{
		Gateway: gatewaySubagentService,
	})
	automationStore := automationrepo.NewSQLiteAutomationStore(database.Bun)
	automationEngine := gatewayautomation.NewEngine(queueManager, automationStore, gatewayEvents)
	notificationService := notifications.New()
	cronScheduler := gatewaycron.NewScheduler(automationStore, automationEngine)
	cronScheduler.SetAssistantIDResolver(func(ctx context.Context) string {
		if assistantService == nil {
			return ""
		}
		items, err := assistantService.ListAssistants(ctx, true)
		if err != nil || len(items) == 0 {
			return ""
		}
		for _, item := range items {
			if item.IsDefault {
				return strings.TrimSpace(item.ID)
			}
		}
		return strings.TrimSpace(items[0].ID)
	})
	cronScheduler.SetMainSystemEventEnqueuer(func(ctx context.Context, request gatewaycron.MainSystemEventRequest) bool {
		if heartbeatService == nil {
			return false
		}
		return heartbeatService.EnqueueSystemEvent(ctx, gatewayheartbeat.SystemEventInput{
			SessionKey: strings.TrimSpace(request.SessionKey),
			Text:       strings.TrimSpace(request.Text),
			ContextKey: strings.TrimSpace(request.ContextKey),
			RunID:      strings.TrimSpace(request.RunID),
			Source:     "cron",
		})
	})
	cronScheduler.SetWakeTrigger(func(ctx context.Context, request gatewaycron.WakeTriggerRequest) gatewaycron.WakeTriggerResult {
		if heartbeatService == nil {
			return gatewaycron.WakeTriggerResult{
				Accepted:       false,
				ExecutedStatus: "skipped",
				Reason:         "heartbeat unavailable",
			}
		}
		result := heartbeatService.TriggerWithResult(ctx, gatewayheartbeat.TriggerInput{
			Reason:     strings.TrimSpace(request.Reason),
			SessionKey: strings.TrimSpace(request.SessionKey),
			Force:      request.Force,
			Source:     strings.TrimSpace(request.Source),
			RunID:      strings.TrimSpace(request.RunID),
		})
		return gatewaycron.WakeTriggerResult{
			Accepted:       result.Accepted,
			ExecutedStatus: strings.TrimSpace(string(result.ExecutedStatus)),
			Reason:         strings.TrimSpace(result.Reason),
		}
	})
	cronScheduler.SetAnnouncementSender(func(ctx context.Context, request gatewaycron.AnnouncementRequest) error {
		entry, ok := resolveCronAnnouncementSession(ctx, sessionStore, request)
		if !ok {
			return errors.New("cron delivery session not found")
		}
		channel := resolveCronAnnouncementDeliveryChannel(strings.TrimSpace(request.Channel), entry)
		switch channel {
		case "telegram":
			accountID, chatID, threadID, ok := resolveCronTelegramTarget(entry, strings.TrimSpace(request.SessionKey))
			if !ok {
				return errors.New("cron telegram delivery target unavailable")
			}
			return telegramBotService.SendText(ctx, telegrammenu.SendTextInput{
				AccountID: accountID,
				ChatID:    chatID,
				ThreadID:  threadID,
				Text:      strings.TrimSpace(request.Message),
			})
		case "app":
			if notificationService == nil {
				return errors.New("notification service unavailable")
			}
			title := strings.TrimSpace(request.JobName)
			if title == "" {
				title = "Cron"
			}
			body := strings.TrimSpace(request.Message)
			if body == "" {
				return errors.New("cron announcement message is empty")
			}
			notificationID := strings.TrimSpace(request.RunID)
			if notificationID == "" {
				notificationID = time.Now().Format(time.RFC3339Nano)
			}
			return notificationService.SendNotification(notifications.NotificationOptions{
				ID:       notificationID,
				Title:    title,
				Subtitle: "Cron Delivery",
				Body:     body,
				Data: map[string]any{
					"source":      "cron",
					"jobId":       strings.TrimSpace(request.JobID),
					"jobName":     strings.TrimSpace(request.JobName),
					"assistantId": strings.TrimSpace(request.AssistantID),
					"sessionKey":  strings.TrimSpace(request.SessionKey),
					"channel":     channel,
				},
			})
		default:
			return fmt.Errorf("cron delivery channel unsupported: %s", channel)
		}
	})
	cronScheduler.SetWebhookSender(func(ctx context.Context, request gatewaycron.WebhookRequest) error {
		targetURL := strings.TrimSpace(request.URL)
		lowerTarget := strings.ToLower(targetURL)
		if targetURL == "" || (!strings.HasPrefix(lowerTarget, "http://") && !strings.HasPrefix(lowerTarget, "https://")) {
			return errors.New("cron webhook url must start with http:// or https://")
		}
		payload := request.Payload
		if payload == nil {
			payload = map[string]any{
				"runId":       strings.TrimSpace(request.RunID),
				"jobId":       strings.TrimSpace(request.JobID),
				"jobName":     strings.TrimSpace(request.JobName),
				"assistantId": strings.TrimSpace(request.AssistantID),
				"status":      strings.TrimSpace(request.Status),
				"summary":     strings.TrimSpace(request.Summary),
				"error":       strings.TrimSpace(request.Error),
				"sessionKey":  strings.TrimSpace(request.SessionKey),
			}
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		sendCtx := ctx
		if sendCtx == nil {
			sendCtx = context.Background()
		}
		sendCtx, cancel := context.WithTimeout(sendCtx, 10*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(sendCtx, http.MethodPost, targetURL, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		client := proxyManager.HTTPClient()
		if client == nil {
			client = http.DefaultClient
		}
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1024))
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("cron webhook returned status %d", response.StatusCode)
		}
		return nil
	})
	cronScheduler.SetIsolatedExecutor(func(ctx context.Context, request gatewaycron.IsolatedExecutionRequest) (gatewaycron.IsolatedExecutionResult, error) {
		if runtimeService == nil {
			err := errors.New("runtime service unavailable")
			return gatewaycron.IsolatedExecutionResult{
				Status: "failed",
				Error:  err.Error(),
			}, err
		}
		message := strings.TrimSpace(request.Message)
		if message == "" {
			err := errors.New("payload.message is required when payload.kind=agentTurn")
			return gatewaycron.IsolatedExecutionResult{
				Status: "failed",
				Error:  err.Error(),
			}, err
		}

		modelSelection := resolveCronRuntimeModelSelection(strings.TrimSpace(request.Model))

		sessionKey := strings.TrimSpace(request.SessionKey)
		if sessionKey == "" {
			sessionKey = "cron/isolated"
		}

		metadata := map[string]any{
			"cron":            true,
			"runKind":         "cron",
			"queueLane":       gatewayqueue.LaneCron,
			"source":          "cron",
			"persistRun":      true,
			"persistMessages": true,
			"persistEvents":   true,
			"persistUsage":    true,
			"useQueue":        true,
			"cronJobId":       strings.TrimSpace(request.JobID),
			"cronRunId":       strings.TrimSpace(request.RunID),
		}
		if request.TimeoutSeconds > 0 {
			metadata["timeoutSeconds"] = request.TimeoutSeconds
		}
		if request.LightContext {
			metadata["lightContext"] = true
		}
		if request.Delivery != nil {
			metadata["cronDelivery"] = request.Delivery
		}

		promptMode := ""
		if request.LightContext {
			promptMode = "minimal"
		}
		runRequest := gatewayruntimedto.RuntimeRunRequest{
			RunID:       strings.TrimSpace(request.RunID),
			SessionKey:  sessionKey,
			AssistantID: strings.TrimSpace(request.AssistantID),
			RunKind:     "cron",
			PromptMode:  promptMode,
			Input: gatewayruntimedto.RuntimeInput{
				Messages: []gatewayruntimedto.Message{{
					Role:    "user",
					Content: message,
				}},
			},
			Metadata: metadata,
		}
		if modelSelection != nil {
			runRequest.Model = modelSelection
		}
		if thinking := strings.TrimSpace(request.Thinking); thinking != "" {
			runRequest.Thinking = gatewayruntimedto.ThinkingConfig{Mode: thinking}
			metadata["thinking"] = thinking
		}

		runCtx := ctx
		if runCtx == nil {
			runCtx = context.Background()
		}
		if request.TimeoutSeconds > 0 {
			var cancel context.CancelFunc
			runCtx, cancel = context.WithTimeout(runCtx, time.Duration(request.TimeoutSeconds)*time.Second)
			defer cancel()
		}

		runtimeResult, err := runtimeService.Run(runCtx, runRequest)
		response := gatewaycron.IsolatedExecutionResult{
			Status:     strings.TrimSpace(runtimeResult.Status),
			Error:      strings.TrimSpace(runtimeResult.Error),
			Summary:    strings.TrimSpace(runtimeResult.AssistantMessage.Content),
			SessionKey: sessionKey,
		}
		if runtimeResult.Model != nil {
			response.Provider = strings.TrimSpace(runtimeResult.Model.ProviderID)
			response.Model = strings.TrimSpace(runtimeResult.Model.Name)
		}
		if response.Summary == "" {
			response.Summary = strings.TrimSpace(runtimeResult.Error)
		}
		if usageJSON, usageErr := marshalCronRuntimeUsage(runtimeResult.Usage); usageErr == nil {
			response.UsageJSON = usageJSON
		}
		if err != nil {
			if response.Error == "" {
				response.Error = strings.TrimSpace(err.Error())
			}
			if response.Status == "" {
				response.Status = "failed"
			}
			return response, err
		}
		if response.Status == "" {
			if response.Error != "" {
				response.Status = "failed"
			} else {
				response.Status = "completed"
			}
		}
		if strings.EqualFold(response.Status, "failed") && response.Error == "" {
			response.Error = "isolated cron execution failed"
		}
		return response, nil
	})
	cronScheduler.SetRunRealtimeNotifier(func(ctx context.Context, event gatewaycron.RunRealtimeEvent) {
		if gatewayEvents == nil {
			return
		}
		stage := strings.ToLower(strings.TrimSpace(event.Stage))
		switch stage {
		case "job_upserted", "job_deleted", "job_running", "job_state_updated", "job_rescheduled":
			statusEnvelope := appevents.NewGatewayEventEnvelope("cron.scheduler", "cron.status")
			statusEnvelope.RunID = strings.TrimSpace(event.RunID)
			_, _ = gatewayEvents.Publish(ctx, statusEnvelope, event)
			listEnvelope := appevents.NewGatewayEventEnvelope("cron.scheduler", "cron.list")
			listEnvelope.RunID = strings.TrimSpace(event.RunID)
			_, _ = gatewayEvents.Publish(ctx, listEnvelope, event)
		}
		runID := strings.TrimSpace(event.RunID)
		if runID == "" {
			return
		}
		detailEnvelope := appevents.NewGatewayEventEnvelope("cron.run", "cron.runDetail")
		detailEnvelope.RunID = runID
		_, _ = gatewayEvents.Publish(ctx, detailEnvelope, event)
		runsEnvelope := appevents.NewGatewayEventEnvelope("cron.run", "cron.runs")
		runsEnvelope.RunID = runID
		_, _ = gatewayEvents.Publish(ctx, runsEnvelope, event)
	})
	gatewayEvents.Subscribe(gatewayevents.Filter{Type: "heartbeat.event"}, func(record gatewayevents.Record) {
		if len(record.Payload) == 0 {
			return
		}
		payload := append([]byte(nil), record.Payload...)
		go func() {
			event, ok := gatewaycron.DecodeHeartbeatDeliveryEvent(payload)
			if !ok {
				return
			}
			deliverCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cronScheduler.HandleHeartbeatDeliveryEvent(deliverCtx, event)
		}()
	})
	cronScheduler.SetEnabled(currentSettings.Gateway.Cron.Enabled)
	startCronEnabledWatcher(serverCtx, settingsService, cronScheduler)
	gatewaymethods.RegisterCron(gatewayRouter, cronScheduler)
	gatewaytools.RegisterCronTool(ctx, toolService, toolExecutor, cronScheduler)
	cronStop := cronScheduler.Start(serverCtx)
	app.OnShutdown(cronStop)
	channelRegistry := gatewaychannels.NewRegistry(gatewayServer)
	channelRegistry.Register(gatewaychannels.ChannelDescriptor{
		ChannelID:    "telegram",
		DisplayName:  "Telegram",
		Kind:         "telegram",
		Capabilities: []string{"inbound", "outbound"},
		Enabled:      false,
	}, gatewaychannels.ChannelHandlers{
		Status: func(ctx context.Context) (gatewaychannels.ChannelStatus, error) {
			state, lastError, updatedAt := telegramBotService.Status(ctx)
			return gatewaychannels.ChannelStatus{
				ChannelID: "telegram",
				State:     state,
				LastError: lastError,
				UpdatedAt: updatedAt,
			}, nil
		},
		Logout: func(ctx context.Context) error {
			return telegramBotService.Logout(ctx)
		},
		Probe: func(ctx context.Context) (gatewaychannels.ChannelProbeResult, error) {
			ok, errMsg, latency, err := telegramBotService.Probe(ctx)
			result := gatewaychannels.ChannelProbeResult{
				ChannelID: "telegram",
				Success:   ok,
				State:     "offline",
				CheckedAt: time.Now(),
			}
			if ok {
				result.State = "online"
			}
			if errMsg != "" {
				result.Error = errMsg
			}
			if latency > 0 {
				result.CheckedAt = time.Now()
			}
			return result, err
		},
	})
	channelRegistry.Register(gatewaychannels.ChannelDescriptor{
		ChannelID:    "web",
		DisplayName:  "Webhook",
		Kind:         "webhook",
		Capabilities: []string{"inbound"},
		Enabled:      false,
	}, gatewaychannels.ChannelHandlers{
		Status: func(ctx context.Context) (gatewaychannels.ChannelStatus, error) {
			return gatewaychannels.ChannelStatus{ChannelID: "web", State: "online", UpdatedAt: time.Now()}, nil
		},
		Webhook: func(ctx context.Context, request gatewaychannels.WebhookIngestRequest) (gatewaychannels.WebhookIngestResult, error) {
			_, event, err := queueManager.Enqueue(ctx, gatewayqueue.EnqueueRequest{
				SessionKey: request.SessionKey,
				Mode:       string(domainsession.QueueModeFollowup),
				Payload: map[string]any{
					"channelId": request.ChannelID,
					"eventId":   request.EventID,
					"payload":   request.Payload,
				},
			})
			return gatewaychannels.WebhookIngestResult{
				EventID:    request.EventID,
				SessionKey: request.SessionKey,
				Queued:     err == nil,
				QueuedAt:   event.Timestamp,
			}, err
		},
	})
	heartbeatService.SetChannelReadyCheck(func(ctx context.Context, channelID string, accountID string) (bool, string) {
		normalizedChannelID := strings.TrimSpace(strings.ToLower(channelID))
		if normalizedChannelID == "" {
			return false, "channel_not_ready"
		}
		if ready, reason := resolveHeartbeatAccountReadiness(ctx, settingsService, normalizedChannelID, accountID); !ready {
			return false, reason
		}
		statuses, err := channelRegistry.StatusAll(ctx)
		if err != nil {
			return false, "channel_status_unavailable"
		}
		for _, status := range statuses {
			if strings.TrimSpace(strings.ToLower(status.ChannelID)) != normalizedChannelID {
				continue
			}
			state := strings.TrimSpace(strings.ToLower(status.State))
			switch state {
			case "online", "ready", "connected":
				return true, ""
			default:
				if reason := strings.TrimSpace(status.LastError); reason != "" {
					return false, reason
				}
				if state == "" {
					return false, "channel_not_ready"
				}
				return false, state
			}
		}
		return false, "channel_not_registered"
	})
	gatewaymethods.RegisterChannels(gatewayRouter, channelRegistry)
	channelDebugService := gatewaychannels.NewDebugService(channelRegistry)
	channelDebugService.RegisterProvider(telegrammenu.NewDebugProvider(telegramBotService))
	gatewaymethods.RegisterChannelsDebug(gatewayRouter, channelDebugService)
	gatewaymethods.RegisterChannelMenus(gatewayRouter, telegramMenuService)
	if telegramPairingStore != nil {
		telegramPairingService := telegrammenu.NewPairingService(
			settingsService,
			telegramBotService,
			telegramPairingStore,
			proxyManager.HTTPClient(),
		)
		telegramPairingService.SetSettingsUpdatedNotifier(func(updated settingsdto.Settings) {
			if windowManager != nil {
				windowManager.ApplySettings(updated)
			}
		})
		gatewaymethods.RegisterChannelPairing(gatewayRouter, telegramPairingService)
	}
	heartbeatRunner := gatewayautomation.NewHeartbeatRunner(sessionManager, runRepo)
	heartbeatRunner.ConfigureAutomation(automationEngine)
	heartbeatRunner.SetHeartbeatService(heartbeatService)
	heartbeatStop := heartbeatRunner.Start(serverCtx)
	app.OnShutdown(heartbeatStop)
	diagnosticsStore := diagnosticsrepo.NewSQLiteReportStore(database.Bun)
	observabilityService := gatewayobservability.NewService(sessionManager, runRepo, channelRegistry, nodeService, appLogger, diagnosticsStore)
	telegramHandler := telegramchannel.NewWebhookHandler(telegramBotService)
	webhookHandler := webchannel.NewWebhookHandler(channelRegistry, settingsService)
	openAIHandler := openaihttp.NewHandler(runtimeService)
	openResponsesHandler := openresponseshttp.NewHandler(runtimeService)
	realtimeServer.Handle("/v1/chat/completions", openAIHandler)
	realtimeServer.Handle("/v1/chat/completions/", openAIHandler)
	realtimeServer.Handle("/v1/responses", openResponsesHandler)
	realtimeServer.Handle("/v1/responses/", openResponsesHandler)
	threadAPIHandler := presentationhttp.NewThreadAPIHandler(threadService)
	realtimeServer.Handle("/api/threads", threadAPIHandler)
	realtimeServer.Handle("/api/threads/", threadAPIHandler)
	realtimeServer.Handle("/api/channels/telegram", telegramHandler)
	realtimeServer.Handle("/api/channels/telegram/", telegramHandler)
	realtimeServer.Handle("/api/channels/webhook", webhookHandler)
	realtimeServer.Handle("/api/channels/webhook/", webhookHandler)
	realtimeServer.Handle("/api/health", presentationhttp.NewHealthHandler(observabilityService))
	realtimeServer.Handle("/api/status", presentationhttp.NewStatusHandler(observabilityService))
	realtimeServer.Handle("/api/logs/tail", presentationhttp.NewLogsTailHandler(observabilityService))
	realtimeServer.Handle("/api/logs/tail/", presentationhttp.NewLogsTailHandler(observabilityService))
	realtimeServer.Handle("/api/library/asset", presentationhttp.NewLibraryAssetHandler())
	realtimeServer.Handle("/api/library/asset/", presentationhttp.NewLibraryAssetHandler())
	realtimeServer.Handle("/api/memory/avatar", presentationhttp.NewMemoryAvatarHandler(memoryService))
	libraryBridge := gatewayevents.NewLibraryEventBridge(eventBus, operationRepo, gatewayEvents)
	libraryBridgeCancel := libraryBridge.Start()
	app.OnShutdown(libraryBridgeCancel)

	fonts := fontservice.NewFontService()
	systemHandler := wails.NewSystemHandler(fonts, eventBus)
	app.RegisterService(application.NewService(systemHandler))
	app.RegisterService(application.NewService(notificationService))
	app.RegisterService(application.NewService(wails.NewRealtimeHandler(realtimeServer, eventBus, notificationService)))
	app.RegisterService(application.NewService(wails.NewTelemetryHandler(telemetryService, apptelemetry.AppLaunchContext{
		LaunchedByAutoStart: startup.launchedByAutoStart,
	})))
	app.RegisterService(application.NewService(wails.NewUpdateHandler(updateService, telemetryService, app)))

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(_ *application.ApplicationEvent) {
		updateService.PublishCurrentState()
		updateService.ScheduleAutoCheck(ctx, 10*time.Second, time.Hour, appVersion)
	})
	app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(_ *application.ApplicationEvent) {
		updated, err := settingsService.GetSettings(ctx)
		if err != nil {
			return
		}
		windowManager.ApplySettings(updated)
	})

	return app, nil
}

func openDatabase(ctx context.Context) (*persistence.Database, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	appDir := filepath.Join(configDir, "dreamcreator")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, err
	}

	path := filepath.Join(appDir, "data.db")
	return persistence.OpenSQLite(ctx, persistence.SQLiteConfig{Path: path})
}

func ensureGlobalWorkspace(ctx context.Context, repo workspace.Repository) error {
	_, err := repo.GetGlobal(ctx)
	if err == nil {
		return nil
	}
	if err != workspace.ErrWorkspaceNotFound {
		return err
	}

	now := time.Now()
	global, err := workspace.NewGlobalWorkspace(workspace.GlobalWorkspaceParams{
		ID:        1,
		CreatedAt: &now,
		UpdatedAt: &now,
	})
	if err != nil {
		return err
	}
	return repo.SaveGlobal(ctx, global)
}

func buildSoftwareUpdateService(proxyManager *proxy.Manager) *softwareupdate.Service {
	httpClient := proxyManager.HTTPClient()
	return softwareupdate.NewService(softwareupdate.ServiceParams{
		CatalogProvider:      infrastructureupdate.NewManifestCatalogProvider(httpClient, ""),
		AppFallbackProvider:  infrastructureupdate.NewGithubReleaseClient(httpClient),
		ToolFallbackProvider: infrastructureupdate.NewToolFallbackProvider(httpClient),
	})
}

func buildUpdateService(proxyManager *proxy.Manager, bus appevents.Bus, notifier applicationupdate.Notifier, catalog *softwareupdate.Service, currentVersion string) (*applicationupdate.Service, error) {
	httpClient := proxyManager.HTTPClient()
	downloader := infrastructureupdate.NewHTTPDownloader(httpClient)
	installer, err := infrastructureupdate.NewInstaller("")
	if err != nil {
		return nil, err
	}

	service := applicationupdate.NewService(applicationupdate.ServiceParams{
		Catalog:    catalog,
		Downloader: downloader,
		Installer:  installer,
		Bus:        bus,
		Notifier:   notifier,
	})
	service.SetCurrentVersion(currentVersion)
	return service, nil
}

func startCronEnabledWatcher(ctx context.Context, settingsService *service.SettingsService, scheduler *gatewaycron.Scheduler) {
	if settingsService == nil || scheduler == nil {
		return
	}
	initial, err := settingsService.GetSettings(ctx)
	lastEnabled := false
	if err == nil {
		lastEnabled = initial.Gateway.Cron.Enabled
		scheduler.SetEnabled(lastEnabled)
	}

	ticker := time.NewTicker(2 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current, err := settingsService.GetSettings(ctx)
				if err != nil {
					continue
				}
				enabled := current.Gateway.Cron.Enabled
				if enabled == lastEnabled {
					continue
				}
				lastEnabled = enabled
				scheduler.SetEnabled(enabled)
			}
		}
	}()
}

func startAccentColorWatcher(ctx context.Context, settingsService *service.SettingsService, windowManager *wails.WindowManager) {
	initial, err := settingsService.GetSettings(ctx)
	lastAccent := ""
	if err == nil {
		lastAccent = strings.ToLower(strings.TrimSpace(initial.SystemThemeColor))
	}

	ticker := time.NewTicker(2 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				current, err := settingsService.GetSettings(ctx)
				if err != nil {
					continue
				}
				if !settings.IsSystemThemeColor(current.ThemeColor) {
					continue
				}
				accent := strings.ToLower(strings.TrimSpace(current.SystemThemeColor))
				if accent == "" || accent == lastAccent {
					continue
				}
				lastAccent = accent
				windowManager.ApplySettings(current)
			}
		}
	}()
}

func resolveVersion(env string) string {
	if v := strings.TrimSpace(os.Getenv("APP_VERSION")); v != "" {
		return v
	}
	if env == "dev" || env == "development" {
		return "dev"
	}
	v := strings.TrimSpace(AppVersion)
	if v == "" {
		return "dev"
	}
	return v
}

func resolveHeartbeatAccountReadiness(
	ctx context.Context,
	settingsService *service.SettingsService,
	channelID string,
	accountID string,
) (bool, string) {
	normalizedChannel := strings.TrimSpace(strings.ToLower(channelID))
	requestedAccountID := strings.TrimSpace(accountID)
	if requestedAccountID == "" {
		return true, ""
	}
	if settingsService == nil {
		return false, "account_config_unavailable"
	}
	current, err := settingsService.GetSettings(ctx)
	if err != nil {
		return false, "account_config_unavailable"
	}

	if normalizedChannel == "telegram" {
		return resolveHeartbeatTelegramAccountReadiness(current, requestedAccountID)
	}

	channelCfgRaw, ok := current.Channels[normalizedChannel]
	if !ok {
		return true, ""
	}
	channelCfg, ok := channelCfgRaw.(map[string]any)
	if !ok || len(channelCfg) == 0 {
		return true, ""
	}
	accountsRaw, _ := channelCfg["accounts"].(map[string]any)
	if len(accountsRaw) == 0 {
		return true, ""
	}
	accountCfg, found := resolveHeartbeatAccountConfig(accountsRaw, requestedAccountID)
	if !found {
		return false, "unknown_account"
	}
	if enabled, hasEnabled := resolveHeartbeatEnabledFlag(accountCfg["enabled"]); hasEnabled && !enabled {
		return false, "account_disabled"
	}
	return true, ""
}

func resolveHeartbeatTelegramAccountReadiness(current settingsdto.Settings, requestedAccountID string) (bool, string) {
	runtime := telegrammenu.ResolveTelegramRuntimeConfig(current)
	if len(runtime.Accounts) == 0 {
		return false, "unknown_account"
	}

	for _, account := range runtime.Accounts {
		if !strings.EqualFold(strings.TrimSpace(account.AccountID), requestedAccountID) {
			continue
		}
		if !account.Enabled {
			return false, "account_disabled"
		}
		if strings.TrimSpace(account.BotToken) == "" {
			return false, "account_token_missing"
		}
		return true, ""
	}
	return false, "unknown_account"
}

func resolveHeartbeatAccountConfig(accounts map[string]any, accountID string) (map[string]any, bool) {
	for key, raw := range accounts {
		if !strings.EqualFold(strings.TrimSpace(key), accountID) {
			continue
		}
		cfg, _ := raw.(map[string]any)
		return cfg, true
	}
	return nil, false
}

func resolveHeartbeatEnabledFlag(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes", "on", "enabled":
			return true, true
		case "false", "0", "no", "off", "disabled":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func resolveCronAnnouncementSession(
	ctx context.Context,
	store interface {
		List(ctx context.Context) ([]domainsession.Entry, error)
	},
	request gatewaycron.AnnouncementRequest,
) (domainsession.Entry, bool) {
	if store == nil {
		return domainsession.Entry{}, false
	}
	entries, err := store.List(ctx)
	if err != nil || len(entries) == 0 {
		return domainsession.Entry{}, false
	}
	targetSessionKey := strings.TrimSpace(request.SessionKey)
	if targetSessionKey != "" {
		normalizedTarget := targetSessionKey
		if _, normalized, normalizeErr := domainsession.NormalizeSessionKey(targetSessionKey); normalizeErr == nil {
			normalizedTarget = strings.TrimSpace(normalized)
		}
		for _, entry := range entries {
			if strings.EqualFold(strings.TrimSpace(entry.SessionKey), targetSessionKey) ||
				strings.EqualFold(strings.TrimSpace(entry.SessionKey), normalizedTarget) ||
				strings.EqualFold(strings.TrimSpace(entry.SessionID), targetSessionKey) {
				return entry, true
			}
		}
		if !isSyntheticCronAnnouncementSessionKey(targetSessionKey) {
			return domainsession.Entry{}, false
		}
	}

	assistantID := strings.TrimSpace(request.AssistantID)
	for _, entry := range entries {
		if assistantID != "" && !strings.EqualFold(strings.TrimSpace(entry.AssistantID), assistantID) {
			continue
		}
		channel := resolveCronSessionChannel(strings.TrimSpace(entry.SessionKey), strings.TrimSpace(entry.Origin.Channel))
		if channel == "" {
			continue
		}
		return entry, true
	}
	return domainsession.Entry{}, false
}

func resolveCronSessionChannel(sessionKey string, originChannel string) string {
	if parts, _, err := domainsession.NormalizeSessionKey(strings.TrimSpace(sessionKey)); err == nil {
		if channel := normalizeCronChannelAlias(strings.ToLower(strings.TrimSpace(parts.Channel))); channel != "" {
			return channel
		}
	}
	return normalizeCronChannelAlias(strings.ToLower(strings.TrimSpace(originChannel)))
}

func resolveCronAnnouncementDeliveryChannel(requestChannel string, entry domainsession.Entry) string {
	normalizedRequested := normalizeCronDeliveryChannel(requestChannel)
	switch normalizedRequested {
	case "", "default":
		return resolveCronSessionChannel(strings.TrimSpace(entry.SessionKey), strings.TrimSpace(entry.Origin.Channel))
	default:
		return normalizedRequested
	}
}

func normalizeCronChannelAlias(channel string) string {
	normalized := strings.ToLower(strings.TrimSpace(channel))
	switch normalized {
	case "app", "aui":
		return "app"
	default:
		return normalized
	}
}

func normalizeCronDeliveryChannel(channel string) string {
	normalized := strings.ToLower(strings.TrimSpace(channel))
	switch normalized {
	case "", "default":
		return "default"
	case "app", "aui":
		return "app"
	case "telegram":
		return "telegram"
	default:
		return normalized
	}
}

func isSyntheticCronAnnouncementSessionKey(sessionKey string) bool {
	switch strings.ToLower(strings.TrimSpace(sessionKey)) {
	case "cron/main", "cron/isolated", "cron/default":
		return true
	default:
		return false
	}
}

func resolveCronTelegramTarget(entry domainsession.Entry, overrideSessionKey string) (string, int64, int, bool) {
	sessionKey := strings.TrimSpace(overrideSessionKey)
	if sessionKey == "" {
		sessionKey = strings.TrimSpace(entry.SessionKey)
	}
	accountID, chatID, threadID, ok := parseCronTelegramTargetFromSessionKey(sessionKey)
	if ok {
		if strings.TrimSpace(accountID) == "" {
			accountID = strings.TrimSpace(entry.Origin.AccountID)
		}
		return strings.TrimSpace(accountID), chatID, threadID, true
	}
	originPeerID := strings.TrimSpace(entry.Origin.PeerID)
	if originPeerID == "" {
		return "", 0, 0, false
	}
	chatID, err := strconv.ParseInt(originPeerID, 10, 64)
	if err != nil || chatID == 0 {
		return "", 0, 0, false
	}
	return strings.TrimSpace(entry.Origin.AccountID), chatID, 0, true
}

func parseCronTelegramTargetFromSessionKey(sessionKey string) (string, int64, int, bool) {
	parts, _, err := domainsession.NormalizeSessionKey(strings.TrimSpace(sessionKey))
	if err != nil {
		return "", 0, 0, false
	}
	if !strings.EqualFold(strings.TrimSpace(parts.Channel), "telegram") {
		return "", 0, 0, false
	}
	accountID := strings.TrimSpace(parts.AccountID)
	primary := strings.TrimSpace(parts.PrimaryID)
	if primary == "" {
		primary = strings.TrimSpace(parts.ThreadRef)
	}
	segments := strings.Split(primary, ":")
	if len(segments) < 4 || !strings.EqualFold(strings.TrimSpace(segments[0]), "telegram") {
		return accountID, 0, 0, false
	}
	legacyAccountID := strings.TrimSpace(segments[1])
	if accountID == "" {
		accountID = legacyAccountID
	}
	chatID, parseErr := strconv.ParseInt(strings.TrimSpace(segments[3]), 10, 64)
	if parseErr != nil || chatID == 0 {
		return accountID, 0, 0, false
	}
	threadID := 0
	if len(segments) >= 6 && strings.EqualFold(strings.TrimSpace(segments[4]), "thread") {
		if parsedThreadID, threadErr := strconv.ParseInt(strings.TrimSpace(segments[5]), 10, 64); threadErr == nil && parsedThreadID > 0 {
			threadID = int(parsedThreadID)
		}
	}
	return accountID, chatID, threadID, true
}

func resolveCronRuntimeModelSelection(primary string) *gatewayruntimedto.ModelSelection {
	if providerID, modelName, ok := parseCronModelRef(primary); ok {
		return &gatewayruntimedto.ModelSelection{
			ProviderID: providerID,
			Name:       modelName,
		}
	}
	return nil
}

func parseCronModelRef(value string) (string, string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", "", false
	}
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		providerID := strings.TrimSpace(parts[0])
		modelName := strings.TrimSpace(parts[1])
		if providerID == "" || modelName == "" {
			return "", "", false
		}
		return providerID, modelName, true
	}
	if strings.Contains(trimmed, ":") {
		parts := strings.SplitN(trimmed, ":", 2)
		providerID := strings.TrimSpace(parts[0])
		modelName := strings.TrimSpace(parts[1])
		if providerID == "" || modelName == "" {
			return "", "", false
		}
		return providerID, modelName, true
	}
	return "", "", false
}
