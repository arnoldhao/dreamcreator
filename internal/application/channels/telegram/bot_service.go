package telegram

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	assistantdto "dreamcreator/internal/application/assistant/dto"
	channelpairing "dreamcreator/internal/application/channels/pairing"
	"dreamcreator/internal/application/chatevent"
	appcommands "dreamcreator/internal/application/commands"
	gatewayapprovals "dreamcreator/internal/application/gateway/approvals"
	runtimedto "dreamcreator/internal/application/gateway/runtime/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	skillsdto "dreamcreator/internal/application/skills/dto"
	subagentservice "dreamcreator/internal/application/subagent/service"
	threaddto "dreamcreator/internal/application/thread/dto"
	domainproviders "dreamcreator/internal/domain/providers"
	domainsession "dreamcreator/internal/domain/session"
	telegramapi "dreamcreator/internal/infrastructure/telegram"
	"github.com/google/uuid"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"go.uber.org/zap"
)

type Runtime interface {
	Run(ctx context.Context, request runtimedto.RuntimeRunRequest) (runtimedto.RuntimeRunResult, error)
}

type SendTextInput struct {
	AccountID string
	ChatID    int64
	ThreadID  int
	Text      string
	Silent    bool
}

type StreamingRuntime interface {
	RunStream(
		ctx context.Context,
		request runtimedto.RuntimeRunRequest,
		callback runtimedto.RuntimeStreamCallback,
	) (runtimedto.RuntimeRunResult, error)
}

type ExecApprovalResolver interface {
	Resolve(ctx context.Context, id string, decision string, reason string) (gatewayapprovals.Request, error)
}

type ProviderRepository interface {
	List(ctx context.Context) ([]domainproviders.Provider, error)
}

type ModelRepository interface {
	ListByProvider(ctx context.Context, providerID string) ([]domainproviders.Model, error)
}

type SubagentGateway interface {
	Get(ctx context.Context, runID string) (subagentservice.RunRecord, error)
	ListByParent(ctx context.Context, parentSessionKey string) ([]subagentservice.RunRecord, error)
	Kill(ctx context.Context, runID string) error
	KillAll(ctx context.Context, parentSessionKey string) (int, error)
	Steer(ctx context.Context, runID string, message string) error
	Send(ctx context.Context, runID string, message string) error
}

type AssistantService interface {
	ListAssistants(ctx context.Context, includeDisabled bool) ([]assistantdto.Assistant, error)
	UpdateAssistant(ctx context.Context, request assistantdto.UpdateAssistantRequest) (assistantdto.Assistant, error)
}

type SkillPromptResolver interface {
	ResolveSkillPromptItems(ctx context.Context, request skillsdto.ResolveSkillPromptRequest) (skillsdto.ResolveSkillPromptResponse, error)
}

type ThreadService interface {
	ListThreads(ctx context.Context, includeDeleted bool) ([]threaddto.Thread, error)
}

type BotService struct {
	settings            *settingsservice.SettingsService
	runtime             Runtime
	approvals           ExecApprovalResolver
	providers           ProviderRepository
	models              ModelRepository
	subagents           SubagentGateway
	assistants          AssistantService
	skills              SkillPromptResolver
	threads             ThreadService
	httpClient          *http.Client
	now                 func() time.Time
	pairing             *channelpairing.Store
	offsets             *UpdateOffsetStore
	onSettingsUpdated   func(settingsdto.Settings)
	onAssistantsUpdated func()

	mu           sync.Mutex
	accounts     map[string]*telegramAccountState
	lastRefresh  time.Time
	refreshAfter time.Duration
	activeRuns   map[string]telegramActiveRun
	sessionRuns  map[string]string
	commandState map[string]telegramSessionCommandState
	statusCards  map[string]int
	runStatus    map[string]int
}

type telegramAccountState struct {
	config                TelegramAccountConfig
	cancel                context.CancelFunc
	running               bool
	mode                  string
	bot                   *telego.Bot
	botID                 int64
	botUsername           string
	lastError             string
	lastErrorAt           time.Time
	lastInbound           time.Time
	lastOutbound          time.Time
	updateOffset          int64
	replyTracker          *replyTracker
	placeholders          sync.Map
	inboundCount          int
	outboundCount         int
	deniedCount           int
	errorCount            int
	lastInboundType       string
	lastInboundUpdateID   int64
	lastInboundMessageID  int64
	lastInboundChatID     int64
	lastInboundUserID     string
	lastInboundCommand    string
	lastDeniedReason      string
	lastDeniedAt          time.Time
	lastRunID             string
	lastRunError          string
	lastRunAt             time.Time
	lastOutboundMessageID int64
	lastOutboundError     string
	peerProfileCache      sync.Map
}

type telegramActiveRun struct {
	AccountID  string
	SessionKey string
	Cancel     context.CancelFunc
	StartedAt  time.Time
}

type telegramSessionCommandState struct {
	ConversationID  string
	SessionOverride string
	ModelProvider   string
	ModelName       string
	ThinkingLevel   string
	QueueMode       string
	UsageMode       string
	VerboseMode     string
	ReasoningMode   string
	SendMode        string
	ActivationMode  string
	TTSMode         string
	SkillName       string
}

type telegramPeerProfile struct {
	Name      string
	Username  string
	AvatarURL string
}

type telegramPeerProfileCacheEntry struct {
	Profile   telegramPeerProfile
	ExpiresAt time.Time
}

const (
	telegramPeerProfileCacheTTL       = 10 * time.Minute
	telegramPeerProfileLookupTimeout  = 2500 * time.Millisecond
	telegramPeerProfileRefreshTimeout = 8 * time.Second
)

func NewBotService(settings *settingsservice.SettingsService, runtime Runtime, client *http.Client) *BotService {
	return &BotService{
		settings:     settings,
		runtime:      runtime,
		httpClient:   client,
		now:          time.Now,
		offsets:      NewUpdateOffsetStore(""),
		accounts:     make(map[string]*telegramAccountState),
		refreshAfter: 5 * time.Second,
		activeRuns:   make(map[string]telegramActiveRun),
		sessionRuns:  make(map[string]string),
		commandState: make(map[string]telegramSessionCommandState),
		statusCards:  make(map[string]int),
		runStatus:    make(map[string]int),
	}
}

func (service *BotService) SetRuntime(runtime Runtime) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.runtime = runtime
	service.mu.Unlock()
}

func (service *BotService) SendText(ctx context.Context, input SendTextInput) error {
	if service == nil {
		return errors.New("telegram bot unavailable")
	}
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return errors.New("text is required")
	}
	if input.ChatID == 0 {
		return errors.New("chat id is required")
	}
	state := service.resolveAccountState(strings.TrimSpace(input.AccountID))
	if state == nil || state.bot == nil {
		return errors.New("telegram account unavailable")
	}
	chunks := buildTelegramChunks(text, state.config.Chunk)
	if len(chunks) == 0 {
		return errors.New("text is required")
	}
	return service.sendChunksWithOptions(ctx, state, input.ChatID, input.ThreadID, chunks, 0, input.Silent)
}

func (service *BotService) SetPairingStore(store *channelpairing.Store) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.pairing = store
	service.mu.Unlock()
}

func (service *BotService) SetApprovalResolver(resolver ExecApprovalResolver) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.approvals = resolver
	service.mu.Unlock()
}

func (service *BotService) SetModelRepositories(providerRepo ProviderRepository, modelRepo ModelRepository) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.providers = providerRepo
	service.models = modelRepo
	service.mu.Unlock()
}

func (service *BotService) SetSubagentGateway(gateway SubagentGateway) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.subagents = gateway
	service.mu.Unlock()
}

func (service *BotService) SetAssistantService(assistants AssistantService) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.assistants = assistants
	service.mu.Unlock()
}

func (service *BotService) SetSkillPromptResolver(skills SkillPromptResolver) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.skills = skills
	service.mu.Unlock()
}

func (service *BotService) SetThreadService(threads ThreadService) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.threads = threads
	service.mu.Unlock()
}

func (service *BotService) SetModelSelectionNotifiers(
	onSettingsUpdated func(settingsdto.Settings),
	onAssistantsUpdated func(),
) {
	if service == nil {
		return
	}
	service.mu.Lock()
	service.onSettingsUpdated = onSettingsUpdated
	service.onAssistantsUpdated = onAssistantsUpdated
	service.mu.Unlock()
}

func (service *BotService) notifySettingsUpdated(updated settingsdto.Settings) {
	if service == nil {
		return
	}
	service.mu.Lock()
	notify := service.onSettingsUpdated
	service.mu.Unlock()
	if notify != nil {
		notify(updated)
	}
}

func (service *BotService) Refresh(ctx context.Context) error {
	if service == nil || service.settings == nil {
		return errors.New("settings service unavailable")
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return err
	}
	return service.RefreshFromSettings(ctx, current)
}

func (service *BotService) RefreshFromSettings(ctx context.Context, settings settingsdto.Settings) error {
	if service == nil {
		return errors.New("telegram service unavailable")
	}
	config := ResolveTelegramRuntimeConfig(settings)
	desired := make(map[string]TelegramAccountConfig, len(config.Accounts))
	for _, account := range config.Accounts {
		id := strings.TrimSpace(account.AccountID)
		if id == "" {
			id = DefaultTelegramAccountID
			account.AccountID = id
		}
		desired[id] = account
	}

	service.mu.Lock()
	defer service.mu.Unlock()

	for accountID, state := range service.accounts {
		account, ok := desired[accountID]
		if !ok {
			service.stopAccount(state)
			service.clearAccountError(state)
			service.clearUpdateOffset(accountID)
			delete(service.accounts, accountID)
			continue
		}
		previousToken := strings.TrimSpace(state.config.BotToken)
		nextToken := strings.TrimSpace(account.BotToken)
		tokenChanged := previousToken != nextToken
		if tokenChanged {
			service.clearUpdateOffset(accountID)
			service.clearAccountError(state)
			clearTelegramPeerProfileCache(state)
		}
		if !account.Enabled || nextToken == "" {
			service.stopAccount(state)
			state.config = account
			service.clearAccountError(state)
			continue
		}
		if stateNeedsRestart(state, account) {
			service.stopAccount(state)
			state.config = account
			service.startAccount(ctx, state)
			continue
		}
		state.config = account
	}

	for accountID, account := range desired {
		if _, ok := service.accounts[accountID]; ok {
			continue
		}
		state := &telegramAccountState{config: account}
		service.accounts[accountID] = state
		if account.Enabled && strings.TrimSpace(account.BotToken) != "" {
			service.startAccount(ctx, state)
		}
	}
	return nil
}

