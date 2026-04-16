package service

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"time"

	"dreamcreator/internal/application/settings/dto"
	"dreamcreator/internal/domain/settings"
	"go.uber.org/zap"
)

type ThemeProvider interface {
	IsDarkMode(ctx context.Context) (bool, error)
	AccentColor(ctx context.Context) (string, error)
}

type SettingsService struct {
	repo          settings.Repository
	themeProvider ThemeProvider
	defaults      settings.Settings
}

func NewSettingsService(repo settings.Repository, themeProvider ThemeProvider, defaults settings.Settings) *SettingsService {
	return &SettingsService{
		repo:          repo,
		themeProvider: themeProvider,
		defaults:      defaults,
	}
}

func (service *SettingsService) GetSettings(ctx context.Context) (dto.Settings, error) {
	current, err := service.repo.Get(ctx)
	if err != nil {
		if err == settings.ErrSettingsNotFound {
			current = service.defaults
			if err := service.repo.Save(ctx, current); err != nil {
				return dto.Settings{}, err
			}
		} else {
			return dto.Settings{}, err
		}
	}

	effectiveAppearance, err := service.resolveAppearance(ctx, current.Appearance())
	if err != nil {
		return dto.Settings{}, err
	}

	systemThemeColor := service.resolveSystemThemeColor(ctx)
	return toDTO(current, effectiveAppearance, systemThemeColor), nil
}

