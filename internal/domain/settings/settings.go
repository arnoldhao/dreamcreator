package settings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AppearanceMode string

type LogLevel string

type WindowBounds struct {
	x      int
	y      int
	width  int
	height int
}

type Settings struct {
	appearance            AppearanceMode
	fontFamily            string
	themeColor            string
	colorScheme           ColorScheme
	fontSize              int
	language              Language
	downloadDirectory     string
	mainBounds            WindowBounds
	settingsBounds        WindowBounds
	version               int
	logLevel              LogLevel
	logMaxSizeMB          int
	logMaxBackups         int
	logMaxAgeDays         int
	logCompress           bool
	proxy                 ProxySettings
	menuBarVisibility     MenuBarVisibility
	autoStart             bool
	minimizeToTrayOnStart bool
	agentModelProviderID  string
	agentModelName        string
	agentStreamEnabled    bool
	chatTemperature       float32
	chatMaxTokens         int
	skills                []SkillSpec
	gateway               GatewaySettings
	memory                MemorySettings
	toolsConfig           map[string]any
	skillsConfig          map[string]any
	commands              CommandsSettings
	channels              ChannelsSettings
}

type SettingsParams struct {
	Appearance            string
	FontFamily            string
	ThemeColor            string
	ColorScheme           string
	FontSize              int
	Language              string
	DownloadDirectory     string
	MainBounds            WindowBounds
	SettingsBounds        WindowBounds
	Version               int
	LogLevel              string
	LogMaxSizeMB          int
	LogMaxBackups         int
	LogMaxAgeDays         int
	LogCompress           *bool
	Proxy                 ProxySettingsParams
	MenuBarVisibility     *string
	AutoStart             *bool
	MinimizeToTrayOnStart *bool
	AgentModelProviderID  string
	AgentModelName        string
	AgentStreamEnabled    *bool
	ChatTemperature       *float32
	ChatMaxTokens         *int
	Skills                []SkillSpec
	Gateway               GatewaySettingsParams
	Memory                MemorySettingsParams
	ToolsConfig           map[string]any
	SkillsConfig          map[string]any
	Commands              CommandsSettingsParams
	Channels              ChannelsSettingsParams
}

const (
	AppearanceLight AppearanceMode = "light"
	AppearanceDark  AppearanceMode = "dark"
	AppearanceAuto  AppearanceMode = "auto"
)

type Language string

const (
	LanguageEnglish           Language = "en"
	LanguageChineseSimplified Language = "zh-CN"
	DefaultLanguage                    = LanguageEnglish
)

const (
	DefaultMainWidth        = 1280
	DefaultMainHeight       = 800
	DefaultSettingsWidth    = 960
	DefaultSettingsHeight   = 640
	MinMainWindowWidth      = 1280
	MinMainWindowHeight     = 800
	MinSettingsWindowWidth  = 960
	MinSettingsWindowHeight = 640
	DefaultFontSize         = 15
	MinFontSize             = 12
	MaxFontSize             = 24
)

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"

	DefaultLogLevel = LogLevelInfo
)

const (
	DefaultLogMaxSizeMB  = 50
	DefaultLogMaxBackups = 5
	DefaultLogMaxAgeDays = 7
	DefaultLogCompress   = true
)

const (
	ThemeColorSystem = "system"
)

type ColorScheme string

const (
	ColorSchemeDefault  ColorScheme = "default"
	ColorSchemeContrast ColorScheme = "contrast"
	ColorSchemeSlate    ColorScheme = "slate"
	ColorSchemeWarm     ColorScheme = "warm"

	DefaultColorScheme = ColorSchemeDefault
)

func (scheme ColorScheme) String() string {
	return string(scheme)
}

const (
	MenuBarVisibilityAlways      MenuBarVisibility = "always"
	MenuBarVisibilityWhenRunning MenuBarVisibility = "whenRunning"
	MenuBarVisibilityNever       MenuBarVisibility = "never"

	DefaultMenuBarVisibility = MenuBarVisibilityWhenRunning
)

const (
	DefaultProxyTimeoutSeconds = 30
)

