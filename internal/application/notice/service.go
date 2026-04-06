package notice

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	appevents "dreamcreator/internal/application/events"
	domainnotice "dreamcreator/internal/domain/notice"
)

type CreateNoticeInput struct {
	Kind      domainnotice.Kind
	Category  domainnotice.Category
	Code      string
	Severity  domainnotice.Severity
	I18n      *domainnotice.I18n
	Source    domainnotice.Source
	Action    domainnotice.Action
	Surfaces  []domainnotice.Surface
	DedupKey  string
	Metadata  map[string]any
	ExpiresAt *time.Time
}

type MarkReadInput struct {
	IDs  []string
	Read bool
}

type ArchiveInput struct {
	IDs      []string
	Archived bool
}

type Service struct {
	store domainnotice.Store
	bus   appevents.Bus
	now   func() time.Time
	newID func() string
}

func NewService(store domainnotice.Store, bus appevents.Bus) *Service {
	return &Service{
		store: store,
		bus:   bus,
		now:   time.Now,
		newID: func() string { return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000"))) },
	}
}

func (service *Service) Create(ctx context.Context, input CreateNoticeInput) (domainnotice.Notice, error) {
	if service == nil || service.store == nil {
		return domainnotice.Notice{}, errors.New("notice store unavailable")
	}
	if input.I18n == nil {
		return domainnotice.Notice{}, errors.New("notice i18n is required")
	}
	now := service.now()
	normalized := normalizeCreateInput(input)
	if normalized.Code == "" {
		return domainnotice.Notice{}, errors.New("notice code is required")
	}
	if normalized.Kind == "" {
		normalized.Kind = domainnotice.KindRuntimeEvent
	}
	if normalized.Severity == "" {
		normalized.Severity = domainnotice.SeverityInfo
	}
	if len(normalized.Surfaces) == 0 {
		normalized.Surfaces = []domainnotice.Surface{domainnotice.SurfaceCenter}
	}
	if normalized.DedupKey != "" {
		existing, err := service.store.FindByDedupKey(ctx, normalized.DedupKey)
		if err == nil {
			existing.Kind = normalized.Kind
			existing.Category = normalized.Category
			existing.Code = normalized.Code
			existing.Severity = normalized.Severity
			existing.I18n = *normalized.I18n
			existing.Source = normalized.Source
			existing.Action = normalized.Action
			existing.Surfaces = normalized.Surfaces
			existing.Metadata = normalized.Metadata
			existing.UpdatedAt = now
			existing.LastOccurredAt = now
			existing.OccurrenceCount++
			existing.Status = domainnotice.StatusUnread
			existing.ReadAt = nil
			existing.ArchivedAt = nil
			existing.ExpiresAt = normalized.ExpiresAt
			if err := service.store.Save(ctx, existing); err != nil {
				return domainnotice.Notice{}, err
			}
			service.publishUpdated(ctx, existing)
			return existing, nil
		}
		if !errors.Is(err, domainnotice.ErrNoticeNotFound) {
			return domainnotice.Notice{}, err
		}
	}

	item := domainnotice.Notice{
		ID:              service.newID(),
		Kind:            normalized.Kind,
		Category:        normalized.Category,
		Code:            normalized.Code,
		Severity:        normalized.Severity,
		Status:          domainnotice.StatusUnread,
		I18n:            *normalized.I18n,
		Source:          normalized.Source,
		Action:          normalized.Action,
		Surfaces:        normalized.Surfaces,
		DedupKey:        normalized.DedupKey,
		OccurrenceCount: 1,
		Metadata:        normalized.Metadata,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastOccurredAt:  now,
		ExpiresAt:       normalized.ExpiresAt,
	}
	if err := service.store.Save(ctx, item); err != nil {
		return domainnotice.Notice{}, err
	}
	service.publishCreated(ctx, item)
	return item, nil
}

func (service *Service) List(ctx context.Context, filter domainnotice.ListFilter) ([]domainnotice.Notice, error) {
	if service == nil || service.store == nil {
		return nil, errors.New("notice store unavailable")
	}
	return service.store.List(ctx, filter)
}

func (service *Service) UnreadCount(ctx context.Context, surface domainnotice.Surface) (int, error) {
	if service == nil || service.store == nil {
		return 0, errors.New("notice store unavailable")
	}
	return service.store.CountUnread(ctx, surface)
}

func (service *Service) MarkRead(ctx context.Context, input MarkReadInput) error {
	if service == nil || service.store == nil {
		return errors.New("notice store unavailable")
	}
	ids := normalizeIDs(input.IDs)
	if len(ids) == 0 {
		return nil
	}
	if err := service.store.MarkRead(ctx, ids, input.Read, service.now()); err != nil {
		return err
	}
	service.publishUnread(ctx)
	return nil
}

func (service *Service) Archive(ctx context.Context, input ArchiveInput) error {
	if service == nil || service.store == nil {
		return errors.New("notice store unavailable")
	}
	ids := normalizeIDs(input.IDs)
	if len(ids) == 0 {
		return nil
	}
	if err := service.store.Archive(ctx, ids, input.Archived, service.now()); err != nil {
		return err
	}
	service.publishUnread(ctx)
	return nil
}

func (service *Service) MarkAllRead(ctx context.Context, surface domainnotice.Surface) error {
	if service == nil || service.store == nil {
		return errors.New("notice store unavailable")
	}
	if err := service.store.MarkAllRead(ctx, surface, service.now()); err != nil {
		return err
	}
	service.publishUnread(ctx)
	return nil
}

func normalizeCreateInput(input CreateNoticeInput) CreateNoticeInput {
	input.Code = strings.TrimSpace(input.Code)
	input.DedupKey = strings.TrimSpace(input.DedupKey)
	input.Source.Producer = strings.TrimSpace(input.Source.Producer)
	input.Source.SessionKey = strings.TrimSpace(input.Source.SessionKey)
	input.Source.ThreadID = strings.TrimSpace(input.Source.ThreadID)
	input.Source.RunID = strings.TrimSpace(input.Source.RunID)
	input.Source.JobID = strings.TrimSpace(input.Source.JobID)
	input.Source.Channel = strings.TrimSpace(input.Source.Channel)
	input.Action.Type = strings.TrimSpace(input.Action.Type)
	input.Action.LabelKey = strings.TrimSpace(input.Action.LabelKey)
	input.Action.Target = strings.TrimSpace(input.Action.Target)
	input.Surfaces = normalizeSurfaces(input.Surfaces)
	if input.I18n != nil {
		input.I18n.TitleKey = strings.TrimSpace(input.I18n.TitleKey)
		input.I18n.SummaryKey = strings.TrimSpace(input.I18n.SummaryKey)
		input.I18n.BodyKey = strings.TrimSpace(input.I18n.BodyKey)
		input.I18n.Params = cloneStringMap(input.I18n.Params)
	}
	input.Action.Params = cloneStringMap(input.Action.Params)
	input.Metadata = cloneAnyMap(input.Metadata)
	return input
}

func normalizeSurfaces(input []domainnotice.Surface) []domainnotice.Surface {
	if len(input) == 0 {
		return nil
	}
	seen := make(map[domainnotice.Surface]struct{}, len(input))
	result := make([]domainnotice.Surface, 0, len(input))
	for _, surface := range input {
		trimmed := domainnotice.Surface(strings.TrimSpace(string(surface)))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func normalizeIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	result := make(map[string]string, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = value
	}
	return result
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(input))
	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		result[trimmedKey] = value
	}
	return result
}

func (service *Service) publishCreated(ctx context.Context, item domainnotice.Notice) {
	service.publishEvent(ctx, "notice.created", map[string]any{
		"notice": item,
	})
	service.publishUnread(ctx)
}

func (service *Service) publishUpdated(ctx context.Context, item domainnotice.Notice) {
	service.publishEvent(ctx, "notice.updated", map[string]any{
		"notice": item,
	})
	service.publishUnread(ctx)
}

func (service *Service) publishUnread(ctx context.Context) {
	if service == nil || service.store == nil {
		return
	}
	center, _ := service.store.CountUnread(ctx, domainnotice.SurfaceCenter)
	service.publishEvent(ctx, "notice.unread", map[string]any{
		"surface": domainnotice.SurfaceCenter,
		"count":   center,
	})
}

func (service *Service) publishEvent(ctx context.Context, topic string, payload map[string]any) {
	if service == nil || service.bus == nil {
		return
	}
	_ = service.bus.Publish(ctx, appevents.Event{
		Topic:   topic,
		Type:    topic,
		Payload: payload,
	})
}