func (service *BotService) Close() {
	if service == nil {
		return
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	for _, state := range service.accounts {
		service.stopAccount(state)
	}
}

func (service *BotService) Status(ctx context.Context) (string, string, time.Time) {
	if service == nil {
		return "unknown", "telegram service unavailable", time.Time{}
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	state := "offline"
	lastError := ""
	updatedAt := time.Time{}
	for _, account := range service.accounts {
		if account.running {
			state = "online"
			if !account.lastOutbound.IsZero() {
				updatedAt = account.lastOutbound
			} else if !account.lastInbound.IsZero() {
				updatedAt = account.lastInbound
			} else if updatedAt.IsZero() {
				updatedAt = service.now()
			}
		}
		if account.lastError != "" {
			lastError = account.lastError
			if account.lastErrorAt.After(updatedAt) {
				updatedAt = account.lastErrorAt
			}
		}
	}
	if updatedAt.IsZero() {
		updatedAt = service.now()
	}
	return state, lastError, updatedAt
}

func (service *BotService) Probe(ctx context.Context) (bool, string, time.Duration, error) {
	account := service.resolvePrimaryAccount()
	if account == nil {
		return false, "telegram account missing", 0, errors.New("telegram account missing")
	}
	if !account.config.Enabled {
		return false, "telegram channel disabled", 0, errors.New("telegram channel disabled")
	}
	token := strings.TrimSpace(account.config.BotToken)
	if token == "" {
		return false, "telegram bot token missing", 0, errors.New("telegram bot token missing")
	}
	httpClient := buildTelegramHTTPClient(service.httpClient, account.config.Network)
	client := telegramapi.NewClient(token, httpClient)
	start := service.now()
	var bot telegramapi.User
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		bot, err = client.GetMe(ctx)
		if err == nil {
			break
		}
		if attempt < 2 {
			sleepWithContext(ctx, 200*time.Millisecond)
		}
	}
	latency := service.now().Sub(start)
	if err != nil {
		service.setAccountError(account, err.Error())
		return false, account.lastError, latency, err
	}
	service.setAccountIdentity(account, bot)
	service.clearAccountError(account)
	return true, "", latency, nil
}

func (service *BotService) Logout(ctx context.Context) error {
	if service == nil || service.settings == nil {
		return errors.New("settings service unavailable")
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return err
	}
	channels := cloneMap(current.Channels)
	telegram := resolveMap(channels["telegram"])
	if telegram == nil {
		return nil
	}
	delete(telegram, "botToken")
	telegram["enabled"] = false
	channels["telegram"] = telegram
	updated, err := service.settings.UpdateSettings(ctx, settingsdto.UpdateSettingsRequest{
		Channels: channels,
	})
	if err == nil {
		service.clearAllAccountErrors()
		service.clearAllUpdateOffsets()
		service.notifySettingsUpdated(updated)
		if refreshErr := service.RefreshFromSettings(ctx, updated); refreshErr != nil {
			return refreshErr
		}
	}
	return err
}

func (service *BotService) RefreshPrincipalProfile(
	ctx context.Context,
	channel string,
	accountID string,
	principalType string,
	principalID string,
) (string, string, string, error) {
	if service == nil {
		return "", "", "", errors.New("telegram service unavailable")
	}
	if !strings.EqualFold(strings.TrimSpace(channel), "telegram") {
		return "", "", "", errors.New("unsupported channel")
	}
	normalizedPrincipalType := strings.ToLower(strings.TrimSpace(principalType))
	peerKind := ""
	switch normalizedPrincipalType {
	case "user":
		peerKind = "direct"
	case "group":
		peerKind = "group"
	default:
		return "", "", "", errors.New("principal type is required")
	}
	peerID := strings.TrimSpace(principalID)
	if peerID == "" {
		return "", "", "", errors.New("principal id is required")
	}
	state := service.resolveAccountState(strings.TrimSpace(accountID))
	if state == nil || state.bot == nil {
		return "", "", "", errors.New("telegram account unavailable")
	}
	cacheKey := buildTelegramPeerProfileCacheKey(peerKind, peerID, nil)
	if cacheKey != "" {
		state.peerProfileCache.Delete(cacheKey)
	}
	lookupCtx := ctx
	if lookupCtx == nil {
		lookupCtx = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(lookupCtx, telegramPeerProfileRefreshTimeout)
	defer cancel()
	profile := fetchTelegramPeerProfileFromAPI(timeoutCtx, state, nil, peerKind, peerID)
	if strings.TrimSpace(profile.AvatarURL) == "" {
		// Retry once for transient latency/Telegram side delays during manual refresh.
		retryCtx, retryCancel := context.WithTimeout(lookupCtx, telegramPeerProfileRefreshTimeout)
		retryProfile := fetchTelegramPeerProfileFromAPI(retryCtx, state, nil, peerKind, peerID)
		retryCancel()
		profile = mergeTelegramPeerProfile(retryProfile, profile)
	}
	if strings.TrimSpace(profile.AvatarURL) == "" {
		if stableAvatarURL := resolveTelegramPeerAvatarURL(profile.Username); stableAvatarURL != "" {
			profile.AvatarURL = stableAvatarURL
		}
	}
	if cacheKey != "" {
		storeTelegramPeerProfileToCache(service, state, cacheKey, profile)
	}
	name := strings.TrimSpace(profile.Name)
	username := strings.TrimSpace(profile.Username)
	avatarURL := strings.TrimSpace(profile.AvatarURL)
	if name == "" && username == "" && avatarURL == "" {
		return "", "", "", errors.New("unable to load principal profile from telegram")
	}
	return name, username, avatarURL, nil
}

func (service *BotService) HandleWebhook(ctx context.Context, accountID string, update telegramapi.Update) error {
	state := service.resolveAccountState(accountID)
	if state == nil {
		return errors.New("telegram account unavailable")
	}
	if err := service.handleUpdate(ctx, state, update); err != nil {
		service.setAccountError(state, err.Error())
		return err
	}
	return nil
}

func (service *BotService) ResolveWebhookAccountID(pathAccountID string, secret string) (string, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	if pathAccountID != "" {
		if state, ok := service.accounts[pathAccountID]; ok {
			expected := strings.TrimSpace(state.config.WebhookSecret)
			if expected != "" && strings.TrimSpace(secret) == "" {
				return "", errors.New("telegram webhook secret required")
			}
			if expected != "" && strings.TrimSpace(secret) != expected {
				return "", errors.New("telegram webhook secret mismatch")
			}
			return pathAccountID, nil
		}
	}
	if secret != "" {
		for id, state := range service.accounts {
			if strings.TrimSpace(state.config.WebhookSecret) == secret {
				return id, nil
			}
		}
	}
	if len(service.accounts) == 1 {
		for id := range service.accounts {
			if state := service.accounts[id]; state != nil {
				if strings.TrimSpace(state.config.WebhookSecret) != "" {
					return "", errors.New("telegram webhook secret required")
				}
			}
			return id, nil
		}
	}
	return "", errors.New("unable to resolve telegram account")
}

func (service *BotService) resolvePrimaryAccount() *telegramAccountState {
	service.mu.Lock()
	defer service.mu.Unlock()
	if state, ok := service.accounts[DefaultTelegramAccountID]; ok && accountConfigured(state) {
		return state
	}
	for _, state := range service.accounts {
		if accountConfigured(state) {
			return state
		}
	}
	if state, ok := service.accounts[DefaultTelegramAccountID]; ok {
		return state
	}
	for _, state := range service.accounts {
		return state
	}
	return nil
}

func (service *BotService) resolveAccountState(accountID string) *telegramAccountState {
	service.mu.Lock()
	defer service.mu.Unlock()
	if accountID != "" {
		if state, ok := service.accounts[accountID]; ok {
			return state
		}
	}
	if state, ok := service.accounts[DefaultTelegramAccountID]; ok {
		return state
	}
	for _, state := range service.accounts {
		return state
	}
	return nil
}

func (service *BotService) startAccount(ctx context.Context, state *telegramAccountState) {
	if service == nil || state == nil {
		return
	}
	if strings.TrimSpace(state.config.BotToken) == "" {
		return
	}
	httpClient := buildTelegramHTTPClient(service.httpClient, state.config.Network)
	bot, err := telegramapi.NewBot(telegramapi.BotConfig{
		Token:      state.config.BotToken,
		HTTPClient: httpClient,
	})
	if err != nil {
		service.setAccountError(state, err.Error())
		return
	}
	state.bot = bot
	if state.replyTracker == nil {
		state.replyTracker = newReplyTracker()
	}
	user, err := bot.GetMe(ctx)
	if err != nil {
		service.setAccountError(state, err.Error())
		return
	}
	if user == nil {
		service.setAccountError(state, "telegram bot identity unavailable")
		return
	}
	service.setAccountIdentity(state, *user)
	if service.offsets != nil {
		state.updateOffset = service.offsets.Get(state.config.AccountID)
	}
	webhookURL := resolveWebhookURL(state.config)
	if strings.TrimSpace(webhookURL) != "" {
		if strings.TrimSpace(state.config.WebhookSecret) == "" {
			zap.L().Warn("telegram webhook secret missing; falling back to polling", zap.String("accountId", state.config.AccountID))
		} else {
			allowed := defaultAllowedUpdates()
			if err := bot.SetWebhook(ctx, &telego.SetWebhookParams{
				URL:            webhookURL,
				SecretToken:    state.config.WebhookSecret,
				AllowedUpdates: allowed,
			}); err != nil {
				service.setAccountError(state, err.Error())
				return
			}
			state.mode = "webhook"
			state.running = true
			service.clearAccountError(state)
			return
		}
	}
	_ = bot.DeleteWebhook(ctx, &telego.DeleteWebhookParams{DropPendingUpdates: false})
	pollCtx, cancel := context.WithCancel(context.Background())
	state.cancel = cancel
	state.running = true
	state.mode = "polling"
	service.clearAccountError(state)
	go service.pollLoop(pollCtx, state)
}

func (service *BotService) stopAccount(state *telegramAccountState) {
	if state == nil {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	state.running = false
	state.cancel = nil
	state.bot = nil
	state.replyTracker = nil
	state.placeholders = sync.Map{}
}

func (service *BotService) pollLoop(ctx context.Context, state *telegramAccountState) {
	if service == nil || state == nil {
		return
	}
	bot := state.bot
	if bot == nil {
		return
	}
	polling := state.config.Polling
	limit := polling.Limit
	if limit <= 0 {
		limit = 50
	}
	timeoutSeconds := polling.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = 20
	}
	concurrency := polling.Concurrency
	concurrency = resolveTelegramPollingWorkerCount(concurrency)
	queueSize := polling.QueueSize
	if queueSize <= 0 {
		queueSize = 100
	}
	backoff := time.Second
	if polling.BackoffSeconds > 0 {
		backoff = time.Duration(polling.BackoffSeconds) * time.Second
	}
	initialBackoff := backoff
	maxBackoff := 30 * time.Second
	if polling.MaxBackoffSeconds > 0 {
		maxBackoff = time.Duration(polling.MaxBackoffSeconds) * time.Second
	}
	updatesCh := make(chan telegramapi.Update, queueSize)
	var wg sync.WaitGroup
	if concurrency > 1 {
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					case update, ok := <-updatesCh:
						if !ok {
							return
						}
						if err := service.handleUpdate(ctx, state, update); err != nil {
							service.setAccountError(state, err.Error())
						}
					}
				}
			}()
		}
	}
	for {
		select {
		case <-ctx.Done():
			close(updatesCh)
			wg.Wait()
			return
		default:
		}
		updates, err := bot.GetUpdates(ctx, &telego.GetUpdatesParams{
			Offset:         int(state.updateOffset),
			Limit:          limit,
			Timeout:        timeoutSeconds,
			AllowedUpdates: defaultAllowedUpdates(),
		})
		if err != nil {
			if ctx.Err() != nil {
				close(updatesCh)
				wg.Wait()
				return
			}
			if isContextCanceledError(err) {
				sleepWithContext(ctx, backoff)
				continue
			}
			service.setAccountError(state, err.Error())
			sleepWithContext(ctx, backoff)
			if backoff < maxBackoff {
				backoff = backoff * 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}
		backoff = initialBackoff
		for _, update := range updates {
			if int64(update.UpdateID) >= state.updateOffset {
				state.updateOffset = int64(update.UpdateID) + 1
				if service.offsets != nil {
					_ = service.offsets.Set(state.config.AccountID, state.updateOffset)
				}
			}
			// Exec approval updates are routed through a fast lane so a long-running
			// runtime turn cannot deadlock the approval action itself.
			if isExecApprovalUpdate(update) {
				service.dispatchUpdateAsync(ctx, state, update)
				continue
			}
			select {
			case updatesCh <- update:
			default:
				if err := service.handleUpdate(ctx, state, update); err != nil {
					service.setAccountError(state, err.Error())
				}
			}
		}
	}
}

func resolveTelegramPollingWorkerCount(configured int) int {
	if configured <= 1 {
		// Single-worker polling can deadlock when a run is waiting for approval,
		// because the /approve update cannot be processed by the same blocked worker.
		return 2
	}
	return configured
}

func (service *BotService) dispatchUpdateAsync(ctx context.Context, state *telegramAccountState, update telegramapi.Update) {
	if service == nil || state == nil {
		return
	}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				zap.L().Warn(
					"telegram update async handler panicked",
					zap.String("accountId", state.config.AccountID),
					zap.Any("panic", recovered),
				)
			}
		}()
		if err := service.handleUpdate(ctx, state, update); err != nil {
			service.setAccountError(state, err.Error())
		}
	}()
}

func (service *BotService) handleUpdate(ctx context.Context, state *telegramAccountState, update telegramapi.Update) error {
	if service == nil || state == nil {
		return nil
	}
	updateType := resolveUpdateType(update)
	if update.CallbackQuery != nil {
		service.answerCallbackQuery(ctx, state, update.CallbackQuery)
	}
	message := firstMessage(update)
	if message == nil {
		service.recordInbound(state, update, nil, updateType, "")
		return nil
	}
	senderID := ""
	if update.CallbackQuery != nil && update.CallbackQuery.From.ID != 0 {
		senderID = fmt.Sprintf("%d", update.CallbackQuery.From.ID)
	}
	primaryText := ""
	if update.CallbackQuery != nil {
		primaryText = strings.TrimSpace(update.CallbackQuery.Data)
	}
	if primaryText == "" {
		primaryText = strings.TrimSpace(message.Text)
	}
	if primaryText == "" {
		primaryText = strings.TrimSpace(message.Caption)
	}
	primarySlashLike := strings.HasPrefix(strings.TrimSpace(primaryText), "/")
	content := buildMessageContent(message, primaryText)
	service.recordInbound(state, update, message, updateType, primaryText)
	if primaryText == "" && content == "" {
		return nil
	}
	commandCtx := ctx
	if update.CallbackQuery != nil && message.MessageID > 0 {
		commandCtx = withTelegramEditTarget(ctx, message.Chat.ID, message.MessageID)
	}
	// Callback approval actions should never block the polling loop.
	if isCallbackApprovalCommand(update, primaryText) {
		service.handleExecApprovalCommandAsync(commandCtx, state, message, primaryText, senderID)
		return nil
	}
	baseSessionKey := buildSessionKey(state.config.AccountID, message.Chat, int64(message.MessageThreadID))
	commandState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	parsedCommand, hasParsedCommand := parseTelegramInboundCommand(primaryText)
	nativeLookup := map[string]appcommands.NativeCommandSpec(nil)
	if hasParsedCommand {
		_, nativeLookup = service.resolveTelegramNativeCommandSpecs(ctx)
	}
	_, _, isApprovalCommand, _ := parseExecApprovalCommand(primaryText)
	allowWithoutMention := isApprovalCommand || hasParsedCommand || primarySlashLike
	if normalizeTelegramActivationMode(commandState.ActivationMode) == "always" {
		allowWithoutMention = true
	}
	allowed, reason, policyCtx := allowMessage(state, message, primaryText, senderID, allowWithoutMention)
	if !allowed {
		service.recordDenied(state, reason)
		if reason == "dm_pairing_required" {
			service.handlePairing(ctx, state, message)
		}
		return nil
	}
	if handled := service.handleExecApprovalCommand(commandCtx, state, message, primaryText, senderID); handled {
		return nil
	}
	if hasParsedCommand {
		handled, err := service.handleTelegramNativeCommand(
			commandCtx,
			state,
			message,
			senderID,
			baseSessionKey,
			parsedCommand,
			nativeLookup,
		)
		if err != nil {
			return err
		}
		if handled {
			return nil
		}
	}
	if primarySlashLike {
		if !hasParsedCommand {
			return service.sendSystemMessage(
				commandCtx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				"Unknown command. Use /help to see available commands.",
				nil,
			)
		}
		if _, ok := nativeLookup[parsedCommand.Name]; !ok {
			if !service.isTelegramRegisteredCustomCommand(ctx, parsedCommand.Name, nativeLookup) {
				return service.sendSystemMessage(
					commandCtx,
					state,
					message.Chat.ID,
					message.MessageThreadID,
					message.MessageID,
					fmt.Sprintf("Unknown command /%s. Use /help to see available commands.", parsedCommand.Name),
					nil,
				)
			}
		}
	}
	commandState = service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	if normalizeTelegramSendMode(commandState.SendMode) == "off" {
		return nil
	}
	if !service.runtimeReady() {
		service.recordRunError(state, "", errors.New("runtime unavailable"))
		return errors.New("runtime unavailable")
	}

	runSessionKey := service.resolveTelegramRuntimeSessionKey(baseSessionKey, commandState)
	streamMode := resolveTelegramStreamMode(state.config.StreamMode)
	runID := uuid.NewString()
	replyToStatus := 0
	if strings.ToLower(strings.TrimSpace(state.config.ReplyToMode)) != "off" {
		replyToStatus = message.MessageID
	}
	stopTyping := service.startTypingKeepalive(state, message.Chat.ID, message.MessageThreadID)
	defer stopTyping()
	runStatusID := service.sendRunStatusCard(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		replyToStatus,
		runID,
	)
	runStatusText := telegramRunStatusDoneText
	defer func() {
		service.finishTelegramRunStatusCard(
			state,
			state.config.AccountID,
			message.Chat.ID,
			message.MessageThreadID,
			runStatusID,
			runStatusText,
		)
	}()
	messages := make([]runtimedto.Message, 0, 2)
	if strings.TrimSpace(policyCtx.SystemPrompt) != "" {
		messages = append(messages, runtimedto.Message{Role: "system", Content: policyCtx.SystemPrompt})
	}
	if skill := strings.TrimSpace(commandState.SkillName); skill != "" {
		messages = append(messages, runtimedto.Message{
			Role:    "system",
			Content: fmt.Sprintf("Prefer the skill named \"%s\" when it is relevant and available.", skill),
		})
	}
	if content == "" {
		content = primaryText
	}
	messages = append(messages, runtimedto.Message{Role: "user", Content: content})
	effectiveSenderID := strings.TrimSpace(senderID)
	if effectiveSenderID == "" {
		effectiveSenderID = resolveMessageUserID(message)
	}
	peerKind := "direct"
	peerID := effectiveSenderID
	if strings.ToLower(strings.TrimSpace(message.Chat.Type)) != "private" {
		peerKind = "group"
		peerID = fmt.Sprintf("%d", message.Chat.ID)
	}
	peerProfile := service.resolveTelegramPeerProfile(ctx, state, message, peerKind, peerID)
	request := runtimedto.RuntimeRunRequest{
		RunID:      runID,
		SessionID:  runSessionKey,
		SessionKey: runSessionKey,
		Input: runtimedto.RuntimeInput{
			Messages: messages,
		},
		Metadata: map[string]any{
			"channel":       "telegram",
			"chatId":        fmt.Sprintf("%d", message.Chat.ID),
			"userId":        effectiveSenderID,
			"messageId":     message.MessageID,
			"accountId":     state.config.AccountID,
			"peerKind":      peerKind,
			"peerId":        peerID,
			"peerName":      peerProfile.Name,
			"peerUsername":  peerProfile.Username,
			"peerAvatarUrl": peerProfile.AvatarURL,
			"topicId":       message.MessageThreadID,
		},
	}
	service.applyTelegramCommandStateToRequest(&request, commandState)
	if len(policyCtx.Tools) > 0 {
		request.Tools.AllowList = append([]string(nil), policyCtx.Tools...)
	}
	runCtx, runCancel := context.WithCancel(ctx)
	service.registerActiveRun(state.config.AccountID, baseSessionKey, runID, runCancel)
	defer func() {
		service.unregisterActiveRun(runID)
		runCancel()
	}()
	var result runtimedto.RuntimeRunResult
	var err error
	var draftStream *telegramDraftStream
	var draftStreamer *telegramDraftStreamer
	streamStatusUpdated := false
	currentRunStatusText := telegramRunStatusThinkingText
	updateRunStatus := func(nextText string, withStop bool) {
		trimmed := strings.TrimSpace(nextText)
		if runStatusID <= 0 || trimmed == "" || trimmed == currentRunStatusText {
			return
		}
		currentRunStatusText = trimmed
		service.updateTelegramRunStatusCard(
			state,
			message.Chat.ID,
			runStatusID,
			runID,
			trimmed,
			withStop,
		)
	}
	if streamMode != "off" && state.bot != nil {
		if streamingRuntime, ok := service.runtime.(StreamingRuntime); ok {
			replyToDraft := 0
			if strings.ToLower(strings.TrimSpace(state.config.ReplyToMode)) != "off" {
				replyToDraft = message.MessageID
			}
			draftMaxChars := state.config.Chunk.TextChunkLimit
			if draftMaxChars <= 0 {
				draftMaxChars = 3800
			}
			if draftMaxChars > 4096 {
				draftMaxChars = 4096
			}
			minInitial := 1
			if streamMode == "block" {
				minInitial = telegramDraftStreamMinInitialChars
			}
			draftStream = newTelegramDraftStream(
				ctx,
				service,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				replyToDraft,
				draftMaxChars,
				minInitial,
				telegramDraftStreamDefaultThrottle,
			)
			draftStreamer = newTelegramDraftStreamer(draftStream, streamMode, state.config.DraftChunk)
			result, err = streamingRuntime.RunStream(runCtx, request, func(event runtimedto.RuntimeStreamEvent) {
				switch event.Type {
				case runtimedto.RuntimeStreamEventDelta:
					if !streamStatusUpdated && strings.TrimSpace(event.Delta) != "" {
						streamStatusUpdated = true
					}
					if strings.TrimSpace(event.Delta) != "" {
						updateRunStatus(telegramRunStatusStreamingText, true)
					}
				case runtimedto.RuntimeStreamEventToolStart:
					updateRunStatus(buildTelegramToolStatusText(event.ToolName), true)
				case runtimedto.RuntimeStreamEventToolResult:
					if streamStatusUpdated {
						updateRunStatus(telegramRunStatusStreamingText, true)
					} else {
						updateRunStatus(telegramRunStatusThinkingText, true)
					}
				}
				draftStreamer.HandleEvent(event)
			})
			draftStreamer.Flush()
		} else {
			result, err = service.runtime.Run(runCtx, request)
		}
	} else {
		result, err = service.runtime.Run(runCtx, request)
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			runStatusText = telegramRunStatusStoppedText
			if draftStream != nil {
				draftStream.Clear()
			}
			return nil
		}
		runStatusText = telegramRunStatusFailedText
		service.recordRunError(state, runID, err)
		service.sendRuntimeErrorReply(ctx, state, message, draftStream, err)
		return err
	}
	service.recordRunSuccess(state, runID)
	reply := buildTelegramReply(result.AssistantMessage.Content, result.AssistantMessage.Parts)
	reply = decorateTelegramReply(reply, result, commandState)
	if reply == "" {
		if draftStream != nil {
			draftStream.Clear()
		}
		return nil
	}
	if draftStream != nil {
		draftStream.Stop()
	}
	if err := service.sendReply(ctx, state, message, baseSessionKey, reply, draftStream); err != nil {
		runStatusText = telegramRunStatusFailedText
		service.recordOutboundError(state, err)
		return err
	}
	return nil
}

