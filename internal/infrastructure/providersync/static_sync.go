package providersync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dreamcreator/internal/domain/providers"
)

type StaticSyncer struct {
	now func() time.Time
}

func NewStaticSyncer() *StaticSyncer {
	return &StaticSyncer{now: time.Now}
}

func (syncer *StaticSyncer) Sync(_ context.Context, provider providers.Provider, _ string) ([]providers.Model, error) {
	var models []string
	switch strings.ToLower(strings.TrimSpace(provider.Name)) {
	case "deepseek":
		models = []string{"deepseek-chat", "deepseek-reasoner"}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider.Name)
	}

	now := syncer.now()
	result := make([]providers.Model, 0, len(models))
	for _, name := range models {
		modelID := buildModelID(provider.ID, name)
		model, err := providers.NewModel(providers.ModelParams{
			ID:         modelID,
			ProviderID: provider.ID,
			Name:       name,
			Enabled:    false,
			ShowInUI:   false,
			UpdatedAt:  &now,
		})
		if err != nil {
			return nil, err
		}
		result = append(result, model)
	}
	return result, nil
}

func buildModelID(providerID, modelName string) string {
	trimmed := strings.TrimSpace(modelName)
	if trimmed == "" {
		return providerID
	}
	return fmt.Sprintf("%s:%s", providerID, trimmed)
}