const (
	DefaultAgentStreamEnabled = true
	DefaultChatStreamEnabled  = DefaultAgentStreamEnabled
	DefaultChatTemperature    = float32(0.7)
	MinChatTemperature        = float32(0.0)
	MaxChatTemperature        = float32(2.0)
	DefaultChatMaxTokens      = 2048
	MinChatMaxTokens          = 1
	MaxChatMaxTokens          = 8192
)

func DefaultDownloadDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	trimmed := strings.TrimSpace(home)
	if trimmed == "" {
		return ""
	}
	return filepath.Join(trimmed, "Downloads")
}

type ProxyMode string

const (
	ProxyModeNone   ProxyMode = "none"
	ProxyModeSystem ProxyMode = "system"
	ProxyModeManual ProxyMode = "manual"
)

type ProxyScheme string

const (
	ProxySchemeHTTP   ProxyScheme = "http"
	ProxySchemeHTTPS  ProxyScheme = "https"
	ProxySchemeSocks5 ProxyScheme = "socks5"
)

type ProxySettings struct {
	mode     ProxyMode
	scheme   ProxyScheme
	host     string
	port     int
	username string
	password string
	noProxy  []string
	timeout  time.Duration

	lastTestedAt time.Time
	testSuccess  bool
	testMessage  string
}

type SkillSpec struct {
	ID          string
	ProviderID  string
	Name        string
	Description string
	Version     string
	Enabled     bool
}

type ProxySettingsParams struct {
	Mode           string
	Scheme         string
	Host           string
	Port           int
	Username       string
	Password       string
	NoProxy        []string
	TimeoutSeconds int

	LastTestedAt *time.Time
	TestSuccess  *bool
	TestMessage  string
}

type MenuBarVisibility string

func (visibility MenuBarVisibility) String() string {
	return string(visibility)
}

func NewWindowBounds(x, y, width, height int) (WindowBounds, error) {
	return NewWindowBoundsWithMin(x, y, width, height, MinMainWindowWidth, MinMainWindowHeight)
}

func NewMainWindowBounds(x, y, width, height int) (WindowBounds, error) {
	return NewWindowBoundsWithMin(x, y, width, height, MinMainWindowWidth, MinMainWindowHeight)
}

func NewSettingsWindowBounds(x, y, width, height int) (WindowBounds, error) {
	return NewWindowBoundsWithMin(x, y, width, height, MinSettingsWindowWidth, MinSettingsWindowHeight)
}

func NewWindowBoundsWithMin(x, y, width, height, minWidth, minHeight int) (WindowBounds, error) {
	if width < minWidth || height < minHeight {
		return WindowBounds{}, fmt.Errorf("%w: window size", ErrInvalidSettings)
	}

	return WindowBounds{
		x:      x,
		y:      y,
		width:  width,
		height: height,
	}, nil
}

