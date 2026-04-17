package wails

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"dreamcreator/internal/application/events"
	llmrecord "dreamcreator/internal/application/llmrecord"
	"dreamcreator/internal/application/thread/dto"
	"dreamcreator/internal/application/thread/service"
	domainthread "dreamcreator/internal/domain/thread"
)

const maxChatAttachmentBytes int64 = 20 * 1024 * 1024

type SelectedAttachmentFile struct {
	Path          string `json:"path"`
	FileName      string `json:"fileName"`
	Mime          string `json:"mime"`
	SizeBytes     int64  `json:"sizeBytes"`
	ContentBase64 string `json:"contentBase64"`
}

type ThreadHandler struct {
	service *service.ThreadService
	calls   *llmrecord.Service
	bus     events.Bus
	windows *WindowManager
}

const (
	threadChangeUpsert = "upsert"
	threadChangePurge  = "purge"
)

func NewThreadHandler(service *service.ThreadService, calls *llmrecord.Service, bus events.Bus, windows *WindowManager) *ThreadHandler {
	return &ThreadHandler{
		service: service,
		calls:   calls,
		bus:     bus,
		windows: windows,
	}
}

func (handler *ThreadHandler) ServiceName() string {
	return "ThreadHandler"
}

func (handler *ThreadHandler) NewThread(ctx context.Context, request dto.NewThreadRequest) (dto.NewThreadResponse, error) {
	response, err := handler.service.NewThread(ctx, request)
	if err != nil {
		return dto.NewThreadResponse{}, err
	}
	handler.emitThreadUpdated(ctx, response.ThreadID, threadChangeUpsert, "new-thread")
	return response, nil
}

func (handler *ThreadHandler) ListThreads(ctx context.Context, includeDeleted bool) ([]dto.Thread, error) {
	return handler.service.ListThreads(ctx, includeDeleted)
}

func (handler *ThreadHandler) ListMessages(ctx context.Context, threadID string, limit int) ([]dto.Message, error) {
	return handler.service.ListMessages(ctx, threadID, limit)
}

func (handler *ThreadHandler) ListThreadRunEvents(ctx context.Context, request dto.ListThreadRunEventsRequest) ([]dto.ThreadRunEvent, error) {
	return handler.service.ListThreadRunEvents(ctx, request)
}

func (handler *ThreadHandler) GetThreadContextTokens(ctx context.Context, threadID string) (dto.ContextTokensSnapshot, error) {
	return handler.service.GetContextTokensSnapshot(ctx, threadID)
}

func (handler *ThreadHandler) ListLLMCallRecords(ctx context.Context, request llmrecord.ListRequest) ([]llmrecord.Record, error) {
	if handler.calls == nil {
		return nil, errors.New("llm call record service is not configured")
	}
	return handler.calls.List(ctx, request)
}

func (handler *ThreadHandler) GetLLMCallRecord(ctx context.Context, id string) (llmrecord.Record, error) {
	if handler.calls == nil {
		return llmrecord.Record{}, errors.New("llm call record service is not configured")
	}
	return handler.calls.Get(ctx, id)
}

func (handler *ThreadHandler) PruneExpiredLLMCallRecords(ctx context.Context) (int, error) {
	if handler.calls == nil {
		return 0, errors.New("llm call record service is not configured")
	}
	return handler.calls.PruneExpired(ctx)
}

func (handler *ThreadHandler) ClearLLMCallRecords(ctx context.Context) (int, error) {
	if handler.calls == nil {
		return 0, errors.New("llm call record service is not configured")
	}
	return handler.calls.Clear(ctx)
}

func (handler *ThreadHandler) AppendMessage(ctx context.Context, request dto.AppendMessageRequest) error {
	if err := handler.service.AppendMessage(ctx, request); err != nil {
		return err
	}
	handler.emitThreadUpdated(ctx, request.ThreadID, threadChangeUpsert, "append-message")
	return nil
}

func (handler *ThreadHandler) SoftDeleteThread(ctx context.Context, threadID string) error {
	if err := handler.service.SoftDeleteThread(ctx, threadID); err != nil {
		return err
	}
	handler.emitThreadUpdated(ctx, threadID, threadChangeUpsert, "soft-delete")
	return nil
}

func (handler *ThreadHandler) RenameThread(ctx context.Context, request dto.RenameThreadRequest) error {
	if err := handler.service.RenameThread(ctx, request); err != nil {
		return err
	}
	handler.emitThreadUpdated(ctx, request.ThreadID, threadChangeUpsert, "rename")
	return nil
}