func (service *BotService) sendRuntimeErrorReply(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	draftStream *telegramDraftStream,
	runErr error,
) {
	if service == nil || state == nil || message == nil || runErr == nil {
		return
	}
	errorText := buildTelegramRunFailureText(state, runErr)
	if draftStream != nil {
		draftStream.Stop()
		previewID := draftStream.MessageID()
		if previewID > 0 {
			chunk := telegramapi.FormattedChunk{
				HTML: telegramapi.RenderTelegramHTML(errorText),
				Text: errorText,
			}
			if strings.TrimSpace(chunk.HTML) != "" {
				if editedID, err := service.editPlaceholder(ctx, state, message.Chat.ID, previewID, chunk, true); err == nil {
					service.recordOutboundSuccess(state, int64(editedID))
					return
				}
			}
		}
	}
	if err := service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		errorText,
		nil,
	); err != nil {
		service.recordOutboundError(state, err)
		return
	}
	if draftStream != nil {
		draftStream.Clear()
	}
}

func buildTelegramRunFailureText(state *telegramAccountState, runErr error) string {
	base := "Request failed. Please retry."
	if runErr == nil {
		return base
	}
	reason := strings.TrimSpace(runErr.Error())
	if reason == "" {
		return base
	}
	token := ""
	if state != nil {
		token = state.config.BotToken
	}
	reason = redactTelegramToken(reason, token)
	reason = truncateTelegramText(reason, 600)
	if reason == "" {
		return base
	}
	return fmt.Sprintf("Request failed: %s", reason)
}

func (service *BotService) runtimeReady() bool {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.runtime != nil
}

func (service *BotService) sendTypingOnce(state *telegramAccountState, chatID int64, threadID int) {
	if service == nil || state == nil || state.bot == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	params := &telego.SendChatActionParams{
		ChatID: tu.ID(chatID),
		Action: telego.ChatActionTyping,
	}
	if threadID > 0 {
		params.MessageThreadID = threadID
	}
	if err := state.bot.SendChatAction(ctx, params); err != nil {
		redacted := redactTelegramToken(err.Error(), state.config.BotToken)
		zap.L().Debug("telegram typing failed", zap.String("accountId", state.config.AccountID), zap.String("error", redacted))
	}
}

func (service *BotService) startTypingKeepalive(state *telegramAccountState, chatID int64, threadID int) func() {
	if service == nil || state == nil || state.bot == nil {
		return func() {}
	}
	service.sendTypingOnce(state, chatID, threadID)
	stop := make(chan struct{})
	var once sync.Once
	go func() {
		ticker := time.NewTicker(telegramTypingKeepaliveInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				service.sendTypingOnce(state, chatID, threadID)
			}
		}
	}()
	return func() {
		once.Do(func() {
			close(stop)
		})
	}
}

func shouldSendAckReaction(state *telegramAccountState, message *telegramapi.Message, text string) bool {
	if state == nil || message == nil {
		return false
	}
	reaction := strings.TrimSpace(state.config.AckReaction)
	if reaction == "" {
		return false
	}
	scope := strings.ToLower(strings.TrimSpace(state.config.AckReactionScope))
	chatType := strings.ToLower(strings.TrimSpace(message.Chat.Type))
	isDirect := chatType == "private"
	if isDirect {
		return scope == "direct" || scope == "all"
	}
	switch scope {
	case "all", "group-all":
		return true
	case "group-mentions":
		groupCfg, _ := resolveGroupConfig(state.config.Groups, message.Chat.ID)
		topicCfg, topicMatched, _ := resolveTopicConfig(groupCfg, int64(message.MessageThreadID))
		if !resolveRequireMention(groupCfg, topicCfg, topicMatched) {
			return false
		}
		return mentionsBot(state, message, text)
	default:
		return false
	}
}

func (service *BotService) sendAckReaction(state *telegramAccountState, message *telegramapi.Message) bool {
	if service == nil || state == nil || state.bot == nil || message == nil {
		return false
	}
	if message.MessageID <= 0 {
		return false
	}
	reaction := strings.TrimSpace(state.config.AckReaction)
	if reaction == "" {
		return false
	}
	params := &telego.SetMessageReactionParams{
		ChatID:    tu.ID(message.Chat.ID),
		MessageID: message.MessageID,
		Reaction: []telego.ReactionType{
			&telego.ReactionTypeEmoji{Type: telego.ReactionEmoji, Emoji: reaction},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := state.bot.SetMessageReaction(ctx, params); err != nil {
		redacted := redactTelegramToken(err.Error(), state.config.BotToken)
		zap.L().Debug("telegram ack reaction failed", zap.String("accountId", state.config.AccountID), zap.String("error", redacted))
		return false
	}
	return true
}

func (service *BotService) clearAckReaction(state *telegramAccountState, message *telegramapi.Message) {
	if service == nil || state == nil || state.bot == nil || message == nil {
		return
	}
	if message.MessageID <= 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Reaction omitted on purpose to clear any existing bot reaction.
	if err := state.bot.SetMessageReaction(ctx, &telego.SetMessageReactionParams{
		ChatID:    tu.ID(message.Chat.ID),
		MessageID: message.MessageID,
	}); err != nil {
		redacted := redactTelegramToken(err.Error(), state.config.BotToken)
		zap.L().Debug("telegram ack reaction clear failed", zap.String("accountId", state.config.AccountID), zap.String("error", redacted))
	}
}

func (service *BotService) recordInbound(state *telegramAccountState, update telegramapi.Update, message *telegramapi.Message, updateType, text string) {
	if service == nil || state == nil {
		return
	}
	state.lastInbound = service.now()
	state.inboundCount++
	state.lastInboundType = updateType
	state.lastInboundUpdateID = int64(update.UpdateID)
	state.lastInboundCommand = extractCommand(text)
	if message == nil {
		return
	}
	state.lastInboundMessageID = int64(message.MessageID)
	state.lastInboundChatID = message.Chat.ID
	state.lastInboundUserID = resolveMessageUserID(message)
}

func (service *BotService) recordDenied(state *telegramAccountState, reason string) {
	if service == nil || state == nil {
		return
	}
	state.deniedCount++
	state.lastDeniedReason = strings.TrimSpace(reason)
	state.lastDeniedAt = service.now()
}

func (service *BotService) recordRunError(state *telegramAccountState, runID string, err error) {
	if service == nil || state == nil || err == nil {
		return
	}
	state.errorCount++
	state.lastRunID = runID
	state.lastRunError = redactTelegramToken(err.Error(), state.config.BotToken)
	state.lastRunAt = service.now()
}

func (service *BotService) recordRunSuccess(state *telegramAccountState, runID string) {
	if service == nil || state == nil {
		return
	}
	state.lastRunID = runID
	state.lastRunError = ""
	state.lastRunAt = service.now()
}

func (service *BotService) recordOutboundError(state *telegramAccountState, err error) {
	if service == nil || state == nil || err == nil {
		return
	}
	state.errorCount++
	state.lastOutboundError = redactTelegramToken(err.Error(), state.config.BotToken)
	state.lastOutbound = service.now()
}

func (service *BotService) recordOutboundSuccess(state *telegramAccountState, messageID int64) {
	if service == nil || state == nil {
		return
	}
	state.outboundCount++
	state.lastOutboundError = ""
	state.lastOutboundMessageID = messageID
	state.lastOutbound = service.now()
}

func resolveUpdateType(update telegramapi.Update) string {
	switch {
	case update.Message != nil:
		return "message"
	case update.EditedMessage != nil:
		return "edited_message"
	case update.ChannelPost != nil:
		return "channel_post"
	case update.EditedChannelPost != nil:
		return "edited_channel_post"
	case update.CallbackQuery != nil:
		return "callback_query"
	default:
		return ""
	}
}

func extractCommand(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "/") {
		return ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

type telegramInboundCommand struct {
	Name string
	Args string
	Raw  string
}

type telegramEditTargetContextKey struct{}

type telegramEditTarget struct {
	ChatID    int64
	MessageID int
}

func parseTelegramInboundCommand(text string) (telegramInboundCommand, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return telegramInboundCommand{}, false
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return telegramInboundCommand{}, false
	}
	rawName := strings.TrimPrefix(strings.TrimSpace(parts[0]), "/")
	if at := strings.Index(rawName, "@"); at >= 0 {
		rawName = rawName[:at]
	}
	name := NormalizeTelegramCommandName(rawName)
	if name == "" {
		return telegramInboundCommand{}, false
	}
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(strings.Join(parts[1:], " "))
	}
	return telegramInboundCommand{
		Name: name,
		Args: args,
		Raw:  trimmed,
	}, true
}

func withTelegramEditTarget(ctx context.Context, chatID int64, messageID int) context.Context {
	if ctx == nil || messageID <= 0 {
		return ctx
	}
	target := telegramEditTarget{ChatID: chatID, MessageID: messageID}
	return context.WithValue(ctx, telegramEditTargetContextKey{}, target)
}

func telegramEditTargetFromContext(ctx context.Context) (telegramEditTarget, bool) {
	if ctx == nil {
		return telegramEditTarget{}, false
	}
	value := ctx.Value(telegramEditTargetContextKey{})
	target, ok := value.(telegramEditTarget)
	if !ok || target.MessageID <= 0 {
		return telegramEditTarget{}, false
	}
	return target, true
}

const telegramCallbackDataMaxBytes = 64
const telegramRunStatusThinkingText = "Thinking... 💭"
const telegramRunStatusStreamingText = "Streaming... ⏳"
const telegramRunStatusDoneText = "Completed ✅"
const telegramRunStatusStoppedText = "Stopped ⏹️"
const telegramRunStatusFailedText = "Failed ❌"
const telegramTypingKeepaliveInterval = 4 * time.Second
const telegramToolStatusNameLimit = 42

var telegramToolStatusWebTokens = []string{
	"web",
	"browser",
	"search",
	"crawl",
	"http",
	"url",
	"navigate",
}

var telegramToolStatusCodingTokens = []string{
	"exec",
	"bash",
	"read",
	"write",
	"edit",
	"file",
	"terminal",
	"shell",
	"process",
}

type telegramCommandButton struct {
	Text     string
	Callback string
}

type telegramModelRef struct {
	ProviderID   string
	ProviderName string
	ModelName    string
}

func normalizeTelegramThinkingLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "off", "none", "disabled", "disable", "false", "0":
		return "off"
	case "minimal", "min":
		return "minimal"
	case "low", "on", "enabled", "enable", "true", "1":
		return "low"
	case "medium", "med":
		return "medium"
	case "high", "max":
		return "high"
	case "xhigh", "x-high", "x_high", "extra-high", "extra_high", "extrahigh":
		return "xhigh"
	default:
		return ""
	}
}

func normalizeTelegramUsageMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "off", "none", "disable", "disabled", "false", "0":
		return "off"
	case "tokens", "token":
		return "tokens"
	case "full":
		return "full"
	case "cost":
		return "cost"
	default:
		return ""
	}
}

func normalizeTelegramQueueMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "steer", "steer-backlog", "steer_backlog", "steerbacklog":
		return "steer"
	case "collect":
		return "collect"
	case "followup", "follow-up", "interrupt":
		return "followup"
	default:
		return ""
	}
}

func normalizeTelegramVerboseMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1", "yes", "enable", "enabled":
		return "on"
	case "off", "false", "0", "no", "disable", "disabled":
		return "off"
	default:
		return ""
	}
}

func normalizeTelegramReasoningMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1", "yes":
		return "on"
	case "stream":
		return "stream"
	case "off", "false", "0", "no":
		return "off"
	default:
		return ""
	}
}

func normalizeTelegramActivationMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "always", "all", "on":
		return "always"
	case "mention", "mentions", "default", "inherit", "off":
		return "mention"
	default:
		return ""
	}
}

func normalizeTelegramSendMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1":
		return "on"
	case "off", "false", "0":
		return "off"
	case "inherit", "default":
		return "inherit"
	default:
		return ""
	}
}

func normalizeTelegramTTSMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "1":
		return "on"
	case "off", "false", "0":
		return "off"
	default:
		return ""
	}
}