func NewSettings(params SettingsParams) (Settings, error) {
	appearance, err := ParseAppearanceMode(params.Appearance)
	if err != nil {
		return Settings{}, err
	}

	fontFamily := strings.TrimSpace(params.FontFamily)
	themeColor := strings.TrimSpace(params.ThemeColor)
	colorScheme, err := ParseColorScheme(params.ColorScheme)
	if err != nil {
		return Settings{}, err
	}
	fontSize := params.FontSize
	if fontSize <= 0 {
		fontSize = DefaultFontSize
	}
	if fontSize < MinFontSize || fontSize > MaxFontSize {
		return Settings{}, fmt.Errorf("%w: font size", ErrInvalidSettings)
	}

	parsedLanguage, err := ParseLanguage(params.Language)
	if err != nil {
		return Settings{}, err
	}

	downloadDirectory := strings.TrimSpace(params.DownloadDirectory)
	if downloadDirectory == "" {
		downloadDirectory = DefaultDownloadDirectory()
	}

	logLevel, err := ParseLogLevel(params.LogLevel)
	if err != nil {
		return Settings{}, err
	}

	if params.Version <= 0 {
		params.Version = 1
	}

	logMaxSizeMB := params.LogMaxSizeMB
	if logMaxSizeMB <= 0 {
		logMaxSizeMB = DefaultLogMaxSizeMB
	}

	logMaxBackups := params.LogMaxBackups
	if logMaxBackups <= 0 {
		logMaxBackups = DefaultLogMaxBackups
	}

	logMaxAgeDays := params.LogMaxAgeDays
	if logMaxAgeDays <= 0 {
		logMaxAgeDays = DefaultLogMaxAgeDays
	}

	logCompress := DefaultLogCompress
	if params.LogCompress != nil {
		logCompress = *params.LogCompress
	}

	proxySettings, err := NewProxySettings(params.Proxy)
	if err != nil {
		return Settings{}, err
	}

	menuBarVisibility := DefaultMenuBarVisibility
	if params.MenuBarVisibility != nil {
		menuBarVisibility, err = ParseMenuBarVisibility(*params.MenuBarVisibility)
		if err != nil {
			return Settings{}, err
		}
	}

	autoStart := false
	if params.AutoStart != nil {
		autoStart = *params.AutoStart
	}

	minimizeToTrayOnStart := false
	if params.MinimizeToTrayOnStart != nil {
		minimizeToTrayOnStart = *params.MinimizeToTrayOnStart
	}

	agentModelProviderID := strings.TrimSpace(params.AgentModelProviderID)
	agentModelName := strings.TrimSpace(params.AgentModelName)
	agentStreamEnabled := DefaultAgentStreamEnabled
	if params.AgentStreamEnabled != nil {
		agentStreamEnabled = *params.AgentStreamEnabled
	}

	chatTemperature := DefaultChatTemperature
	if params.ChatTemperature != nil {
		chatTemperature = *params.ChatTemperature
	}
	if chatTemperature < MinChatTemperature || chatTemperature > MaxChatTemperature {
		return Settings{}, fmt.Errorf("%w: chat temperature", ErrInvalidSettings)
	}

	chatMaxTokens := DefaultChatMaxTokens
	if params.ChatMaxTokens != nil {
		chatMaxTokens = *params.ChatMaxTokens
	}
	if chatMaxTokens < MinChatMaxTokens || chatMaxTokens > MaxChatMaxTokens {
		return Settings{}, fmt.Errorf("%w: chat max tokens", ErrInvalidSettings)
	}

	gatewaySettings := ResolveGatewaySettings(params.Gateway)
	memorySettings := ResolveMemorySettings(params.Memory)
	toolsConfig := normalizeToolsConfig(params.ToolsConfig)
	skillsConfig := normalizeSettingsAnyMap(params.SkillsConfig)
	commandsSettings := NewCommandsSettings(params.Commands)
	channelsSettings := NewChannelsSettings(params.Channels)

	return Settings{
		appearance:            appearance,
		fontFamily:            fontFamily,
		themeColor:            themeColor,
		colorScheme:           colorScheme,
		fontSize:              fontSize,
		language:              parsedLanguage,
		downloadDirectory:     downloadDirectory,
		mainBounds:            params.MainBounds,
		settingsBounds:        params.SettingsBounds,
		version:               params.Version,
		logLevel:              logLevel,
		logMaxSizeMB:          logMaxSizeMB,
		logMaxBackups:         logMaxBackups,
		logMaxAgeDays:         logMaxAgeDays,
		logCompress:           logCompress,
		proxy:                 proxySettings,
		menuBarVisibility:     menuBarVisibility,
		autoStart:             autoStart,
		minimizeToTrayOnStart: minimizeToTrayOnStart,
		agentModelProviderID:  agentModelProviderID,
		agentModelName:        agentModelName,
		agentStreamEnabled:    agentStreamEnabled,
		chatTemperature:       chatTemperature,
		chatMaxTokens:         chatMaxTokens,
		skills:                normalizeSkillSpecs(params.Skills),
		gateway:               gatewaySettings,
		memory:                memorySettings,
		toolsConfig:           toolsConfig,
		skillsConfig:          skillsConfig,
		commands:              commandsSettings,
		channels:              channelsSettings,
	}, nil
}

