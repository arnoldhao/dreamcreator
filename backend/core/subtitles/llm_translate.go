package subtitles

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/types"

	"crypto/sha1"
	"encoding/hex"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TranslateSubtitleLLM performs AI translation for a project's subtitles from origin -> target
// using an OpenAI-compatible provider identified by providerID and model name.
// It persists progress to Bolt and emits websocket events via EventBus.
func (s *Service) TranslateSubtitleLLM(projectID, originLang, targetLang, providerID, model string) error {
	logger.Info("LLM translate requested (batched)",
		zap.String("projectID", projectID),
		zap.String("origin", originLang),
		zap.String("target", targetLang),
		zap.String("providerID", providerID),
		zap.String("model", model),
	)
	// default batch size 20; no profile; no filter; glossary hint mode
	return s.TranslateSubtitleLLMBatchedWithAnalysis(projectID, originLang, targetLang, providerID, model, nil, nil, 20, "", nil, false)
}

// TranslateSubtitleLLMWithOptions allows selecting glossary set IDs and providing task-only extra entries.
// strictGlossary 控制是否在 prompt glossary 中暴露占位符映射（hint vs strict 模式）。
func (s *Service) TranslateSubtitleLLMWithOptions(projectID, originLang, targetLang, providerID, model string, setIDs []string, extraEntries []types.GlossaryEntry, strictGlossary bool) error {
	logger.Info("LLM translate (batched, with options)",
		zap.String("projectID", projectID),
		zap.String("origin", originLang),
		zap.String("target", targetLang),
		zap.String("providerID", providerID),
		zap.String("model", model),
		zap.Int("glossarySetIDs", len(setIDs)),
		zap.Int("extraTerms", len(extraEntries)),
	)
	return s.TranslateSubtitleLLMBatchedWithAnalysis(projectID, originLang, targetLang, providerID, model, setIDs, extraEntries, 20, "", nil, strictGlossary)
}

// TranslateSubtitleLLMFailedOnlyWithOptions 仅重试上次失败/回退的片段
func (s *Service) TranslateSubtitleLLMFailedOnlyWithOptions(projectID, originLang, targetLang, providerID, model string, setIDs []string, extraEntries []types.GlossaryEntry, strictGlossary bool) error {
	// filter: include segments considered unfinished/failed for targetLang
	// Rules:
	// - if targetLang does not exist on this segment -> retry
	// - if language exists but Process is nil -> retry
	// - if Process.Status in {fallback, error} -> retry
	filter := func(seg *types.SubtitleSegment) bool {
		if seg == nil {
			return false
		}
		lc, ok := seg.Languages[targetLang]
		if !ok {
			return true
		}
		if lc.Process == nil {
			return true
		}
		st := strings.ToLower(strings.TrimSpace(lc.Process.Status))
		return st == "fallback" || st == "error"
	}
	return s.TranslateSubtitleLLMBatchedWithAnalysis(projectID, originLang, targetLang, providerID, model, setIDs, extraEntries, 20, "", filter, strictGlossary)
}

// translateSubtitleLLMCore: shared core with optional per-segment filter
func (s *Service) translateSubtitleLLMCore(projectID, originLang, targetLang, providerID, model string, setIDs []string, extraEntries []types.GlossaryEntry, filter func(*types.SubtitleSegment) bool, prof *types.LLMProfile) error {
	// Keep API compatibility: delegate to batched pipeline with default batch size.
	profileID := ""
	if prof != nil {
		profileID = prof.ID
	}
	// 当前核心路径保持 hint 模式（strictGlossary=false），严格模式通过 WithOptions 系列控制。
	return s.TranslateSubtitleLLMBatchedWithAnalysis(projectID, originLang, targetLang, providerID, model, setIDs, extraEntries, 20, profileID, filter, false)
}

