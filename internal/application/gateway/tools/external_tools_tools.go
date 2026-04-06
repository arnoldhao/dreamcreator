package tools

import (
	"context"
	"strings"

	externaltoolsdto "dreamcreator/internal/application/externaltools/dto"
	externaltoolsservice "dreamcreator/internal/application/externaltools/service"
	"dreamcreator/internal/domain/externaltools"
)

type externalToolsToolResult struct {
	Ok     bool   `json:"ok"`
	Action string `json:"action,omitempty"`
	Tool   string `json:"tool,omitempty"`
	Data   any    `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

type externalToolWithReadiness struct {
	externaltoolsdto.ExternalTool
	Ready  bool   `json:"ready"`
	Reason string `json:"reason,omitempty"`
}

func runExternalToolsQueryTool(service *externaltoolsservice.ExternalToolsService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if service == nil {
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     false,
				Action: "list",
				Error:  "external tools service unavailable",
			}), nil
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:    false,
				Error: err.Error(),
			}), nil
		}
		action := strings.ToLower(strings.TrimSpace(getStringArg(payload, "action", "type")))
		if action == "" {
			action = "list"
		}
		toolName := strings.TrimSpace(getStringArg(payload, "name", "tool"))
		switch action {
		case "list":
			items, listErr := service.ListTools(ctx)
			if listErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Error:  listErr.Error(),
				}), nil
			}
			result := make([]externalToolWithReadiness, 0, len(items))
			for _, item := range items {
				ready, reason, readyErr := service.ToolReadiness(ctx, externaltools.ToolName(item.Name))
				if readyErr != nil {
					return marshalExternalToolsResult(externalToolsToolResult{
						Ok:     false,
						Action: action,
						Error:  readyErr.Error(),
					}), nil
				}
				result = append(result, externalToolWithReadiness{
					ExternalTool: item,
					Ready:        ready,
					Reason:       reason,
				})
			}
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     true,
				Action: action,
				Data:   result,
			}), nil
		case "status":
			if toolName == "" {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Error:  "name is required",
				}), nil
			}
			tool, findErr := findExternalToolStatus(ctx, service, toolName)
			if findErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  findErr.Error(),
				}), nil
			}
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     true,
				Action: action,
				Tool:   toolName,
				Data:   tool,
				Ready:  tool.Ready,
				Reason: tool.Reason,
			}), nil
		case "install_state":
			if toolName == "" {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Error:  "name is required",
				}), nil
			}
			state, stateErr := service.GetInstallState(ctx, externaltoolsdto.GetExternalToolInstallStateRequest{Name: toolName})
			if stateErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  stateErr.Error(),
				}), nil
			}
			ready, reason, readinessErr := service.ToolReadiness(ctx, externaltools.ToolName(toolName))
			if readinessErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  readinessErr.Error(),
				}), nil
			}
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     true,
				Action: action,
				Tool:   toolName,
				Data:   state,
				Ready:  ready,
				Reason: reason,
			}), nil
		case "updates":
			updates, updatesErr := service.ListToolUpdates(ctx)
			if updatesErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  updatesErr.Error(),
				}), nil
			}
			if toolName != "" {
				for _, item := range updates {
					if strings.EqualFold(strings.TrimSpace(item.Name), toolName) {
						ready, reason, readinessErr := service.ToolReadiness(ctx, externaltools.ToolName(toolName))
						if readinessErr != nil {
							return marshalExternalToolsResult(externalToolsToolResult{
								Ok:     false,
								Action: action,
								Tool:   toolName,
								Error:  readinessErr.Error(),
							}), nil
						}
						return marshalExternalToolsResult(externalToolsToolResult{
							Ok:     true,
							Action: action,
							Tool:   toolName,
							Data:   item,
							Ready:  ready,
							Reason: reason,
						}), nil
					}
				}
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  "tool not found",
				}), nil
			}
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     true,
				Action: action,
				Data:   updates,
			}), nil
		default:
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     false,
				Action: action,
				Tool:   toolName,
				Error:  "unknown action",
			}), nil
		}
	}
}

func runExternalToolsManageTool(service *externaltoolsservice.ExternalToolsService) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if service == nil {
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:    false,
				Error: "external tools service unavailable",
			}), nil
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:    false,
				Error: err.Error(),
			}), nil
		}
		action := strings.ToLower(strings.TrimSpace(getStringArg(payload, "action", "type")))
		toolName := strings.TrimSpace(getStringArg(payload, "name", "tool"))
		version := strings.TrimSpace(getStringArg(payload, "version"))
		manager := strings.TrimSpace(getStringArg(payload, "manager"))
		execPath := strings.TrimSpace(getStringArg(payload, "execPath", "exec_path"))
		if action == "" {
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:    false,
				Error: "action is required",
			}), nil
		}
		if toolName == "" {
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     false,
				Action: action,
				Error:  "name is required",
			}), nil
		}
		switch action {
		case "install":
			installed, installErr := service.InstallTool(ctx, externaltoolsdto.InstallExternalToolRequest{
				Name:    toolName,
				Version: version,
				Manager: manager,
			})
			if installErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  installErr.Error(),
				}), nil
			}
			return marshalExternalToolMutationResult(ctx, service, action, toolName, installed)
		case "verify":
			verified, verifyErr := service.VerifyTool(ctx, externaltoolsdto.VerifyExternalToolRequest{Name: toolName})
			if verifyErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  verifyErr.Error(),
				}), nil
			}
			return marshalExternalToolMutationResult(ctx, service, action, toolName, verified)
		case "reinstall":
			reinstallVersion := version
			if reinstallVersion == "" {
				if current, findErr := findExternalToolStatus(ctx, service, toolName); findErr == nil {
					reinstallVersion = strings.TrimSpace(current.Version)
				}
			}
			if removeErr := service.RemoveTool(ctx, externaltoolsdto.RemoveExternalToolRequest{Name: toolName}); removeErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  removeErr.Error(),
				}), nil
			}
			reinstalled, installErr := service.InstallTool(ctx, externaltoolsdto.InstallExternalToolRequest{
				Name:    toolName,
				Version: reinstallVersion,
				Manager: manager,
			})
			if installErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  installErr.Error(),
				}), nil
			}
			return marshalExternalToolMutationResult(ctx, service, action, toolName, reinstalled)
		case "remove":
			if removeErr := service.RemoveTool(ctx, externaltoolsdto.RemoveExternalToolRequest{Name: toolName}); removeErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  removeErr.Error(),
				}), nil
			}
			ready, reason, readinessErr := service.ToolReadiness(ctx, externaltools.ToolName(toolName))
			if readinessErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  readinessErr.Error(),
				}), nil
			}
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     true,
				Action: action,
				Tool:   toolName,
				Data:   map[string]any{"removed": true},
				Ready:  ready,
				Reason: reason,
			}), nil
		case "set_path":
			if execPath == "" {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  "execPath is required",
				}), nil
			}
			updated, setErr := service.SetToolPath(ctx, externaltoolsdto.SetExternalToolPathRequest{
				Name:     toolName,
				ExecPath: execPath,
			})
			if setErr != nil {
				return marshalExternalToolsResult(externalToolsToolResult{
					Ok:     false,
					Action: action,
					Tool:   toolName,
					Error:  setErr.Error(),
				}), nil
			}
			return marshalExternalToolMutationResult(ctx, service, action, toolName, updated)
		default:
			return marshalExternalToolsResult(externalToolsToolResult{
				Ok:     false,
				Action: action,
				Tool:   toolName,
				Error:  "unknown action",
			}), nil
		}
	}
}

func marshalExternalToolMutationResult(ctx context.Context, service *externaltoolsservice.ExternalToolsService, action string, toolName string, data any) (string, error) {
	ready, reason, readinessErr := service.ToolReadiness(ctx, externaltools.ToolName(toolName))
	if readinessErr != nil {
		return marshalExternalToolsResult(externalToolsToolResult{
			Ok:     false,
			Action: action,
			Tool:   toolName,
			Error:  readinessErr.Error(),
		}), nil
	}
	return marshalExternalToolsResult(externalToolsToolResult{
		Ok:     true,
		Action: action,
		Tool:   toolName,
		Data:   data,
		Ready:  ready,
		Reason: reason,
	}), nil
}

func marshalExternalToolsResult(result externalToolsToolResult) string {
	return marshalResult(result)
}

func findExternalToolStatus(ctx context.Context, service *externaltoolsservice.ExternalToolsService, name string) (externalToolWithReadiness, error) {
	items, err := service.ListTools(ctx)
	if err != nil {
		return externalToolWithReadiness{}, err
	}
	toolName := strings.TrimSpace(name)
	for _, item := range items {
		if !strings.EqualFold(strings.TrimSpace(item.Name), toolName) {
			continue
		}
		ready, reason, readyErr := service.ToolReadiness(ctx, externaltools.ToolName(item.Name))
		if readyErr != nil {
			return externalToolWithReadiness{}, readyErr
		}
		return externalToolWithReadiness{
			ExternalTool: item,
			Ready:        ready,
			Reason:       reason,
		}, nil
	}
	return externalToolWithReadiness{}, externaltools.ErrToolNotFound
}