func DefaultSettingsWithLanguage(language string) Settings {
	mainBounds, _ := NewMainWindowBounds(0, 0, DefaultMainWidth, DefaultMainHeight)
	settingsBounds, _ := NewSettingsWindowBounds(0, 0, DefaultSettingsWidth, DefaultSettingsHeight)
	parsedLanguage, _ := ParseLanguage(language)
	return Settings{
		appearance:            AppearanceAuto,
		fontFamily:            "",
		themeColor:            ThemeColorSystem,
		colorScheme:           DefaultColorScheme,
		fontSize:              DefaultFontSize,
		language:              parsedLanguage,
		downloadDirectory:     DefaultDownloadDirectory(),
		mainBounds:            mainBounds,
		settingsBounds:        settingsBounds,
		version:               1,
		logLevel:              DefaultLogLevel,
		logMaxSizeMB:          DefaultLogMaxSizeMB,
		logMaxBackups:         DefaultLogMaxBackups,
		logMaxAgeDays:         DefaultLogMaxAgeDays,
		logCompress:           DefaultLogCompress,
		proxy:                 DefaultProxySettings(),
		menuBarVisibility:     DefaultMenuBarVisibility,
		autoStart:             false,
		minimizeToTrayOnStart: false,
		agentModelProviderID:  "",
		agentModelName:        "",
		agentStreamEnabled:    DefaultAgentStreamEnabled,
		chatTemperature:       DefaultChatTemperature,
		chatMaxTokens:         DefaultChatMaxTokens,
		skills:                nil,
		gateway:               DefaultGatewaySettings(),
		memory:                DefaultMemorySettings(),
		toolsConfig:           normalizeToolsConfig(nil),
		skillsConfig:          nil,
		commands:              NewCommandsSettings(CommandsSettingsParams{}),
		channels:              NewChannelsSettings(ChannelsSettingsParams{Config: DefaultChannelsConfig()}),
	}
}

func DefaultSettings() Settings {
	return DefaultSettingsWithLanguage(DefaultLanguage.String())
}

func ParseAppearanceMode(value string) (AppearanceMode, error) {
	switch AppearanceMode(value) {
	case AppearanceLight, AppearanceDark, AppearanceAuto:
		return AppearanceMode(value), nil
	default:
		return "", fmt.Errorf("%w: appearance", ErrInvalidSettings)
	}
}

func ParseLanguage(value string) (Language, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return DefaultLanguage, nil
	}

	switch Language(trimmed) {
	case LanguageEnglish, LanguageChineseSimplified:
		return Language(trimmed), nil
	default:
		return "", fmt.Errorf("%w: language", ErrInvalidSettings)
	}
}

func ParseColorScheme(value string) (ColorScheme, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return DefaultColorScheme, nil
	}

	switch ColorScheme(trimmed) {
	case ColorSchemeDefault, ColorSchemeContrast, ColorSchemeSlate, ColorSchemeWarm:
		return ColorScheme(trimmed), nil
	default:
		return "", fmt.Errorf("%w: color scheme", ErrInvalidSettings)
	}
}

func ParseLogLevel(value string) (LogLevel, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return DefaultLogLevel, nil
	}

	switch LogLevel(trimmed) {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
		return LogLevel(trimmed), nil
	default:
		return "", fmt.Errorf("%w: log level", ErrInvalidSettings)
	}
}

