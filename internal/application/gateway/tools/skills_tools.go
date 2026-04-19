package tools

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	assistantservice "dreamcreator/internal/application/assistant/service"
	externaltoolsdto "dreamcreator/internal/application/externaltools/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
	skillsdto "dreamcreator/internal/application/skills/dto"
	skillsservice "dreamcreator/internal/application/skills/service"
	"dreamcreator/internal/domain/externaltools"
	domainskills "dreamcreator/internal/domain/skills"
)

const clawhubUnavailableReason = "clawhub_unavailable"

const (
	skillsToolErrorInvalidArgs      = "invalid_arguments"
	skillsToolErrorUnsupported      = "unsupported_action"
	skillsToolErrorNotImplemented   = "not_implemented"
	skillsToolErrorMissingParameter = "missing_parameter"
	skillsToolErrorPolicyDenied     = "policy_denied"
	skillsToolErrorApprovalRequired = "approval_required"
	skillsToolErrorDependencyFailed = "dependency_failed"
)

const (
	defaultSkillsDepsInstallTimeoutMs = 120_000
	defaultSkillsAuditMaxEntries      = 200
	defaultSkillsAuditRetentionDays   = 14
	maxSkillsAuditRetentionDays       = 365
	skillsDepsCommandOutputLimit      = 4_096
)

var allowedSkillsToolArgs = map[string]struct{}{
	"action":       {},
	"assistantid":  {},
	"assistant_id": {},
	"providerid":   {},
	"provider_id":  {},
	"skill":        {},
	"skillkey":     {},
	"skill_key":    {},
	"id":           {},
	"name":         {},
	"enabled":      {},
	"apikey":       {},
	"api_key":      {},
	"env":          {},
	"config":       {},
	"installid":    {},
	"install_id":   {},
	"timeoutms":    {},
	"timeout_ms":   {},
}

var allowedSkillManageToolArgs = map[string]struct{}{
	"action":       {},
	"assistantid":  {},
	"assistant_id": {},
	"providerid":   {},
	"provider_id":  {},
	"skill":        {},
	"skillkey":     {},
	"skill_key":    {},
	"id":           {},
	"name":         {},
	"query":        {},
	"limit":        {},
	"version":      {},
	"force":        {},
}

var skillsKnownActions = map[string]struct{}{
	"status":          {},
	"catalog":         {},
	"search_packages": {},
	"install_package": {},
	"update_package":  {},
	"remove_package":  {},
	"sync_packages":   {},
	"bins":            {},
	"install_deps":    {},
	"update_config":   {},
}

var skillsExternalToolByBin = map[string]externaltools.ToolName{
	"clawhub": externaltools.ToolClawHub,
	"bun":     externaltools.ToolBun,
	"ffmpeg":  externaltools.ToolFFmpeg,
	"ffprobe": externaltools.ToolFFmpeg,
	"yt-dlp":  externaltools.ToolYTDLP,
	"ytdlp":   externaltools.ToolYTDLP,
}

// skillsDepsCommandRunner is injectable for tests.
var skillsDepsCommandRunner = runCommandWithInput

type skillsToolResult struct {
	Ok            bool   `json:"ok"`
	Action        string `json:"action,omitempty"`
	Skill         string `json:"skill,omitempty"`
	Data          any    `json:"data,omitempty"`
	Error         string `json:"error,omitempty"`
	ErrorCode     string `json:"errorCode,omitempty"`
	Hint          string `json:"hint,omitempty"`
	RequiresForce bool   `json:"requiresForce,omitempty"`
	ClawhubReady  bool   `json:"clawhubReady"`
	Reason        string `json:"reason,omitempty"`
}

type skillsSettingsUpdater interface {
	SettingsReader
	UpdateSettings(ctx context.Context, request settingsdto.UpdateSettingsRequest) (settingsdto.Settings, error)
}

type skillsSettingsApplier interface {
	ApplySettings(settingsdto.Settings)
}

type skillsExternalToolInstaller interface {
	InstallTool(ctx context.Context, request externaltoolsdto.InstallExternalToolRequest) (externaltoolsdto.ExternalTool, error)
	ToolReadiness(ctx context.Context, name externaltools.ToolName) (bool, string, error)
}

type skillsDepsPolicy struct {
	AllowExternalToolsOnly bool
	AllowGenericInstaller  bool
	AllowedBins            map[string]struct{}
}

type skillsInstallPolicy struct {
	ScannerMode       string
	AllowForceInstall bool
	RequireApproval   bool
}

type skillsDepsInstallResult struct {
	HandledBy   string
	Actions     []map[string]any
	SuccessBins []string
	FailedBins  []string
}

func runSkillsTool(skills *skillsservice.SkillsService, assistants *assistantservice.AssistantService, settings SettingsReader, externalTools skillsExternalToolInstaller) func(ctx context.Context, args string) (string, error) {
	return runSkillsToolWithName("skills", skills, assistants, settings, externalTools)
}

func runSkillsManageTool(skills *skillsservice.SkillsService, assistants *assistantservice.AssistantService, settings SettingsReader, externalTools skillsExternalToolInstaller) func(ctx context.Context, args string) (string, error) {
	return runSkillsToolWithName("skills_manage", skills, assistants, settings, externalTools)
}

func runSkillManageTool(skills *skillsservice.SkillsService, assistants *assistantservice.AssistantService, settings SettingsReader, externalTools skillsExternalToolInstaller) func(ctx context.Context, args string) (string, error) {
	return runSkillsManageTool(skills, assistants, settings, externalTools)
}

