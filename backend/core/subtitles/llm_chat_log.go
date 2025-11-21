package subtitles

import (
	"time"

	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/types"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// appendLLMChatMessage 追加一条会话消息并通过事件总线广播，失败时仅记录日志，不影响主流程。
func (s *Service) appendLLMChatMessage(projectID, lang, taskID, providerName, providerID, model, role, kind, content, stage string, meta map[string]any) {
	if projectID == "" || lang == "" || taskID == "" || content == "" {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Warn("appendLLMChatMessage panic", zap.Any("recover", r))
		}
	}()

	proj, err := s.boltStorage.GetSubtitle(projectID)
	if err != nil || proj == nil {
		return
	}
	if proj.LanguageMetadata == nil {
		return
	}
	m, ok := proj.LanguageMetadata[lang]
	if !ok {
		return
	}
	if m.Status.LLMConversations == nil {
		m.Status.LLMConversations = make(map[string]types.LLMConversation)
	}
	conv, ok := m.Status.LLMConversations[taskID]
	if !ok {
		conv = types.LLMConversation{
			ID:         taskID,
			ProjectID:  projectID,
			Language:   lang,
			TaskID:     taskID,
			Provider:   providerName,
			ProviderID: providerID,
			Model:      model,
			Status:     types.LLMConversationStatusRunning,
			StartedAt:  time.Now().Unix(),
		}
	}
	// 补全元信息
	if conv.Provider == "" && providerName != "" {
		conv.Provider = providerName
	}
	if conv.ProviderID == "" && providerID != "" {
		conv.ProviderID = providerID
	}
	if conv.Model == "" && model != "" {
		conv.Model = model
	}

	now := time.Now().Unix()
	msg := types.LLMChatMessage{
		ID:        uuid.New().String(),
		Role:      types.LLMChatRole(role),
		Kind:      kind,
		Content:   content,
		CreatedAt: now,
	}
	if stage != "" || len(meta) > 0 {
		msg.Metadata = map[string]any{}
		if stage != "" {
			msg.Metadata["stage"] = stage
		}
		for k, v := range meta {
			msg.Metadata[k] = v
		}
	}

	conv.Messages = append(conv.Messages, msg)
	m.Status.LLMConversations[taskID] = conv
	proj.LanguageMetadata[lang] = m
	if err := s.boltStorage.SaveSubtitle(proj); err != nil {
		logger.Warn("save subtitle (conversation) failed", zap.Error(err))
	}

	// 推送 WS 事件
	ev := &types.LLMChatEvent{
		ProjectID:      projectID,
		Language:       lang,
		TaskID:         taskID,
		ConversationID: conv.ID,
		Status:         string(conv.Status),
		Message:        &msg,
		MessagesTotal:  len(conv.Messages),
	}
	evt := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      consts.TopicSubtitleConversation,
		Source:    "subtitles",
		Timestamp: time.Now(),
		Data:      ev,
		Metadata: map[string]any{
			"project_id": projectID,
			"language":   lang,
			"task_id":    taskID,
		},
	}
	s.eventBus.Publish(s.ctx, evt)
}

// pushLLMStreamDelta 仅用于推送流式增量内容到前端，不修改会话持久化数据。
// 适用于 LLM SSE 流：将每个 delta 作为临时消息发送给前端，以便实时展示，
// 但仅在流结束后由调用方写入一次聚合后的完整回复。
func (s *Service) pushLLMStreamDelta(projectID, lang, taskID, providerName, providerID, model, role, kind, content, stage string, meta map[string]any) {
	if projectID == "" || lang == "" || taskID == "" || content == "" {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Warn("pushLLMStreamDelta panic", zap.Any("recover", r))
		}
	}()
	now := time.Now().Unix()
	msg := types.LLMChatMessage{
		ID:        uuid.New().String(),
		Role:      types.LLMChatRole(role),
		Kind:      kind,
		Content:   content,
		CreatedAt: now,
	}
	if stage != "" || len(meta) > 0 {
		msg.Metadata = map[string]any{}
		if stage != "" {
			msg.Metadata["stage"] = stage
		}
		for k, v := range meta {
			msg.Metadata[k] = v
		}
	}
	ev := &types.LLMChatEvent{
		ProjectID:      projectID,
		Language:       lang,
		TaskID:         taskID,
		ConversationID: taskID,
		Status:         string(types.LLMConversationStatusRunning),
		Message:        &msg,
	}
	evt := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      consts.TopicSubtitleConversation,
		Source:    "subtitles",
		Timestamp: time.Now(),
		Data:      ev,
		Metadata: map[string]any{
			"project_id": projectID,
			"language":   lang,
			"task_id":    taskID,
		},
	}
	s.eventBus.Publish(s.ctx, evt)
}

// markLLMConversationFinished 更新会话状态为 finished/failed 并广播一次状态更新。
func (s *Service) markLLMConversationFinished(projectID, lang, taskID string, status types.LLMConversationStatus) {
	if projectID == "" || lang == "" || taskID == "" {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Warn("markLLMConversationFinished panic", zap.Any("recover", r))
		}
	}()

	proj, err := s.boltStorage.GetSubtitle(projectID)
	if err != nil || proj == nil {
		return
	}
	m, ok := proj.LanguageMetadata[lang]
	if !ok || m.Status.LLMConversations == nil {
		return
	}
	conv, ok := m.Status.LLMConversations[taskID]
	if !ok {
		return
	}
	conv.Status = status
	if conv.EndedAt == 0 {
		conv.EndedAt = time.Now().Unix()
	}
	m.Status.LLMConversations[taskID] = conv
	proj.LanguageMetadata[lang] = m
	if err := s.boltStorage.SaveSubtitle(proj); err != nil {
		logger.Warn("save subtitle (conversation status) failed", zap.Error(err))
	}

	ev := &types.LLMChatEvent{
		ProjectID:      projectID,
		Language:       lang,
		TaskID:         taskID,
		ConversationID: conv.ID,
		Status:         string(conv.Status),
		MessagesTotal:  len(conv.Messages),
	}
	evt := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      consts.TopicSubtitleConversation,
		Source:    "subtitles",
		Timestamp: time.Now(),
		Data:      ev,
		Metadata: map[string]any{
			"project_id": projectID,
			"language":   lang,
			"task_id":    taskID,
		},
	}
	s.eventBus.Publish(s.ctx, evt)
}