func newTelegramConversationID() string {
	value := strings.ReplaceAll(uuid.NewString(), "-", "")
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func buildTelegramInlineKeyboard(rows ...[]telegramCommandButton) telego.ReplyMarkup {
	if len(rows) == 0 {
		return nil
	}
	keyboardRows := make([][]telego.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		buttons := make([]telego.InlineKeyboardButton, 0, len(row))
		for _, button := range row {
			text := strings.TrimSpace(button.Text)
			callback := strings.TrimSpace(button.Callback)
			if text == "" || callback == "" {
				continue
			}
			if len([]byte(callback)) > telegramCallbackDataMaxBytes {
				continue
			}
			buttons = append(buttons, tu.InlineKeyboardButton(text).WithCallbackData(callback))
		}
		if len(buttons) > 0 {
			keyboardRows = append(keyboardRows, buttons)
		}
	}
	if len(keyboardRows) == 0 {
		return nil
	}
	return &telego.InlineKeyboardMarkup{InlineKeyboard: keyboardRows}
}

func (service *BotService) getSessionCommandState(accountID string, baseSessionKey string) telegramSessionCommandState {
	if service == nil {
		return telegramSessionCommandState{}
	}
	key := buildSessionRunKey(accountID, baseSessionKey)
	if strings.TrimSpace(key) == "::" {
		return telegramSessionCommandState{}
	}
	service.mu.Lock()
	state := service.commandState[key]
	service.mu.Unlock()
	return state
}

func (service *BotService) updateSessionCommandState(
	accountID string,
	baseSessionKey string,
	update func(state *telegramSessionCommandState),
) telegramSessionCommandState {
	if service == nil || update == nil {
		return telegramSessionCommandState{}
	}
	key := buildSessionRunKey(accountID, baseSessionKey)
	if strings.TrimSpace(key) == "::" {
		return telegramSessionCommandState{}
	}
	service.mu.Lock()
	current := service.commandState[key]
	update(&current)
	service.commandState[key] = current
	service.mu.Unlock()
	return current
}

func (service *BotService) resolveTelegramRuntimeSessionKey(baseSessionKey string, state telegramSessionCommandState) string {
	if override := strings.TrimSpace(state.SessionOverride); override != "" {
		return override
	}
	trimmedBase := strings.TrimSpace(baseSessionKey)
	conversationID := strings.TrimSpace(state.ConversationID)
	if trimmedBase == "" || conversationID == "" {
		return trimmedBase
	}
	return fmt.Sprintf("%s:conv:%s", trimmedBase, conversationID)
}

func (service *BotService) applyTelegramCommandStateToRequest(request *runtimedto.RuntimeRunRequest, state telegramSessionCommandState) {
	if request == nil {
		return
	}
	modelProvider := strings.TrimSpace(state.ModelProvider)
	modelName := strings.TrimSpace(state.ModelName)
	if modelProvider != "" && modelName != "" {
		request.Model = &runtimedto.ModelSelection{
			ProviderID: modelProvider,
			Name:       modelName,
		}
	}
	if level := normalizeTelegramThinkingLevel(state.ThinkingLevel); level != "" {
		request.Thinking.Mode = level
	}
	if request.Metadata == nil {
		request.Metadata = make(map[string]any)
	}
	if queueMode := normalizeTelegramQueueMode(state.QueueMode); queueMode != "" {
		request.Metadata["queueMode"] = queueMode
	}
	if verboseMode := normalizeTelegramVerboseMode(state.VerboseMode); verboseMode != "" {
		request.Metadata["verbose"] = verboseMode == "on"
	}
	if reasoningMode := normalizeTelegramReasoningMode(state.ReasoningMode); reasoningMode != "" {
		request.Metadata["reasoningMode"] = reasoningMode
	}
	if usageMode := normalizeTelegramUsageMode(state.UsageMode); usageMode != "" {
		request.Metadata["usageMode"] = usageMode
	}
	if sendMode := normalizeTelegramSendMode(state.SendMode); sendMode != "" {
		request.Metadata["telegramSendMode"] = sendMode
	}
	if activationMode := normalizeTelegramActivationMode(state.ActivationMode); activationMode != "" {
		request.Metadata["telegramActivationMode"] = activationMode
	}
	if ttsMode := normalizeTelegramTTSMode(state.TTSMode); ttsMode != "" {
		request.Metadata["telegramTTSMode"] = ttsMode
	}
	if skill := strings.TrimSpace(state.SkillName); skill != "" {
		request.Metadata["telegramSkillName"] = skill
	}
}

func (service *BotService) resolveModelCatalogRepositories() (ProviderRepository, ModelRepository) {
	if service == nil {
		return nil, nil
	}
	service.mu.Lock()
	providerRepo := service.providers
	modelRepo := service.models
	service.mu.Unlock()
	return providerRepo, modelRepo
}

func (service *BotService) listTelegramModelCatalog(ctx context.Context) ([]telegramModelRef, error) {
	providerRepo, modelRepo := service.resolveModelCatalogRepositories()
	if providerRepo == nil || modelRepo == nil {
		return nil, nil
	}
	providersList, err := providerRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]telegramModelRef, 0, 32)
	for _, provider := range providersList {
		if !provider.Enabled {
			continue
		}
		providerID := strings.TrimSpace(provider.ID)
		if providerID == "" {
			continue
		}
		providerName := strings.TrimSpace(provider.Name)
		if providerName == "" {
			providerName = providerID
		}
		modelsList, listErr := modelRepo.ListByProvider(ctx, providerID)
		if listErr != nil {
			return nil, listErr
		}
		for _, model := range modelsList {
			if !model.Enabled {
				continue
			}
			modelName := strings.TrimSpace(model.Name)
			if modelName == "" {
				continue
			}
			resolvedProviderID := strings.TrimSpace(model.ProviderID)
			if resolvedProviderID == "" {
				resolvedProviderID = providerID
			}
			items = append(items, telegramModelRef{
				ProviderID:   resolvedProviderID,
				ProviderName: providerName,
				ModelName:    modelName,
			})
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		leftProvider := strings.ToLower(strings.TrimSpace(items[i].ProviderName))
		rightProvider := strings.ToLower(strings.TrimSpace(items[j].ProviderName))
		if leftProvider == "" {
			leftProvider = strings.ToLower(strings.TrimSpace(items[i].ProviderID))
		}
		if rightProvider == "" {
			rightProvider = strings.ToLower(strings.TrimSpace(items[j].ProviderID))
		}
		if leftProvider == rightProvider {
			return strings.ToLower(strings.TrimSpace(items[i].ModelName)) < strings.ToLower(strings.TrimSpace(items[j].ModelName))
		}
		return leftProvider < rightProvider
	})
	return items, nil
}

func (service *BotService) resolveDefaultModel(ctx context.Context) (string, string) {
	if service == nil || service.settings == nil {
		return "", ""
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return "", ""
	}
	return strings.TrimSpace(current.AgentModelProviderID), strings.TrimSpace(current.AgentModelName)
}

func (service *BotService) describeEffectiveModel(ctx context.Context, state telegramSessionCommandState) string {
	providerID := strings.TrimSpace(state.ModelProvider)
	modelName := strings.TrimSpace(state.ModelName)
	if providerID != "" && modelName != "" {
		if item, ok := service.resolveModelCatalogItem(ctx, providerID, modelName); ok {
			return buildTelegramModelDisplayNameOnly(item) + " (session override)"
		}
		return buildTelegramModelRef(providerID, modelName) + " (session override)"
	}
	defaultProvider, defaultModel := service.resolveDefaultModel(ctx)
	if defaultProvider == "" || defaultModel == "" {
		return "not configured"
	}
	if item, ok := service.resolveModelCatalogItem(ctx, defaultProvider, defaultModel); ok {
		return buildTelegramModelDisplayNameOnly(item)
	}
	return buildTelegramModelRef(defaultProvider, defaultModel)
}

func (service *BotService) syncGlobalModelSelection(ctx context.Context, providerID string, modelName string) error {
	if service == nil || service.settings == nil {
		return nil
	}
	trimmedProvider := strings.TrimSpace(providerID)
	trimmedModel := strings.TrimSpace(modelName)
	if trimmedProvider == "" || trimmedModel == "" {
		return errors.New("invalid model selection")
	}
	current, err := service.settings.GetSettings(ctx)
	if err == nil {
		if strings.EqualFold(strings.TrimSpace(current.AgentModelProviderID), trimmedProvider) &&
			strings.EqualFold(strings.TrimSpace(current.AgentModelName), trimmedModel) {
			return nil
		}
	}
	updated, updateErr := service.settings.UpdateSettings(ctx, settingsdto.UpdateSettingsRequest{
		AgentModelProviderID: &trimmedProvider,
		AgentModelName:       &trimmedModel,
	})
	if updateErr == nil {
		service.notifySettingsUpdated(updated)
	}
	return updateErr
}

func (service *BotService) syncDefaultAssistantPrimaryModel(ctx context.Context, providerID string, modelName string) error {
	if service == nil {
		return nil
	}
	service.mu.Lock()
	assistants := service.assistants
	notify := service.onAssistantsUpdated
	service.mu.Unlock()
	if assistants == nil {
		return nil
	}
	items, err := assistants.ListAssistants(ctx, true)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	target := items[0]
	for _, item := range items {
		if item.IsDefault {
			target = item
			break
		}
	}
	nextModel := target.Model
	nextPrimary := buildTelegramModelRef(providerID, modelName)
	if strings.EqualFold(strings.TrimSpace(nextModel.Agent.Primary), strings.TrimSpace(nextPrimary)) {
		return nil
	}
	nextModel.Agent.Primary = nextPrimary
	if _, err := assistants.UpdateAssistant(ctx, assistantdto.UpdateAssistantRequest{
		ID:    target.ID,
		Model: &nextModel,
	}); err != nil {
		return err
	}
	if notify != nil {
		notify()
	}
	return nil
}

func (service *BotService) resolveModelSelection(
	ctx context.Context,
	currentState telegramSessionCommandState,
	input string,
) (telegramModelRef, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return telegramModelRef{}, errors.New("model value is required")
	}
	requestedProvider := ""
	requestedModel := ""
	if strings.Contains(trimmed, "/") {
		parts := strings.SplitN(trimmed, "/", 2)
		requestedProvider = strings.TrimSpace(parts[0])
		requestedModel = strings.TrimSpace(parts[1])
	} else {
		requestedModel = trimmed
		requestedProvider = strings.TrimSpace(currentState.ModelProvider)
		if requestedProvider == "" {
			requestedProvider, _ = service.resolveDefaultModel(ctx)
		}
	}
	if requestedModel == "" {
		return telegramModelRef{}, errors.New("model value is required")
	}

	catalog, err := service.listTelegramModelCatalog(ctx)
	if err != nil {
		return telegramModelRef{}, err
	}
	if len(catalog) == 0 {
		if requestedProvider == "" {
			return telegramModelRef{}, errors.New("provider prefix required (use provider/model)")
		}
		requestedModel = strings.TrimSpace(strings.TrimPrefix(requestedModel, requestedProvider+"/"))
		return telegramModelRef{
			ProviderID: requestedProvider,
			ModelName:  requestedModel,
		}, nil
	}

	requestedRef := buildTelegramModelRef(requestedProvider, requestedModel)
	if requestedProvider != "" {
		matches := make([]telegramModelRef, 0, 2)
		for _, item := range catalog {
			itemRef := buildTelegramModelRef(item.ProviderID, item.ModelName)
			itemDisplayRef := buildTelegramModelRef(item.ProviderName, item.ModelName)
			providerMatch := strings.EqualFold(strings.TrimSpace(item.ProviderID), requestedProvider) ||
				strings.EqualFold(strings.TrimSpace(item.ProviderName), requestedProvider)
			modelMatch := strings.EqualFold(strings.TrimSpace(item.ModelName), requestedModel) ||
				strings.EqualFold(itemRef, requestedRef) ||
				strings.EqualFold(itemDisplayRef, requestedRef)
			if providerMatch && modelMatch {
				matches = append(matches, item)
			}
		}
		switch len(matches) {
		case 0:
			return telegramModelRef{}, errors.New("model not found in enabled catalog")
		case 1:
			return matches[0], nil
		default:
			return telegramModelRef{}, errors.New("model selection is ambiguous; use provider id/model")
		}
	}

	matches := make([]telegramModelRef, 0, 2)
	for _, item := range catalog {
		itemRef := buildTelegramModelRef(item.ProviderID, item.ModelName)
		if strings.EqualFold(strings.TrimSpace(item.ModelName), requestedModel) || strings.EqualFold(itemRef, requestedRef) {
			matches = append(matches, item)
		}
	}
	switch len(matches) {
	case 0:
		return telegramModelRef{}, errors.New("model not found in enabled catalog")
	case 1:
		return matches[0], nil
	default:
		return telegramModelRef{}, errors.New("model name is ambiguous; use provider/model")
	}
}

func (service *BotService) resolveModelCatalogItem(ctx context.Context, providerID string, modelName string) (telegramModelRef, bool) {
	if service == nil {
		return telegramModelRef{}, false
	}
	trimmedProvider := strings.TrimSpace(providerID)
	trimmedModel := strings.TrimSpace(modelName)
	if trimmedProvider == "" || trimmedModel == "" {
		return telegramModelRef{}, false
	}
	catalog, err := service.listTelegramModelCatalog(ctx)
	if err != nil || len(catalog) == 0 {
		return telegramModelRef{}, false
	}
	targetRef := buildTelegramModelRef(trimmedProvider, trimmedModel)
	for _, item := range catalog {
		itemProviderID := strings.TrimSpace(item.ProviderID)
		itemModel := strings.TrimSpace(item.ModelName)
		itemRef := buildTelegramModelRef(itemProviderID, itemModel)
		if strings.EqualFold(itemProviderID, trimmedProvider) &&
			(strings.EqualFold(itemModel, trimmedModel) || strings.EqualFold(itemRef, targetRef)) {
			return item, true
		}
	}
	return telegramModelRef{}, false
}

func resolveTelegramChoiceLabel(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return strings.ToUpper(trimmed[:1]) + trimmed[1:]
}

func buildTelegramModelRef(providerID string, modelName string) string {
	provider := strings.TrimSpace(providerID)
	model := strings.TrimSpace(modelName)
	if provider == "" {
		return model
	}
	if model == "" {
		return provider
	}
	prefix := strings.ToLower(provider) + "/"
	if strings.HasPrefix(strings.ToLower(model), prefix) {
		return model
	}
	return provider + "/" + model
}

func buildTelegramModelDisplayRef(item telegramModelRef) string {
	return buildTelegramModelDisplayNameOnly(item)
}

func buildTelegramModelDisplayNameOnly(item telegramModelRef) string {
	providerID := strings.TrimSpace(item.ProviderID)
	providerName := strings.TrimSpace(item.ProviderName)
	modelName := strings.TrimSpace(item.ModelName)
	if providerName == "" {
		return buildTelegramModelRef(providerID, modelName)
	}
	if modelName == "" {
		return providerName
	}
	return providerName + "/" + modelName
}

func shortTelegramModelLabel(modelRef string) string {
	trimmed := strings.TrimSpace(modelRef)
	if trimmed == "" {
		return ""
	}
	if utf8.RuneCountInString(trimmed) <= 24 {
		return trimmed
	}
	return truncateToRunes(trimmed, 24)
}

func parseTelegramModelsPage(args string) int {
	fields := strings.Fields(strings.TrimSpace(args))
	if len(fields) == 0 {
		return 1
	}
	for idx, field := range fields {
		trimmed := strings.TrimSpace(field)
		if trimmed == "" {
			continue
		}
		if value, err := strconv.Atoi(trimmed); err == nil && value > 0 {
			return value
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "page=") {
			if value, err := strconv.Atoi(strings.TrimSpace(trimmed[5:])); err == nil && value > 0 {
				return value
			}
		}
		if lower == "page" && idx+1 < len(fields) {
			if value, err := strconv.Atoi(strings.TrimSpace(fields[idx+1])); err == nil && value > 0 {
				return value
			}
		}
	}
	return 1
}

func buildCommandModeButtons(command string, values []string) telego.ReplyMarkup {
	if len(values) == 0 {
		return nil
	}
	rows := make([][]telegramCommandButton, 0, (len(values)+1)/2)
	for i := 0; i < len(values); i += 2 {
		row := make([]telegramCommandButton, 0, 2)
		for j := i; j < i+2 && j < len(values); j++ {
			value := strings.TrimSpace(values[j])
			if value == "" {
				continue
			}
			row = append(row, telegramCommandButton{
				Text:     resolveTelegramChoiceLabel(value),
				Callback: fmt.Sprintf("/%s %s", command, value),
			})
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return buildTelegramInlineKeyboard(rows...)
}

func buildTelegramSystemUsage(command string, usage string) string {
	trimmedCommand := strings.TrimSpace(command)
	trimmedUsage := strings.TrimSpace(usage)
	if trimmedCommand == "" {
		return trimmedUsage
	}
	if trimmedUsage == "" {
		return "Usage: /" + trimmedCommand
	}
	return fmt.Sprintf("Usage: /%s %s", trimmedCommand, trimmedUsage)
}

func truncateTelegramText(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= limit {
		return strings.TrimSpace(value)
	}
	return truncateToRunes(value, limit)
}

func (service *BotService) handleTelegramModelCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	trimmedArgs := strings.TrimSpace(args)
	if trimmedArgs == "" {
		text := fmt.Sprintf(
			"Current model: %s\nUse /model <provider/model> to set a session override.\nUse /model default to clear the override.",
			service.describeEffectiveModel(ctx, currentState),
		)
		markup := buildTelegramInlineKeyboard(
			[]telegramCommandButton{{Text: "Browse", Callback: "/models"}},
			[]telegramCommandButton{{Text: "Default", Callback: "/model default"}},
		)
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			markup,
		)
	}
	switch strings.ToLower(strings.TrimSpace(trimmedArgs)) {
	case "default", "reset", "clear", "inherit":
		updated := service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
			item.ModelProvider = ""
			item.ModelName = ""
		})
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Model override cleared. Active model: %s", service.describeEffectiveModel(ctx, updated)),
			nil,
		)
	}
	modelRef, err := service.resolveModelSelection(ctx, currentState, trimmedArgs)
	if err != nil {
		text := fmt.Sprintf("Failed to set model: %s\n%s", err.Error(), buildTelegramSystemUsage("model", "<provider/model>|default"))
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			nil,
		)
	}
	if err := service.syncGlobalModelSelection(ctx, modelRef.ProviderID, modelRef.ModelName); err != nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Failed to persist model selection: %s", err.Error()),
			nil,
		)
	}
	if err := service.syncDefaultAssistantPrimaryModel(ctx, modelRef.ProviderID, modelRef.ModelName); err != nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Failed to sync assistant primary model: %s", err.Error()),
			nil,
		)
	}
	updated := service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
		item.ModelProvider = modelRef.ProviderID
		item.ModelName = modelRef.ModelName
	})
	displayModel := buildTelegramModelDisplayNameOnly(modelRef)
	if strings.TrimSpace(displayModel) == "" {
		displayModel = service.describeEffectiveModel(ctx, updated)
	}
	return service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		fmt.Sprintf("Model set to %s (session override).\nGlobal settings + default assistant primary model synced.", displayModel),
		nil,
	)
}

