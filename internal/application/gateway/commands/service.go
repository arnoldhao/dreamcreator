package commands

import (
	"context"
	"errors"
	"strings"

	appcommands "dreamcreator/internal/application/commands"
	settingsservice "dreamcreator/internal/application/settings/service"
)

type Service struct {
	settings *settingsservice.SettingsService
}

type ListRequest struct {
	Provider        string `json:"provider,omitempty"`
	IncludeDisabled bool   `json:"includeDisabled,omitempty"`
}

func NewService(settings *settingsservice.SettingsService) *Service {
	return &Service{settings: settings}
}

func (service *Service) List(ctx context.Context, request ListRequest) ([]appcommands.CommandDescriptor, error) {
	if service == nil || service.settings == nil {
		return nil, errors.New("settings service unavailable")
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	items := appcommands.ListCommandDescriptors(current, request.Provider)
	if request.IncludeDisabled {
		return items, nil
	}
	filtered := make([]appcommands.CommandDescriptor, 0, len(items))
	for _, item := range items {
		if item.Enabled {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (service *Service) Run(ctx context.Context, request appcommands.CommandRunRequest) (appcommands.CommandRunResponse, error) {
	if service == nil || service.settings == nil {
		return appcommands.CommandRunResponse{}, errors.New("settings service unavailable")
	}
	key := strings.TrimSpace(request.Key)
	if key == "" {
		return appcommands.CommandRunResponse{}, errors.New("command key required")
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return appcommands.CommandRunResponse{}, err
	}
	enabled := appcommands.ListEnabledCommands(current, "")
	found := false
	for _, command := range enabled {
		if strings.EqualFold(command.Key, key) {
			found = true
			break
		}
	}
	if !found {
		return appcommands.CommandRunResponse{}, errors.New("command not available")
	}
	if key == "help" || key == "commands" {
		return appcommands.CommandRunResponse{
			Success: true,
			Data: map[string]any{
				"commands": enabled,
			},
		}, nil
	}
	return appcommands.CommandRunResponse{
		Success: false,
		Message: "command execution not implemented",
	}, nil
}