func (s *Service) ensureLangMaps(seg *types.SubtitleSegment) {
	if seg.Languages == nil {
		seg.Languages = make(map[string]types.LanguageContent)
	}
	if seg.GuidelineStandard == nil {
		seg.GuidelineStandard = make(map[string]types.GuideLineStandard)
	}
}

func (s *Service) findSegmentByID(p *types.SubtitleProject, id string) *types.SubtitleSegment {
	if p == nil || id == "" {
		return nil
	}
	for i := range p.Segments {
		if p.Segments[i].ID == id {
			return &p.Segments[i]
		}
	}
	return nil
}

func (s *Service) pushLlmProgress(task types.ConversionTask) {
	evt := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      consts.TopicSubtitleProgress,
		Source:    "subtitles",
		Timestamp: time.Now(),
		Data:      task,
		Metadata:  map[string]interface{}{"progress": task},
	}
	s.eventBus.Publish(s.ctx, evt)
}

// --- legacy single-shot helpers removed ---

// glossary support
type restoreMap map[string]string // placeholder -> replacement

// applyGlossaryMaskBatch 替换 enforce 术语为占位符，并返回每个术语在本批中是否命中。
// 为了在 prompt 中稳定反查，占位符编号使用 entries 的原始索引（⟦G%03d⟧），
// 而匹配顺序则按术语长度降序处理，避免短词截断长词。
func (s *Service) applyGlossaryMaskBatch(lines []string, entries []*types.GlossaryEntry, targetLang string) ([]string, restoreMap, []bool) {
	placeholder := func(i int) string { return fmt.Sprintf("⟦G%03d⟧", i) }
	rm := make(restoreMap)
	out := make([]string, len(lines))
	copy(out, lines)
	used := make([]bool, len(entries))
	if len(entries) == 0 || len(lines) == 0 {
		return out, rm, used
	}
	// 按 source 长度降序排序索引，避免短词优先替换长词的一部分
	idxs := make([]int, len(entries))
	for i := range entries {
		idxs[i] = i
	}
	sort.SliceStable(idxs, func(a, b int) bool {
		sa, sb := 0, 0
		if entries[idxs[a]] != nil {
			sa = len(entries[idxs[a]].Source)
		}
		if entries[idxs[b]] != nil {
			sb = len(entries[idxs[b]].Source)
		}
		return sa > sb
	})
	for _, idx := range idxs {
		e := entries[idx]
		if e == nil || strings.TrimSpace(e.Source) == "" {
			continue
		}
		ph := placeholder(idx)
		replacement := ""
		if e.DoNotTranslate {
			replacement = e.Source
		} else if v, ok := e.Translations[targetLang]; ok && strings.TrimSpace(v) != "" {
			replacement = v
		} else if v, ok := e.Translations["all"]; ok && strings.TrimSpace(v) != "" { // wildcard: all targets
			replacement = v
		} else if v, ok := e.Translations["*"]; ok && strings.TrimSpace(v) != "" { // alternate wildcard key
			replacement = v
		} else {
			replacement = e.Source
		}
		rm[ph] = replacement
		for j := range out {
			if out[j] == "" {
				continue
			}
			before := out[j]
			if e.CaseSensitive {
				out[j] = strings.ReplaceAll(out[j], e.Source, ph)
			} else {
				pat := regexp.MustCompile("(?i)" + regexp.QuoteMeta(e.Source))
				out[j] = pat.ReplaceAllString(out[j], ph)
			}
			if out[j] != before {
				used[idx] = true
			}
		}
	}
	return out, rm, used
}

func (s *Service) restoreGlossaryPlaceholders(text string, rm restoreMap) string {
	if text == "" || len(rm) == 0 {
		return text
	}
	out := text
	for ph, repl := range rm {
		out = strings.ReplaceAll(out, ph, repl)
	}
	return out
}