func (service *BotService) handleTelegramModelsCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	catalog, err := service.listTelegramModelCatalog(ctx)
	if err != nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Failed to load models: %s", err.Error()),
			nil,
		)
	}
	if len(catalog) == 0 {
		text := "Model catalog is unavailable in Telegram right now.\nUse /model <provider/model> manually."
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			nil,
		)
	}
	modelRefs := make([]telegramModelRef, 0, len(catalog))
	modelRefs = append(modelRefs, catalog...)
	page := parseTelegramModelsPage(args)
	pageSize := 20
	totalPages := (len(modelRefs) + pageSize - 1) / pageSize
	if totalPages <= 0 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	if start > len(modelRefs) {
		start = 0
	}
	end := start + pageSize
	if end > len(modelRefs) {
		end = len(modelRefs)
	}
	pageItems := modelRefs[start:end]

	commandState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	activeProvider := strings.TrimSpace(commandState.ModelProvider)
	activeModel := strings.TrimSpace(commandState.ModelName)
	if activeProvider == "" || activeModel == "" {
		activeProvider, activeModel = service.resolveDefaultModel(ctx)
	}
	activeModelRef := buildTelegramModelRef(activeProvider, activeModel)

	rows := make([][]telegramCommandButton, 0, 12)
	for _, item := range pageItems {
		modelRef := buildTelegramModelRef(item.ProviderID, item.ModelName)
		callback := fmt.Sprintf("/model %s", modelRef)
		if len([]byte(callback)) > telegramCallbackDataMaxBytes {
			continue
		}
		label := shortTelegramModelLabel(buildTelegramModelDisplayRef(item))
		if strings.EqualFold(modelRef, activeModelRef) {
			label = "• " + truncateToRunes(label, 22)
		}
		rows = append(rows, []telegramCommandButton{
			{Text: label, Callback: callback},
		})
		if len(rows) >= 10 {
			break
		}
	}
	paging := make([]telegramCommandButton, 0, 2)
	if page > 1 {
		paging = append(paging, telegramCommandButton{Text: "Prev", Callback: fmt.Sprintf("/models %d", page-1)})
	}
	if page < totalPages {
		paging = append(paging, telegramCommandButton{Text: "Next", Callback: fmt.Sprintf("/models %d", page+1)})
	}
	if len(paging) > 0 {
		rows = append(rows, paging)
	}

	return service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		fmt.Sprintf("Select a model to use (%d total, page %d/%d).", len(modelRefs), page, totalPages),
		buildTelegramInlineKeyboard(rows...),
	)
}

func (service *BotService) handleTelegramNewCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	action := strings.ToLower(strings.TrimSpace(firstField(args)))
	switch action {
	case "confirm", "yes", "ok":
		updated := service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
			item.ConversationID = newTelegramConversationID()
			item.SessionOverride = ""
		})
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Started a new session branch: %s", service.resolveTelegramRuntimeSessionKey(baseSessionKey, updated)),
			nil,
		)
	case "cancel", "no":
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"New session canceled.",
			nil,
		)
	default:
		text := "Start a new session branch? This keeps your current command toggles."
		markup := buildTelegramInlineKeyboard(
			[]telegramCommandButton{
				{Text: "Confirm", Callback: "/new confirm"},
				{Text: "Cancel", Callback: "/new cancel"},
			},
		)
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			markup,
		)
	}
}

func (service *BotService) handleTelegramResetCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	action := strings.ToLower(strings.TrimSpace(firstField(args)))
	switch action {
	case "confirm", "yes", "ok":
		updated := service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
			item.ConversationID = newTelegramConversationID()
			item.SessionOverride = ""
			item.ModelProvider = ""
			item.ModelName = ""
			item.ThinkingLevel = ""
			item.QueueMode = ""
			item.UsageMode = ""
			item.VerboseMode = ""
			item.ReasoningMode = ""
			item.SkillName = ""
		})
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Session reset. New branch: %s", service.resolveTelegramRuntimeSessionKey(baseSessionKey, updated)),
			nil,
		)
	case "cancel", "no":
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Reset canceled.",
			nil,
		)
	default:
		text := "Reset this Telegram session? This starts a fresh branch and clears model/think/queue/usage overrides."
		markup := buildTelegramInlineKeyboard(
			[]telegramCommandButton{
				{Text: "Confirm", Callback: "/reset confirm"},
				{Text: "Cancel", Callback: "/reset cancel"},
			},
		)
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			markup,
		)
	}
}

type telegramSessionThread struct {
	ThreadID  string
	Title     string
	Channel   string
	UpdatedAt string
}

func telegramSessionSelectToken(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ""
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(trimmed))
	return fmt.Sprintf("%016x", hash.Sum64())
}

func resolveTelegramSessionByToken(
	items []telegramSessionThread,
	token string,
) (telegramSessionThread, bool, bool) {
	normalized := strings.ToLower(strings.TrimSpace(token))
	if normalized == "" {
		return telegramSessionThread{}, false, false
	}
	var matched telegramSessionThread
	matches := 0
	for _, item := range items {
		if telegramSessionSelectToken(item.ThreadID) != normalized {
			continue
		}
		matched = item
		matches++
		if matches > 1 {
			return telegramSessionThread{}, false, true
		}
	}
	if matches == 1 {
		return matched, true, false
	}
	return telegramSessionThread{}, false, false
}

func resolveTelegramSessionChannel(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return "unknown"
	}
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "telegram:"):
		return "telegram"
	case strings.HasPrefix(lower, "discord:"):
		return "discord"
	case strings.HasPrefix(lower, "slack:"):
		return "slack"
	}
	if parts, err := domainsession.ParseSessionKey(trimmed); err == nil {
		channel := strings.TrimSpace(parts.Channel)
		if channel != "" {
			return channel
		}
	}
	if !strings.Contains(trimmed, ":") {
		return "app"
	}
	return "unknown"
}

func shortTelegramSessionID(sessionID string) string {
	trimmed := strings.TrimSpace(sessionID)
	if trimmed == "" {
		return ""
	}
	if utf8.RuneCountInString(trimmed) <= 32 {
		return trimmed
	}
	head := truncateToRunes(trimmed, 14)
	tail := trimmed
	runes := []rune(trimmed)
	if len(runes) > 10 {
		tail = string(runes[len(runes)-10:])
	}
	head = strings.TrimSuffix(head, "…")
	return strings.TrimSpace(head + "…" + tail)
}

func (service *BotService) resolveTelegramThreadService() ThreadService {
	if service == nil {
		return nil
	}
	service.mu.Lock()
	threads := service.threads
	service.mu.Unlock()
	return threads
}

func (service *BotService) listTelegramSessionThreads(ctx context.Context) ([]telegramSessionThread, error) {
	threads := service.resolveTelegramThreadService()
	if threads == nil {
		return nil, nil
	}
	items, err := threads.ListThreads(ctx, false)
	if err != nil {
		return nil, err
	}
	result := make([]telegramSessionThread, 0, len(items))
	for _, item := range items {
		threadID := strings.TrimSpace(item.ID)
		if threadID == "" {
			continue
		}
		result = append(result, telegramSessionThread{
			ThreadID:  threadID,
			Title:     strings.TrimSpace(item.Title),
			Channel:   resolveTelegramSessionChannel(threadID),
			UpdatedAt: strings.TrimSpace(item.UpdatedAt),
		})
	}
	return result, nil
}

func (service *BotService) handleTelegramSessionsCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	trimmedArgs := strings.TrimSpace(args)
	fields := strings.Fields(trimmedArgs)
	action := strings.ToLower(strings.TrimSpace(firstField(trimmedArgs)))
	commandState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	currentSessionKey := service.resolveTelegramRuntimeSessionKey(baseSessionKey, commandState)

	sessions, err := service.listTelegramSessionThreads(ctx)
	if err != nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Failed to load sessions: %s", err.Error()),
			nil,
		)
	}
	if len(sessions) == 0 {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"No sessions found yet.",
			nil,
		)
	}

	if action == "use" || action == "switch" || action == "select" || action == "useid" {
		if len(fields) < 2 {
			return service.sendSystemMessage(
				ctx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				buildTelegramSystemUsage("sessions", "[page] | use <index>"),
				nil,
			)
		}
		target := telegramSessionThread{}
		index := 0
		if action == "useid" {
			token := strings.TrimSpace(fields[1])
			resolved, ok, collision := resolveTelegramSessionByToken(sessions, token)
			if !ok {
				errorText := "Session selection is no longer valid. Use /sessions to refresh."
				if collision {
					errorText = "Session token is ambiguous. Use /sessions use <index>."
				}
				return service.sendSystemMessage(
					ctx,
					state,
					message.Chat.ID,
					message.MessageThreadID,
					message.MessageID,
					errorText,
					nil,
				)
			}
			target = resolved
			for i, item := range sessions {
				if strings.EqualFold(strings.TrimSpace(item.ThreadID), strings.TrimSpace(target.ThreadID)) {
					index = i + 1
					break
				}
			}
		} else {
			parsedIndex, parseErr := strconv.Atoi(strings.TrimSpace(fields[1]))
			if parseErr != nil || parsedIndex <= 0 || parsedIndex > len(sessions) {
				return service.sendSystemMessage(
					ctx,
					state,
					message.Chat.ID,
					message.MessageThreadID,
					message.MessageID,
					fmt.Sprintf("Invalid session index %q. Use /sessions to view available sessions.", strings.TrimSpace(fields[1])),
					nil,
				)
			}
			index = parsedIndex
			target = sessions[index-1]
		}
		updated := service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
			item.ConversationID = ""
			item.SessionOverride = strings.TrimSpace(target.ThreadID)
		})
		switchedSession := service.resolveTelegramRuntimeSessionKey(baseSessionKey, updated)
		title := strings.TrimSpace(target.Title)
		if title == "" {
			title = "New Chat"
		}
		channel := strings.TrimSpace(target.Channel)
		if channel == "" {
			channel = "unknown"
		}
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Switched to session %d: %s\nChannel: %s\nTitle: %s", index, switchedSession, channel, title),
			nil,
		)
	}

	page := parseTelegramModelsPage(trimmedArgs)
	pageSize := 15
	totalPages := (len(sessions) + pageSize - 1) / pageSize
	if totalPages <= 0 {
		totalPages = 1
	}
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > len(sessions) {
		end = len(sessions)
	}
	pageItems := sessions[start:end]

	rows := make([][]telegramCommandButton, 0, 12)
	for _, item := range pageItems {
		label := strings.TrimSpace(item.Title)
		if label == "" {
			label = "New Chat"
		}
		channel := strings.TrimSpace(item.Channel)
		if channel == "" {
			channel = "unknown"
		}
		label = fmt.Sprintf("%s/%s", channel, truncateToRunes(label, 20))
		callback := fmt.Sprintf("/sessions useid %s", telegramSessionSelectToken(item.ThreadID))
		if strings.EqualFold(strings.TrimSpace(item.ThreadID), currentSessionKey) {
			label = "• " + truncateToRunes(label, 22)
		}
		rows = append(rows, []telegramCommandButton{{Text: label, Callback: callback}})
		if len(rows) >= 10 {
			break
		}
	}
	paging := make([]telegramCommandButton, 0, 2)
	if page > 1 {
		paging = append(paging, telegramCommandButton{Text: "Prev", Callback: fmt.Sprintf("/sessions %d", page-1)})
	}
	if page < totalPages {
		paging = append(paging, telegramCommandButton{Text: "Next", Callback: fmt.Sprintf("/sessions %d", page+1)})
	}
	if len(paging) > 0 {
		rows = append(rows, paging)
	}

	return service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		fmt.Sprintf("Select a session to continue (%d total, page %d/%d).", len(sessions), page, totalPages),
		buildTelegramInlineKeyboard(rows...),
	)
}

func (service *BotService) handleTelegramCompactCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	action := strings.ToLower(strings.TrimSpace(firstField(args)))
	switch action {
	case "confirm", "yes", "ok":
		updated := service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
			item.ConversationID = newTelegramConversationID()
			item.SessionOverride = ""
		})
		text := fmt.Sprintf(
			"Compaction completed by rotating to a fresh session branch: %s",
			service.resolveTelegramRuntimeSessionKey(baseSessionKey, updated),
		)
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			nil,
		)
	case "cancel", "no":
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Compaction canceled.",
			nil,
		)
	default:
		text := "Compact current context now? Telegram performs this by creating a new branch for subsequent runs."
		markup := buildTelegramInlineKeyboard(
			[]telegramCommandButton{
				{Text: "Confirm", Callback: "/compact confirm"},
				{Text: "Cancel", Callback: "/compact cancel"},
			},
		)
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			markup,
		)
	}
}

func (service *BotService) handleTelegramChoiceCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
	command string,
	currentValue string,
	usage string,
	choices []string,
	normalize func(string) string,
	apply func(item *telegramSessionCommandState, normalized string),
) error {
	if service == nil || state == nil || message == nil || normalize == nil || apply == nil {
		return nil
	}
	argValue := strings.TrimSpace(firstField(args))
	if argValue == "" {
		text := fmt.Sprintf(
			"/%s current: %s\n%s",
			command,
			currentValue,
			buildTelegramSystemUsage(command, usage),
		)
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			buildCommandModeButtons(command, choices),
		)
	}
	normalized := normalize(argValue)
	if normalized == "" {
		text := fmt.Sprintf("Invalid /%s value: %s\n%s", command, argValue, buildTelegramSystemUsage(command, usage))
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			buildCommandModeButtons(command, choices),
		)
	}
	updated := service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
		apply(item, normalized)
	})
	current := currentValue
	switch command {
	case "think":
		current = normalizeTelegramThinkingLevel(updated.ThinkingLevel)
	case "usage":
		current = normalizeTelegramUsageMode(updated.UsageMode)
	case "reasoning":
		current = normalizeTelegramReasoningMode(updated.ReasoningMode)
	case "verbose":
		current = normalizeTelegramVerboseMode(updated.VerboseMode)
	case "queue":
		current = normalizeTelegramQueueMode(updated.QueueMode)
	case "send":
		current = normalizeTelegramSendMode(updated.SendMode)
	case "activation":
		current = normalizeTelegramActivationMode(updated.ActivationMode)
	case "tts":
		current = normalizeTelegramTTSMode(updated.TTSMode)
	}
	if current == "" {
		current = normalized
	}
	return service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		fmt.Sprintf("/%s set to %s.", command, current),
		nil,
	)
}

func (service *BotService) handleTelegramThinkCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	current := normalizeTelegramThinkingLevel(currentState.ThinkingLevel)
	if current == "" {
		current = "default"
	}
	return service.handleTelegramChoiceCommand(
		ctx,
		state,
		message,
		baseSessionKey,
		args,
		"think",
		current,
		"<off|minimal|low|medium|high|xhigh>",
		[]string{"off", "minimal", "low", "medium", "high", "xhigh"},
		normalizeTelegramThinkingLevel,
		func(item *telegramSessionCommandState, normalized string) {
			item.ThinkingLevel = normalized
		},
	)
}

func (service *BotService) handleTelegramUsageCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	current := normalizeTelegramUsageMode(currentState.UsageMode)
	if current == "" {
		current = "off"
	}
	return service.handleTelegramChoiceCommand(
		ctx,
		state,
		message,
		baseSessionKey,
		args,
		"usage",
		current,
		"<off|tokens|full|cost>",
		[]string{"off", "tokens", "full", "cost"},
		normalizeTelegramUsageMode,
		func(item *telegramSessionCommandState, normalized string) {
			item.UsageMode = normalized
		},
	)
}

func (service *BotService) handleTelegramReasoningCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	current := normalizeTelegramReasoningMode(currentState.ReasoningMode)
	if current == "" {
		current = "off"
	}
	return service.handleTelegramChoiceCommand(
		ctx,
		state,
		message,
		baseSessionKey,
		args,
		"reasoning",
		current,
		"<off|on|stream>",
		[]string{"off", "on", "stream"},
		normalizeTelegramReasoningMode,
		func(item *telegramSessionCommandState, normalized string) {
			item.ReasoningMode = normalized
		},
	)
}