func (handler *ThreadHandler) GenerateThreadTitle(ctx context.Context, request dto.GenerateThreadTitleRequest) (dto.GenerateThreadTitleResponse, error) {
	response, err := handler.service.GenerateThreadTitle(ctx, request)
	if err != nil {
		return dto.GenerateThreadTitleResponse{}, err
	}
	if response.Updated {
		handler.emitThreadUpdated(ctx, response.ThreadID, threadChangeUpsert, "generate-title")
	}
	return response, nil
}

func (handler *ThreadHandler) SetThreadStatus(ctx context.Context, request dto.SetThreadStatusRequest) error {
	if err := handler.service.SetThreadStatus(ctx, request); err != nil {
		return err
	}
	handler.emitThreadUpdated(ctx, request.ThreadID, threadChangeUpsert, "set-status")
	return nil
}

func (handler *ThreadHandler) RestoreThread(ctx context.Context, threadID string) error {
	if err := handler.service.RestoreThread(ctx, threadID); err != nil {
		return err
	}
	handler.emitThreadUpdated(ctx, threadID, threadChangeUpsert, "restore")
	return nil
}

func (handler *ThreadHandler) PurgeThread(ctx context.Context, threadID string) error {
	if err := handler.service.PurgeThread(ctx, threadID); err != nil {
		return err
	}
	handler.emitThreadUpdated(ctx, threadID, threadChangePurge, "purge")
	return nil
}

func (handler *ThreadHandler) SelectAttachmentFiles(_ context.Context, title string) ([]SelectedAttachmentFile, error) {
	if handler.windows == nil {
		return nil, errors.New("window manager not available")
	}
	normalizedTitle := strings.TrimSpace(title)
	paths, err := handler.windows.SelectFilesDialog(normalizedTitle, "")
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return []SelectedAttachmentFile{}, nil
	}
	result := make([]SelectedAttachmentFile, 0, len(paths))
	for _, path := range paths {
		item, itemErr := readSelectedAttachmentFile(path)
		if itemErr != nil {
			return nil, itemErr
		}
		result = append(result, item)
	}
	return result, nil
}

func (handler *ThreadHandler) emitThreadUpdated(ctx context.Context, threadID string, change string, reason string) {
	threadID = strings.TrimSpace(threadID)
	if handler.bus == nil || threadID == "" {
		return
	}

	payload := map[string]any{
		"threadId": threadID,
	}
	if strings.TrimSpace(change) != "" {
		payload["change"] = change
	}
	if strings.TrimSpace(reason) != "" {
		payload["reason"] = reason
	}
	if strings.EqualFold(change, threadChangeUpsert) && handler.service != nil {
		threadItem, err := handler.service.GetThread(ctx, threadID)
		if err == nil {
			payload["thread"] = threadItem
			if strings.TrimSpace(threadItem.UpdatedAt) != "" {
				payload["threadVersion"] = threadItem.UpdatedAt
			}
		} else if !errors.Is(err, domainthread.ErrThreadNotFound) {
			zap.L().Debug("thread update event skipped thread snapshot",
				zap.String("threadID", threadID),
				zap.Error(err),
			)
		}
	}

	_ = handler.bus.Publish(ctx, events.Event{
		Topic:   "chat.thread.updated",
		Type:    "thread-updated",
		Payload: payload,
	})
}

func readSelectedAttachmentFile(path string) (SelectedAttachmentFile, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return SelectedAttachmentFile{}, errors.New("path is required")
	}
	info, err := os.Stat(trimmed)
	if err != nil {
		return SelectedAttachmentFile{}, err
	}
	if info.IsDir() {
		return SelectedAttachmentFile{}, errors.New("path is a directory")
	}
	if info.Size() > maxChatAttachmentBytes {
		return SelectedAttachmentFile{}, fmt.Errorf("file too large (max %d bytes): %s", maxChatAttachmentBytes, filepath.Base(trimmed))
	}
	payload, err := os.ReadFile(trimmed)
	if err != nil {
		return SelectedAttachmentFile{}, err
	}
	mimeType := detectAttachmentMime(trimmed, payload)
	return SelectedAttachmentFile{
		Path:          trimmed,
		FileName:      filepath.Base(trimmed),
		Mime:          mimeType,
		SizeBytes:     info.Size(),
		ContentBase64: base64.StdEncoding.EncodeToString(payload),
	}, nil
}

func detectAttachmentMime(path string, payload []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		if byExt := mime.TypeByExtension(ext); strings.TrimSpace(byExt) != "" {
			return byExt
		}
	}
	if len(payload) == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(payload)
}