func (service *SettingsService) UpdateSettings(ctx context.Context, request dto.UpdateSettingsRequest) (dto.Settings, error) {
	current, err := service.repo.Get(ctx)
	if err != nil {
		if err == settings.ErrSettingsNotFound {
			current = service.defaults
		} else {
			return dto.Settings{}, err
		}
	}

	appearance := current.Appearance().String()
	if request.Appearance != nil {
		appearance = *request.Appearance
	}

	fontFamily := current.FontFamily()
	if request.FontFamily != nil {
		fontFamily = *request.FontFamily
	}

	themeColor := current.ThemeColor()
	if request.ThemeColor != nil {
		themeColor = *request.ThemeColor
	}

	colorScheme := current.ColorScheme().String()
	if request.ColorScheme != nil {
		colorScheme = *request.ColorScheme
	}

	language := current.Language().String()
	if request.Language != nil {
		language = *request.Language
	}

	downloadDirectory := current.DownloadDirectory()
	if request.DownloadDirectory != nil {
		downloadDirectory = strings.TrimSpace(*request.DownloadDirectory)
	}

	fontSize := current.FontSize()
	if request.FontSize != nil {
		fontSize = *request.FontSize
	}

	logLevel := current.LogLevel().String()
	if request.LogLevel != nil {
		logLevel = *request.LogLevel
	}

	logMaxSizeMB := current.LogMaxSizeMB()
	if request.LogMaxSizeMB != nil {
		logMaxSizeMB = *request.LogMaxSizeMB
	}

	logMaxBackups := current.LogMaxBackups()
	if request.LogMaxBackups != nil {
		logMaxBackups = *request.LogMaxBackups
	}

	menuBarVisibility := sanitizeMenuBarVisibility(current.MenuBarVisibility().String())
	if request.MenuBarVisibility != nil {
		menuBarVisibility = *request.MenuBarVisibility
	}
	menuBarVisibility = sanitizeMenuBarVisibility(menuBarVisibility)

	logMaxAgeDays := current.LogMaxAgeDays()
	if request.LogMaxAgeDays != nil {
		logMaxAgeDays = *request.LogMaxAgeDays
	}

	logCompress := current.LogCompress()
	if request.LogCompress != nil {
		logCompress = *request.LogCompress
	}

	autoStart := current.AutoStart()
	if request.AutoStart != nil {
		autoStart = *request.AutoStart
	}

	minimizeToTrayOnStart := current.MinimizeToTrayOnStart()
	if request.MinimizeToTrayOnStart != nil {
		minimizeToTrayOnStart = *request.MinimizeToTrayOnStart
	}

	agentModelProviderID := current.AgentModelProviderID()
	if request.AgentModelProviderID != nil {
		agentModelProviderID = strings.TrimSpace(*request.AgentModelProviderID)
	}

	agentModelName := current.AgentModelName()
	if request.AgentModelName != nil {
		agentModelName = strings.TrimSpace(*request.AgentModelName)
	}

	agentStreamEnabled := current.AgentStreamEnabled()
	if request.AgentStreamEnabled != nil {
		agentStreamEnabled = *request.AgentStreamEnabled
	}

	chatTemperature := current.ChatTemperature()
	if request.ChatTemperature != nil {
		chatTemperature = *request.ChatTemperature
	}

	chatMaxTokens := current.ChatMaxTokens()
	if request.ChatMaxTokens != nil {
		chatMaxTokens = *request.ChatMaxTokens
	}

	currentTools := cloneSettingsAnyMap(current.ToolsConfig())
	currentSkills := cloneSettingsAnyMap(current.SkillsConfig())
	nextTools := cloneSettingsAnyMap(currentTools)
	nextSkills := cloneSettingsAnyMap(currentSkills)

	if request.Tools != nil {
		nextTools = cloneSettingsAnyMap(request.Tools)
	}
	if request.Skills != nil {
		nextSkills = cloneSettingsAnyMap(request.Skills)
	}
	memory := current.Memory()
	if request.Memory != nil {
		if request.Memory.Enabled != nil {
			memory.Enabled = *request.Memory.Enabled
		}
		if request.Memory.EmbeddingProvider != nil {
			memory.EmbeddingProvider = strings.TrimSpace(*request.Memory.EmbeddingProvider)
		}
		if request.Memory.EmbeddingModel != nil {
			memory.EmbeddingModel = strings.TrimSpace(*request.Memory.EmbeddingModel)
		}
		if request.Memory.LLMProvider != nil {
			memory.LLMProvider = strings.TrimSpace(*request.Memory.LLMProvider)
		}
		if request.Memory.LLMModel != nil {
			memory.LLMModel = strings.TrimSpace(*request.Memory.LLMModel)
		}
		if request.Memory.RecallTopK != nil {
			memory.RecallTopK = *request.Memory.RecallTopK
		}
		if request.Memory.VectorWeight != nil {
			memory.VectorWeight = *request.Memory.VectorWeight
		}
		if request.Memory.TextWeight != nil {
			memory.TextWeight = *request.Memory.TextWeight
		}
		if request.Memory.RecencyWeight != nil {
			memory.RecencyWeight = *request.Memory.RecencyWeight
		}
		if request.Memory.RecencyHalfLife != nil {
			memory.RecencyHalfLife = *request.Memory.RecencyHalfLife
		}
		if request.Memory.MinScore != nil {
			memory.MinScore = *request.Memory.MinScore
		}
		if request.Memory.AutoRecall != nil {
			memory.AutoRecall = *request.Memory.AutoRecall
		}
		if request.Memory.AutoCapture != nil {
			memory.AutoCapture = *request.Memory.AutoCapture
		}
		if request.Memory.SessionLifecycle != nil {
			memory.SessionLifecycle = *request.Memory.SessionLifecycle
		}
		if request.Memory.CaptureMaxEntries != nil {
			memory.CaptureMaxEntries = *request.Memory.CaptureMaxEntries
		}
	}
	currentCommands := current.Commands()
	commandsParams := settings.CommandsSettingsParams{Flags: currentCommands.Flags()}
	if request.Commands != nil {
		commandsParams.Flags = request.Commands
	}
	currentChannels := current.Channels()
	channelsParams := settings.ChannelsSettingsParams{Config: currentChannels.Config()}
	if request.Channels != nil {
		channelsParams.Config = request.Channels
	}

	currentProxy := current.Proxy()
	proxyParams := settings.ProxySettingsParams{
		Mode:           currentProxy.Mode().String(),
		Scheme:         currentProxy.Scheme().String(),
		Host:           currentProxy.Host(),
		Port:           currentProxy.Port(),
		Username:       currentProxy.Username(),
		Password:       currentProxy.Password(),
		NoProxy:        currentProxy.NoProxy(),
		TimeoutSeconds: int(currentProxy.Timeout().Seconds()),
		TestMessage:    currentProxy.TestMessage(),
	}
	if !currentProxy.LastTestedAt().IsZero() {
		testedAt := currentProxy.LastTestedAt()
		proxyParams.LastTestedAt = &testedAt
	}
	testSuccess := currentProxy.TestSuccess()
	proxyParams.TestSuccess = &testSuccess

	if request.Proxy != nil {
		proxyParams.Mode = request.Proxy.Mode
		proxyParams.Scheme = request.Proxy.Scheme
		proxyParams.Host = request.Proxy.Host
		proxyParams.Port = request.Proxy.Port
		proxyParams.Username = request.Proxy.Username
		proxyParams.Password = request.Proxy.Password
		proxyParams.NoProxy = request.Proxy.NoProxy
		if request.Proxy.TimeoutSeconds > 0 {
			proxyParams.TimeoutSeconds = request.Proxy.TimeoutSeconds
		}
		proxyParams.TestSuccess = &request.Proxy.TestSuccess
		proxyParams.TestMessage = request.Proxy.TestMessage
		proxyParams.LastTestedAt = parseProxyTestedAt(request.Proxy.TestedAt)
	}

	mainBounds := current.MainBounds()
	if request.MainBounds != nil {
		bounds, err := settings.NewMainWindowBounds(
			request.MainBounds.X,
			request.MainBounds.Y,
			request.MainBounds.Width,
			request.MainBounds.Height,
		)
		if err != nil {
			return dto.Settings{}, err
		}
		mainBounds = bounds
	}

	settingsBounds := current.SettingsBounds()
	if request.SettingsBounds != nil {
		bounds, err := settings.NewSettingsWindowBounds(
			request.SettingsBounds.X,
			request.SettingsBounds.Y,
			request.SettingsBounds.Width,
			request.SettingsBounds.Height,
		)
		if err != nil {
			return dto.Settings{}, err
		}
		settingsBounds = bounds
	}

	gateway := normalizeGatewaySettings(current.Gateway())
	if request.Gateway != nil {
		if request.Gateway.ControlPlaneEnabled != nil {
			gateway.ControlPlaneEnabled = *request.Gateway.ControlPlaneEnabled
		}
		if request.Gateway.VoiceEnabled != nil {
			gateway.VoiceEnabled = *request.Gateway.VoiceEnabled
		}
		if request.Gateway.SandboxEnabled != nil {
			gateway.SandboxEnabled = *request.Gateway.SandboxEnabled
		}
		if request.Gateway.VoiceWakeEnabled != nil {
			gateway.VoiceWakeEnabled = *request.Gateway.VoiceWakeEnabled
		}
		if request.Gateway.HTTP != nil && request.Gateway.HTTP.Endpoints != nil {
			if request.Gateway.HTTP.Endpoints.ChatCompletions != nil {
				if request.Gateway.HTTP.Endpoints.ChatCompletions.Enabled != nil {
					gateway.HTTP.Endpoints.ChatCompletions.Enabled = *request.Gateway.HTTP.Endpoints.ChatCompletions.Enabled
				}
			}
			if request.Gateway.HTTP.Endpoints.Responses != nil {
				if request.Gateway.HTTP.Endpoints.Responses.Enabled != nil {
					gateway.HTTP.Endpoints.Responses.Enabled = *request.Gateway.HTTP.Endpoints.Responses.Enabled
				}
				if request.Gateway.HTTP.Endpoints.Responses.MaxBodyBytes != nil {
					gateway.HTTP.Endpoints.Responses.MaxBodyBytes = *request.Gateway.HTTP.Endpoints.Responses.MaxBodyBytes
				}
				if request.Gateway.HTTP.Endpoints.Responses.MaxURLParts != nil {
					gateway.HTTP.Endpoints.Responses.MaxURLParts = *request.Gateway.HTTP.Endpoints.Responses.MaxURLParts
				}
				if request.Gateway.HTTP.Endpoints.Responses.Files != nil {
					if request.Gateway.HTTP.Endpoints.Responses.Files.AllowURL != nil {
						gateway.HTTP.Endpoints.Responses.Files.AllowURL = *request.Gateway.HTTP.Endpoints.Responses.Files.AllowURL
					}
					if request.Gateway.HTTP.Endpoints.Responses.Files.URLAllowlist != nil {
						gateway.HTTP.Endpoints.Responses.Files.URLAllowlist = normalizeStringList(*request.Gateway.HTTP.Endpoints.Responses.Files.URLAllowlist)
					}
					if request.Gateway.HTTP.Endpoints.Responses.Files.AllowedMimes != nil {
						gateway.HTTP.Endpoints.Responses.Files.AllowedMimes = normalizeStringList(*request.Gateway.HTTP.Endpoints.Responses.Files.AllowedMimes)
					}
					if request.Gateway.HTTP.Endpoints.Responses.Files.MaxBytes != nil {
						gateway.HTTP.Endpoints.Responses.Files.MaxBytes = *request.Gateway.HTTP.Endpoints.Responses.Files.MaxBytes
					}
					if request.Gateway.HTTP.Endpoints.Responses.Files.MaxChars != nil {
						gateway.HTTP.Endpoints.Responses.Files.MaxChars = *request.Gateway.HTTP.Endpoints.Responses.Files.MaxChars
					}
					if request.Gateway.HTTP.Endpoints.Responses.Files.MaxRedirects != nil {
						gateway.HTTP.Endpoints.Responses.Files.MaxRedirects = *request.Gateway.HTTP.Endpoints.Responses.Files.MaxRedirects
					}
					if request.Gateway.HTTP.Endpoints.Responses.Files.TimeoutMs != nil {
						gateway.HTTP.Endpoints.Responses.Files.TimeoutMs = *request.Gateway.HTTP.Endpoints.Responses.Files.TimeoutMs
					}
					if request.Gateway.HTTP.Endpoints.Responses.Files.PDF != nil {
						if request.Gateway.HTTP.Endpoints.Responses.Files.PDF.MaxPages != nil {
							gateway.HTTP.Endpoints.Responses.Files.PDF.MaxPages = *request.Gateway.HTTP.Endpoints.Responses.Files.PDF.MaxPages
						}
						if request.Gateway.HTTP.Endpoints.Responses.Files.PDF.MaxPixels != nil {
							gateway.HTTP.Endpoints.Responses.Files.PDF.MaxPixels = *request.Gateway.HTTP.Endpoints.Responses.Files.PDF.MaxPixels
						}
						if request.Gateway.HTTP.Endpoints.Responses.Files.PDF.MinTextChars != nil {
							gateway.HTTP.Endpoints.Responses.Files.PDF.MinTextChars = *request.Gateway.HTTP.Endpoints.Responses.Files.PDF.MinTextChars
						}
					}
				}
				if request.Gateway.HTTP.Endpoints.Responses.Images != nil {
					if request.Gateway.HTTP.Endpoints.Responses.Images.AllowURL != nil {
						gateway.HTTP.Endpoints.Responses.Images.AllowURL = *request.Gateway.HTTP.Endpoints.Responses.Images.AllowURL
					}
					if request.Gateway.HTTP.Endpoints.Responses.Images.URLAllowlist != nil {
						gateway.HTTP.Endpoints.Responses.Images.URLAllowlist = normalizeStringList(*request.Gateway.HTTP.Endpoints.Responses.Images.URLAllowlist)
					}
					if request.Gateway.HTTP.Endpoints.Responses.Images.AllowedMimes != nil {
						gateway.HTTP.Endpoints.Responses.Images.AllowedMimes = normalizeStringList(*request.Gateway.HTTP.Endpoints.Responses.Images.AllowedMimes)
					}
					if request.Gateway.HTTP.Endpoints.Responses.Images.MaxBytes != nil {
						gateway.HTTP.Endpoints.Responses.Images.MaxBytes = *request.Gateway.HTTP.Endpoints.Responses.Images.MaxBytes
					}
					if request.Gateway.HTTP.Endpoints.Responses.Images.MaxRedirects != nil {
						gateway.HTTP.Endpoints.Responses.Images.MaxRedirects = *request.Gateway.HTTP.Endpoints.Responses.Images.MaxRedirects
					}
					if request.Gateway.HTTP.Endpoints.Responses.Images.TimeoutMs != nil {
						gateway.HTTP.Endpoints.Responses.Images.TimeoutMs = *request.Gateway.HTTP.Endpoints.Responses.Images.TimeoutMs
					}
				}
			}
		}
		if request.Gateway.ChannelHealthCheckMinutes != nil {
			gateway.ChannelHealthCheckMinutes = *request.Gateway.ChannelHealthCheckMinutes
		}
		if request.Gateway.Runtime != nil {
			if request.Gateway.Runtime.MaxSteps != nil {
				gateway.Runtime.MaxSteps = *request.Gateway.Runtime.MaxSteps
			}
			if request.Gateway.Runtime.DebugMode != nil {
				gateway.Runtime.DebugMode = settings.ApplyGatewayDebugModeOverride(
					gateway.Runtime.DebugMode,
					request.Gateway.Runtime.DebugMode,
					nil,
				)
				gateway.Runtime.RecordPrompt = settings.GatewayDebugModeRecordsPrompt(gateway.Runtime.DebugMode)
			} else if request.Gateway.Runtime.RecordPrompt != nil {
				gateway.Runtime.DebugMode = settings.ApplyGatewayDebugModeOverride(
					gateway.Runtime.DebugMode,
					nil,
					request.Gateway.Runtime.RecordPrompt,
				)
				gateway.Runtime.RecordPrompt = settings.GatewayDebugModeRecordsPrompt(gateway.Runtime.DebugMode)
			}
			if request.Gateway.Runtime.ToolLoopDetection != nil {
				if request.Gateway.Runtime.ToolLoopDetection.Enabled != nil {
					gateway.Runtime.ToolLoopDetection.Enabled = *request.Gateway.Runtime.ToolLoopDetection.Enabled
				}
				if request.Gateway.Runtime.ToolLoopDetection.WarnThreshold != nil {
					gateway.Runtime.ToolLoopDetection.WarnThreshold = *request.Gateway.Runtime.ToolLoopDetection.WarnThreshold
				}
				if request.Gateway.Runtime.ToolLoopDetection.CriticalThreshold != nil {
					gateway.Runtime.ToolLoopDetection.CriticalThreshold = *request.Gateway.Runtime.ToolLoopDetection.CriticalThreshold
					gateway.Runtime.ToolLoopDetection.AbortThreshold = *request.Gateway.Runtime.ToolLoopDetection.CriticalThreshold
				}
				if request.Gateway.Runtime.ToolLoopDetection.GlobalCircuitBreakerThreshold != nil {
					gateway.Runtime.ToolLoopDetection.GlobalCircuitBreakerThreshold = *request.Gateway.Runtime.ToolLoopDetection.GlobalCircuitBreakerThreshold
				}
				if request.Gateway.Runtime.ToolLoopDetection.HistorySize != nil {
					gateway.Runtime.ToolLoopDetection.HistorySize = *request.Gateway.Runtime.ToolLoopDetection.HistorySize
					gateway.Runtime.ToolLoopDetection.WindowSize = *request.Gateway.Runtime.ToolLoopDetection.HistorySize
				}
				if request.Gateway.Runtime.ToolLoopDetection.Detectors != nil {
					if request.Gateway.Runtime.ToolLoopDetection.Detectors.GenericRepeat != nil {
						gateway.Runtime.ToolLoopDetection.Detectors.GenericRepeat = *request.Gateway.Runtime.ToolLoopDetection.Detectors.GenericRepeat
					}
					if request.Gateway.Runtime.ToolLoopDetection.Detectors.KnownPollNoProgress != nil {
						gateway.Runtime.ToolLoopDetection.Detectors.KnownPollNoProgress = *request.Gateway.Runtime.ToolLoopDetection.Detectors.KnownPollNoProgress
					}
					if request.Gateway.Runtime.ToolLoopDetection.Detectors.PingPong != nil {
						gateway.Runtime.ToolLoopDetection.Detectors.PingPong = *request.Gateway.Runtime.ToolLoopDetection.Detectors.PingPong
					}
				}
				if request.Gateway.Runtime.ToolLoopDetection.AbortThreshold != nil {
					gateway.Runtime.ToolLoopDetection.AbortThreshold = *request.Gateway.Runtime.ToolLoopDetection.AbortThreshold
					if gateway.Runtime.ToolLoopDetection.CriticalThreshold == 0 {
						gateway.Runtime.ToolLoopDetection.CriticalThreshold = gateway.Runtime.ToolLoopDetection.AbortThreshold
					}
				}
				if request.Gateway.Runtime.ToolLoopDetection.WindowSize != nil {
					gateway.Runtime.ToolLoopDetection.WindowSize = *request.Gateway.Runtime.ToolLoopDetection.WindowSize
					if gateway.Runtime.ToolLoopDetection.HistorySize == 0 {
						gateway.Runtime.ToolLoopDetection.HistorySize = gateway.Runtime.ToolLoopDetection.WindowSize
					}
				}
			}
			if request.Gateway.Runtime.ContextWindow != nil {
				if request.Gateway.Runtime.ContextWindow.WarnTokens != nil {
					gateway.Runtime.ContextWindow.WarnTokens = *request.Gateway.Runtime.ContextWindow.WarnTokens
				}
				if request.Gateway.Runtime.ContextWindow.HardTokens != nil {
					gateway.Runtime.ContextWindow.HardTokens = *request.Gateway.Runtime.ContextWindow.HardTokens
				}
			}
			if request.Gateway.Runtime.Compaction != nil {
				if request.Gateway.Runtime.Compaction.Mode != nil {
					gateway.Runtime.Compaction.Mode = strings.TrimSpace(*request.Gateway.Runtime.Compaction.Mode)
				}
				if request.Gateway.Runtime.Compaction.ReserveTokens != nil {
					gateway.Runtime.Compaction.ReserveTokens = *request.Gateway.Runtime.Compaction.ReserveTokens
				}
				if request.Gateway.Runtime.Compaction.KeepRecentTokens != nil {
					gateway.Runtime.Compaction.KeepRecentTokens = *request.Gateway.Runtime.Compaction.KeepRecentTokens
				}
				if request.Gateway.Runtime.Compaction.ReserveTokensFloor != nil {
					gateway.Runtime.Compaction.ReserveTokensFloor = *request.Gateway.Runtime.Compaction.ReserveTokensFloor
				}
				if request.Gateway.Runtime.Compaction.MaxHistoryShare != nil {
					gateway.Runtime.Compaction.MaxHistoryShare = *request.Gateway.Runtime.Compaction.MaxHistoryShare
				}
				if request.Gateway.Runtime.Compaction.MemoryFlush != nil {
					if request.Gateway.Runtime.Compaction.MemoryFlush.Enabled != nil {
						gateway.Runtime.Compaction.MemoryFlush.Enabled = *request.Gateway.Runtime.Compaction.MemoryFlush.Enabled
					}
					if request.Gateway.Runtime.Compaction.MemoryFlush.SoftThresholdTokens != nil {
						gateway.Runtime.Compaction.MemoryFlush.SoftThresholdTokens = *request.Gateway.Runtime.Compaction.MemoryFlush.SoftThresholdTokens
					}
					if request.Gateway.Runtime.Compaction.MemoryFlush.Prompt != nil {
						gateway.Runtime.Compaction.MemoryFlush.Prompt = strings.TrimSpace(*request.Gateway.Runtime.Compaction.MemoryFlush.Prompt)
					}
					if request.Gateway.Runtime.Compaction.MemoryFlush.SystemPrompt != nil {
						gateway.Runtime.Compaction.MemoryFlush.SystemPrompt = strings.TrimSpace(*request.Gateway.Runtime.Compaction.MemoryFlush.SystemPrompt)
					}
				}
			}
		}
		if request.Gateway.Queue != nil {
			if request.Gateway.Queue.GlobalConcurrency != nil {
				gateway.Queue.GlobalConcurrency = *request.Gateway.Queue.GlobalConcurrency
			}
			if request.Gateway.Queue.SessionConcurrency != nil {
				gateway.Queue.SessionConcurrency = *request.Gateway.Queue.SessionConcurrency
			}
			if request.Gateway.Queue.Lanes != nil {
				if request.Gateway.Queue.Lanes.Main != nil {
					gateway.Queue.Lanes.Main = *request.Gateway.Queue.Lanes.Main
				}
				if request.Gateway.Queue.Lanes.Subagent != nil {
					gateway.Queue.Lanes.Subagent = *request.Gateway.Queue.Lanes.Subagent
				}
				if request.Gateway.Queue.Lanes.Cron != nil {
					gateway.Queue.Lanes.Cron = *request.Gateway.Queue.Lanes.Cron
				}
			}
		}
		if request.Gateway.Heartbeat != nil {
			if request.Gateway.Heartbeat.Enabled != nil {
				gateway.Heartbeat.Enabled = *request.Gateway.Heartbeat.Enabled
			}
			if request.Gateway.Heartbeat.EveryMinutes != nil {
				gateway.Heartbeat.EveryMinutes = *request.Gateway.Heartbeat.EveryMinutes
			}
			if request.Gateway.Heartbeat.Every != nil {
				gateway.Heartbeat.Every = strings.TrimSpace(*request.Gateway.Heartbeat.Every)
			}
			if request.Gateway.Heartbeat.Target != nil {
				gateway.Heartbeat.Target = strings.TrimSpace(*request.Gateway.Heartbeat.Target)
			}
			if request.Gateway.Heartbeat.To != nil {
				gateway.Heartbeat.To = strings.TrimSpace(*request.Gateway.Heartbeat.To)
			}
			if request.Gateway.Heartbeat.AccountID != nil {
				gateway.Heartbeat.AccountID = strings.TrimSpace(*request.Gateway.Heartbeat.AccountID)
			}
			if request.Gateway.Heartbeat.Model != nil {
				gateway.Heartbeat.Model = strings.TrimSpace(*request.Gateway.Heartbeat.Model)
			}
			if request.Gateway.Heartbeat.Session != nil {
				gateway.Heartbeat.Session = strings.TrimSpace(*request.Gateway.Heartbeat.Session)
			}
			if request.Gateway.Heartbeat.RunSession != nil {
				gateway.Heartbeat.RunSession = strings.TrimSpace(*request.Gateway.Heartbeat.RunSession)
			}
			if request.Gateway.Heartbeat.Prompt != nil {
				gateway.Heartbeat.Prompt = strings.TrimSpace(*request.Gateway.Heartbeat.Prompt)
			}
			if request.Gateway.Heartbeat.PromptAppend != nil {
				gateway.Heartbeat.PromptAppend = strings.TrimSpace(*request.Gateway.Heartbeat.PromptAppend)
			}
			if request.Gateway.Heartbeat.IncludeReasoning != nil {
				gateway.Heartbeat.IncludeReasoning = *request.Gateway.Heartbeat.IncludeReasoning
			}
			if request.Gateway.Heartbeat.SuppressToolErrorWarnings != nil {
				gateway.Heartbeat.SuppressToolErrorWarnings = *request.Gateway.Heartbeat.SuppressToolErrorWarnings
			}
			if request.Gateway.Heartbeat.ActiveHours != nil {
				if request.Gateway.Heartbeat.ActiveHours.Start != nil {
					gateway.Heartbeat.ActiveHours.Start = strings.TrimSpace(*request.Gateway.Heartbeat.ActiveHours.Start)
				}
				if request.Gateway.Heartbeat.ActiveHours.End != nil {
					gateway.Heartbeat.ActiveHours.End = strings.TrimSpace(*request.Gateway.Heartbeat.ActiveHours.End)
				}
				if request.Gateway.Heartbeat.ActiveHours.Timezone != nil {
					gateway.Heartbeat.ActiveHours.Timezone = strings.TrimSpace(*request.Gateway.Heartbeat.ActiveHours.Timezone)
				}
			}
			if request.Gateway.Heartbeat.Periodic != nil {
				if request.Gateway.Heartbeat.Periodic.Enabled != nil {
					gateway.Heartbeat.Periodic.Enabled = *request.Gateway.Heartbeat.Periodic.Enabled
				}
				if request.Gateway.Heartbeat.Periodic.Every != nil {
					gateway.Heartbeat.Periodic.Every = strings.TrimSpace(*request.Gateway.Heartbeat.Periodic.Every)
				}
			}
			if request.Gateway.Heartbeat.Delivery != nil {
				if request.Gateway.Heartbeat.Delivery.Periodic != nil {
					if request.Gateway.Heartbeat.Delivery.Periodic.Center != nil {
						gateway.Heartbeat.Delivery.Periodic.Center = *request.Gateway.Heartbeat.Delivery.Periodic.Center
					}
					if request.Gateway.Heartbeat.Delivery.Periodic.PopupMinSeverity != nil {
						gateway.Heartbeat.Delivery.Periodic.PopupMinSeverity = strings.TrimSpace(*request.Gateway.Heartbeat.Delivery.Periodic.PopupMinSeverity)
					}
					if request.Gateway.Heartbeat.Delivery.Periodic.ToastMinSeverity != nil {
						gateway.Heartbeat.Delivery.Periodic.ToastMinSeverity = strings.TrimSpace(*request.Gateway.Heartbeat.Delivery.Periodic.ToastMinSeverity)
					}
					if request.Gateway.Heartbeat.Delivery.Periodic.OSMinSeverity != nil {
						gateway.Heartbeat.Delivery.Periodic.OSMinSeverity = strings.TrimSpace(*request.Gateway.Heartbeat.Delivery.Periodic.OSMinSeverity)
					}
				}
				if request.Gateway.Heartbeat.Delivery.EventDriven != nil {
					if request.Gateway.Heartbeat.Delivery.EventDriven.Center != nil {
						gateway.Heartbeat.Delivery.EventDriven.Center = *request.Gateway.Heartbeat.Delivery.EventDriven.Center
					}
					if request.Gateway.Heartbeat.Delivery.EventDriven.PopupMinSeverity != nil {
						gateway.Heartbeat.Delivery.EventDriven.PopupMinSeverity = strings.TrimSpace(*request.Gateway.Heartbeat.Delivery.EventDriven.PopupMinSeverity)
					}
					if request.Gateway.Heartbeat.Delivery.EventDriven.ToastMinSeverity != nil {
						gateway.Heartbeat.Delivery.EventDriven.ToastMinSeverity = strings.TrimSpace(*request.Gateway.Heartbeat.Delivery.EventDriven.ToastMinSeverity)
					}
					if request.Gateway.Heartbeat.Delivery.EventDriven.OSMinSeverity != nil {
						gateway.Heartbeat.Delivery.EventDriven.OSMinSeverity = strings.TrimSpace(*request.Gateway.Heartbeat.Delivery.EventDriven.OSMinSeverity)
					}
				}
				if request.Gateway.Heartbeat.Delivery.ThreadReplyMode != nil {
					gateway.Heartbeat.Delivery.ThreadReplyMode = strings.TrimSpace(*request.Gateway.Heartbeat.Delivery.ThreadReplyMode)
				}
			}
			if request.Gateway.Heartbeat.Events != nil {
				if request.Gateway.Heartbeat.Events.CronWakeMode != nil {
					gateway.Heartbeat.Events.CronWakeMode = strings.TrimSpace(*request.Gateway.Heartbeat.Events.CronWakeMode)
				}
				if request.Gateway.Heartbeat.Events.ExecWakeMode != nil {
					gateway.Heartbeat.Events.ExecWakeMode = strings.TrimSpace(*request.Gateway.Heartbeat.Events.ExecWakeMode)
				}
				if request.Gateway.Heartbeat.Events.SubagentWakeMode != nil {
					gateway.Heartbeat.Events.SubagentWakeMode = strings.TrimSpace(*request.Gateway.Heartbeat.Events.SubagentWakeMode)
				}
			}
			if request.Gateway.Heartbeat.Checklist != nil {
				items := make([]settings.GatewayHeartbeatChecklistItem, 0, len(request.Gateway.Heartbeat.Checklist.Items))
				for _, item := range request.Gateway.Heartbeat.Checklist.Items {
					id := strings.TrimSpace(item.ID)
					text := strings.TrimSpace(item.Text)
					if id == "" && text == "" {
						continue
					}
					items = append(items, settings.GatewayHeartbeatChecklistItem{
						ID:       id,
						Text:     text,
						Done:     item.Done,
						Priority: strings.TrimSpace(item.Priority),
					})
				}
				gateway.Heartbeat.Checklist = settings.GatewayHeartbeatChecklist{
					Title:     strings.TrimSpace(request.Gateway.Heartbeat.Checklist.Title),
					Items:     items,
					Notes:     strings.TrimSpace(request.Gateway.Heartbeat.Checklist.Notes),
					Version:   request.Gateway.Heartbeat.Checklist.Version,
					UpdatedAt: strings.TrimSpace(request.Gateway.Heartbeat.Checklist.UpdatedAt),
				}
				if gateway.Heartbeat.Checklist.Version < 0 {
					gateway.Heartbeat.Checklist.Version = 0
				}
			}
		}
		if request.Gateway.Subagents != nil {
			if request.Gateway.Subagents.MaxDepth != nil {
				gateway.Subagents.MaxDepth = *request.Gateway.Subagents.MaxDepth
			}
			if request.Gateway.Subagents.MaxChildren != nil {
				gateway.Subagents.MaxChildren = *request.Gateway.Subagents.MaxChildren
			}
			if request.Gateway.Subagents.MaxConcurrent != nil {
				gateway.Subagents.MaxConcurrent = *request.Gateway.Subagents.MaxConcurrent
			}
			if request.Gateway.Subagents.Model != nil {
				gateway.Subagents.Model = strings.TrimSpace(*request.Gateway.Subagents.Model)
			}
			if request.Gateway.Subagents.Thinking != nil {
				gateway.Subagents.Thinking = strings.TrimSpace(*request.Gateway.Subagents.Thinking)
			}
			if request.Gateway.Subagents.Tools != nil {
				if request.Gateway.Subagents.Tools.Allow != nil {
					gateway.Subagents.Tools.Allow = normalizeStringList(*request.Gateway.Subagents.Tools.Allow)
				}
				if request.Gateway.Subagents.Tools.AlsoAllow != nil {
					gateway.Subagents.Tools.AlsoAllow = normalizeStringList(*request.Gateway.Subagents.Tools.AlsoAllow)
				}
				if request.Gateway.Subagents.Tools.Deny != nil {
					gateway.Subagents.Tools.Deny = normalizeStringList(*request.Gateway.Subagents.Tools.Deny)
				}
			}
		}
		if request.Gateway.Cron != nil {
			if request.Gateway.Cron.Enabled != nil {
				gateway.Cron.Enabled = *request.Gateway.Cron.Enabled
			}
			if request.Gateway.Cron.MaxConcurrentRuns != nil {
				gateway.Cron.MaxConcurrentRuns = *request.Gateway.Cron.MaxConcurrentRuns
			}
			if request.Gateway.Cron.SessionRetention != nil {
				gateway.Cron.SessionRetention = strings.TrimSpace(*request.Gateway.Cron.SessionRetention)
			}
			if request.Gateway.Cron.RunLog != nil {
				if request.Gateway.Cron.RunLog.MaxBytes != nil {
					gateway.Cron.RunLog.MaxBytes = strings.TrimSpace(*request.Gateway.Cron.RunLog.MaxBytes)
				}
				if request.Gateway.Cron.RunLog.KeepLines != nil {
					gateway.Cron.RunLog.KeepLines = *request.Gateway.Cron.RunLog.KeepLines
				}
			}
		}
	}

	nextVersion := current.Version() + 1
	if nextVersion <= 0 {
		nextVersion = 1
	}

	gatewayParams, err := gatewaySettingsParamsFromSettings(gateway)
	if err != nil {
		return dto.Settings{}, err
	}

	updated, err := settings.NewSettings(settings.SettingsParams{
		Appearance:            appearance,
		FontFamily:            fontFamily,
		FontSize:              fontSize,
		ThemeColor:            themeColor,
		ColorScheme:           colorScheme,
		Language:              language,
		DownloadDirectory:     downloadDirectory,
		MainBounds:            mainBounds,
		SettingsBounds:        settingsBounds,
		Version:               nextVersion,
		LogLevel:              logLevel,
		LogMaxSizeMB:          logMaxSizeMB,
		LogMaxBackups:         logMaxBackups,
		LogMaxAgeDays:         logMaxAgeDays,
		LogCompress:           &logCompress,
		Proxy:                 proxyParams,
		MenuBarVisibility:     &menuBarVisibility,
		AutoStart:             &autoStart,
		MinimizeToTrayOnStart: &minimizeToTrayOnStart,
		AgentModelProviderID:  agentModelProviderID,
		AgentModelName:        agentModelName,
		AgentStreamEnabled:    &agentStreamEnabled,
		ChatTemperature:       &chatTemperature,
		ChatMaxTokens:         &chatMaxTokens,
		Skills:                current.Skills(),
		ToolsConfig:           nextTools,
		SkillsConfig:          nextSkills,
		Commands:              commandsParams,
		Channels:              channelsParams,
		Gateway:               gatewayParams,
		Memory:                memorySettingsParamsFromSettings(memory),
	})
	if err != nil {
		return dto.Settings{}, err
	}

	if err := service.repo.Save(ctx, updated); err != nil {
		return dto.Settings{}, err
	}

	effectiveAppearance, err := service.resolveAppearance(ctx, updated.Appearance())
	if err != nil {
		return dto.Settings{}, err
	}

	systemThemeColor := service.resolveSystemThemeColor(ctx)
	return toDTO(updated, effectiveAppearance, systemThemeColor), nil
}