func runSkillsToolWithName(toolName string, skills *skillsservice.SkillsService, assistants *assistantservice.AssistantService, settings SettingsReader, externalTools skillsExternalToolInstaller) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if skills == nil {
			return marshalSkillsResult(skillsToolResult{
				Ok:        false,
				Error:     "skills service unavailable",
				ErrorCode: skillsToolErrorNotImplemented,
			}), nil
		}

		payload, err := parseToolArgs(args)
		if err != nil {
			return marshalSkillsResult(skillsToolResult{
				Ok:        false,
				Error:     err.Error(),
				ErrorCode: skillsToolErrorInvalidArgs,
			}), nil
		}
		if err := validateSkillsToolArgs(toolName, payload); err != nil {
			return marshalSkillsResult(skillsToolResult{
				Ok:        false,
				Error:     err.Error(),
				ErrorCode: skillsToolErrorInvalidArgs,
			}), nil
		}

		action := canonicalSkillsAction(toolName, getStringArg(payload, "action"))
		if action == "" {
			return marshalSkillsResult(skillsToolResult{
				Ok:        false,
				Action:    normalizeSkillsActionKey(toolName, getStringArg(payload, "action")),
				Error:     "unsupported action",
				ErrorCode: skillsToolErrorUnsupported,
			}), nil
		}
		actionKey := normalizeSkillsActionKey(toolName, action)
		ctx = skillsservice.WithSkillsAuditSource(ctx, skillsservice.SkillsAuditSourceToolCall)

		assistantID, err := resolveAssistantID(ctx, assistants, getStringArg(payload, "assistantId", "assistant_id"))
		if err != nil {
			return marshalSkillsResult(skillsToolResult{
				Ok:        false,
				Action:    actionKey,
				Error:     err.Error(),
				ErrorCode: skillsToolErrorInvalidArgs,
			}), nil
		}
		providerID := strings.TrimSpace(getStringArg(payload, "providerId", "provider_id"))
		skill := strings.TrimSpace(getStringArg(payload, "skill", "skillKey", "skill_key", "id", "name"))
		version := strings.TrimSpace(getStringArg(payload, "version"))
		force, _ := getBoolArg(payload, "force")
		enabled, enabledProvided := getBoolArg(payload, "enabled")
		installID := strings.TrimSpace(getStringArg(payload, "installId", "install_id"))
		timeoutMs, timeoutProvided := getIntArg(payload, "timeoutMs", "timeout_ms")
		if !timeoutProvided || timeoutMs <= 0 {
			timeoutMs = defaultSkillsDepsInstallTimeoutMs
		}

		auditContext := map[string]any{
			"assistantId": assistantID,
			"providerId":  providerID,
			"skill":       skill,
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
		}
		respond := func(result skillsToolResult) (string, error) {
			if strings.TrimSpace(result.Action) == "" {
				result.Action = actionKey
			} else {
				result.Action = normalizeSkillsActionKey(toolName, result.Action)
			}
			if strings.TrimSpace(result.Skill) == "" && strings.TrimSpace(skill) != "" {
				result.Skill = skill
			}
			record := map[string]any{
				"action":      result.Action,
				"group":       resolveSkillsActionGroup(result.Action),
				"tool":        toolName,
				"skill":       result.Skill,
				"assistantId": auditContext["assistantId"],
				"providerId":  auditContext["providerId"],
				"source":      skillsservice.SkillsAuditSourceToolCall,
				"ok":          result.Ok,
				"errorCode":   result.ErrorCode,
				"error":       strings.TrimSpace(result.Error),
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			}
			appendSkillsAuditRecord(ctx, settings, record)
			return marshalSkillsResult(result), nil
		}

		if _, known := skillsKnownActions[action]; known {
			actionGroup := resolveSkillsActionGroup(actionKey)
			actionMode := resolveSkillsActionMode(ctx, settings, actionKey, actionGroup)
			if actionMode == "deny" {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    actionKey,
					Skill:     skill,
					Error:     "action blocked by skills security policy",
					ErrorCode: skillsToolErrorPolicyDenied,
					Data: map[string]any{
						"actionGroup": actionGroup,
						"mode":        actionMode,
					},
				})
			}
			if actionMode == "ask" {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    actionKey,
					Skill:     skill,
					Error:     "action requires approval by skills security policy",
					ErrorCode: skillsToolErrorApprovalRequired,
					Data: map[string]any{
						"actionGroup": actionGroup,
						"mode":        actionMode,
					},
				})
			}
		}

		var status skillsdto.SkillsStatus
		statusLoaded := false
		loadStatus := func() (skillsdto.SkillsStatus, error) {
			if statusLoaded {
				return status, nil
			}
			resolved, statusErr := skills.GetSkillsStatus(ctx, skillsdto.SkillsStatusRequest{
				ProviderID:  providerID,
				AssistantID: assistantID,
			})
			if statusErr != nil {
				return skillsdto.SkillsStatus{}, statusErr
			}
			status = resolved
			statusLoaded = true
			return status, nil
		}
		loadCatalog := func(workspaceRoot string) ([]skillsdto.ProviderSkillSpec, error) {
			return skills.ResolveSkillsForProviderInWorkspace(ctx, skillsdto.ResolveSkillsRequest{ProviderID: providerID}, workspaceRoot)
		}

		switch action {
		case "status":
			resolvedStatus, statusErr := loadStatus()
			if statusErr != nil {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Error:     statusErr.Error(),
					ErrorCode: skillsToolErrorNotImplemented,
				})
			}
			catalog, catalogErr := loadCatalog(resolvedStatus.WorkspaceRoot)
			if catalogErr != nil {
				return respond(skillsToolResult{
					Ok:           false,
					Action:       action,
					Error:        catalogErr.Error(),
					ErrorCode:    skillsToolErrorNotImplemented,
					ClawhubReady: resolvedStatus.ClawhubReady,
					Reason:       resolvedStatus.Reason,
				})
			}
			return respond(skillsToolResult{
				Ok:           true,
				Action:       action,
				Data:         buildSkillsStatusPayload(resolvedStatus, catalog),
				ClawhubReady: resolvedStatus.ClawhubReady,
				Reason:       resolvedStatus.Reason,
			})
		case "catalog":
			resolvedStatus, statusErr := loadStatus()
			if statusErr != nil {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Error:     statusErr.Error(),
					ErrorCode: skillsToolErrorNotImplemented,
				})
			}
			catalog, listErr := loadCatalog(resolvedStatus.WorkspaceRoot)
			if listErr != nil {
				return respond(skillsToolResult{
					Ok:           false,
					Action:       action,
					Error:        listErr.Error(),
					ErrorCode:    skillsToolErrorNotImplemented,
					ClawhubReady: resolvedStatus.ClawhubReady,
					Reason:       resolvedStatus.Reason,
				})
			}
			return respond(skillsToolResult{
				Ok:           true,
				Action:       action,
				Data:         map[string]any{"items": catalog, "status": resolvedStatus},
				ClawhubReady: resolvedStatus.ClawhubReady,
				Reason:       resolvedStatus.Reason,
			})
		case "search_packages":
			query := strings.TrimSpace(getStringArg(payload, "query"))
			if query == "" {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Error:     "query is required",
					ErrorCode: skillsToolErrorMissingParameter,
				})
			}
			resolvedStatus, statusErr := loadStatus()
			if statusErr != nil {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Error:     statusErr.Error(),
					ErrorCode: skillsToolErrorNotImplemented,
				})
			}
			if !resolvedStatus.ClawhubReady {
				return respond(clawhubUnavailableSkillsResult(action, ""))
			}
			limit, _ := getIntArg(payload, "limit")
			items, searchErr := skills.SearchSkills(ctx, skillsdto.SearchSkillsRequest{
				Query:         query,
				Limit:         limit,
				AssistantID:   assistantID,
				WorkspaceRoot: resolvedStatus.WorkspaceRoot,
			})
			if searchErr != nil {
				if errors.Is(searchErr, skillsservice.ErrClawHubUnavailable) {
					return respond(clawhubUnavailableSkillsResult(action, ""))
				}
				return respond(skillsToolResult{
					Ok:           false,
					Action:       action,
					Error:        searchErr.Error(),
					ErrorCode:    skillsToolErrorNotImplemented,
					ClawhubReady: resolvedStatus.ClawhubReady,
					Reason:       resolvedStatus.Reason,
				})
			}
			return respond(skillsToolResult{
				Ok:           true,
				Action:       action,
				Data:         items,
				ClawhubReady: true,
			})
		case "install_package", "update_package", "remove_package", "sync_packages":
			resolvedStatus, statusErr := loadStatus()
			if statusErr != nil {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Skill:     skill,
					Error:     statusErr.Error(),
					ErrorCode: skillsToolErrorNotImplemented,
				})
			}
			if !resolvedStatus.ClawhubReady {
				return respond(clawhubUnavailableSkillsResult(action, skill))
			}
			installPolicy := resolveSkillsInstallPolicy(ctx, settings)
			var data any = map[string]any{"updated": true}
			switch action {
			case "install_package":
				if skill == "" {
					return respond(skillsToolResult{
						Ok:        false,
						Action:    action,
						Error:     "skill is required",
						ErrorCode: skillsToolErrorMissingParameter,
					})
				}
				if force {
					if !installPolicy.AllowForceInstall {
						return respond(skillsToolResult{
							Ok:        false,
							Action:    action,
							Skill:     skill,
							Error:     "force install is disabled by skills security policy",
							ErrorCode: skillsToolErrorPolicyDenied,
							Data: map[string]any{
								"scannerMode":       installPolicy.ScannerMode,
								"allowForceInstall": installPolicy.AllowForceInstall,
								"requireApproval":   installPolicy.RequireApproval,
							},
						})
					}
					if installPolicy.RequireApproval {
						return respond(skillsToolResult{
							Ok:        false,
							Action:    action,
							Skill:     skill,
							Error:     "force install requires approval by skills security policy",
							ErrorCode: skillsToolErrorApprovalRequired,
							Data: map[string]any{
								"scannerMode":       installPolicy.ScannerMode,
								"allowForceInstall": installPolicy.AllowForceInstall,
								"requireApproval":   installPolicy.RequireApproval,
							},
						})
					}
				}
				if installErr := skills.InstallSkill(ctx, skillsdto.InstallSkillRequest{
					Skill:         skill,
					Version:       version,
					Force:         force,
					AssistantID:   assistantID,
					WorkspaceRoot: resolvedStatus.WorkspaceRoot,
				}); installErr != nil {
					if result, handled := resolveSkillManageInstallPolicyError(action, skill, installErr, installPolicy, resolvedStatus); handled {
						return respond(result)
					}
					return respond(toSkillsErrorResult(action, skill, installErr, resolvedStatus))
				}
			case "update_package":
				if skill == "" {
					return respond(skillsToolResult{
						Ok:        false,
						Action:    action,
						Error:     "skill is required",
						ErrorCode: skillsToolErrorMissingParameter,
					})
				}
				if force {
					if !installPolicy.AllowForceInstall {
						return respond(skillsToolResult{
							Ok:        false,
							Action:    action,
							Skill:     skill,
							Error:     "force update is disabled by skills security policy",
							ErrorCode: skillsToolErrorPolicyDenied,
							Data: map[string]any{
								"scannerMode":       installPolicy.ScannerMode,
								"allowForceInstall": installPolicy.AllowForceInstall,
								"requireApproval":   installPolicy.RequireApproval,
							},
						})
					}
					if installPolicy.RequireApproval {
						return respond(skillsToolResult{
							Ok:        false,
							Action:    action,
							Skill:     skill,
							Error:     "force update requires approval by skills security policy",
							ErrorCode: skillsToolErrorApprovalRequired,
							Data: map[string]any{
								"scannerMode":       installPolicy.ScannerMode,
								"allowForceInstall": installPolicy.AllowForceInstall,
								"requireApproval":   installPolicy.RequireApproval,
							},
						})
					}
				}
				if updateErr := skills.UpdateSkill(ctx, skillsdto.UpdateSkillRequest{
					Skill:         skill,
					Version:       version,
					Force:         force,
					AssistantID:   assistantID,
					WorkspaceRoot: resolvedStatus.WorkspaceRoot,
				}); updateErr != nil {
					if result, handled := resolveSkillManageInstallPolicyError(action, skill, updateErr, installPolicy, resolvedStatus); handled {
						return respond(result)
					}
					return respond(toSkillsErrorResult(action, skill, updateErr, resolvedStatus))
				}
			case "remove_package":
				if skill == "" {
					return respond(skillsToolResult{
						Ok:        false,
						Action:    action,
						Error:     "skill is required",
						ErrorCode: skillsToolErrorMissingParameter,
					})
				}
				if removeErr := skills.RemoveSkill(ctx, skillsdto.RemoveSkillRequest{
					Skill:         skill,
					AssistantID:   assistantID,
					WorkspaceRoot: resolvedStatus.WorkspaceRoot,
				}); removeErr != nil {
					return respond(toSkillsErrorResult(action, skill, removeErr, resolvedStatus))
				}
				_ = skills.DeleteSkill(ctx, skillsdto.DeleteSkillRequest{ID: skill})
			case "sync_packages":
				synced, syncErr := skills.SyncSkills(ctx, skillsdto.SyncSkillsRequest{
					ProviderID:    providerID,
					AssistantID:   assistantID,
					WorkspaceRoot: resolvedStatus.WorkspaceRoot,
				})
				if syncErr != nil {
					return respond(toSkillsErrorResult(action, skill, syncErr, resolvedStatus))
				}
				data = synced
			}
			catalog, catalogErr := loadCatalog(resolvedStatus.WorkspaceRoot)
			if catalogErr != nil {
				return respond(skillsToolResult{
					Ok:           false,
					Action:       action,
					Skill:        skill,
					Error:        catalogErr.Error(),
					ErrorCode:    skillsToolErrorNotImplemented,
					ClawhubReady: resolvedStatus.ClawhubReady,
					Reason:       resolvedStatus.Reason,
				})
			}
			return respond(skillsToolResult{
				Ok:           true,
				Action:       action,
				Skill:        skill,
				Data:         map[string]any{"result": data, "catalog": catalog},
				ClawhubReady: true,
			})
		case "bins":
			resolvedStatus, statusErr := loadStatus()
			if statusErr != nil {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Skill:     skill,
					Error:     statusErr.Error(),
					ErrorCode: skillsToolErrorNotImplemented,
				})
			}
			if !resolvedStatus.ClawhubReady {
				return respond(clawhubUnavailableSkillsResult(action, skill))
			}
			if skill != "" {
				detail, inspectErr := skills.InspectSkill(ctx, skillsdto.InspectSkillRequest{
					Skill:         skill,
					AssistantID:   assistantID,
					WorkspaceRoot: resolvedStatus.WorkspaceRoot,
				})
				if inspectErr != nil {
					return respond(toSkillsErrorResult(action, skill, inspectErr, resolvedStatus))
				}
				bins, anyBins := extractRuntimeBins(detail.Runtime)
				return respond(skillsToolResult{
					Ok:           true,
					Action:       action,
					Skill:        skill,
					Data:         map[string]any{"skill": skill, "bins": bins, "anyBins": anyBins, "runtime": detail.Runtime},
					ClawhubReady: true,
				})
			}

			catalog, catalogErr := loadCatalog(resolvedStatus.WorkspaceRoot)
			if catalogErr != nil {
				return respond(skillsToolResult{
					Ok:           false,
					Action:       action,
					Error:        catalogErr.Error(),
					ErrorCode:    skillsToolErrorNotImplemented,
					ClawhubReady: resolvedStatus.ClawhubReady,
					Reason:       resolvedStatus.Reason,
				})
			}
			items := make([]map[string]any, 0, len(catalog))
			failed := make([]map[string]any, 0)
			allBins := make([]string, 0)
			allAnyBins := make([]string, 0)
			for _, item := range catalog {
				detail, inspectErr := skills.InspectSkill(ctx, skillsdto.InspectSkillRequest{
					Skill:         item.ID,
					AssistantID:   assistantID,
					WorkspaceRoot: resolvedStatus.WorkspaceRoot,
				})
				if inspectErr != nil {
					failed = append(failed, map[string]any{"skill": item.ID, "error": inspectErr.Error()})
					continue
				}
				bins, anyBins := extractRuntimeBins(detail.Runtime)
				allBins = append(allBins, bins...)
				allAnyBins = append(allAnyBins, anyBins...)
				items = append(items, map[string]any{
					"skill":   item.ID,
					"bins":    bins,
					"anyBins": anyBins,
				})
			}
			return respond(skillsToolResult{
				Ok:           true,
				Action:       action,
				Data:         map[string]any{"items": items, "failed": failed, "bins": uniqueStrings(allBins), "anyBins": uniqueStrings(allAnyBins)},
				ClawhubReady: true,
			})
		case "install_deps":
			if skill == "" {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Error:     "skill is required",
					ErrorCode: skillsToolErrorMissingParameter,
				})
			}
			resolvedStatus, statusErr := loadStatus()
			if statusErr != nil {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Skill:     skill,
					Error:     statusErr.Error(),
					ErrorCode: skillsToolErrorNotImplemented,
				})
			}
			if !resolvedStatus.ClawhubReady {
				return respond(clawhubUnavailableSkillsResult(action, skill))
			}
			detail, inspectErr := skills.InspectSkill(ctx, skillsdto.InspectSkillRequest{
				Skill:         skill,
				AssistantID:   assistantID,
				WorkspaceRoot: resolvedStatus.WorkspaceRoot,
			})
			if inspectErr != nil {
				return respond(toSkillsErrorResult(action, skill, inspectErr, resolvedStatus))
			}
			depsResult, depsErr := installSkillDependencies(ctx, detail.Runtime, resolveInstallHints(detail.Runtime), timeoutMs, settings, externalTools)
			data := map[string]any{
				"installId":           installID,
				"runtimeRequirements": detail.Runtime,
				"installHints":        resolveInstallHints(detail.Runtime),
				"handledBy":           depsResult.HandledBy,
				"actions":             depsResult.Actions,
				"successBins":         depsResult.SuccessBins,
				"failedBins":          depsResult.FailedBins,
			}
			if depsErr != nil {
				return respond(skillsToolResult{
					Ok:           false,
					Action:       action,
					Skill:        skill,
					Error:        depsErr.Error(),
					ErrorCode:    skillsToolErrorDependencyFailed,
					Data:         data,
					ClawhubReady: true,
				})
			}
			if len(depsResult.FailedBins) > 0 {
				return respond(skillsToolResult{
					Ok:           false,
					Action:       action,
					Skill:        skill,
					Error:        "some dependencies failed to install",
					ErrorCode:    skillsToolErrorDependencyFailed,
					Data:         data,
					ClawhubReady: true,
				})
			}
			return respond(skillsToolResult{
				Ok:           true,
				Action:       action,
				Skill:        skill,
				Data:         data,
				ClawhubReady: true,
			})
		case "update_config":
			if skill == "" {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Error:     "skill is required",
					ErrorCode: skillsToolErrorMissingParameter,
				})
			}

			apiKeyProvided := hasAnyArg(payload, "apiKey", "api_key")
			apiKey := strings.TrimSpace(getStringArg(payload, "apiKey", "api_key"))
			envProvided := hasAnyArg(payload, "env")
			envPayload := getMapArg(payload, "env")
			configProvided := hasAnyArg(payload, "config")
			configPayload := getMapArg(payload, "config")
			if !enabledProvided && !apiKeyProvided && !envProvided && !configProvided {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Skill:     skill,
					Error:     "at least one of enabled/apiKey/env/config is required",
					ErrorCode: skillsToolErrorMissingParameter,
				})
			}

			if enabledProvided {
				if enableErr := setSkillEnabled(ctx, skills, skill, enabled); enableErr != nil {
					return respond(skillsToolResult{
						Ok:        false,
						Action:    action,
						Skill:     skill,
						Error:     enableErr.Error(),
						ErrorCode: skillsToolErrorNotImplemented,
					})
				}
			}

			updatedEntry := map[string]any{}
			if apiKeyProvided || envProvided || configProvided || enabledProvided {
				entry, updateErr := updateSkillConfigEntry(ctx, settings, skill, skillConfigPatchInput{
					EnabledProvided: enabledProvided,
					Enabled:         enabled,
					APIKeyProvided:  apiKeyProvided,
					APIKey:          apiKey,
					EnvProvided:     envProvided,
					Env:             envPayload,
					ConfigProvided:  configProvided,
					Config:          configPayload,
				})
				if updateErr != nil {
					return respond(skillsToolResult{
						Ok:        false,
						Action:    action,
						Skill:     skill,
						Error:     updateErr.Error(),
						ErrorCode: skillsToolErrorNotImplemented,
					})
				}
				updatedEntry = entry
			}

			resolvedStatus, statusErr := loadStatus()
			if statusErr != nil {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Skill:     skill,
					Error:     statusErr.Error(),
					ErrorCode: skillsToolErrorNotImplemented,
				})
			}
			catalog, catalogErr := loadCatalog(resolvedStatus.WorkspaceRoot)
			if catalogErr != nil {
				return respond(skillsToolResult{
					Ok:        false,
					Action:    action,
					Skill:     skill,
					Error:     catalogErr.Error(),
					ErrorCode: skillsToolErrorNotImplemented,
				})
			}
			return respond(skillsToolResult{
				Ok:           true,
				Action:       action,
				Skill:        skill,
				Data:         map[string]any{"updated": true, "enabled": enabled, "entry": updatedEntry, "catalog": catalog},
				ClawhubReady: resolvedStatus.ClawhubReady,
				Reason:       resolvedStatus.Reason,
			})
		default:
			return respond(skillsToolResult{
				Ok:        false,
				Action:    action,
				Skill:     skill,
				Error:     "unsupported action",
				ErrorCode: skillsToolErrorUnsupported,
			})
		}
	}
}