func ParseMenuBarVisibility(value string) (MenuBarVisibility, error) {
	switch MenuBarVisibility(strings.TrimSpace(value)) {
	case MenuBarVisibilityAlways, MenuBarVisibilityWhenRunning, MenuBarVisibilityNever:
		return MenuBarVisibility(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("%w: menu bar visibility", ErrInvalidSettings)
	}
}

func ParseProxyMode(value string) (ProxyMode, error) {
	switch ProxyMode(strings.TrimSpace(value)) {
	case ProxyModeNone, ProxyModeSystem, ProxyModeManual:
		return ProxyMode(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("%w: proxy mode", ErrInvalidSettings)
	}
}

func ParseProxyScheme(value string) (ProxyScheme, error) {
	switch ProxyScheme(strings.TrimSpace(value)) {
	case ProxySchemeHTTP, ProxySchemeHTTPS, ProxySchemeSocks5:
		return ProxyScheme(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("%w: proxy scheme", ErrInvalidSettings)
	}
}

func DefaultProxySettings() ProxySettings {
	return ProxySettings{
		mode:    ProxyModeNone,
		timeout: DefaultProxyTimeoutSeconds * time.Second,
		noProxy: []string{"localhost", "127.0.0.1"},
	}
}

func NewProxySettings(params ProxySettingsParams) (ProxySettings, error) {
	mode := ProxyModeNone
	if params.Mode != "" {
		parsedMode, err := ParseProxyMode(params.Mode)
		if err != nil {
			return ProxySettings{}, err
		}
		mode = parsedMode
	}

	scheme := ProxySchemeHTTP
	if params.Scheme != "" {
		parsedScheme, err := ParseProxyScheme(params.Scheme)
		if err != nil {
			return ProxySettings{}, err
		}
		scheme = parsedScheme
	}

	timeoutSeconds := params.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = DefaultProxyTimeoutSeconds
	}

	if mode == ProxyModeManual {
		if strings.TrimSpace(params.Host) == "" {
			return ProxySettings{}, fmt.Errorf("%w: proxy host", ErrInvalidSettings)
		}
		if params.Port <= 0 || params.Port > 65535 {
			return ProxySettings{}, fmt.Errorf("%w: proxy port", ErrInvalidSettings)
		}
	}

	noProxy := make([]string, 0, len(params.NoProxy))
	for _, entry := range params.NoProxy {
		trimmed := strings.TrimSpace(entry)
		if trimmed != "" {
			noProxy = append(noProxy, trimmed)
		}
	}
	if len(noProxy) == 0 {
		noProxy = []string{"localhost", "127.0.0.1"}
	}

	testSuccess := false
	if params.TestSuccess != nil {
		testSuccess = *params.TestSuccess
	}

	var testedAt time.Time
	if params.LastTestedAt != nil {
		testedAt = *params.LastTestedAt
	}

	return ProxySettings{
		mode:         mode,
		scheme:       scheme,
		host:         strings.TrimSpace(params.Host),
		port:         params.Port,
		username:     strings.TrimSpace(params.Username),
		password:     params.Password,
		noProxy:      noProxy,
		timeout:      time.Duration(timeoutSeconds) * time.Second,
		lastTestedAt: testedAt,
		testSuccess:  testSuccess,
		testMessage:  strings.TrimSpace(params.TestMessage),
	}, nil
}

func (proxy ProxySettings) Mode() ProxyMode {
	return proxy.mode
}

func (proxy ProxySettings) Scheme() ProxyScheme {
	return proxy.scheme
}

func (proxy ProxySettings) Host() string {
	return proxy.host
}

func (proxy ProxySettings) Port() int {
	return proxy.port
}

func (proxy ProxySettings) Username() string {
	return proxy.username
}

func (proxy ProxySettings) Password() string {
	return proxy.password
}

func (proxy ProxySettings) NoProxy() []string {
	copied := make([]string, len(proxy.noProxy))
	copy(copied, proxy.noProxy)
	return copied
}

func (proxy ProxySettings) Timeout() time.Duration {
	return proxy.timeout
}

func (proxy ProxySettings) LastTestedAt() time.Time {
	return proxy.lastTestedAt
}

func (proxy ProxySettings) TestSuccess() bool {
	return proxy.testSuccess
}

func (proxy ProxySettings) TestMessage() string {
	return proxy.testMessage
}

func (proxy ProxySettings) WithTestResult(success bool, message string, testedAt time.Time) ProxySettings {
	proxy.testSuccess = success
	proxy.testMessage = strings.TrimSpace(message)
	proxy.lastTestedAt = testedAt
	return proxy
}

func (settings Settings) Appearance() AppearanceMode {
	return settings.appearance
}

func (settings Settings) FontFamily() string {
	return settings.fontFamily
}

func (settings Settings) FontSize() int {
	return settings.fontSize
}

func (settings Settings) ThemeColor() string {
	return settings.themeColor
}

func (settings Settings) ColorScheme() ColorScheme {
	return settings.colorScheme
}

func (settings Settings) Language() Language {
	return settings.language
}

func (settings Settings) DownloadDirectory() string {
	return settings.downloadDirectory
}

func (settings Settings) MainBounds() WindowBounds {
	return settings.mainBounds
}

func (settings Settings) SettingsBounds() WindowBounds {
	return settings.settingsBounds
}

func (settings Settings) Version() int {
	return settings.version
}

func (settings Settings) LogLevel() LogLevel {
	return settings.logLevel
}

func (settings Settings) LogMaxSizeMB() int {
	return settings.logMaxSizeMB
}

func (settings Settings) LogMaxBackups() int {
	return settings.logMaxBackups
}

func (settings Settings) LogMaxAgeDays() int {
	return settings.logMaxAgeDays
}

func (settings Settings) LogCompress() bool {
	return settings.logCompress
}

func (settings Settings) Proxy() ProxySettings {
	return settings.proxy
}

func (settings Settings) MenuBarVisibility() MenuBarVisibility {
	return settings.menuBarVisibility
}

func IsSystemThemeColor(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), ThemeColorSystem)
}