func (service *SettingsService) resolveAppearance(ctx context.Context, appearance settings.AppearanceMode) (settings.AppearanceMode, error) {
	if appearance != settings.AppearanceAuto {
		return appearance, nil
	}

	isDark, err := service.themeProvider.IsDarkMode(ctx)
	if err != nil {
		return settings.AppearanceLight, err
	}

	if isDark {
		return settings.AppearanceDark, nil
	}

	return settings.AppearanceLight, nil
}

func sanitizeMenuBarVisibility(value string) string {
	if runtime.GOOS == "windows" && value == settings.MenuBarVisibilityNever.String() {
		return settings.MenuBarVisibilityWhenRunning.String()
	}
	return value
}

func toDTO(current settings.Settings, effective settings.AppearanceMode, systemThemeColor string) dto.Settings {
	menuBarVisibility := sanitizeMenuBarVisibility(current.MenuBarVisibility().String())
	gateway := current.Gateway()
	toolsConfig := cloneSettingsAnyMap(current.ToolsConfig())
	skillsConfig := cloneSettingsAnyMap(current.SkillsConfig())
	return dto.Settings{
		Appearance:          current.Appearance().String(),
		EffectiveAppearance: effective.String(),
		FontFamily:          current.FontFamily(),
		FontSize:            current.FontSize(),
		ThemeColor:          current.ThemeColor(),
		ColorScheme:         current.ColorScheme().String(),
		SystemThemeColor:    systemThemeColor,
		Language:            current.Language().String(),
		DownloadDirectory:   current.DownloadDirectory(),
		MainBounds: dto.WindowBounds{
			X:      current.MainBounds().X(),
			Y:      current.MainBounds().Y(),
			Width:  current.MainBounds().Width(),
			Height: current.MainBounds().Height(),
		},
		SettingsBounds: dto.WindowBounds{
			X:      current.SettingsBounds().X(),
			Y:      current.SettingsBounds().Y(),
			Width:  current.SettingsBounds().Width(),
			Height: current.SettingsBounds().Height(),
		},
		Version:               current.Version(),
		LogLevel:              current.LogLevel().String(),
		LogMaxSizeMB:          current.LogMaxSizeMB(),
		LogMaxBackups:         current.LogMaxBackups(),
		LogMaxAgeDays:         current.LogMaxAgeDays(),
		LogCompress:           current.LogCompress(),
		MenuBarVisibility:     menuBarVisibility,
		AutoStart:             current.AutoStart(),
		MinimizeToTrayOnStart: current.MinimizeToTrayOnStart(),
		AgentModelProviderID:  current.AgentModelProviderID(),
		AgentModelName:        current.AgentModelName(),
		AgentStreamEnabled:    current.AgentStreamEnabled(),
		ChatTemperature:       current.ChatTemperature(),
		ChatMaxTokens:         current.ChatMaxTokens(),
		Proxy:                 toProxyDTO(current.Proxy()),
		Gateway:               toGatewayDTO(gateway),
		Memory:                toMemoryDTO(current.Memory()),
		Tools:                 toolsConfig,
		Skills:                skillsConfig,
		Commands:              current.Commands().Flags(),
		Channels:              current.Channels().Config(),
	}
}

func cloneSettingsAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func mergeSettingsAnyMap(base map[string]any, overlay map[string]any) map[string]any {
	result := cloneSettingsAnyMap(base)
	if result == nil {
		result = map[string]any{}
	}
	for key, value := range overlay {
		overlayMap, ok := value.(map[string]any)
		if !ok {
			result[key] = value
			continue
		}
		if existing, ok := result[key].(map[string]any); ok {
			result[key] = mergeSettingsAnyMap(existing, overlayMap)
			continue
		}
		result[key] = mergeSettingsAnyMap(nil, overlayMap)
	}
	return result
}

func (service *SettingsService) resolveSystemThemeColor(ctx context.Context) string {
	if service.themeProvider == nil {
		return ""
	}
	accent, err := service.themeProvider.AccentColor(ctx)
	if err != nil {
		return ""
	}
	return accent
}

func toProxyDTO(proxy settings.ProxySettings) dto.Proxy {
	testedAt := ""
	if !proxy.LastTestedAt().IsZero() {
		testedAt = proxy.LastTestedAt().Format(time.RFC3339)
	}
	return dto.Proxy{
		Mode:           proxy.Mode().String(),
		Scheme:         proxy.Scheme().String(),
		Host:           proxy.Host(),
		Port:           proxy.Port(),
		Username:       proxy.Username(),
		Password:       proxy.Password(),
		NoProxy:        proxy.NoProxy(),
		TimeoutSeconds: int(proxy.Timeout().Seconds()),
		TestedAt:       testedAt,
		TestSuccess:    proxy.TestSuccess(),
		TestMessage:    proxy.TestMessage(),
	}
}