func canonicalSkillsAction(toolName string, action string) string {
	normalized := strings.ToLower(strings.TrimSpace(action))
	if normalized == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(toolName)) {
	case "skills":
		switch normalized {
		case "status":
			return "status"
		case "bins":
			return "bins"
		case "install":
			return "install_deps"
		case "update":
			return "update_config"
		default:
			return ""
		}
	case "skill_manage", "skills_manage":
		switch normalized {
		case "list":
			return "catalog"
		case "search":
			return "search_packages"
		case "install":
			return "install_package"
		case "update":
			return "update_package"
		case "remove":
			return "remove_package"
		case "sync":
			return "sync_packages"
		default:
			return ""
		}
	default:
		return ""
	}
}

func normalizeSkillsActionKey(toolName string, action string) string {
	normalized := strings.ToLower(strings.TrimSpace(action))
	if normalized == "" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(toolName)) {
	case "skills":
		switch normalized {
		case "status":
			return "skills.status"
		case "bins":
			return "skills.bins"
		case "install_deps":
			return "skills.install"
		case "update_config":
			return "skills.update"
		default:
			return normalized
		}
	case "skill_manage", "skills_manage":
		switch normalized {
		case "catalog":
			return "skills_manage.list"
		case "search_packages":
			return "skills_manage.search"
		case "install_package":
			return "skills_manage.install"
		case "update_package":
			return "skills_manage.update"
		case "remove_package":
			return "skills_manage.remove"
		case "sync_packages":
			return "skills_manage.sync"
		default:
			return normalized
		}
	default:
		return normalized
	}
}