func (service *BotService) handleTelegramVerboseCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	current := normalizeTelegramVerboseMode(currentState.VerboseMode)
	if current == "" {
		current = "off"
	}
	return service.handleTelegramChoiceCommand(
		ctx,
		state,
		message,
		baseSessionKey,
		args,
		"verbose",
		current,
		"<off|on>",
		[]string{"off", "on"},
		normalizeTelegramVerboseMode,
		func(item *telegramSessionCommandState, normalized string) {
			item.VerboseMode = normalized
		},
	)
}

func (service *BotService) handleTelegramQueueCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	current := normalizeTelegramQueueMode(currentState.QueueMode)
	if current == "" {
		current = "default"
	}
	return service.handleTelegramChoiceCommand(
		ctx,
		state,
		message,
		baseSessionKey,
		args,
		"queue",
		current,
		"<steer|followup|collect>",
		[]string{"steer", "followup", "collect"},
		normalizeTelegramQueueMode,
		func(item *telegramSessionCommandState, normalized string) {
			item.QueueMode = normalized
		},
	)
}

func (service *BotService) handleTelegramSkillCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		current := strings.TrimSpace(service.getSessionCommandState(state.config.AccountID, baseSessionKey).SkillName)
		if current == "" {
			current = "none"
		}
		text := fmt.Sprintf("Current skill preference: %s\nUsage: /skill <name> | /skill off", current)
		markup := buildTelegramInlineKeyboard(
			[]telegramCommandButton{
				{Text: "Clear", Callback: "/skill off"},
			},
		)
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			text,
			markup,
		)
	}
	first := strings.TrimSpace(firstField(trimmed))
	switch strings.ToLower(first) {
	case "off", "none", "clear", "default", "inherit":
		service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
			item.SkillName = ""
		})
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Skill preference cleared.",
			nil,
		)
	default:
		service.updateSessionCommandState(state.config.AccountID, baseSessionKey, func(item *telegramSessionCommandState) {
			item.SkillName = first
		})
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Skill preference set to %s. Next runs will prioritize this skill when available.", first),
			nil,
		)
	}
}

func (service *BotService) handleTelegramTTSCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	current := normalizeTelegramTTSMode(currentState.TTSMode)
	if current == "" {
		current = "off"
	}
	return service.handleTelegramChoiceCommand(
		ctx,
		state,
		message,
		baseSessionKey,
		args,
		"tts",
		current,
		"<off|on>",
		[]string{"off", "on"},
		normalizeTelegramTTSMode,
		func(item *telegramSessionCommandState, normalized string) {
			item.TTSMode = normalized
		},
	)
}

func (service *BotService) handleTelegramActivationCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	current := normalizeTelegramActivationMode(currentState.ActivationMode)
	if current == "" {
		current = "mention"
	}
	return service.handleTelegramChoiceCommand(
		ctx,
		state,
		message,
		baseSessionKey,
		args,
		"activation",
		current,
		"<mention|always>",
		[]string{"mention", "always"},
		normalizeTelegramActivationMode,
		func(item *telegramSessionCommandState, normalized string) {
			item.ActivationMode = normalized
		},
	)
}

func (service *BotService) handleTelegramSendCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	currentState := service.getSessionCommandState(state.config.AccountID, baseSessionKey)
	current := normalizeTelegramSendMode(currentState.SendMode)
	if current == "" {
		current = "inherit"
	}
	return service.handleTelegramChoiceCommand(
		ctx,
		state,
		message,
		baseSessionKey,
		args,
		"send",
		current,
		"<on|off|inherit>",
		[]string{"on", "off", "inherit"},
		normalizeTelegramSendMode,
		func(item *telegramSessionCommandState, normalized string) {
			item.SendMode = normalized
		},
	)
}

func (service *BotService) resolveSubagentGateway() SubagentGateway {
	if service == nil {
		return nil
	}
	service.mu.Lock()
	gateway := service.subagents
	service.mu.Unlock()
	return gateway
}

func (service *BotService) resolveSubagentParentSession(accountID string, baseSessionKey string) string {
	state := service.getSessionCommandState(accountID, baseSessionKey)
	return service.resolveTelegramRuntimeSessionKey(baseSessionKey, state)
}

func parseTargetAndMessage(args string) (string, string) {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return "", ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", ""
	}
	target := strings.TrimSpace(parts[0])
	message := ""
	if len(parts) > 1 {
		message = strings.TrimSpace(strings.Join(parts[1:], " "))
	}
	return target, message
}

func sortSubagentRecords(items []subagentservice.RunRecord) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return strings.ToLower(strings.TrimSpace(items[i].RunID)) > strings.ToLower(strings.TrimSpace(items[j].RunID))
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func resolveSubagentRecordTarget(target string, records []subagentservice.RunRecord) (subagentservice.RunRecord, bool) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" || len(records) == 0 {
		return subagentservice.RunRecord{}, false
	}
	if index, err := strconv.Atoi(trimmed); err == nil && index > 0 && index <= len(records) {
		return records[index-1], true
	}
	for _, record := range records {
		if strings.EqualFold(strings.TrimSpace(record.RunID), trimmed) {
			return record, true
		}
		if label := strings.TrimSpace(record.Label); label != "" && strings.EqualFold(label, trimmed) {
			return record, true
		}
	}
	for _, record := range records {
		runID := strings.TrimSpace(record.RunID)
		if runID != "" && strings.HasPrefix(strings.ToLower(runID), strings.ToLower(trimmed)) {
			return record, true
		}
	}
	return subagentservice.RunRecord{}, false
}

func (service *BotService) handleTelegramSubagentsCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	gateway := service.resolveSubagentGateway()
	if gateway == nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Subagent service is unavailable.",
			nil,
		)
	}
	action := strings.ToLower(strings.TrimSpace(firstField(args)))
	remaining := strings.TrimSpace(args)
	if action != "" {
		remaining = strings.TrimSpace(strings.TrimPrefix(remaining, firstField(args)))
	}
	parentSessionKey := service.resolveSubagentParentSession(state.config.AccountID, baseSessionKey)
	switch action {
	case "", "list":
		records, err := gateway.ListByParent(ctx, parentSessionKey)
		if err != nil {
			return service.sendSystemMessage(
				ctx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				fmt.Sprintf("Failed to list subagents: %s", err.Error()),
				nil,
			)
		}
		if len(records) == 0 {
			return service.sendSystemMessage(
				ctx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				"No subagent runs for this session.",
				nil,
			)
		}
		sortSubagentRecords(records)
		var builder strings.Builder
		builder.WriteString("Subagent runs:\n")
		limit := len(records)
		if limit > 20 {
			limit = 20
		}
		for index := 0; index < limit; index++ {
			record := records[index]
			label := strings.TrimSpace(record.Label)
			if label == "" {
				label = strings.TrimSpace(record.Task)
			}
			if label == "" {
				label = "-"
			}
			builder.WriteString(fmt.Sprintf("%d. [%s] %s (%s)\n", index+1, string(record.Status), record.RunID, label))
		}
		if len(records) > limit {
			builder.WriteString(fmt.Sprintf("... and %d more\n", len(records)-limit))
		}
		builder.WriteString("\nUse /kill <id|index|all> or /steer <id|index> <message>.")
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			truncateTelegramText(builder.String(), 3600),
			nil,
		)
	case "kill":
		return service.handleTelegramKillCommand(ctx, state, message, baseSessionKey, remaining)
	case "steer":
		return service.handleTelegramSteerCommand(ctx, state, message, baseSessionKey, remaining)
	case "send":
		target, msgText := parseTargetAndMessage(remaining)
		if strings.TrimSpace(target) == "" || strings.TrimSpace(msgText) == "" {
			return service.sendSystemMessage(
				ctx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				buildTelegramSystemUsage("subagents send", "<id|index> <message>"),
				nil,
			)
		}
		records, _ := gateway.ListByParent(ctx, parentSessionKey)
		sortSubagentRecords(records)
		record, ok := resolveSubagentRecordTarget(target, records)
		runID := strings.TrimSpace(target)
		if ok {
			runID = strings.TrimSpace(record.RunID)
		}
		if runID == "" {
			return service.sendSystemMessage(
				ctx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				"Subagent target not found.",
				nil,
			)
		}
		if err := gateway.Send(ctx, runID, msgText); err != nil {
			return service.sendSystemMessage(
				ctx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				fmt.Sprintf("Failed to send to subagent %s: %s", runID, err.Error()),
				nil,
			)
		}
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Sent follow-up to subagent %s.", runID),
			nil,
		)
	case "info":
		target := strings.TrimSpace(firstField(remaining))
		if target == "" {
			return service.sendSystemMessage(
				ctx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				buildTelegramSystemUsage("subagents info", "<id|index>"),
				nil,
			)
		}
		records, _ := gateway.ListByParent(ctx, parentSessionKey)
		sortSubagentRecords(records)
		record, ok := resolveSubagentRecordTarget(target, records)
		if !ok {
			loaded, err := gateway.Get(ctx, target)
			if err != nil {
				return service.sendSystemMessage(
					ctx,
					state,
					message.Chat.ID,
					message.MessageThreadID,
					message.MessageID,
					"Subagent target not found.",
					nil,
				)
			}
			record = loaded
		}
		infoLines := []string{
			fmt.Sprintf("Run: %s", strings.TrimSpace(record.RunID)),
			fmt.Sprintf("Status: %s", string(record.Status)),
			fmt.Sprintf("Task: %s", strings.TrimSpace(record.Task)),
			fmt.Sprintf("Label: %s", strings.TrimSpace(record.Label)),
			fmt.Sprintf("Model: %s", strings.TrimSpace(record.Model)),
			fmt.Sprintf("Thinking: %s", strings.TrimSpace(record.Thinking)),
		}
		if record.RuntimeMs > 0 {
			infoLines = append(infoLines, fmt.Sprintf("Runtime: %dms", record.RuntimeMs))
		}
		if strings.TrimSpace(record.Error) != "" {
			infoLines = append(infoLines, fmt.Sprintf("Error: %s", strings.TrimSpace(record.Error)))
		}
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			truncateTelegramText(strings.Join(infoLines, "\n"), 3600),
			nil,
		)
	default:
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Usage: /subagents list | /subagents kill <id|index|all> | /subagents steer <id|index> <message> | /subagents send <id|index> <message> | /subagents info <id|index>",
			nil,
		)
	}
}

func (service *BotService) handleTelegramKillCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	gateway := service.resolveSubagentGateway()
	if gateway == nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Subagent service is unavailable.",
			nil,
		)
	}
	target := strings.TrimSpace(firstField(args))
	if target == "" {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			buildTelegramSystemUsage("kill", "<id|index|all>"),
			nil,
		)
	}
	parentSessionKey := service.resolveSubagentParentSession(state.config.AccountID, baseSessionKey)
	if strings.EqualFold(target, "all") {
		stopped, err := gateway.KillAll(ctx, parentSessionKey)
		if err != nil {
			return service.sendSystemMessage(
				ctx,
				state,
				message.Chat.ID,
				message.MessageThreadID,
				message.MessageID,
				fmt.Sprintf("Failed to kill subagents: %s", err.Error()),
				nil,
			)
		}
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Kill signal sent to %d subagent run(s).", stopped),
			nil,
		)
	}
	records, _ := gateway.ListByParent(ctx, parentSessionKey)
	sortSubagentRecords(records)
	record, ok := resolveSubagentRecordTarget(target, records)
	runID := strings.TrimSpace(target)
	if ok {
		runID = strings.TrimSpace(record.RunID)
	}
	if runID == "" {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Subagent target not found.",
			nil,
		)
	}
	if err := gateway.Kill(ctx, runID); err != nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Failed to kill subagent %s: %s", runID, err.Error()),
			nil,
		)
	}
	return service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		fmt.Sprintf("Kill signal sent for subagent %s.", runID),
		nil,
	)
}

func (service *BotService) handleTelegramSteerCommand(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	baseSessionKey string,
	args string,
) error {
	if service == nil || state == nil || message == nil {
		return nil
	}
	gateway := service.resolveSubagentGateway()
	if gateway == nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Subagent service is unavailable.",
			nil,
		)
	}
	target, guidance := parseTargetAndMessage(args)
	if strings.TrimSpace(target) == "" || strings.TrimSpace(guidance) == "" {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			buildTelegramSystemUsage("steer", "<id|index> <message>"),
			nil,
		)
	}
	parentSessionKey := service.resolveSubagentParentSession(state.config.AccountID, baseSessionKey)
	records, _ := gateway.ListByParent(ctx, parentSessionKey)
	sortSubagentRecords(records)
	record, ok := resolveSubagentRecordTarget(target, records)
	runID := strings.TrimSpace(target)
	if ok {
		runID = strings.TrimSpace(record.RunID)
	}
	if runID == "" {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			"Subagent target not found.",
			nil,
		)
	}
	if err := gateway.Steer(ctx, runID, guidance); err != nil {
		return service.sendSystemMessage(
			ctx,
			state,
			message.Chat.ID,
			message.MessageThreadID,
			message.MessageID,
			fmt.Sprintf("Failed to steer subagent %s: %s", runID, err.Error()),
			nil,
		)
	}
	return service.sendSystemMessage(
		ctx,
		state,
		message.Chat.ID,
		message.MessageThreadID,
		message.MessageID,
		fmt.Sprintf("Steering sent to subagent %s.", runID),
		nil,
	)
}

func buildSessionRunKey(accountID, sessionKey string) string {
	return strings.TrimSpace(accountID) + "::" + strings.TrimSpace(sessionKey)
}

func (service *BotService) registerActiveRun(accountID, sessionKey, runID string, cancel context.CancelFunc) {
	if service == nil || strings.TrimSpace(runID) == "" || cancel == nil {
		return
	}
	entry := telegramActiveRun{
		AccountID:  strings.TrimSpace(accountID),
		SessionKey: strings.TrimSpace(sessionKey),
		Cancel:     cancel,
		StartedAt:  service.now(),
	}
	service.mu.Lock()
	service.activeRuns[strings.TrimSpace(runID)] = entry
	sessionRunKey := buildSessionRunKey(entry.AccountID, entry.SessionKey)
	if sessionRunKey != "::" {
		service.sessionRuns[sessionRunKey] = strings.TrimSpace(runID)
	}
	service.mu.Unlock()
}

func (service *BotService) unregisterActiveRun(runID string) {
	if service == nil {
		return
	}
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return
	}
	service.mu.Lock()
	entry, ok := service.activeRuns[trimmedRunID]
	if ok {
		sessionRunKey := buildSessionRunKey(entry.AccountID, entry.SessionKey)
		if currentRunID, exists := service.sessionRuns[sessionRunKey]; exists && currentRunID == trimmedRunID {
			delete(service.sessionRuns, sessionRunKey)
		}
		delete(service.activeRuns, trimmedRunID)
	}
	service.mu.Unlock()
}

func (service *BotService) abortActiveRun(
	ctx context.Context,
	accountID string,
	sessionKey string,
	runID string,
	reason string,
) (bool, string, error) {
	_ = ctx
	_ = reason
	if service == nil {
		return false, "", nil
	}
	targetAccountID := strings.TrimSpace(accountID)
	targetSessionKey := strings.TrimSpace(sessionKey)
	targetRunID := strings.TrimSpace(runID)

	var cancel context.CancelFunc
	resolvedRunID := targetRunID

	service.mu.Lock()
	if targetRunID != "" {
		if entry, ok := service.activeRuns[targetRunID]; ok {
			if strings.EqualFold(entry.AccountID, targetAccountID) &&
				(targetSessionKey == "" || targetSessionKey == entry.SessionKey) {
				cancel = entry.Cancel
				resolvedRunID = targetRunID
			}
		}
	} else if targetSessionKey != "" {
		sessionRunKey := buildSessionRunKey(targetAccountID, targetSessionKey)
		if matchedRunID, ok := service.sessionRuns[sessionRunKey]; ok {
			if entry, exists := service.activeRuns[matchedRunID]; exists && strings.EqualFold(entry.AccountID, targetAccountID) {
				cancel = entry.Cancel
				resolvedRunID = matchedRunID
			}
		}
	}
	service.mu.Unlock()

	if cancel != nil {
		cancel()
		return true, resolvedRunID, nil
	}
	return false, resolvedRunID, nil
}

