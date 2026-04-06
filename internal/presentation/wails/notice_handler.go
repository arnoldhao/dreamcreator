package wails

import (
	"context"

	appnotice "dreamcreator/internal/application/notice"
	domainnotice "dreamcreator/internal/domain/notice"
)

type NoticeHandler struct {
	service *appnotice.Service
}

type NoticeListRequest struct {
	Statuses   []domainnotice.Status   `json:"statuses,omitempty"`
	Kinds      []domainnotice.Kind     `json:"kinds,omitempty"`
	Categories []domainnotice.Category `json:"categories,omitempty"`
	Severities []domainnotice.Severity `json:"severities,omitempty"`
	Surface    domainnotice.Surface    `json:"surface,omitempty"`
	Query      string                  `json:"query,omitempty"`
	Limit      int                     `json:"limit,omitempty"`
}

type NoticeMarkReadRequest struct {
	IDs  []string `json:"ids"`
	Read bool     `json:"read"`
}

type NoticeArchiveRequest struct {
	IDs      []string `json:"ids"`
	Archived bool     `json:"archived"`
}

func NewNoticeHandler(service *appnotice.Service) *NoticeHandler {
	return &NoticeHandler{service: service}
}

func (handler *NoticeHandler) ServiceName() string {
	return "NoticeHandler"
}

func (handler *NoticeHandler) List(ctx context.Context, request NoticeListRequest) ([]domainnotice.Notice, error) {
	if handler == nil || handler.service == nil {
		return []domainnotice.Notice{}, nil
	}
	return handler.service.List(ctx, domainnotice.ListFilter{
		Statuses:   request.Statuses,
		Kinds:      request.Kinds,
		Categories: request.Categories,
		Severities: request.Severities,
		Surface:    request.Surface,
		Query:      request.Query,
		Limit:      request.Limit,
	})
}

func (handler *NoticeHandler) UnreadCount(ctx context.Context, surface string) (int, error) {
	if handler == nil || handler.service == nil {
		return 0, nil
	}
	return handler.service.UnreadCount(ctx, domainnotice.Surface(surface))
}

func (handler *NoticeHandler) MarkRead(ctx context.Context, request NoticeMarkReadRequest) error {
	if handler == nil || handler.service == nil {
		return nil
	}
	return handler.service.MarkRead(ctx, appnotice.MarkReadInput{
		IDs:  request.IDs,
		Read: request.Read,
	})
}

func (handler *NoticeHandler) MarkAllRead(ctx context.Context, surface string) error {
	if handler == nil || handler.service == nil {
		return nil
	}
	return handler.service.MarkAllRead(ctx, domainnotice.Surface(surface))
}

func (handler *NoticeHandler) Archive(ctx context.Context, request NoticeArchiveRequest) error {
	if handler == nil || handler.service == nil {
		return nil
	}
	return handler.service.Archive(ctx, appnotice.ArchiveInput{
		IDs:      request.IDs,
		Archived: request.Archived,
	})
}
