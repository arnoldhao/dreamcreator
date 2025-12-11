package subtitles

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/provider"
	"dreamcreator/backend/types"

	"go.uber.org/zap"
)

// BuildProjectAnalysisPrompts 构建“项目级初始化（仅分析，不翻译）”的提示词
func BuildProjectAnalysisPrompts(srcLang, sysTpl string, segments []types.SubtitleSegment) (system, user string) {
	// 将字幕压缩为 id+text 的要点列表，避免过多无关字段
	var b strings.Builder
	for _, seg := range segments {
		txt := strings.TrimSpace(seg.GetText(srcLang))
		if txt == "" {
			continue
		}
		fmt.Fprintf(&b, "- {\"id\": \"%s\", \"text\": %q}\n", seg.ID, txt)
	}
	// 要求模型输出一个 JSON 对象，包含 genre/tone/scene_outline/style_guide/roles/initial_glossary
	baseSys := "You are a subtitle project analyst. Read all items and output only ONE JSON object with keys: genre, tone, style_guide (array of short rules), scene_outline (array of {start_id,end_id,summary}), roles (array of {name,person,notes}), initial_glossary (array of glossary entries). Do NOT translate lines individually."
	// 指示 glossary 的最小结构，复用已有 GlossaryEntry 语义（source/translations/do_not_translate/case_sensitive）
	baseSys += " The initial_glossary should use fields: source, translations (map target_lang->text, allow 'all' as wildcard), do_not_translate (bool), case_sensitive (bool). Keep the list concise."
	system = baseSys
	if strings.TrimSpace(sysTpl) != "" {
		system = strings.TrimSpace(sysTpl) + "\n\n" + baseSys
	}
	user = "Analyze the following subtitle bullets (id + text). Return ONLY the JSON object, no code fences, no comments.\n" + b.String()
	return
}

// EnsureProjectAnalysis 如果项目还没有分析结果，则调用 LLM 生成一次全局分析，并保存到项目元数据中
func (s *Service) EnsureProjectAnalysis(ctx context.Context, proj *types.SubtitleProject, srcLang, providerID, model string, prof *types.LLMProfile) (*types.ProjectAnalysis, provider.TokenUsage, error) {
	var tu provider.TokenUsage
	if proj == nil {
		return nil, tu, fmt.Errorf("project nil")
	}
	if proj.Metadata.Analysis != nil {
		return proj.Metadata.Analysis, tu, nil
	}

	sys, user := BuildProjectAnalysisPrompts(srcLang, func() string {
		if prof != nil {
			return prof.SysPromptTpl
		}
		return ""
	}(), proj.Segments)

	msgs := []provider.ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: user}}
	// 使用 JSONMode 以提高 JSON 合法性
	opts := provider.ChatOptions{}
	if prof != nil {
		opts.Temperature = prof.Temperature
		opts.TopP = prof.TopP
		opts.MaxTokens = prof.MaxTokens
		opts.JSONMode = true
	} else {
		opts.Temperature = 0.1
		opts.JSONMode = true
	}
	content, u, err := s.providerService.ChatCompletionWithOptionsUsage(ctx, providerID, model, msgs, opts)
	if err != nil {
		return nil, tu, err
	}
	tu = u
	payload := strings.TrimSpace(content)
	if strings.HasPrefix(payload, "```") {
		// 容错：去掉 code fence
		if p := strings.Index(payload, "\n"); p > -1 {
			payload = strings.TrimSpace(payload[p+1:])
		}
		payload = strings.TrimSuffix(payload, "```")
		payload = strings.TrimSpace(payload)
	}
	var parsed types.ProjectAnalysis
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		// 容错：如果外层包了一层对象，尝试寻找第一个对象字段
		var obj map[string]any
		if e2 := json.Unmarshal([]byte(payload), &obj); e2 == nil {
			if b, e3 := json.Marshal(obj); e3 == nil {
				_ = json.Unmarshal(b, &parsed)
			}
		}
		if parsed.Genre == "" && len(parsed.StyleGuide) == 0 && len(parsed.SceneOutline) == 0 {
			logger.Warn("analysis: failed to parse JSON", zap.Error(err), zap.String("preview", func() string {
				if len(payload) > 300 {
					return payload[:300] + "..."
				}
				return payload
			}()))
			return nil, tu, fmt.Errorf("analysis json parse failed: %w", err)
		}
	}

	// 保存到项目（在保存前同步可能存在的会话，避免覆盖 talk_log 写入）
	proj.Metadata.Analysis = &parsed
	s.syncLLMConversationsFromStore(proj.ID, proj)
	if err := s.boltStorage.SaveSubtitle(proj); err != nil {
		return nil, tu, err
	}
	return &parsed, tu, nil
}