type telegramPolicyContext struct {
	SystemPrompt string
	Tools        []string
}

func allowMessage(state *telegramAccountState, message *telegramapi.Message, text string, senderIDOverride string, allowWithoutMention bool) (bool, string, telegramPolicyContext) {
	if state == nil || message == nil {
		return false, "invalid_message", telegramPolicyContext{}
	}
	chatType := strings.ToLower(strings.TrimSpace(message.Chat.Type))
	senderID := strings.TrimSpace(senderIDOverride)
	if senderID == "" {
		senderID = resolveMessageUserID(message)
	}
	if chatType == "private" {
		allowed, reason := allowDM(state, senderID)
		return allowed, reason, telegramPolicyContext{}
	}

	groupCfg, groupMatched := resolveGroupConfig(state.config.Groups, message.Chat.ID)
	topicCfg, topicMatched, topicAllowed := resolveTopicConfig(groupCfg, int64(message.MessageThreadID))
	policyCtx := resolvePolicyContext(groupCfg, topicCfg, topicMatched)

	if allowedByList(state.config.GroupAllowFrom, senderID) {
		return true, "", policyCtx
	}
	if allowedByList(groupCfg.AllowFrom, senderID) {
		return true, "", policyCtx
	}
	if topicMatched && allowedByList(topicCfg.AllowFrom, senderID) {
		return true, "", policyCtx
	}

	switch state.config.GroupPolicy {
	case GroupPolicyDisabled:
		return false, "group_policy_disabled", policyCtx
	case GroupPolicyAllowlist:
		if !groupMatched || !isGroupEnabled(groupCfg) {
			return false, "group_not_allowed", policyCtx
		}
		if topicMatched && !topicAllowed {
			return false, "topic_not_allowed", policyCtx
		}
		if !allowWithoutMention && resolveRequireMention(groupCfg, topicCfg, topicMatched) && !mentionsBot(state, message, text) {
			return false, "group_requires_mention", policyCtx
		}
		return true, "", policyCtx
	case GroupPolicyOpen:
		if topicMatched && !topicAllowed {
			return false, "topic_not_allowed", policyCtx
		}
		if !allowWithoutMention && resolveRequireMention(groupCfg, topicCfg, topicMatched) && !mentionsBot(state, message, text) {
			return false, "group_requires_mention", policyCtx
		}
		return true, "", policyCtx
	default:
		return true, "", policyCtx
	}
}

func allowDM(state *telegramAccountState, senderID string) (bool, string) {
	switch state.config.DMPolicy {
	case DMPolicyDisabled:
		return false, "dm_policy_disabled"
	case DMPolicyOpen:
		return true, ""
	case DMPolicyAllowlist, DMPolicyPairing:
		if allowedByList(state.config.AllowFrom, senderID) {
			return true, ""
		}
		if state.config.DMPolicy == DMPolicyAllowlist {
			return false, "dm_allowlist"
		}
		return false, "dm_pairing_required"
	default:
		return true, ""
	}
}

func allowedByList(list map[string]struct{}, senderID string) bool {
	if len(list) == 0 {
		return false
	}
	if _, ok := list["*"]; ok {
		return true
	}
	if senderID == "" {
		return false
	}
	_, ok := list[senderID]
	return ok
}

func resolveGroupConfig(groups map[string]TelegramGroupConfig, chatID int64) (TelegramGroupConfig, bool) {
	if len(groups) == 0 {
		return TelegramGroupConfig{}, false
	}
	key := fmt.Sprintf("%d", chatID)
	if entry, ok := groups[key]; ok {
		return entry, true
	}
	if entry, ok := groups["*"]; ok {
		return entry, true
	}
	return TelegramGroupConfig{}, false
}

func resolveTopicConfig(group TelegramGroupConfig, threadID int64) (TelegramTopicConfig, bool, bool) {
	if threadID <= 0 || len(group.Topics) == 0 {
		return TelegramTopicConfig{}, false, true
	}
	key := fmt.Sprintf("%d", threadID)
	if entry, ok := group.Topics[key]; ok {
		return entry, true, isTopicEnabled(entry)
	}
	if entry, ok := group.Topics["*"]; ok {
		return entry, true, isTopicEnabled(entry)
	}
	return TelegramTopicConfig{}, false, true
}

func resolvePolicyContext(group TelegramGroupConfig, topic TelegramTopicConfig, topicMatched bool) telegramPolicyContext {
	ctx := telegramPolicyContext{
		SystemPrompt: strings.TrimSpace(group.SystemPrompt),
		Tools:        append([]string(nil), group.Tools...),
	}
	if topicMatched {
		if strings.TrimSpace(topic.SystemPrompt) != "" {
			ctx.SystemPrompt = strings.TrimSpace(topic.SystemPrompt)
		}
		if len(topic.Tools) > 0 {
			ctx.Tools = append([]string(nil), topic.Tools...)
		}
	}
	return ctx
}

func resolveRequireMention(group TelegramGroupConfig, topic TelegramTopicConfig, topicMatched bool) bool {
	if topicMatched && topic.RequireMention != nil {
		return *topic.RequireMention
	}
	if group.RequireMention != nil {
		return *group.RequireMention
	}
	return true
}

func isGroupEnabled(group TelegramGroupConfig) bool {
	if group.Enabled == nil {
		return true
	}
	return *group.Enabled
}

func isTopicEnabled(topic TelegramTopicConfig) bool {
	if topic.Enabled == nil {
		return true
	}
	return *topic.Enabled
}

func mentionsBot(state *telegramAccountState, message *telegramapi.Message, text string) bool {
	if state == nil || message == nil {
		return false
	}
	if message.ReplyToMessage != nil && message.ReplyToMessage.From != nil {
		if state.botID != 0 && message.ReplyToMessage.From.ID == state.botID {
			return true
		}
	}
	mention := strings.ToLower(strings.TrimSpace("@" + state.botUsername))
	if mention != "@" && strings.Contains(strings.ToLower(text), mention) {
		return true
	}
	return false
}

func shouldReplyToMessage(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "reply", "always":
		return true
	default:
		return false
	}
}

func resolveMessageUserID(message *telegramapi.Message) string {
	if message == nil {
		return ""
	}
	if message.From != nil {
		return fmt.Sprintf("%d", message.From.ID)
	}
	return ""
}

func resolveTelegramPeerName(message *telegramapi.Message, peerKind string) string {
	if message == nil {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(peerKind), "group") {
		return strings.TrimSpace(message.Chat.Title)
	}
	parts := make([]string, 0, 2)
	if message.From != nil {
		if first := strings.TrimSpace(message.From.FirstName); first != "" {
			parts = append(parts, first)
		}
		if last := strings.TrimSpace(message.From.LastName); last != "" {
			parts = append(parts, last)
		}
	}
	if len(parts) > 0 {
		return strings.TrimSpace(strings.Join(parts, " "))
	}
	if message.From != nil {
		return strings.TrimSpace(message.From.Username)
	}
	return ""
}

func resolveTelegramPeerUsername(message *telegramapi.Message, peerKind string) string {
	if message == nil {
		return ""
	}
	if strings.EqualFold(strings.TrimSpace(peerKind), "group") {
		return strings.TrimSpace(message.Chat.Username)
	}
	if message.From != nil {
		return strings.TrimSpace(message.From.Username)
	}
	return ""
}

func resolveTelegramPeerAvatarURL(username string) string {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return ""
	}
	escaped := url.PathEscape(strings.TrimPrefix(trimmed, "@"))
	if escaped == "" {
		return ""
	}
	return "https://t.me/i/userpic/320/" + escaped + ".jpg"
}

func (service *BotService) resolveTelegramPeerProfile(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	peerKind string,
	peerID string,
) telegramPeerProfile {
	fallback := telegramPeerProfile{
		Name:      resolveTelegramPeerName(message, peerKind),
		Username:  resolveTelegramPeerUsername(message, peerKind),
		AvatarURL: "",
	}
	fallback.AvatarURL = resolveTelegramPeerAvatarURL(fallback.Username)

	cacheKey := buildTelegramPeerProfileCacheKey(peerKind, peerID, message)
	if cached, ok := loadTelegramPeerProfileFromCache(service, state, cacheKey); ok {
		return mergeTelegramPeerProfile(cached, fallback)
	}
	if state == nil || state.bot == nil {
		return fallback
	}

	lookupCtx := ctx
	if lookupCtx == nil {
		lookupCtx = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(lookupCtx, telegramPeerProfileLookupTimeout)
	defer cancel()

	resolved := fetchTelegramPeerProfileFromAPI(timeoutCtx, state, message, peerKind, peerID)
	resolved = mergeTelegramPeerProfile(resolved, fallback)
	if strings.TrimSpace(resolved.AvatarURL) == "" {
		if stableAvatarURL := resolveTelegramPeerAvatarURL(resolved.Username); stableAvatarURL != "" {
			resolved.AvatarURL = stableAvatarURL
		}
	}
	if cacheKey != "" {
		storeTelegramPeerProfileToCache(service, state, cacheKey, resolved)
	}
	return resolved
}

func buildTelegramPeerProfileCacheKey(peerKind string, peerID string, message *telegramapi.Message) string {
	kind := strings.ToLower(strings.TrimSpace(peerKind))
	if kind == "" {
		kind = "direct"
	}
	id := strings.TrimSpace(peerID)
	if id == "" && message != nil {
		if kind == "group" {
			id = fmt.Sprintf("%d", message.Chat.ID)
		} else if message.From != nil {
			id = fmt.Sprintf("%d", message.From.ID)
		}
	}
	if id == "" {
		return ""
	}
	return kind + ":" + id
}

func loadTelegramPeerProfileFromCache(
	service *BotService,
	state *telegramAccountState,
	cacheKey string,
) (telegramPeerProfile, bool) {
	if service == nil || state == nil || cacheKey == "" {
		return telegramPeerProfile{}, false
	}
	raw, ok := state.peerProfileCache.Load(cacheKey)
	if !ok {
		return telegramPeerProfile{}, false
	}
	entry, ok := raw.(telegramPeerProfileCacheEntry)
	if !ok {
		state.peerProfileCache.Delete(cacheKey)
		return telegramPeerProfile{}, false
	}
	now := time.Now()
	if service.now != nil {
		now = service.now()
	}
	if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
		state.peerProfileCache.Delete(cacheKey)
		return telegramPeerProfile{}, false
	}
	return entry.Profile, true
}

func storeTelegramPeerProfileToCache(
	service *BotService,
	state *telegramAccountState,
	cacheKey string,
	profile telegramPeerProfile,
) {
	if service == nil || state == nil || cacheKey == "" {
		return
	}
	now := time.Now()
	if service.now != nil {
		now = service.now()
	}
	state.peerProfileCache.Store(cacheKey, telegramPeerProfileCacheEntry{
		Profile:   profile,
		ExpiresAt: now.Add(telegramPeerProfileCacheTTL),
	})
}

func clearTelegramPeerProfileCache(state *telegramAccountState) {
	if state == nil {
		return
	}
	state.peerProfileCache.Range(func(key any, _ any) bool {
		state.peerProfileCache.Delete(key)
		return true
	})
}

func fetchTelegramPeerProfileFromAPI(
	ctx context.Context,
	state *telegramAccountState,
	message *telegramapi.Message,
	peerKind string,
	peerID string,
) telegramPeerProfile {
	result := telegramPeerProfile{}
	if state == nil || state.bot == nil {
		return result
	}

	normalizedKind := strings.ToLower(strings.TrimSpace(peerKind))
	if normalizedKind == "group" {
		groupID, ok := resolveTelegramGroupID(peerID, message)
		if !ok {
			return result
		}
		chat, err := state.bot.GetChat(ctx, &telego.GetChatParams{ChatID: tu.ID(groupID)})
		if err != nil || chat == nil {
			return result
		}
		result.Name = strings.TrimSpace(chat.Title)
		result.Username = strings.TrimSpace(chat.Username)
		fileID := resolveTelegramChatPhotoFileID(chat.Photo)
		result.AvatarURL = resolveTelegramFileURLByFileID(ctx, state, fileID)
		return result
	}

	userID, ok := resolveTelegramUserID(peerID, message)
	if !ok {
		return result
	}
	if chat, err := state.bot.GetChat(ctx, &telego.GetChatParams{ChatID: tu.ID(userID)}); err == nil && chat != nil {
		parts := make([]string, 0, 2)
		if first := strings.TrimSpace(chat.FirstName); first != "" {
			parts = append(parts, first)
		}
		if last := strings.TrimSpace(chat.LastName); last != "" {
			parts = append(parts, last)
		}
		if len(parts) > 0 {
			result.Name = strings.TrimSpace(strings.Join(parts, " "))
		}
		result.Username = strings.TrimSpace(chat.Username)
		if result.Name == "" {
			result.Name = result.Username
		}
		fileID := resolveTelegramChatPhotoFileID(chat.Photo)
		result.AvatarURL = resolveTelegramFileURLByFileID(ctx, state, fileID)
	}
	if strings.TrimSpace(result.AvatarURL) != "" {
		return result
	}
	photos, err := state.bot.GetUserProfilePhotos(ctx, &telego.GetUserProfilePhotosParams{
		UserID: userID,
		Limit:  1,
	})
	if err != nil {
		return result
	}
	fileID := resolveTelegramUserProfilePhotoFileID(photos)
	if fileID == "" {
		return result
	}
	result.AvatarURL = resolveTelegramFileURLByFileID(ctx, state, fileID)
	return result
}

func resolveTelegramGroupID(peerID string, message *telegramapi.Message) (int64, bool) {
	trimmed := strings.TrimSpace(peerID)
	if trimmed != "" {
		if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil && parsed != 0 {
			return parsed, true
		}
	}
	if message != nil && message.Chat.ID != 0 {
		return message.Chat.ID, true
	}
	return 0, false
}

func resolveTelegramUserID(peerID string, message *telegramapi.Message) (int64, bool) {
	trimmed := strings.TrimSpace(peerID)
	if trimmed != "" {
		if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil && parsed != 0 {
			return parsed, true
		}
	}
	if message != nil && message.From != nil && message.From.ID != 0 {
		return message.From.ID, true
	}
	return 0, false
}

func resolveTelegramChatPhotoFileID(photo *telego.ChatPhoto) string {
	if photo == nil {
		return ""
	}
	if fileID := strings.TrimSpace(photo.SmallFileID); fileID != "" {
		return fileID
	}
	return strings.TrimSpace(photo.BigFileID)
}

func resolveTelegramUserProfilePhotoFileID(photos *telego.UserProfilePhotos) string {
	if photos == nil || len(photos.Photos) == 0 {
		return ""
	}
	sizes := photos.Photos[0]
	if len(sizes) == 0 {
		return ""
	}
	return strings.TrimSpace(sizes[len(sizes)-1].FileID)
}

func resolveTelegramFileURLByFileID(ctx context.Context, state *telegramAccountState, fileID string) string {
	if state == nil || state.bot == nil {
		return ""
	}
	trimmed := strings.TrimSpace(fileID)
	if trimmed == "" {
		return ""
	}
	file, err := state.bot.GetFile(ctx, &telego.GetFileParams{FileID: trimmed})
	if err != nil || file == nil {
		return ""
	}
	filePath := strings.TrimSpace(file.FilePath)
	if filePath == "" {
		return ""
	}
	return strings.TrimSpace(state.bot.FileDownloadURL(filePath))
}

func mergeTelegramPeerProfile(primary telegramPeerProfile, fallback telegramPeerProfile) telegramPeerProfile {
	result := primary
	if strings.TrimSpace(result.Name) == "" {
		result.Name = strings.TrimSpace(fallback.Name)
	}
	if strings.TrimSpace(result.Username) == "" {
		result.Username = strings.TrimSpace(fallback.Username)
	}
	if strings.TrimSpace(result.AvatarURL) == "" {
		result.AvatarURL = strings.TrimSpace(fallback.AvatarURL)
	}
	return result
}