func validateSkillsToolArgs(toolName string, payload toolArgs) error {
	if len(payload) == 0 {
		return nil
	}
	allowed := allowedSkillsToolArgs
	if strings.EqualFold(strings.TrimSpace(toolName), "skill_manage") || strings.EqualFold(strings.TrimSpace(toolName), "skills_manage") {
		allowed = allowedSkillManageToolArgs
	}
	unknown := make([]string, 0)
	for key := range payload {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if _, ok := allowed[normalized]; ok {
			continue
		}
		unknown = append(unknown, key)
	}
	if len(unknown) == 0 {
		return nil
	}
	sort.Strings(unknown)
	return fmt.Errorf("unsupported fields: %s", strings.Join(unknown, ","))
}

func buildSkillsStatusPayload(status skillsdto.SkillsStatus, catalog []skillsdto.ProviderSkillSpec) map[string]any {
	disabled := make([]string, 0)
	enabledCount := 0
	for _, item := range catalog {
		if item.Enabled {
			enabledCount++
			continue
		}
		disabled = append(disabled, item.ID)
	}
	eligible := status.ClawhubReady && enabledCount > 0
	missing := make([]string, 0)
	installHints := make([]string, 0)
	if !status.ClawhubReady {
		missing = append(missing, clawhubUnavailableReason)
		installHints = append(installHints, "install or configure clawhub in external tools")
	}
	return map[string]any{
		"status":             status,
		"eligible":           eligible,
		"catalogCount":       len(catalog),
		"enabledCount":       enabledCount,
		"disabled":           disabled,
		"missing":            missing,
		"configChecks":       []string{},
		"installHints":       installHints,
		"blockedByAllowlist": []string{},
	}
}