func parseProxyTestedAt(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		zap.L().Warn("invalid proxy testedAt, ignoring", zap.String("testedAt", value), zap.Error(err))
		return nil
	}
	return &parsed
}

func gatewaySettingsParamsFromSettings(gateway settings.GatewaySettings) (settings.GatewaySettingsParams, error) {
	raw, err := json.Marshal(gateway)
	if err != nil {
		return settings.GatewaySettingsParams{}, err
	}
	var params settings.GatewaySettingsParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return settings.GatewaySettingsParams{}, err
	}
	return params, nil
}

func toGatewayDTO(gateway settings.GatewaySettings) dto.GatewaySettings {
	normalized := normalizeGatewaySettings(gateway)
	raw, err := json.Marshal(normalized)
	if err != nil {
		zap.L().Warn("failed to serialize gateway settings", zap.Error(err))
		return dto.GatewaySettings{}
	}
	var result dto.GatewaySettings
	if err := json.Unmarshal(raw, &result); err != nil {
		zap.L().Warn("failed to deserialize gateway settings", zap.Error(err))
		return dto.GatewaySettings{}
	}
	return result
}

func toMemoryDTO(memory settings.MemorySettings) dto.MemorySettings {
	return dto.MemorySettings{
		Enabled:           memory.Enabled,
		EmbeddingProvider: strings.TrimSpace(memory.EmbeddingProvider),
		EmbeddingModel:    strings.TrimSpace(memory.EmbeddingModel),
		LLMProvider:       strings.TrimSpace(memory.LLMProvider),
		LLMModel:          strings.TrimSpace(memory.LLMModel),
		RecallTopK:        memory.RecallTopK,
		VectorWeight:      memory.VectorWeight,
		TextWeight:        memory.TextWeight,
		RecencyWeight:     memory.RecencyWeight,
		RecencyHalfLife:   memory.RecencyHalfLife,
		MinScore:          memory.MinScore,
		AutoRecall:        memory.AutoRecall,
		AutoCapture:       memory.AutoCapture,
		SessionLifecycle:  memory.SessionLifecycle,
		CaptureMaxEntries: memory.CaptureMaxEntries,
	}
}