func (service *BotService) handlePairing(ctx context.Context, state *telegramAccountState, message *telegramapi.Message) {
	if service == nil || state == nil || message == nil {
		return
	}
	service.mu.Lock()
	store := service.pairing
	httpClient := service.httpClient
	service.mu.Unlock()
	if store == nil || httpClient == nil {
		return
	}
	senderID := resolveMessageUserID(message)
	if senderID == "" {
		return
	}
	meta := map[string]string{}
	if message.From != nil {
		if message.From.Username != "" {
			meta["username"] = message.From.Username
		}
		if message.From.FirstName != "" {
			meta["firstName"] = message.From.FirstName
		}
		if message.From.LastName != "" {
			meta["lastName"] = message.From.LastName
		}
	}
	result, err := store.Upsert(channelpairing.UpsertParams{
		ID:        senderID,
		AccountID: state.config.AccountID,
		Meta:      meta,
	})
	if err != nil {
		zap.L().Warn(
			"telegram pairing request failed",
			zap.String("accountId", state.config.AccountID),
			zap.String("error", redactTelegramToken(err.Error(), state.config.BotToken)),
		)
		return
	}
	if !result.Created || strings.TrimSpace(result.Code) == "" {
		return
	}
	reply := channelpairing.BuildPairingReply("telegram", fmt.Sprintf("Your Telegram user id: %s", senderID), result.Code)
	client := telegramapi.NewClient(state.config.BotToken, httpClient)
	if _, err := client.SendMessage(ctx, telegramapi.SendMessageParams{
		ChatID: message.Chat.ID,
		Text:   reply,
	}); err != nil {
		zap.L().Warn(
			"telegram pairing reply failed",
			zap.String("accountId", state.config.AccountID),
			zap.String("error", redactTelegramToken(err.Error(), state.config.BotToken)),
		)
	}
}

func buildSessionKey(accountID string, chat telegramapi.Chat, threadID int64) string {
	kind := strings.ToLower(strings.TrimSpace(chat.Type))
	if kind == "" {
		kind = "chat"
	}
	if accountID == "" {
		accountID = DefaultTelegramAccountID
	}
	base := fmt.Sprintf("telegram:%s:%s:%d", accountID, kind, chat.ID)
	if threadID > 0 {
		return fmt.Sprintf("%s:thread:%d", base, threadID)
	}
	return base
}

func parseTelegramSessionKey(raw string) (accountID string, chatID int64, threadID int, ok bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", 0, 0, false
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) < 4 || !strings.EqualFold(parts[0], "telegram") {
		return "", 0, 0, false
	}
	accountID = strings.TrimSpace(parts[1])
	if accountID == "" {
		accountID = DefaultTelegramAccountID
	}
	chatValue := strings.TrimSpace(parts[3])
	parsedChatID, err := strconv.ParseInt(chatValue, 10, 64)
	if err != nil {
		return "", 0, 0, false
	}
	threadID = 0
	if len(parts) >= 6 && strings.EqualFold(strings.TrimSpace(parts[4]), "thread") {
		if parsedThreadID, threadErr := strconv.ParseInt(strings.TrimSpace(parts[5]), 10, 64); threadErr == nil && parsedThreadID > 0 {
			threadID = int(parsedThreadID)
		}
	}
	return accountID, parsedChatID, threadID, true
}

func (service *BotService) ForwardExecApprovalRequested(ctx context.Context, request gatewayapprovals.Request) error {
	if service == nil {
		return nil
	}
	accountID, chatID, threadID, ok := parseTelegramSessionKey(request.SessionKey)
	if !ok {
		return nil
	}
	service.mu.Lock()
	state := service.accounts[accountID]
	service.mu.Unlock()
	if state == nil || !state.running || state.bot == nil {
		return nil
	}
	commandSummary := strings.TrimSpace(request.ToolName)
	if action := strings.TrimSpace(request.Action); action != "" {
		if commandSummary != "" {
			commandSummary += " "
		}
		commandSummary += action
	}
	if commandSummary == "" {
		commandSummary = "tool execution"
	}
	lines := []string{
		"Exec approval required.",
		fmt.Sprintf("ID: %s", request.ID),
		fmt.Sprintf("Target: %s", commandSummary),
	}
	if args := strings.TrimSpace(request.Args); args != "" {
		lines = append(lines, fmt.Sprintf("Args: %s", truncateTelegramApprovalArgs(args)))
	}
	lines = append(lines, "Click a button below to continue.")
	lines = append(lines, fmt.Sprintf("Or reply: /approve %s approve|deny", request.ID))
	keyboard := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Approve").WithCallbackData(fmt.Sprintf("/approve %s approve", request.ID)),
			tu.InlineKeyboardButton("Deny").WithCallbackData(fmt.Sprintf("/approve %s deny", request.ID)),
		),
	)
	return service.sendSystemMessage(
		ctx,
		state,
		chatID,
		threadID,
		0,
		strings.Join(lines, "\n"),
		keyboard,
	)
}

func (service *BotService) ForwardExecApprovalResolved(ctx context.Context, request gatewayapprovals.Request) error {
	if service == nil {
		return nil
	}
	if shouldSuppressTelegramResolvedForward(request.Reason) {
		return nil
	}
	accountID, chatID, threadID, ok := parseTelegramSessionKey(request.SessionKey)
	if !ok {
		return nil
	}
	service.mu.Lock()
	state := service.accounts[accountID]
	service.mu.Unlock()
	if state == nil || !state.running || state.bot == nil {
		return nil
	}
	statusText := "denied"
	if request.Status == gatewayapprovals.StatusApproved {
		statusText = "approved"
	}
	reason := strings.TrimSpace(request.Reason)
	text := fmt.Sprintf("Exec approval %s: %s", statusText, request.ID)
	if reason != "" {
		text = fmt.Sprintf("%s (%s)", text, reason)
	}
	return service.sendSystemMessage(ctx, state, chatID, threadID, 0, text, nil)
}

func shouldSuppressTelegramResolvedForward(reason string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(reason))
	return strings.HasPrefix(trimmed, "telegram")
}

func truncateTelegramApprovalArgs(input string) string {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) <= 120 {
		return trimmed
	}
	return trimmed[:117] + "..."
}

func firstMessage(update telegramapi.Update) *telegramapi.Message {
	if update.Message != nil {
		return update.Message
	}
	if update.EditedMessage != nil {
		return update.EditedMessage
	}
	if update.ChannelPost != nil {
		return update.ChannelPost
	}
	if update.EditedChannelPost != nil {
		return update.EditedChannelPost
	}
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		if msg := update.CallbackQuery.Message.Message(); msg != nil {
			return msg
		}
		// Callback queries may reference an inaccessible message (date=0). Build a
		// minimal message envelope so callback-driven commands can still be handled.
		if inaccessible := update.CallbackQuery.Message.InaccessibleMessage(); inaccessible != nil {
			return &telegramapi.Message{
				MessageID: inaccessible.GetMessageID(),
				Chat:      inaccessible.GetChat(),
			}
		}
	}
	return nil
}

func stateNeedsRestart(state *telegramAccountState, config TelegramAccountConfig) bool {
	if state == nil {
		return true
	}
	if !state.running && config.Enabled && strings.TrimSpace(config.BotToken) != "" {
		return true
	}
	if strings.TrimSpace(state.config.BotToken) != strings.TrimSpace(config.BotToken) {
		return true
	}
	if strings.TrimSpace(state.config.WebhookURL) != strings.TrimSpace(config.WebhookURL) {
		return true
	}
	if strings.TrimSpace(state.config.WebhookSecret) != strings.TrimSpace(config.WebhookSecret) {
		return true
	}
	if strings.TrimSpace(state.config.WebhookHost) != strings.TrimSpace(config.WebhookHost) {
		return true
	}
	if strings.TrimSpace(state.config.WebhookPath) != strings.TrimSpace(config.WebhookPath) {
		return true
	}
	if state.config.Network != config.Network {
		return true
	}
	if state.config.Polling != config.Polling {
		return true
	}
	return false
}

func (service *BotService) setAccountError(state *telegramAccountState, err string) {
	if state == nil {
		return
	}
	message := redactTelegramToken(err, state.config.BotToken)
	state.lastError = message
	state.lastErrorAt = service.now()
	zap.L().Warn("telegram account error", zap.String("accountId", state.config.AccountID), zap.String("error", message))
}

func (service *BotService) clearAccountError(state *telegramAccountState) {
	if state == nil {
		return
	}
	state.lastError = ""
	state.lastErrorAt = time.Time{}
}

func (service *BotService) clearAllAccountErrors() {
	service.mu.Lock()
	defer service.mu.Unlock()
	for _, state := range service.accounts {
		state.lastError = ""
		state.lastErrorAt = time.Time{}
	}
}

func (service *BotService) clearUpdateOffset(accountID string) {
	if service == nil || service.offsets == nil {
		return
	}
	if err := service.offsets.Delete(accountID); err != nil {
		zap.L().Warn("telegram update offset clear failed", zap.String("accountId", accountID), zap.Error(err))
	}
}

func (service *BotService) clearAllUpdateOffsets() {
	if service == nil || service.offsets == nil {
		return
	}
	if err := service.offsets.ClearAll(); err != nil {
		zap.L().Warn("telegram update offset clear failed", zap.Error(err))
	}
}

func accountConfigured(state *telegramAccountState) bool {
	if state == nil {
		return false
	}
	if !state.config.Enabled {
		return false
	}
	return strings.TrimSpace(state.config.BotToken) != ""
}

func redactTelegramToken(message string, token string) string {
	if message == "" {
		return message
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return message
	}
	masked := maskTelegramToken(trimmed)
	redacted := strings.ReplaceAll(message, "bot"+trimmed, "bot"+masked)
	if trimmed != masked {
		redacted = strings.ReplaceAll(redacted, trimmed, masked)
	}
	return redacted
}

func maskTelegramToken(token string) string {
	const keepStart = 6
	const keepEnd = 4
	if len(token) < keepStart+keepEnd+1 {
		return "***"
	}
	return token[:keepStart] + "..." + token[len(token)-keepEnd:]
}

func isContextCanceledError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "context canceled") ||
		strings.Contains(message, "context deadline exceeded")
}

func (service *BotService) setAccountIdentity(state *telegramAccountState, user telegramapi.User) {
	if state == nil {
		return
	}
	state.botID = user.ID
	state.botUsername = strings.TrimSpace(user.Username)
}

type replyTracker struct {
	mu       sync.Mutex
	sessions map[string]struct{}
}

func newReplyTracker() *replyTracker {
	return &replyTracker{sessions: make(map[string]struct{})}
}

func (tracker *replyTracker) ShouldReplyFirst(sessionKey string) bool {
	if tracker == nil {
		return true
	}
	key := strings.TrimSpace(sessionKey)
	if key == "" {
		return true
	}
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	if _, ok := tracker.sessions[key]; ok {
		return false
	}
	tracker.sessions[key] = struct{}{}
	return true
}

func buildMessageContent(message *telegramapi.Message, primaryText string) string {
	content := strings.TrimSpace(primaryText)
	if message == nil {
		return content
	}
	if len(message.Photo) > 0 {
		content = appendPlaceholder(content, "[image]")
	}
	if message.Voice != nil {
		content = appendPlaceholder(content, "[voice]")
	}
	if message.Audio != nil {
		content = appendPlaceholder(content, "[audio]")
	}
	if message.Document != nil {
		content = appendPlaceholder(content, "[file]")
	}
	if message.Video != nil {
		content = appendPlaceholder(content, "[video]")
	}
	if message.Sticker != nil {
		content = appendPlaceholder(content, "[sticker]")
	}
	return content
}

func appendPlaceholder(content string, placeholder string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return placeholder
	}
	return trimmed + "\n" + placeholder
}

type telegramSourceItem struct {
	URL   string
	Title string
}

func collectTelegramReasoning(parts []chatevent.MessagePart) string {
	if len(parts) == 0 {
		return ""
	}
	values := make([]string, 0, 2)
	for _, part := range parts {
		if !strings.EqualFold(strings.TrimSpace(part.Type), "reasoning") {
			continue
		}
		text := strings.TrimSpace(part.Text)
		if text == "" {
			continue
		}
		values = append(values, text)
	}
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(strings.Join(values, "\n"))
}

func formatTelegramUsageSummary(usage runtimedto.RuntimeUsage, mode string) string {
	switch mode {
	case "tokens":
		return fmt.Sprintf(
			"Usage:\n- prompt: %d\n- completion: %d\n- total: %d",
			usage.PromptTokens,
			usage.CompletionTokens,
			usage.TotalTokens,
		)
	case "full":
		lines := []string{
			fmt.Sprintf("Usage prompt: %d", usage.PromptTokens),
			fmt.Sprintf("Usage completion: %d", usage.CompletionTokens),
			fmt.Sprintf("Usage total: %d", usage.TotalTokens),
		}
		if usage.ContextPromptTokens > 0 {
			lines = append(lines, fmt.Sprintf("Context prompt tokens: %d", usage.ContextPromptTokens))
		}
		if usage.ContextTotalTokens > 0 {
			lines = append(lines, fmt.Sprintf("Context total tokens: %d", usage.ContextTotalTokens))
		}
		if usage.ContextWindowTokens > 0 {
			lines = append(lines, fmt.Sprintf("Context window tokens: %d", usage.ContextWindowTokens))
		}
		return strings.Join(lines, "\n")
	case "cost":
		if usage.TotalTokens > 0 {
			return fmt.Sprintf("Usage total tokens: %d\nCost details are unavailable in Telegram response payload.", usage.TotalTokens)
		}
		return "Cost details are unavailable in Telegram response payload."
	default:
		return ""
	}
}

func decorateTelegramReply(
	reply string,
	result runtimedto.RuntimeRunResult,
	commandState telegramSessionCommandState,
) string {
	trimmed := strings.TrimSpace(reply)
	sections := make([]string, 0, 3)
	if trimmed != "" {
		sections = append(sections, trimmed)
	}
	if reasoningMode := normalizeTelegramReasoningMode(commandState.ReasoningMode); reasoningMode != "" && reasoningMode != "off" {
		reasoning := collectTelegramReasoning(result.AssistantMessage.Parts)
		if reasoning != "" {
			sections = append(sections, "Reasoning:\n"+truncateTelegramText(reasoning, 1200))
		}
	}
	if usageMode := normalizeTelegramUsageMode(commandState.UsageMode); usageMode != "" && usageMode != "off" {
		if usageSummary := strings.TrimSpace(formatTelegramUsageSummary(result.Usage, usageMode)); usageSummary != "" {
			sections = append(sections, usageSummary)
		}
	}
	if normalizeTelegramVerboseMode(commandState.VerboseMode) == "on" {
		lines := []string{
			fmt.Sprintf("Runtime status: %s", strings.TrimSpace(result.Status)),
		}
		if strings.TrimSpace(result.FinishReason) != "" {
			lines = append(lines, fmt.Sprintf("Finish reason: %s", strings.TrimSpace(result.FinishReason)))
		}
		if result.Model != nil {
			providerID := strings.TrimSpace(result.Model.ProviderID)
			modelName := strings.TrimSpace(result.Model.Name)
			if providerID != "" || modelName != "" {
				lines = append(lines, fmt.Sprintf("Model: %s/%s", providerID, modelName))
			}
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}
	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func buildTelegramHTTPClient(base *http.Client, cfg TelegramNetworkConfig) *http.Client {
	client := base
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	proxy := strings.TrimSpace(cfg.Proxy)
	if proxy == "" && cfg.TimeoutSeconds <= 0 {
		return client
	}
	cloned := *client
	if cfg.TimeoutSeconds > 0 {
		cloned.Timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}
	transport := cloneHTTPTransport(client.Transport)
	if proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		} else {
			zap.L().Warn("telegram: invalid proxy url", zap.String("proxy", proxy), zap.Error(err))
		}
	}
	cloned.Transport = transport
	return &cloned
}

func cloneHTTPTransport(rt http.RoundTripper) *http.Transport {
	if rt == nil {
		return http.DefaultTransport.(*http.Transport).Clone()
	}
	if transport, ok := rt.(*http.Transport); ok {
		return transport.Clone()
	}
	return http.DefaultTransport.(*http.Transport).Clone()
}

func resolveWebhookURL(cfg TelegramAccountConfig) string {
	trimmed := strings.TrimSpace(cfg.WebhookURL)
	if trimmed != "" {
		return trimmed
	}
	host := strings.TrimSpace(cfg.WebhookHost)
	path := strings.TrimSpace(cfg.WebhookPath)
	if host == "" || path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(host, "/") + path
}

func defaultAllowedUpdates() []string {
	return []string{
		telego.MessageUpdates,
		telego.EditedMessageUpdates,
		telego.ChannelPostUpdates,
		telego.EditedChannelPostUpdates,
		telego.CallbackQueryUpdates,
	}
}

func sleepWithContext(ctx context.Context, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