func extractRuntimeBins(runtime *skillsdto.SkillRuntimeRequirements) ([]string, []string) {
	if runtime == nil {
		return nil, nil
	}
	return uniqueStrings(runtime.Bins), uniqueStrings(runtime.AnyBins)
}

func resolveInstallHints(runtime *skillsdto.SkillRuntimeRequirements) []skillsdto.SkillRuntimeInstallSpec {
	if runtime == nil {
		return nil
	}
	return runtime.Install
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		key := strings.ToLower(normalized)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, normalized)
	}
	sort.Strings(result)
	if len(result) == 0 {
		return nil
	}
	return result
}

func hasAnyArg(args toolArgs, keys ...string) bool {
	for _, key := range keys {
		if key == "" {
			continue
		}
		if _, ok := args[key]; ok {
			return true
		}
	}
	return false
}

func toSkillsErrorResult(action string, skill string, err error, status skillsdto.SkillsStatus) skillsToolResult {
	if errors.Is(err, skillsservice.ErrClawHubUnavailable) {
		return clawhubUnavailableSkillsResult(action, skill)
	}
	result := skillsToolResult{
		Ok:           false,
		Action:       action,
		Skill:        skill,
		Error:        err.Error(),
		ErrorCode:    skillsToolErrorNotImplemented,
		ClawhubReady: status.ClawhubReady,
		Reason:       status.Reason,
	}
	if detail, ok := skillsservice.ExtractClawHubErrorDetail(err); ok {
		result.ErrorCode = detail.Code
		result.Hint = detail.Hint
		result.RequiresForce = detail.Code == skillsservice.ClawHubErrorCodeRequireForce
	}
	return result
}

func resolveSkillManageInstallPolicyError(
	action string,
	skill string,
	err error,
	policy skillsInstallPolicy,
	status skillsdto.SkillsStatus,
) (skillsToolResult, bool) {
	if !isSkillRequireForceError(err) {
		return skillsToolResult{}, false
	}
	data := map[string]any{
		"scannerMode":       policy.ScannerMode,
		"allowForceInstall": policy.AllowForceInstall,
		"requireApproval":   policy.RequireApproval,
	}
	if policy.ScannerMode == "block" || !policy.AllowForceInstall {
		return skillsToolResult{
			Ok:            false,
			Action:        action,
			Skill:         skill,
			Error:         "suspicious skill installation is blocked by skills security policy",
			ErrorCode:     skillsToolErrorPolicyDenied,
			RequiresForce: true,
			Data:          data,
			ClawhubReady:  status.ClawhubReady,
			Reason:        status.Reason,
		}, true
	}
	if policy.RequireApproval {
		return skillsToolResult{
			Ok:            false,
			Action:        action,
			Skill:         skill,
			Error:         "suspicious skill installation requires approval by skills security policy",
			ErrorCode:     skillsToolErrorApprovalRequired,
			RequiresForce: true,
			Data:          data,
			ClawhubReady:  status.ClawhubReady,
			Reason:        status.Reason,
		}, true
	}
	return skillsToolResult{}, false
}

func isSkillRequireForceError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, skillsservice.ErrClawHubRequireForce) {
		return true
	}
	if detail, ok := skillsservice.ExtractClawHubErrorDetail(err); ok {
		return detail.Code == skillsservice.ClawHubErrorCodeRequireForce
	}
	return false
}

func setSkillEnabled(ctx context.Context, skills *skillsservice.SkillsService, skill string, enabled bool) error {
	skill = strings.TrimSpace(skill)
	if skill == "" {
		return errors.New("skill is required")
	}
	err := skills.EnableSkill(ctx, skillsdto.EnableSkillRequest{
		ID:      skill,
		Enabled: enabled,
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, domainskills.ErrSkillNotFound) {
		return err
	}
	_, registerErr := skills.RegisterSkill(ctx, skillsdto.RegisterSkillRequest{
		Spec: skillsdto.ProviderSkillSpec{
			ID:         skill,
			ProviderID: "workspace",
			Name:       skill,
			Enabled:    enabled,
		},
	})
	return registerErr
}

func resolveAssistantID(ctx context.Context, assistants *assistantservice.AssistantService, requested string) (string, error) {
	if id := strings.TrimSpace(requested); id != "" {
		return id, nil
	}
	if assistants == nil {
		return "", nil
	}
	items, err := assistants.ListAssistants(ctx, true)
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item.IsDefault {
			return item.ID, nil
		}
	}
	for _, item := range items {
		if item.Enabled {
			return item.ID, nil
		}
	}
	if len(items) > 0 {
		return items[0].ID, nil
	}
	return "", nil
}

func clawhubUnavailableSkillsResult(action string, skill string) skillsToolResult {
	return skillsToolResult{
		Ok:           false,
		Action:       action,
		Skill:        skill,
		Error:        clawhubUnavailableReason,
		ErrorCode:    clawhubUnavailableReason,
		ClawhubReady: false,
		Reason:       clawhubUnavailableReason,
	}
}

func marshalSkillsResult(result skillsToolResult) string {
	return marshalResult(result)
}

func resolveSkillsActionGroup(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "skills.status", "skills.bins", "skills_manage.search", "skills_manage.list":
		return "read"
	case "skills_manage.install", "skills_manage.update", "skills_manage.remove", "skills_manage.sync":
		return "package_write"
	case "skills.install":
		return "deps_write"
	case "skills.update":
		return "config_write"
	case "source_write":
		return "source_write"
	default:
		return "read"
	}
}

func normalizeSkillsActionMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "allow", "on", "enabled", "true":
		return "allow"
	case "ask", "approval", "prompt":
		return "ask"
	case "deny", "off", "disabled", "false":
		return "deny"
	default:
		return ""
	}
}

func resolveSkillsActionMode(ctx context.Context, settings SettingsReader, action string, actionGroup string) string {
	config := resolveSkillsToolConfig(ctx, settings)
	if config == nil {
		return "allow"
	}
	actionModes := getNestedMap(config, "security", "actionModes")
	if actionModes == nil {
		return "allow"
	}
	trimmedAction := strings.TrimSpace(action)
	lowerAction := strings.ToLower(trimmedAction)
	keys := []string{trimmedAction, lowerAction}
	if dot := strings.Index(lowerAction, "."); dot > 0 && dot+1 < len(lowerAction) {
		unqualified := strings.TrimSpace(lowerAction[dot+1:])
		if unqualified != "" {
			keys = append(keys, unqualified)
		}
	}
	keys = append(keys, strings.TrimSpace(actionGroup), strings.ToLower(strings.TrimSpace(actionGroup)))
	for _, key := range keys {
		if key == "" {
			continue
		}
		if raw, ok := actionModes[key]; ok {
			if value, ok := raw.(string); ok {
				if mode := normalizeSkillsActionMode(value); mode != "" {
					return mode
				}
			}
		}
	}
	return "allow"
}