func memorySettingsParamsFromSettings(memory settings.MemorySettings) settings.MemorySettingsParams {
	return settings.MemorySettingsParams{
		Enabled:           memory.Enabled,
		EmbeddingProvider: strings.TrimSpace(memory.EmbeddingProvider),
		EmbeddingModel:    strings.TrimSpace(memory.EmbeddingModel),
		LLMProvider:       strings.TrimSpace(memory.LLMProvider),
		LLMModel:          strings.TrimSpace(memory.LLMModel),
		RecallTopK:        memory.RecallTopK,
		VectorWeight:      memory.VectorWeight,
		TextWeight:        memory.TextWeight,
		RecencyWeight:     memory.RecencyWeight,
		RecencyHalfLife:   memory.RecencyHalfLife,
		MinScore:          memory.MinScore,
		AutoRecall:        memory.AutoRecall,
		AutoCapture:       memory.AutoCapture,
		SessionLifecycle:  memory.SessionLifecycle,
		CaptureMaxEntries: memory.CaptureMaxEntries,
	}
}

func normalizeGatewaySettings(gateway settings.GatewaySettings) settings.GatewaySettings {
	defaults := settings.DefaultGatewaySettings()
	gateway.Runtime.DebugMode = settings.ResolveGatewayDebugMode(
		gateway.Runtime.DebugMode.String(),
		gateway.Runtime.RecordPrompt,
	)
	gateway.Runtime.RecordPrompt = settings.GatewayDebugModeRecordsPrompt(gateway.Runtime.DebugMode)

	toolLoop := gateway.Runtime.ToolLoopDetection
	if toolLoop.HistorySize <= 0 && toolLoop.WarnThreshold <= 0 && toolLoop.CriticalThreshold <= 0 &&
		toolLoop.GlobalCircuitBreakerThreshold <= 0 &&
		!toolLoop.Detectors.GenericRepeat && !toolLoop.Detectors.KnownPollNoProgress && !toolLoop.Detectors.PingPong {
		toolLoop = defaults.Runtime.ToolLoopDetection
	} else {
		if toolLoop.HistorySize <= 0 {
			toolLoop.HistorySize = defaults.Runtime.ToolLoopDetection.HistorySize
		}
		if toolLoop.WarnThreshold <= 0 {
			toolLoop.WarnThreshold = defaults.Runtime.ToolLoopDetection.WarnThreshold
		}
		if toolLoop.CriticalThreshold <= 0 {
			toolLoop.CriticalThreshold = defaults.Runtime.ToolLoopDetection.CriticalThreshold
		}
		if toolLoop.GlobalCircuitBreakerThreshold <= 0 {
			toolLoop.GlobalCircuitBreakerThreshold = defaults.Runtime.ToolLoopDetection.GlobalCircuitBreakerThreshold
		}
	}
	gateway.Runtime.ToolLoopDetection = toolLoop

	if gateway.Runtime.ContextWindow.WarnTokens <= 0 {
		gateway.Runtime.ContextWindow.WarnTokens = defaults.Runtime.ContextWindow.WarnTokens
	}
	if gateway.Runtime.ContextWindow.HardTokens <= 0 {
		gateway.Runtime.ContextWindow.HardTokens = defaults.Runtime.ContextWindow.HardTokens
	}

	compaction := gateway.Runtime.Compaction
	compactionEmpty := strings.TrimSpace(compaction.Mode) == "" &&
		compaction.ReserveTokens == 0 && compaction.KeepRecentTokens == 0 && compaction.ReserveTokensFloor == 0 &&
		compaction.MaxHistoryShare == 0 && !compaction.MemoryFlush.Enabled &&
		compaction.MemoryFlush.SoftThresholdTokens == 0 && strings.TrimSpace(compaction.MemoryFlush.Prompt) == "" &&
		strings.TrimSpace(compaction.MemoryFlush.SystemPrompt) == ""
	if compactionEmpty {
		compaction = defaults.Runtime.Compaction
	} else {
		if strings.TrimSpace(compaction.Mode) == "" {
			compaction.Mode = defaults.Runtime.Compaction.Mode
		}
		if compaction.ReserveTokens <= 0 {
			compaction.ReserveTokens = defaults.Runtime.Compaction.ReserveTokens
		}
		if compaction.KeepRecentTokens <= 0 {
			compaction.KeepRecentTokens = defaults.Runtime.Compaction.KeepRecentTokens
		}
		if compaction.ReserveTokensFloor <= 0 {
			compaction.ReserveTokensFloor = defaults.Runtime.Compaction.ReserveTokensFloor
		}
		if compaction.MaxHistoryShare <= 0 {
			compaction.MaxHistoryShare = defaults.Runtime.Compaction.MaxHistoryShare
		}
		if compaction.MemoryFlush.SoftThresholdTokens <= 0 {
			compaction.MemoryFlush.SoftThresholdTokens = defaults.Runtime.Compaction.MemoryFlush.SoftThresholdTokens
		}
		if strings.TrimSpace(compaction.MemoryFlush.Prompt) == "" {
			compaction.MemoryFlush.Prompt = defaults.Runtime.Compaction.MemoryFlush.Prompt
		}
		if strings.TrimSpace(compaction.MemoryFlush.SystemPrompt) == "" {
			compaction.MemoryFlush.SystemPrompt = defaults.Runtime.Compaction.MemoryFlush.SystemPrompt
		}
		if !compaction.MemoryFlush.Enabled && compaction.MemoryFlush.SoftThresholdTokens == 0 &&
			strings.TrimSpace(compaction.MemoryFlush.Prompt) == "" && strings.TrimSpace(compaction.MemoryFlush.SystemPrompt) == "" {
			compaction.MemoryFlush.Enabled = defaults.Runtime.Compaction.MemoryFlush.Enabled
		}
	}
	gateway.Runtime.Compaction = compaction

	if gateway.Queue.Lanes.Main <= 0 {
		gateway.Queue.Lanes.Main = defaults.Queue.Lanes.Main
	}
	if gateway.Queue.Lanes.Subagent <= 0 {
		gateway.Queue.Lanes.Subagent = defaults.Queue.Lanes.Subagent
	}
	if gateway.Queue.Lanes.Cron <= 0 {
		gateway.Queue.Lanes.Cron = defaults.Queue.Lanes.Cron
	}
	if gateway.Queue.SessionConcurrency < 0 {
		gateway.Queue.SessionConcurrency = defaults.Queue.SessionConcurrency
	}

	heartbeat := gateway.Heartbeat
	rawPeriodicEnabled := heartbeat.Periodic.Enabled
	rawPeriodicEvery := strings.TrimSpace(heartbeat.Periodic.Every)
	checklistEmpty := strings.TrimSpace(heartbeat.Checklist.Title) == "" &&
		len(heartbeat.Checklist.Items) == 0 &&
		strings.TrimSpace(heartbeat.Checklist.Notes) == "" &&
		heartbeat.Checklist.Version == 0 &&
		strings.TrimSpace(heartbeat.Checklist.UpdatedAt) == ""
	heartbeatEmpty := !heartbeat.Enabled && heartbeat.EveryMinutes == 0 && strings.TrimSpace(heartbeat.Every) == "" &&
		strings.TrimSpace(heartbeat.Target) == "" && strings.TrimSpace(heartbeat.Prompt) == "" &&
		strings.TrimSpace(heartbeat.To) == "" && strings.TrimSpace(heartbeat.AccountID) == "" &&
		strings.TrimSpace(heartbeat.Model) == "" && strings.TrimSpace(heartbeat.Session) == "" &&
		strings.TrimSpace(heartbeat.RunSession) == "" && strings.TrimSpace(heartbeat.PromptAppend) == "" &&
		!heartbeat.IncludeReasoning && !heartbeat.SuppressToolErrorWarnings &&
		strings.TrimSpace(heartbeat.ActiveHours.Start) == "" && strings.TrimSpace(heartbeat.ActiveHours.End) == "" &&
		strings.TrimSpace(heartbeat.ActiveHours.Timezone) == "" &&
		strings.TrimSpace(heartbeat.Periodic.Every) == "" && !heartbeat.Periodic.Enabled &&
		!heartbeat.Delivery.Periodic.Center &&
		strings.TrimSpace(heartbeat.Delivery.Periodic.PopupMinSeverity) == "" &&
		strings.TrimSpace(heartbeat.Delivery.Periodic.ToastMinSeverity) == "" &&
		strings.TrimSpace(heartbeat.Delivery.Periodic.OSMinSeverity) == "" &&
		!heartbeat.Delivery.EventDriven.Center &&
		strings.TrimSpace(heartbeat.Delivery.EventDriven.PopupMinSeverity) == "" &&
		strings.TrimSpace(heartbeat.Delivery.EventDriven.ToastMinSeverity) == "" &&
		strings.TrimSpace(heartbeat.Delivery.EventDriven.OSMinSeverity) == "" &&
		strings.TrimSpace(heartbeat.Delivery.ThreadReplyMode) == "" &&
		strings.TrimSpace(heartbeat.Events.CronWakeMode) == "" &&
		strings.TrimSpace(heartbeat.Events.ExecWakeMode) == "" &&
		strings.TrimSpace(heartbeat.Events.SubagentWakeMode) == "" &&
		checklistEmpty
	if heartbeatEmpty {
		heartbeat = defaults.Heartbeat
	} else {
		if strings.TrimSpace(heartbeat.Every) == "" && heartbeat.EveryMinutes <= 0 {
			heartbeat.Every = defaults.Heartbeat.Every
			heartbeat.EveryMinutes = defaults.Heartbeat.EveryMinutes
		}
		if strings.TrimSpace(heartbeat.Target) == "" {
			heartbeat.Target = defaults.Heartbeat.Target
		}
		if strings.TrimSpace(heartbeat.RunSession) == "" {
			heartbeat.RunSession = strings.TrimSpace(heartbeat.Session)
		}
		if strings.TrimSpace(heartbeat.PromptAppend) == "" && strings.TrimSpace(heartbeat.Prompt) != "" {
			heartbeat.PromptAppend = strings.TrimSpace(heartbeat.Prompt)
		}
		if strings.TrimSpace(heartbeat.ActiveHours.Start) == "" {
			heartbeat.ActiveHours.Start = defaults.Heartbeat.ActiveHours.Start
		}
		if strings.TrimSpace(heartbeat.ActiveHours.End) == "" {
			heartbeat.ActiveHours.End = defaults.Heartbeat.ActiveHours.End
		}
		if strings.TrimSpace(heartbeat.ActiveHours.Timezone) == "" {
			heartbeat.ActiveHours.Timezone = defaults.Heartbeat.ActiveHours.Timezone
		}
		if strings.TrimSpace(heartbeat.Periodic.Every) == "" {
			if strings.TrimSpace(heartbeat.Every) != "" {
				heartbeat.Periodic.Every = strings.TrimSpace(heartbeat.Every)
			} else {
				heartbeat.Periodic.Every = defaults.Heartbeat.Periodic.Every
			}
		}
		if !rawPeriodicEnabled && rawPeriodicEvery == "" {
			heartbeat.Periodic.Enabled = defaults.Heartbeat.Periodic.Enabled
		}
		if !heartbeat.Delivery.Periodic.Center && strings.TrimSpace(heartbeat.Delivery.Periodic.PopupMinSeverity) == "" &&
			strings.TrimSpace(heartbeat.Delivery.Periodic.ToastMinSeverity) == "" &&
			strings.TrimSpace(heartbeat.Delivery.Periodic.OSMinSeverity) == "" {
			heartbeat.Delivery.Periodic = defaults.Heartbeat.Delivery.Periodic
		}
		if !heartbeat.Delivery.EventDriven.Center && strings.TrimSpace(heartbeat.Delivery.EventDriven.PopupMinSeverity) == "" &&
			strings.TrimSpace(heartbeat.Delivery.EventDriven.ToastMinSeverity) == "" &&
			strings.TrimSpace(heartbeat.Delivery.EventDriven.OSMinSeverity) == "" {
			heartbeat.Delivery.EventDriven = defaults.Heartbeat.Delivery.EventDriven
		}
		if strings.TrimSpace(heartbeat.Delivery.ThreadReplyMode) == "" {
			heartbeat.Delivery.ThreadReplyMode = defaults.Heartbeat.Delivery.ThreadReplyMode
		}
		if strings.TrimSpace(heartbeat.Events.CronWakeMode) == "" {
			heartbeat.Events.CronWakeMode = defaults.Heartbeat.Events.CronWakeMode
		}
		if strings.TrimSpace(heartbeat.Events.ExecWakeMode) == "" {
			heartbeat.Events.ExecWakeMode = defaults.Heartbeat.Events.ExecWakeMode
		}
		if strings.TrimSpace(heartbeat.Events.SubagentWakeMode) == "" {
			heartbeat.Events.SubagentWakeMode = defaults.Heartbeat.Events.SubagentWakeMode
		}
		items := make([]settings.GatewayHeartbeatChecklistItem, 0, len(heartbeat.Checklist.Items))
		for _, item := range heartbeat.Checklist.Items {
			id := strings.TrimSpace(item.ID)
			text := strings.TrimSpace(item.Text)
			if id == "" && text == "" {
				continue
			}
			items = append(items, settings.GatewayHeartbeatChecklistItem{
				ID:       id,
				Text:     text,
				Done:     item.Done,
				Priority: strings.TrimSpace(item.Priority),
			})
		}
		heartbeat.Checklist = settings.GatewayHeartbeatChecklist{
			Title:     strings.TrimSpace(heartbeat.Checklist.Title),
			Items:     items,
			Notes:     strings.TrimSpace(heartbeat.Checklist.Notes),
			Version:   heartbeat.Checklist.Version,
			UpdatedAt: strings.TrimSpace(heartbeat.Checklist.UpdatedAt),
		}
		if heartbeat.Checklist.Version < 0 {
			heartbeat.Checklist.Version = 0
		}
	}
	gateway.Heartbeat = heartbeat

	if gateway.Subagents.MaxDepth <= 0 {
		gateway.Subagents.MaxDepth = defaults.Subagents.MaxDepth
	}
	if gateway.Subagents.MaxChildren <= 0 {
		gateway.Subagents.MaxChildren = defaults.Subagents.MaxChildren
	}
	if gateway.Subagents.MaxConcurrent <= 0 {
		gateway.Subagents.MaxConcurrent = defaults.Subagents.MaxConcurrent
	}

	cron := gateway.Cron
	cronEmpty := cron.MaxConcurrentRuns == 0 &&
		strings.TrimSpace(cron.SessionRetention) == "" &&
		strings.TrimSpace(cron.RunLog.MaxBytes) == "" &&
		cron.RunLog.KeepLines == 0 &&
		!cron.Enabled
	if cronEmpty {
		cron = defaults.Cron
	} else {
		if cron.MaxConcurrentRuns <= 0 {
			cron.MaxConcurrentRuns = defaults.Cron.MaxConcurrentRuns
		}
		if strings.TrimSpace(cron.SessionRetention) == "" {
			cron.SessionRetention = defaults.Cron.SessionRetention
		}
		if strings.TrimSpace(cron.RunLog.MaxBytes) == "" {
			cron.RunLog.MaxBytes = defaults.Cron.RunLog.MaxBytes
		}
		if cron.RunLog.KeepLines <= 0 {
			cron.RunLog.KeepLines = defaults.Cron.RunLog.KeepLines
		}
	}
	gateway.Cron = cron

	return gateway
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
