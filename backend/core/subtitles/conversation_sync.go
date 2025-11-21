package subtitles

import (
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/types"

	"go.uber.org/zap"
)

// syncLLMConversationsFromStore 将 Bolt 中最新的 LLM 会话数据合并到内存中的 proj，
// 避免在翻译流程中频繁 SaveSubtitle(proj) 时覆盖掉 appendLLMChatMessage 写入的对话。
// 仅合并 LanguageMetadata[*].Status.LLMConversations，不修改其它字段。
func (s *Service) syncLLMConversationsFromStore(projectID string, proj *types.SubtitleProject) {
	if proj == nil || projectID == "" || s.boltStorage == nil {
		return
	}
	p2, err := s.boltStorage.GetSubtitle(projectID)
	if err != nil || p2 == nil {
		if err != nil {
			logger.Warn("syncLLMConversations: load subtitle failed", zap.String("projectID", projectID), zap.Error(err))
		}
		return
	}
	if p2.LanguageMetadata == nil {
		return
	}
	if proj.LanguageMetadata == nil {
		proj.LanguageMetadata = make(map[string]types.LanguageMetadata)
	}
	for lang, meta2 := range p2.LanguageMetadata {
		if meta2.Status.LLMConversations == nil || len(meta2.Status.LLMConversations) == 0 {
			continue
		}
		m := proj.LanguageMetadata[lang]
		if m.Status.LLMConversations == nil {
			m.Status.LLMConversations = make(map[string]types.LLMConversation)
		}
		// 逐个会话合并：若存储中的会话消息更多或状态更新，则覆盖内存中的对应条目
		for convID, remote := range meta2.Status.LLMConversations {
			local, ok := m.Status.LLMConversations[convID]
			if !ok {
				m.Status.LLMConversations[convID] = remote
				continue
			}
			// 以“消息数量更多”为主的简单策略；如有需要可扩展为比较 EndedAt/Status 等
			if len(remote.Messages) > len(local.Messages) {
				m.Status.LLMConversations[convID] = remote
			}
		}
		proj.LanguageMetadata[lang] = m
	}
}