func resolveSkillsToolConfig(ctx context.Context, settings SettingsReader) map[string]any {
	toolsConfig := resolveToolsConfig(ctx, settings)
	if toolsConfig == nil {
		return nil
	}
	if skills, ok := toolsConfig["skills"].(map[string]any); ok {
		return cloneAnyMap(skills)
	}
	return nil
}

func appendSkillsAuditRecord(ctx context.Context, settings SettingsReader, record map[string]any) {
	if settings == nil || len(record) == 0 {
		return
	}
	_, _ = mutateSkillsToolSettings(ctx, settings, func(skillsConfig map[string]any) error {
		auditEntries := make([]any, 0)
		switch typed := skillsConfig["audit"].(type) {
		case []any:
			auditEntries = append(auditEntries, typed...)
		case []map[string]any:
			for _, item := range typed {
				auditEntries = append(auditEntries, item)
			}
		}
		now := time.Now().UTC()
		auditEntries = pruneSkillsAuditEntriesByRetention(
			auditEntries,
			resolveSkillsAuditRetentionDays(skillsConfig),
			now,
		)
		record["timestamp"] = now.Format(time.RFC3339)
		auditEntries = append(auditEntries, record)
		maxEntries := resolveSkillsAuditMaxEntries(skillsConfig)
		if maxEntries > 0 && len(auditEntries) > maxEntries {
			auditEntries = append([]any(nil), auditEntries[len(auditEntries)-maxEntries:]...)
		}
		skillsConfig["audit"] = auditEntries
		return nil
	})
}

func resolveSkillsAuditMaxEntries(skillsConfig map[string]any) int {
	if value, ok := getNestedInt(skillsConfig, "security", "audit", "maxEntries"); ok && value > 0 {
		return value
	}
	if value, ok := getNestedInt(skillsConfig, "audit", "maxEntries"); ok && value > 0 {
		return value
	}
	return defaultSkillsAuditMaxEntries
}

func resolveSkillsAuditRetentionDays(skillsConfig map[string]any) int {
	if value, ok := getNestedInt(skillsConfig, "auditConfig", "retentionDays"); ok {
		return normalizeSkillsAuditRetentionDays(value)
	}
	if value, ok := getNestedInt(skillsConfig, "audit", "retentionDays"); ok {
		return normalizeSkillsAuditRetentionDays(value)
	}
	return defaultSkillsAuditRetentionDays
}

func normalizeSkillsAuditRetentionDays(value int) int {
	if value <= 0 {
		return defaultSkillsAuditRetentionDays
	}
	if value > maxSkillsAuditRetentionDays {
		return maxSkillsAuditRetentionDays
	}
	return value
}

func pruneSkillsAuditEntriesByRetention(entries []any, retentionDays int, now time.Time) []any {
	if retentionDays <= 0 || len(entries) == 0 {
		return entries
	}
	cutoff := now.AddDate(0, 0, -retentionDays)
	pruned := make([]any, 0, len(entries))
	for _, entry := range entries {
		record, ok := entry.(map[string]any)
		if !ok {
			pruned = append(pruned, entry)
			continue
		}
		rawTimestamp, _ := record["timestamp"].(string)
		timestamp, err := time.Parse(time.RFC3339, strings.TrimSpace(rawTimestamp))
		if err == nil && timestamp.Before(cutoff) {
			continue
		}
		pruned = append(pruned, entry)
	}
	return pruned
}

type skillConfigPatchInput struct {
	EnabledProvided bool
	Enabled         bool
	APIKeyProvided  bool
	APIKey          string
	EnvProvided     bool
	Env             map[string]any
	ConfigProvided  bool
	Config          map[string]any
}

func updateSkillConfigEntry(ctx context.Context, settings SettingsReader, skill string, patch skillConfigPatchInput) (map[string]any, error) {
	skillKey := strings.ToLower(strings.TrimSpace(skill))
	if skillKey == "" {
		return nil, errors.New("skill is required")
	}
	return mutateSkillsToolSettings(ctx, settings, func(skillsConfig map[string]any) error {
		entries := map[string]any{}
		if existing, ok := skillsConfig["entries"].(map[string]any); ok {
			entries = cloneAnyMap(existing)
		}
		entry := map[string]any{}
		if existing, ok := entries[skillKey].(map[string]any); ok {
			entry = cloneAnyMap(existing)
		}
		if patch.EnabledProvided {
			entry["enabled"] = patch.Enabled
		}
		if patch.APIKeyProvided {
			if strings.TrimSpace(patch.APIKey) == "" {
				delete(entry, "apiKey")
			} else {
				entry["apiKey"] = strings.TrimSpace(patch.APIKey)
			}
		}
		if patch.EnvProvided {
			normalizedEnv := normalizeStringAnyMap(patch.Env)
			if len(normalizedEnv) == 0 {
				delete(entry, "env")
			} else {
				entry["env"] = normalizedEnv
			}
		}
		if patch.ConfigProvided {
			normalizedConfig := cloneAnyMap(patch.Config)
			if len(normalizedConfig) == 0 {
				delete(entry, "config")
			} else {
				entry["config"] = normalizedConfig
			}
		}
		if len(entry) == 0 {
			delete(entries, skillKey)
		} else {
			entry["updatedAt"] = time.Now().UTC().Format(time.RFC3339)
			entries[skillKey] = entry
		}
		if len(entries) == 0 {
			delete(skillsConfig, "entries")
		} else {
			skillsConfig["entries"] = entries
		}
		return nil
	})
}