// general placeholders: HTML/ASS tags, variables, timecodes, bracket notes
func (s *Service) applyGeneralProtectionBatch(lines []string) ([]string, restoreMap) {
	out := make([]string, len(lines))
	copy(out, lines)
	rm := make(restoreMap)
	// patterns
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`<[^>]+>`),                        // html tags
		regexp.MustCompile(`\{\\[^}]+\}`),                    // ASS tags {\b1}
		regexp.MustCompile(`\{[^}]+\}`),                      // {var}
		regexp.MustCompile(`\[[^\]]+\]`),                     // [NOTE]
		regexp.MustCompile(`\d{2}:\d{2}:\d{2}[\.,:]\d{2,3}`), // timecodes
	}
	idx := 0
	placeholder := func(i int) string { return fmt.Sprintf("⟦P%03d⟧", i) }
	// collect unique matches
	uniq := make(map[string]string) // match -> placeholder
	for _, re := range patterns {
		for i := range out {
			if out[i] == "" {
				continue
			}
			m := re.FindAllString(out[i], -1)
			for _, s := range m {
				if _, ok := uniq[s]; !ok {
					ph := placeholder(idx)
					idx++
					uniq[s] = ph
					rm[ph] = s
				}
			}
		}
	}
	// replace
	for i := range out {
		for match, ph := range uniq {
			out[i] = strings.ReplaceAll(out[i], match, ph)
		}
	}
	return out, rm
}

// parseJSONArrayText attempts to parse a JSON array of strings; trims code fences if present.
// removed legacy array parsing (single-shot path)

func (s *Service) cacheKey(providerID, model, srcLang, dstLang, text string) string {
	h := sha1.New()
	io := providerID + "|" + model + "|" + srcLang + "|" + dstLang + "|" + text
	_, _ = h.Write([]byte(io))
	return hex.EncodeToString(h.Sum(nil))
}

// --- Legacy single-shot helper notes ---
// The previous single-shot array parsers and prompts have been removed.
// This module now delegates entirely to the batched JSONL pipeline.

// --- Global Profile (model-agnostic) helpers ---
func (s *Service) TranslateSubtitleLLMWithGlobalProfile(projectID, originLang, targetLang, providerID, model, profileID string) error {
	return s.TranslateSubtitleLLMBatchedWithAnalysis(projectID, originLang, targetLang, providerID, model, nil, nil, 20, profileID, nil, false)
}

func (s *Service) TranslateSubtitleLLMWithGlobalProfileWithOptions(projectID, originLang, targetLang, providerID, model, profileID string, setIDs []string, extraEntries []types.GlossaryEntry, strictGlossary bool) error {
	if strings.TrimSpace(profileID) == "" {
		return fmt.Errorf("profile_id is required")
	}
	return s.TranslateSubtitleLLMBatchedWithAnalysis(projectID, originLang, targetLang, providerID, model, setIDs, extraEntries, 20, profileID, nil, strictGlossary)
}

func (s *Service) TranslateSubtitleLLMFailedOnlyWithGlobalProfileWithOptions(projectID, originLang, targetLang, providerID, model, profileID string, setIDs []string, extraEntries []types.GlossaryEntry, strictGlossary bool) error {
	if strings.TrimSpace(profileID) == "" {
		return fmt.Errorf("profile_id is required")
	}
	filter := func(seg *types.SubtitleSegment) bool {
		if seg == nil {
			return false
		}
		lc, ok := seg.Languages[targetLang]
		if !ok {
			return true
		}
		if lc.Process == nil {
			return true
		}
		st := strings.ToLower(strings.TrimSpace(lc.Process.Status))
		return st == "fallback" || st == "error"
	}
	return s.TranslateSubtitleLLMBatchedWithAnalysis(projectID, originLang, targetLang, providerID, model, setIDs, extraEntries, 20, profileID, filter, strictGlossary)
}

// New translation with IDs
// removed legacy single-shot translate helpers

// New review with IDs
// removed legacy single-shot review helpers

// Profile-aware variants
// removed legacy single-shot translate helpers (profile)

// removed legacy single-shot review helpers (profile)
