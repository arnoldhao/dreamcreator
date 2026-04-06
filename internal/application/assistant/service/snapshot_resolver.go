package service

import (
	"context"
	"errors"
	"strings"
	"time"

	assistantdto "dreamcreator/internal/application/assistant/dto"
	"dreamcreator/internal/domain/assistant"
	"dreamcreator/internal/domain/thread"
)

type AssistantSnapshotResolver struct {
	assistants assistant.Repository
	threads    thread.Repository
	now        func() time.Time
}

func NewAssistantSnapshotResolver(
	assistantRepo assistant.Repository,
	threadRepo thread.Repository,
) *AssistantSnapshotResolver {
	return &AssistantSnapshotResolver{
		assistants: assistantRepo,
		threads:    threadRepo,
		now:        time.Now,
	}
}

func (resolver *AssistantSnapshotResolver) ResolveAssistantSnapshot(ctx context.Context, request assistantdto.ResolveAssistantSnapshotRequest) (assistantdto.AssistantSnapshot, error) {
	threadID := strings.TrimSpace(request.ThreadID)
	assistantID := strings.TrimSpace(request.AssistantID)
	if assistantID == "" {
		if threadID == "" {
			return assistantdto.AssistantSnapshot{}, errors.New("assistant id or thread id is required")
		}
		loaded, err := resolver.threads.Get(ctx, threadID)
		if err != nil {
			if errors.Is(err, thread.ErrThreadNotFound) {
				resolved, err := resolver.resolveDefaultAssistant(ctx)
				if err != nil {
					return assistantdto.AssistantSnapshot{}, err
				}
				assistantID = resolved.ID
			} else {
				return assistantdto.AssistantSnapshot{}, err
			}
		} else {
			assistantID = strings.TrimSpace(loaded.AssistantID)
			if assistantID == "" {
				resolved, err := resolver.resolveDefaultAssistant(ctx)
				if err != nil {
					return assistantdto.AssistantSnapshot{}, err
				}
				assistantID = resolved.ID
				loaded.AssistantID = assistantID
				loaded.UpdatedAt = resolver.now()
				_ = resolver.threads.Save(ctx, loaded)
			}
		}
	}
	assistantItem, err := resolver.assistants.Get(ctx, assistantID)
	if err != nil {
		return assistantdto.AssistantSnapshot{}, err
	}

	return assistantdto.AssistantSnapshot{
		AssistantID: assistantItem.ID,
		Builtin:     assistantItem.Builtin,
		Deletable:   assistantItem.Deletable,
		Identity:    assistantItem.Identity,
		Avatar:      assistantItem.Avatar,
		User:        assistantItem.User,
		Model:       assistantItem.Model,
		Tools:       assistantItem.Tools,
		Skills:      assistantItem.Skills,
		Call:        assistantItem.Call,
		Memory:      assistantItem.Memory,
		Enabled:     assistantItem.Enabled,
		IsDefault:   assistantItem.IsDefault,
		CreatedAt:   assistantItem.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   assistantItem.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (resolver *AssistantSnapshotResolver) ResolveDefaultAssistantSnapshot(ctx context.Context) (assistantdto.AssistantSnapshot, error) {
	item, err := resolver.resolveDefaultAssistant(ctx)
	if err != nil {
		return assistantdto.AssistantSnapshot{}, err
	}
	return assistantdto.AssistantSnapshot{
		AssistantID: item.ID,
		Builtin:     item.Builtin,
		Deletable:   item.Deletable,
		Identity:    item.Identity,
		Avatar:      item.Avatar,
		User:        item.User,
		Model:       item.Model,
		Tools:       item.Tools,
		Skills:      item.Skills,
		Call:        item.Call,
		Memory:      item.Memory,
		Enabled:     item.Enabled,
		IsDefault:   item.IsDefault,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (resolver *AssistantSnapshotResolver) resolveDefaultAssistant(ctx context.Context) (assistant.Assistant, error) {
	items, err := resolver.assistants.List(ctx, true)
	if err != nil {
		return assistant.Assistant{}, err
	}
	if len(items) == 0 {
		return assistant.Assistant{}, assistant.ErrAssistantNotFound
	}
	for _, item := range items {
		if item.IsDefault {
			return item, nil
		}
	}
	return items[0], nil
}