func normalizeStringAnyMap(raw map[string]any) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	result := make(map[string]string, len(raw))
	for key, value := range raw {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		text := strings.TrimSpace(toString(value))
		if text == "" {
			continue
		}
		result[normalizedKey] = text
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func mutateSkillsToolSettings(ctx context.Context, settings SettingsReader, mutate func(skillsConfig map[string]any) error) (map[string]any, error) {
	updater, ok := settings.(skillsSettingsUpdater)
	if !ok {
		return nil, errors.New("settings update unavailable")
	}
	current, err := updater.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	toolsConfig := cloneAnyMap(current.Tools)
	if toolsConfig == nil {
		toolsConfig = map[string]any{}
	}
	skillsConfig := cloneAnyMap(current.Skills)
	if skillsConfig == nil {
		skillsConfig = map[string]any{}
	}
	delete(toolsConfig, "skills")
	if err := mutate(skillsConfig); err != nil {
		return nil, err
	}

	updated, err := updater.UpdateSettings(ctx, settingsdto.UpdateSettingsRequest{
		Tools:  toolsConfig,
		Skills: skillsConfig,
	})
	if err != nil {
		return nil, err
	}
	if applier, ok := settings.(skillsSettingsApplier); ok {
		applier.ApplySettings(updated)
	}
	return skillsConfig, nil
}

func resolveSkillsInstallPolicy(ctx context.Context, settings SettingsReader) skillsInstallPolicy {
	policy := skillsInstallPolicy{
		ScannerMode:       "warn",
		AllowForceInstall: true,
		RequireApproval:   false,
	}
	config := resolveSkillsToolConfig(ctx, settings)
	if config == nil {
		return policy
	}
	install := getNestedMap(config, "security", "install")
	if install == nil {
		return policy
	}
	if mode := strings.ToLower(strings.TrimSpace(toString(install["scannerMode"]))); mode != "" {
		switch mode {
		case "off", "warn", "block":
			policy.ScannerMode = mode
		}
	}
	if value, ok := install["allowForceInstall"].(bool); ok {
		policy.AllowForceInstall = value
	}
	if value, ok := install["requireApproval"].(bool); ok {
		policy.RequireApproval = value
	}
	return policy
}

func resolveSkillsDepsPolicy(ctx context.Context, settings SettingsReader) skillsDepsPolicy {
	policy := skillsDepsPolicy{
		AllowExternalToolsOnly: false,
		AllowGenericInstaller:  true,
		AllowedBins:            nil,
	}
	config := resolveSkillsToolConfig(ctx, settings)
	if config == nil {
		return policy
	}
	deps := getNestedMap(config, "security", "deps")
	if deps == nil {
		return policy
	}
	if value, ok := getNestedBool(config, "security", "deps", "allowExternalToolsOnly"); ok {
		policy.AllowExternalToolsOnly = value
	}
	if value, ok := getNestedBool(config, "security", "deps", "allowGenericInstaller"); ok {
		policy.AllowGenericInstaller = value
	}
	if policy.AllowExternalToolsOnly {
		policy.AllowGenericInstaller = false
	}
	if raw, ok := deps["allowedBins"]; ok {
		allowed := map[string]struct{}{}
		for _, item := range normalizeStringSlice(toStringSlice(raw)) {
			allowed[strings.ToLower(item)] = struct{}{}
		}
		if len(allowed) > 0 {
			policy.AllowedBins = allowed
		}
	}
	return policy
}

func toStringSlice(raw any) []string {
	switch typed := raw.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(toString(item))
			if text == "" {
				continue
			}
			result = append(result, text)
		}
		return result
	default:
		return nil
	}
}

func (policy skillsDepsPolicy) allowsBin(bin string) bool {
	if len(policy.AllowedBins) == 0 {
		return true
	}
	_, ok := policy.AllowedBins[strings.ToLower(strings.TrimSpace(bin))]
	return ok
}

func installSkillDependencies(
	ctx context.Context,
	runtimeRequirements *skillsdto.SkillRuntimeRequirements,
	installHints []skillsdto.SkillRuntimeInstallSpec,
	timeoutMs int,
	settings SettingsReader,
	externalTools skillsExternalToolInstaller,
) (skillsDepsInstallResult, error) {
	result := skillsDepsInstallResult{
		HandledBy:   "none",
		Actions:     make([]map[string]any, 0),
		SuccessBins: nil,
		FailedBins:  nil,
	}
	if runtimeRequirements == nil {
		return result, nil
	}

	depsPolicy := resolveSkillsDepsPolicy(ctx, settings)
	bins := uniqueStrings(runtimeRequirements.Bins)
	anyBins := uniqueStrings(runtimeRequirements.AnyBins)
	if len(bins) == 0 && len(anyBins) > 0 {
		selected := ""
		for _, candidate := range anyBins {
			if skillsBinaryExists(candidate) {
				selected = candidate
				break
			}
		}
		if selected == "" {
			selected = anyBins[0]
		}
		bins = []string{selected}
	}
	for _, spec := range installHints {
		for _, bin := range normalizeStringSlice(spec.Bins) {
			bins = append(bins, bin)
		}
	}
	bins = uniqueStrings(bins)
	if len(bins) == 0 {
		return result, nil
	}

	for _, bin := range bins {
		normalizedBin := strings.TrimSpace(bin)
		if normalizedBin == "" {
			continue
		}
		if skillsBinaryExists(normalizedBin) {
			result.SuccessBins = append(result.SuccessBins, normalizedBin)
			result.Actions = append(result.Actions, map[string]any{
				"bin":       normalizedBin,
				"status":    "already_available",
				"handledBy": "none",
			})
			continue
		}
		if !depsPolicy.allowsBin(normalizedBin) {
			result.FailedBins = append(result.FailedBins, normalizedBin)
			result.Actions = append(result.Actions, map[string]any{
				"bin":       normalizedBin,
				"status":    "blocked",
				"handledBy": "policy",
				"error":     "bin is not allowed by skills deps policy",
			})
			continue
		}

		if toolName, ok := skillsExternalToolByBin[strings.ToLower(normalizedBin)]; ok && externalTools != nil {
			result.HandledBy = mergeHandledBy(result.HandledBy, "external-tools")
			ready, reason, readinessErr := externalTools.ToolReadiness(ctx, toolName)
			if readinessErr != nil {
				result.FailedBins = append(result.FailedBins, normalizedBin)
				result.Actions = append(result.Actions, map[string]any{
					"bin":       normalizedBin,
					"status":    "error",
					"handledBy": "external-tools",
					"tool":      string(toolName),
					"error":     readinessErr.Error(),
				})
				continue
			}
			if !ready {
				_, installErr := externalTools.InstallTool(ctx, externaltoolsdto.InstallExternalToolRequest{Name: string(toolName)})
				if installErr != nil {
					result.FailedBins = append(result.FailedBins, normalizedBin)
					result.Actions = append(result.Actions, map[string]any{
						"bin":       normalizedBin,
						"status":    "error",
						"handledBy": "external-tools",
						"tool":      string(toolName),
						"error":     installErr.Error(),
					})
					continue
				}
				ready, reason, readinessErr = externalTools.ToolReadiness(ctx, toolName)
				if readinessErr != nil || !ready {
					if readinessErr != nil {
						reason = readinessErr.Error()
					}
					if strings.TrimSpace(reason) == "" {
						reason = "external tool is still not ready after install"
					}
					result.FailedBins = append(result.FailedBins, normalizedBin)
					result.Actions = append(result.Actions, map[string]any{
						"bin":       normalizedBin,
						"status":    "error",
						"handledBy": "external-tools",
						"tool":      string(toolName),
						"error":     reason,
					})
					continue
				}
			}
			result.SuccessBins = append(result.SuccessBins, normalizedBin)
			result.Actions = append(result.Actions, map[string]any{
				"bin":       normalizedBin,
				"status":    "installed",
				"handledBy": "external-tools",
				"tool":      string(toolName),
			})
			continue
		}

		if depsPolicy.AllowExternalToolsOnly {
			result.FailedBins = append(result.FailedBins, normalizedBin)
			result.Actions = append(result.Actions, map[string]any{
				"bin":       normalizedBin,
				"status":    "blocked",
				"handledBy": "policy",
				"error":     "generic installer is disabled by policy",
			})
			continue
		}
		if !depsPolicy.AllowGenericInstaller {
			result.FailedBins = append(result.FailedBins, normalizedBin)
			result.Actions = append(result.Actions, map[string]any{
				"bin":       normalizedBin,
				"status":    "blocked",
				"handledBy": "policy",
				"error":     "generic installer is disabled by policy",
			})
			continue
		}

		spec, specFound := resolveInstallSpecForBin(installHints, normalizedBin)
		if !specFound {
			result.FailedBins = append(result.FailedBins, normalizedBin)
			result.Actions = append(result.Actions, map[string]any{
				"bin":       normalizedBin,
				"status":    "error",
				"handledBy": "generic",
				"error":     "no install hint found for bin",
			})
			continue
		}
		kind, commands, commandErr := buildSkillsInstallCommands(spec, normalizedBin)
		if commandErr != nil {
			result.FailedBins = append(result.FailedBins, normalizedBin)
			result.Actions = append(result.Actions, map[string]any{
				"bin":       normalizedBin,
				"status":    "error",
				"handledBy": "generic",
				"kind":      kind,
				"error":     commandErr.Error(),
			})
			continue
		}
		result.HandledBy = mergeHandledBy(result.HandledBy, "generic")
		failed := false
		for _, command := range commands {
			exitCode, stdout, stderr, runErr := runSkillsDepsCommand(ctx, timeoutMs, command)
			action := map[string]any{
				"bin":       normalizedBin,
				"kind":      kind,
				"command":   command,
				"handledBy": "generic",
			}
			if runErr != nil {
				action["status"] = "error"
				action["error"] = runErr.Error()
				if strings.TrimSpace(stdout) != "" {
					action["stdout"] = stdout
				}
				if strings.TrimSpace(stderr) != "" {
					action["stderr"] = stderr
				}
				result.Actions = append(result.Actions, action)
				failed = true
				break
			}
			if exitCode != 0 {
				action["status"] = "error"
				action["error"] = fmt.Sprintf("command exited with code %d", exitCode)
				if strings.TrimSpace(stdout) != "" {
					action["stdout"] = stdout
				}
				if strings.TrimSpace(stderr) != "" {
					action["stderr"] = stderr
				}
				result.Actions = append(result.Actions, action)
				failed = true
				break
			}
			action["status"] = "ok"
			result.Actions = append(result.Actions, action)
		}
		if failed || !skillsBinaryExists(normalizedBin) {
			result.FailedBins = append(result.FailedBins, normalizedBin)
			if !failed {
				result.Actions = append(result.Actions, map[string]any{
					"bin":       normalizedBin,
					"handledBy": "generic",
					"status":    "error",
					"error":     "bin still unavailable after installation",
				})
			}
			continue
		}
		result.SuccessBins = append(result.SuccessBins, normalizedBin)
	}

	result.SuccessBins = uniqueStrings(result.SuccessBins)
	result.FailedBins = uniqueStrings(result.FailedBins)
	if len(result.Actions) == 0 {
		result.Actions = nil
	}
	if len(result.SuccessBins) == 0 {
		result.SuccessBins = nil
	}
	if len(result.FailedBins) == 0 {
		result.FailedBins = nil
	}
	if result.HandledBy == "none" {
		if len(result.SuccessBins) > 0 || len(result.FailedBins) > 0 {
			result.HandledBy = "mixed"
		}
	}
	return result, nil
}