func (settings Settings) AutoStart() bool {
	return settings.autoStart
}

func (settings Settings) MinimizeToTrayOnStart() bool {
	return settings.minimizeToTrayOnStart
}

func (settings Settings) AgentModelProviderID() string {
	return settings.agentModelProviderID
}

func (settings Settings) AgentModelName() string {
	return settings.agentModelName
}

func (settings Settings) AgentStreamEnabled() bool {
	return settings.agentStreamEnabled
}

func (settings Settings) ChatStreamEnabled() bool {
	return settings.agentStreamEnabled
}

func (settings Settings) ChatTemperature() float32 {
	return settings.chatTemperature
}

func (settings Settings) ChatMaxTokens() int {
	return settings.chatMaxTokens
}

func (settings Settings) Gateway() GatewaySettings {
	return settings.gateway
}

func (settings Settings) Memory() MemorySettings {
	return settings.memory
}

func (settings Settings) ToolsConfig() map[string]any {
	return cloneAnyMap(settings.toolsConfig)
}

func (settings Settings) SkillsConfig() map[string]any {
	return cloneAnyMap(settings.skillsConfig)
}

func (settings Settings) Skills() []SkillSpec {
	if len(settings.skills) == 0 {
		return nil
	}
	result := make([]SkillSpec, len(settings.skills))
	copy(result, settings.skills)
	return result
}

func (settings Settings) Commands() CommandsSettings {
	return settings.commands
}

func (settings Settings) Channels() ChannelsSettings {
	return settings.channels
}

func (settings Settings) WithSkills(skills []SkillSpec) Settings {
	settings.skills = normalizeSkillSpecs(skills)
	return settings
}

func (bounds WindowBounds) X() int {
	return bounds.x
}

func (bounds WindowBounds) Y() int {
	return bounds.y
}

func (bounds WindowBounds) Width() int {
	return bounds.width
}

func (bounds WindowBounds) Height() int {
	return bounds.height
}

func (mode AppearanceMode) String() string {
	return string(mode)
}

func (language Language) String() string {
	return string(language)
}

func (level LogLevel) String() string {
	return string(level)
}

func (mode ProxyMode) String() string {
	return string(mode)
}

func (scheme ProxyScheme) String() string {
	return string(scheme)
}

func normalizeSkillSpecs(skills []SkillSpec) []SkillSpec {
	if len(skills) == 0 {
		return nil
	}
	result := make([]SkillSpec, 0, len(skills))
	seen := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		id := strings.TrimSpace(skill.ID)
		name := strings.TrimSpace(skill.Name)
		if id == "" || name == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, SkillSpec{
			ID:          id,
			ProviderID:  strings.TrimSpace(skill.ProviderID),
			Name:        name,
			Description: strings.TrimSpace(skill.Description),
			Version:     strings.TrimSpace(skill.Version),
			Enabled:     skill.Enabled,
		})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeSettingsAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	return cloneAnyMap(source)
}

type Repository interface {
	Get(ctx context.Context) (Settings, error)
	Save(ctx context.Context, current Settings) error
}