func runSkillsDepsCommand(ctx context.Context, timeoutMs int, command []string) (int, string, string, error) {
	if len(command) == 0 || strings.TrimSpace(command[0]) == "" {
		return -1, "", "", errors.New("empty install command")
	}
	runCtx := ctx
	cancel := func() {}
	if timeoutMs > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	}
	defer cancel()
	return skillsDepsCommandRunner(runCtx, "", command[0], command[1:], "", nil, skillsDepsCommandOutputLimit)
}

func resolveInstallSpecForBin(specs []skillsdto.SkillRuntimeInstallSpec, bin string) (skillsdto.SkillRuntimeInstallSpec, bool) {
	normalizedBin := strings.ToLower(strings.TrimSpace(bin))
	if normalizedBin == "" {
		return skillsdto.SkillRuntimeInstallSpec{}, false
	}
	for _, spec := range specs {
		for _, candidate := range normalizeStringSlice(spec.Bins) {
			if strings.ToLower(strings.TrimSpace(candidate)) == normalizedBin {
				return spec, true
			}
		}
	}
	if len(specs) == 1 {
		return specs[0], true
	}
	for _, spec := range specs {
		if strings.TrimSpace(spec.ID) == "" && strings.TrimSpace(spec.Package) == "" && strings.TrimSpace(spec.Formula) == "" && strings.TrimSpace(spec.Module) == "" {
			continue
		}
		return spec, true
	}
	return skillsdto.SkillRuntimeInstallSpec{}, false
}

func buildSkillsInstallCommands(spec skillsdto.SkillRuntimeInstallSpec, bin string) (string, [][]string, error) {
	kind := strings.ToLower(strings.TrimSpace(spec.Kind))
	if kind == "" {
		switch {
		case strings.TrimSpace(spec.Formula) != "":
			kind = "brew"
		case strings.TrimSpace(spec.Module) != "":
			kind = "go"
		case strings.TrimSpace(spec.Package) != "":
			kind = "npm"
		default:
			kind = ""
		}
	}
	switch kind {
	case "brew":
		formula := firstNonEmpty(strings.TrimSpace(spec.Formula), strings.TrimSpace(spec.Package), strings.TrimSpace(spec.ID), strings.TrimSpace(bin))
		if formula == "" {
			return kind, nil, errors.New("brew formula is required")
		}
		commands := make([][]string, 0, 2)
		if tap := strings.TrimSpace(spec.Tap); tap != "" {
			commands = append(commands, []string{"brew", "tap", tap})
		}
		commands = append(commands, []string{"brew", "install", formula})
		return kind, commands, nil
	case "npm":
		pkg := firstNonEmpty(strings.TrimSpace(spec.Package), strings.TrimSpace(spec.Module), strings.TrimSpace(spec.ID), strings.TrimSpace(bin))
		if pkg == "" {
			return kind, nil, errors.New("npm package is required")
		}
		return kind, [][]string{{"npm", "install", "-g", pkg}}, nil
	case "go":
		module := firstNonEmpty(strings.TrimSpace(spec.Module), strings.TrimSpace(spec.Package), strings.TrimSpace(spec.ID))
		if module == "" {
			return kind, nil, errors.New("go module is required")
		}
		if !strings.Contains(module, "@") {
			module += "@latest"
		}
		return kind, [][]string{{"go", "install", module}}, nil
	case "uv":
		pkg := firstNonEmpty(strings.TrimSpace(spec.Package), strings.TrimSpace(spec.Module), strings.TrimSpace(spec.ID), strings.TrimSpace(bin))
		if pkg == "" {
			return kind, nil, errors.New("uv package is required")
		}
		return kind, [][]string{{"uv", "tool", "install", pkg}}, nil
	case "pip", "pipx":
		pkg := firstNonEmpty(strings.TrimSpace(spec.Package), strings.TrimSpace(spec.Module), strings.TrimSpace(spec.ID), strings.TrimSpace(bin))
		if pkg == "" {
			return kind, nil, errors.New("pip package is required")
		}
		if kind == "pipx" {
			return kind, [][]string{{"pipx", "install", pkg}}, nil
		}
		return kind, [][]string{{"python3", "-m", "pip", "install", "-U", pkg}}, nil
	default:
		return kind, nil, fmt.Errorf("unsupported install kind: %s", kind)
	}
}

func mergeHandledBy(current string, next string) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if next == "" {
		return current
	}
	if current == "" || current == "none" {
		return next
	}
	if current == next {
		return current
	}
	return "mixed"
}

func skillsBinaryExists(bin string) bool {
	trimmed := strings.TrimSpace(bin)
	if trimmed == "" {
		return false
	}
	_, err := exec.LookPath(trimmed)
	return err == nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		return trimmed
	}
	return ""
}
