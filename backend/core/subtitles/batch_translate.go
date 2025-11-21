package subtitles

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/provider"
	"dreamcreator/backend/types"

	"go.uber.org/zap"
)

// jsonlItem 是批量“自我反思式”翻译的单行结构。
// 为了节省 token，当前协议只要求模型在最终输出中提供 {id, final}；
// draft/reflection 字段仅用于兼容旧输出或宽松解析，不要求模型填充。
type jsonlItem struct {
	ID         string   `json:"id"`
	Src        string   `json:"src,omitempty"`
	Draft      string   `json:"draft,omitempty"`
	Reflection []string `json:"reflection,omitempty"`
	Final      string   `json:"final"`
}

// promptGlossaryEntry 为构造 LLM 提示时使用的一维术语结构。
// origin 标明来源："global" | "task" | "auto"。
// placeholder 仅在本批字幕中实际命中的 enforce 术语上出现，用于反查占位符。
type promptGlossaryEntry struct {
	ID             string            `json:"id,omitempty"`
	SetID          string            `json:"set_id,omitempty"`
	Source         string            `json:"source"`
	DoNotTranslate bool              `json:"do_not_translate"`
	CaseSensitive  bool              `json:"case_sensitive"`
	Translations   map[string]string `json:"translations,omitempty"`
	Placeholder    string            `json:"placeholder,omitempty"`
	Origin         string            `json:"origin,omitempty"`
	CreatedAt      int64             `json:"created_at,omitempty"`
	UpdatedAt      int64             `json:"updated_at,omitempty"`
}

// promptLangLabel builds a display string that combines code + name, avoiding duplicates.
func promptLangLabel(code, name string) string {
	c := strings.TrimSpace(code)
	n := strings.TrimSpace(name)
	if c == "" && n != "" {
		return n
	}
	if n == "" {
		return c
	}
	if strings.EqualFold(c, n) {
		return c
	}
	if c == "" {
		return n
	}
	return fmt.Sprintf("%s (%s)", c, n)
}

// resolveLanguageName tries project metadata first, then falls back to the target language registry.
func (s *Service) resolveLanguageName(code string, proj *types.SubtitleProject) string {
	c := strings.TrimSpace(code)
	if c == "" {
		return ""
	}
	if proj != nil && proj.LanguageMetadata != nil {
		if meta, ok := proj.LanguageMetadata[c]; ok {
			if strings.TrimSpace(meta.LanguageName) != "" {
				return strings.TrimSpace(meta.LanguageName)
			}
		}
	}
	if s != nil && s.boltStorage != nil {
		if tl, err := s.boltStorage.GetTargetLanguage(c); err == nil && tl != nil && strings.TrimSpace(tl.Name) != "" {
			return strings.TrimSpace(tl.Name)
		}
	}
	return c
}

// BuildBatchJSONModePrompts 构建适用于 JSON Mode 的批量提示（要求返回单个 JSON 对象）。
// 要求模型仍然采用“自我反思式”翻译：在内部为每行先生成 draft、进行简短 reflection，再给出 final，
// 但最终输出中只包含 {id, final} 结构，以减少 token。
func BuildBatchJSONModePrompts(sysTpl string, srcLang, dstLang string, globalStyle map[string]any, glossary []promptGlossaryEntry, references []map[string]string, boundarySrc []string, boundaryDst []string, batchIDs []string, batchSrc []string) (system, user string) {
	baseSys := fmt.Sprintf(consts.SubtitleJSONModeSystemPromptTpl, srcLang, dstLang)
	if strings.TrimSpace(sysTpl) != "" {
		system = strings.TrimSpace(sysTpl) + "\n\n" + baseSys
	} else {
		system = baseSys
	}

	var b strings.Builder
	enc := func(v any) string { bt, _ := json.Marshal(v); return string(bt) }
	fmt.Fprintf(&b, "{\n  \"global_style\": %s,\n", enc(globalStyle))
	fmt.Fprintf(&b, "  \"glossary\": %s,\n", enc(glossary))
	fmt.Fprintf(&b, "  \"reference_translations\": %s,\n", enc(references))
	fmt.Fprintf(&b, "  \"boundary_context\": {\"prev_src\": %s, \"prev_dst\": %s},\n", enc(boundarySrc), enc(boundaryDst))
	items := make([]map[string]string, 0, len(batchIDs))
	for i := range batchIDs {
		items = append(items, map[string]string{"id": batchIDs[i], "text": batchSrc[i]})
	}
	fmt.Fprintf(&b, "  \"batch\": %s\n}", enc(items))

	user = consts.SubtitleJSONModeUserPrompt + "\n" + b.String()
	return
}

// BuildBatchJSONLPrompts 构建批量翻译的 system/user 提示
// globalStyle: 从项目分析得来的 genre/tone/style_guide 的精简摘要
// glossary: 选取的关键术语若干（enforce 命中项 + 少量 auto hints）
// references: 少量风格/术语代表性的已译句对
// boundarySrc/boundaryDst: 上一批收尾以及当前开头的边界上下文
func BuildBatchJSONLPrompts(sysTpl string, srcLang, dstLang string, globalStyle map[string]any, glossary []promptGlossaryEntry, references []map[string]string, boundarySrc []string, boundaryDst []string, batchIDs []string, batchSrc []string) (system, user string) {
	// System 侧强调 JSONL，但仅要求输出最终译文
	baseSys := fmt.Sprintf(consts.SubtitleJSONLSystemPromptTpl, srcLang, dstLang)
	if strings.TrimSpace(sysTpl) != "" {
		system = strings.TrimSpace(sysTpl) + "\n\n" + baseSys
	} else {
		system = baseSys
	}

	// 构造输入 JSON（非严格 schema，主要帮助模型吸收上下文）
	// 为避免 token 膨胀，尽量精简
	var b strings.Builder
	enc := func(v any) string { bt, _ := json.Marshal(v); return string(bt) }
	// global summary
	fmt.Fprintf(&b, "{\n  \"global_style\": %s,\n", enc(globalStyle))
	// glossary
	fmt.Fprintf(&b, "  \"glossary\": %s,\n", enc(glossary))
	// references few-shot
	fmt.Fprintf(&b, "  \"reference_translations\": %s,\n", enc(references))
	// boundary context
	fmt.Fprintf(&b, "  \"boundary_context\": {\"prev_src\": %s, \"prev_dst\": %s},\n", enc(boundarySrc), enc(boundaryDst))
	// batch
	items := make([]map[string]string, 0, len(batchIDs))
	for i := range batchIDs {
		items = append(items, map[string]string{"id": batchIDs[i], "text": batchSrc[i]})
	}
	fmt.Fprintf(&b, "  \"batch\": %s\n}", enc(items))

	// user 指令
	user = consts.SubtitleJSONLUserPrompt + "\n" + b.String()
	return
}

// normalizeLLMPayload 负责从 LLM 原始输出中抽取“真正的 JSON/JSONL 内容”。
// 支持以下情况：
// 1) 纯 JSON/JSONL 文本（无 code fence）；
// 2) 以 ``` 或 ```json 等开头的代码块；
// 3) 前面有少量说明文字，后面跟一个以 ``` 开头的代码块（在首个 {/[ 之前出现）。
func normalizeLLMPayload(s string) string {
	txt := strings.TrimSpace(s)
	if txt == "" {
		return txt
	}
	// 快速路径：整体以 ``` 开头（如 ```json）
	if strings.HasPrefix(txt, "```") {
		return stripFirstFenceBlock(txt)
	}
	// 扫描前若干行，若在遇到 JSON 起始行 ({ 或 [) 之前先遇到 ```，则视为“说明 + 代码块”形式
	lines := strings.Split(txt, "\n")
	fenceStart := -1
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "```") {
			fenceStart = i
			break
		}
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
			break
		}
	}
	if fenceStart == -1 {
		return txt
	}
	block := strings.Join(lines[fenceStart:], "\n")
	return stripFirstFenceBlock(block)
}

// stripFirstFenceBlock 去掉首个 markdown 代码块的外壳（```xxx ... ```），仅返回内部内容。
func stripFirstFenceBlock(block string) string {
	b := strings.TrimSpace(block)
	if !strings.HasPrefix(b, "```") {
		return b
	}
	// 去掉第一行 ``` / ```json / ```yaml 等
	if nl := strings.Index(b, "\n"); nl != -1 {
		b = b[nl+1:]
	} else {
		// 单行 ```xxx 情况，内部没有内容
		return ""
	}
	// 去掉最后一个 ``` 之后的内容
	if end := strings.LastIndex(b, "```"); end != -1 {
		b = b[:end]
	}
	return strings.TrimSpace(b)
}

// parseJSONL 解析 JSON Lines，忽略非法行
func parseJSONL(s string) ([]jsonlItem, error) {
	payload := normalizeLLMPayload(s)
	out := make([]jsonlItem, 0)
	rd := bufio.NewScanner(strings.NewReader(payload))
	for rd.Scan() {
		line := strings.TrimSpace(rd.Text())
		if line == "" {
			continue
		}
		var obj jsonlItem
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// 容错：如果是数组/对象整体，尝试解析为切片或对象
			var arr []jsonlItem
			if e2 := json.Unmarshal([]byte(line), &arr); e2 == nil {
				out = append(out, arr...)
				continue
			}
			continue // 跳过坏行
		}
		if strings.TrimSpace(obj.ID) == "" {
			continue
		}
		out = append(out, obj)
	}
	if len(out) > 0 {
		return out, nil
	}
	// Fallback: try to parse the whole payload as a single JSON object/array
	if its, e := parseBatchJSONItems(payload); e == nil && len(its) > 0 {
		return its, nil
	}
	return nil, fmt.Errorf("no jsonl items parsed")
}

// parseBatchJSONItems 解析 JSON 模式输出（单对象 items 或直接数组）
func parseBatchJSONItems(s string) ([]jsonlItem, error) {
	txt := normalizeLLMPayload(s)
	if txt == "" {
		return nil, fmt.Errorf("empty content")
	}
	type wrapper struct {
		Items []jsonlItem `json:"items"`
	}
	var w wrapper
	if err := json.Unmarshal([]byte(txt), &w); err == nil && len(w.Items) > 0 {
		return w.Items, nil
	}
	var arr []jsonlItem
	if err := json.Unmarshal([]byte(txt), &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}
	// Try to be forgiving if provider wrapped with extra object
	var obj map[string]any
	if err := json.Unmarshal([]byte(txt), &obj); err == nil {
		// search first array-like field
		for _, v := range obj {
			if b, e := json.Marshal(v); e == nil {
				var arr2 []jsonlItem
				if e2 := json.Unmarshal(b, &arr2); e2 == nil && len(arr2) > 0 {
					return arr2, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("no items parsed from JSON")
}

// TranslateSubtitleLLMBatchedWithAnalysis 执行“初始化分析 + 分批 JSONL 自我反思翻译”。
// 注意：不替代现有单次大批量路径，作为可选新流程并行存在。
// strictGlossary: 若为 true，则不在 glossary 中暴露占位符映射，仅要求模型严格保留占位符本身；
//                 若为 false，则将命中的 enforce 术语附带 placeholder 字段，作为 hint 模式。
func (s *Service) TranslateSubtitleLLMBatchedWithAnalysis(projectID, originLang, targetLang, providerID, model string, setIDs []string, extraEntries []types.GlossaryEntry, batchSize int, profileID string, filter func(*types.SubtitleSegment) bool, strictGlossary bool) error {
	if batchSize <= 0 {
		batchSize = 50
	}
	if strings.EqualFold(strings.TrimSpace(originLang), strings.TrimSpace(targetLang)) {
		return fmt.Errorf("origin and target languages cannot be the same")
	}
	// 读取项目
	proj, err := s.boltStorage.GetSubtitle(projectID)
	if err != nil {
		return err
	}
	if proj == nil || len(proj.Segments) == 0 {
		return fmt.Errorf("project/segments empty")
	}

	// 组装 profile（可选）
	var prof *types.LLMProfile
	if strings.TrimSpace(profileID) != "" {
		gp, e := s.boltStorage.GetGlobalProfile(profileID)
		if e != nil {
			return e
		}
		prof = &types.LLMProfile{ID: gp.ID, Temperature: gp.Temperature, TopP: gp.TopP, JSONMode: gp.JSONMode, SysPromptTpl: gp.SysPromptTpl, MaxTokens: gp.MaxTokens, Metadata: gp.Metadata}
	}

	// 初始化任务信息（与单次路径一致）
	provRec, err := s.boltStorage.GetProvider(providerID)
	if err != nil {
		return err
	}
	taskID := uuidStr()
	// Determine task-level totals. For failed-only retries, limit to filtered subset but also expose project-level totals.
	totalProject := len(proj.Segments)
	totalTask := totalProject
	preCompleted := 0
	if filter != nil {
		cnt := 0
		for i := range proj.Segments {
			if filter(&proj.Segments[i]) {
				cnt++
			}
		}
		totalTask = cnt
		preCompleted = totalProject - totalTask
	}
	conv := types.ConversionTask{
		ID: taskID, Type: "llm_translate", Status: types.ConversionStatusProcessing, Progress: 0, StartTime: time.Now().Unix(),
		SourceLang: originLang, TargetLang: targetLang, Provider: provRec.Name, ProviderID: provRec.ID, Model: model,
		TotalSegments:            totalTask,
		ProjectTotalSegments:     totalProject,
		ProjectCompletedSegments: preCompleted,
	}
	if proj.LanguageMetadata == nil {
		proj.LanguageMetadata = make(map[string]types.LanguageMetadata)
	}
	srcLangName := s.resolveLanguageName(originLang, proj)
	targetLangName := s.resolveLanguageName(targetLang, proj)
	meta := proj.LanguageMetadata[targetLang]
	if strings.TrimSpace(meta.LanguageName) == "" {
		if targetLangName != "" {
			meta.LanguageName = targetLangName
		} else {
			meta.LanguageName = targetLang
		}
	}
	meta.Translator = fmt.Sprintf("llm/%s", provRec.Name)
	meta.SyncStatus = "translating"
	meta.ActiveTaskID = taskID
	// 目标语言一定不是原始语言
	meta.Status.IsOriginal = false
	meta.Status.LastUpdated = time.Now().Unix()
	meta.Status.ConversionTasks = append(meta.Status.ConversionTasks, conv)
	proj.LanguageMetadata[targetLang] = meta
	promptSrcLabel := promptLangLabel(originLang, srcLangName)
	promptDstLabel := promptLangLabel(targetLang, targetLangName)
	// 首次持久化前，同步可能已有的会话数据，避免覆盖
	s.syncLLMConversationsFromStore(projectID, proj)
	if err := s.boltStorage.SaveSubtitle(proj); err != nil {
		return err
	}
	conv.Stage = "analysis_started"
	conv.StageDetail = "project_overview"
	s.pushLlmProgress(conv)
	s.persistTaskProgress(projectID, targetLang, conv)
	// 会话起始：记录一次元信息消息
	s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleApp), "meta",
		fmt.Sprintf("Start LLM subtitle translation: %s → %s (provider=%s, model=%s, total_segments=%d)", originLang, targetLang, provRec.Name, model, totalTask),
		conv.Stage, map[string]any{"project_total": totalProject, "task_total": totalTask})

	// 异步执行
	go func() {
		// diagnostics aggregator for rich error output
		var errBuf strings.Builder
		logErr := func(msg string) {
			ts := time.Now().Format(time.RFC3339)
			errBuf.WriteString(ts)
			errBuf.WriteString(" - ")
			errBuf.WriteString(msg)
			errBuf.WriteString("\n")
		}
		failedIDsSample := make([]string, 0, 10)

		// 1) 项目级分析（若不存在则生成）
		s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleApp), "meta",
			"Analyze project structure and style before translation (project-level overview).",
			"analysis_start", nil)
		analysis, analysisUsage, aerr := s.EnsureProjectAnalysis(s.ctx, proj, originLang, providerID, model, prof)
		if aerr != nil {
			logger.Warn("analysis failed, continue without it", zap.Error(aerr))
			logErr(fmt.Sprintf("analysis failed: %v", aerr))
			s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleProvider), "error",
				fmt.Sprintf("Project analysis failed: %v", aerr),
				"analysis_error", nil)
		}
		conv.Stage = "analysis_done"
		conv.StageDetail = "project_overview"
		// accumulate analysis token usage
		conv.PromptTokens += analysisUsage.PromptTokens
		conv.CompletionTokens += analysisUsage.CompletionTokens
		conv.TotalTokens += analysisUsage.TotalTokens
		if analysisUsage.TotalTokens > 0 || analysisUsage.PromptTokens > 0 || analysisUsage.CompletionTokens > 0 {
			conv.RequestCount++
		}
		if analysisUsage.TotalTokens > 0 || analysisUsage.PromptTokens > 0 || analysisUsage.CompletionTokens > 0 {
			s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleProvider), "meta",
				fmt.Sprintf("Project analysis completed. Tokens: prompt=%d, completion=%d, total=%d.",
					analysisUsage.PromptTokens, analysisUsage.CompletionTokens, analysisUsage.TotalTokens),
				"analysis_done", nil)
		}
		// persist updated conv to storage so UI can recover after refresh mid-task
		s.persistTaskProgress(projectID, targetLang, conv)
		s.pushLlmProgress(conv)

		// 2) 收集术语（区分“强制掩码遵守”的 enforce 与“仅参考”的 prompt-only）
		isRetry := (filter != nil)
		// helper: convert value slice to pointer slice
		toPtrs := func(arr []types.GlossaryEntry) []*types.GlossaryEntry {
			out := make([]*types.GlossaryEntry, 0, len(arr))
			for i := range arr {
				e := arr[i]
				ee := e
				out = append(out, &ee)
			}
			return out
		}
		// 全局集合（始终严格遵守）：仅当调用方显式选择了 glossary 集合时才使用
		globalEnforce := make([]*types.GlossaryEntry, 0)
		if len(setIDs) > 0 {
			tmp := make([]*types.GlossaryEntry, 0)
			for _, sid := range setIDs {
				if ents, e := s.boltStorage.ListGlossaryEntriesBySet(sid); e == nil {
					tmp = append(tmp, ents...)
				}
			}
			globalEnforce = append(globalEnforce, tmp...)
		}
		// 初始术语（仅参考，不掩码）
		promptOnly := make([]*types.GlossaryEntry, 0)
		if analysis != nil && len(analysis.InitialGlossary) > 0 {
			for i := range analysis.InitialGlossary {
				e := analysis.InitialGlossary[i]
				ee := e
				promptOnly = append(promptOnly, &ee)
			}
		}
		// 组装 enforce 列表并记录来源（global/task）；auto 术语单独保留为 promptOnly。
		enforce := make([]*types.GlossaryEntry, 0)
		enforceOrigins := make([]string, 0)
		// helper: add enforce entry with origin
		addEnforce := func(e *types.GlossaryEntry, origin string) {
			if e == nil {
				return
			}
			enforce = append(enforce, e)
			enforceOrigins = append(enforceOrigins, origin)
		}
		for _, ge := range globalEnforce {
			addEnforce(ge, "global")
		}
		if isRetry {
			if len(extraEntries) > 0 {
				// extras 非空：覆盖项目级 TaskTerms，并作为 enforce
				proj.Metadata.TaskTerms = extraEntries
				s.syncLLMConversationsFromStore(projectID, proj)
				if err := s.boltStorage.SaveSubtitle(proj); err != nil {
					logger.Warn("save subtitle (persist extras to metadata on retry)", zap.Error(err))
				}
				for _, ptr := range toPtrs(extraEntries) {
					addEnforce(ptr, "task")
				}
			} else if len(proj.Metadata.TaskTerms) > 0 {
				for _, ptr := range toPtrs(proj.Metadata.TaskTerms) {
					addEnforce(ptr, "task")
				}
			}
		} else {
			// 新任务：仅在 extras 非空时写入并使用；不读取 ProjectMetadata 的缓存术语
			if len(extraEntries) > 0 {
				proj.Metadata.TaskTerms = extraEntries
				s.syncLLMConversationsFromStore(projectID, proj)
				if err := s.boltStorage.SaveSubtitle(proj); err != nil {
					logger.Warn("save subtitle (persist extras to metadata on new)", zap.Error(err))
				}
				for _, ptr := range toPtrs(extraEntries) {
					addEnforce(ptr, "task")
				}
			}
		}

		// 3) 批量切分
		segs := proj.Segments
		if filter != nil {
			tmp := make([]types.SubtitleSegment, 0, len(segs))
			for i := range segs {
				if filter(&segs[i]) {
					tmp = append(tmp, segs[i])
				}
			}
			segs = tmp
			conv.TotalSegments = len(segs)
			// recompute project aggregates only if not set
			if conv.ProjectTotalSegments == 0 {
				conv.ProjectTotalSegments = len(proj.Segments)
			}
			conv.ProjectCompletedSegments = conv.ProjectTotalSegments - conv.TotalSegments
		} else {
			// full run
			conv.ProjectTotalSegments = len(proj.Segments)
			conv.ProjectCompletedSegments = 0
		}
		total := len(segs)
		bat := func(i int) (st, ed int) {
			st = i
			ed = i + batchSize
			if ed > total {
				ed = total
			}
			return
		}
		processed := 0
		failed := 0
		// 用于 boundary 与示例的累积
		var prevSrcTail []string
		var prevDstTail []string
		styleExamples := make([]map[string]string, 0, 8)

		// JSON Mode 探测：仅在首批（且非失败重试）尝试一次；成功则后续沿用；失败则统一退回 JSONL
		jsonModeDetermined := false
		jsonModeOK := false

		aborted := false
		abortReason := ""
		for i := 0; i < total && !aborted; i += batchSize {
			st, ed := bat(i)
			batch := segs[st:ed]
			// batch index/meta for clear stage messages
			batchIndex := (i / batchSize) + 1
			batchCount := (total + batchSize - 1) / batchSize
			ids := make([]string, 0, len(batch))
			srcs := make([]string, 0, len(batch))
			for _, sg := range batch {
				ids = append(ids, sg.ID)
				srcs = append(srcs, strings.TrimSpace(sg.GetText(originLang)))
			}

			// 保护占位符 + 术语掩码
			genMasked, genRestore := s.applyGeneralProtectionBatch(srcs)
			gloMasked, gloRestore, usedGloss := s.applyGlossaryMaskBatch(genMasked, enforce, targetLang)

			// 组装全局风格摘要
			globalStyle := map[string]any{}
			if analysis != nil {
				if analysis.Genre != "" {
					globalStyle["genre"] = analysis.Genre
				}
				if analysis.Tone != "" {
					globalStyle["tone"] = analysis.Tone
				}
				if len(analysis.StyleGuide) > 0 {
					if len(analysis.StyleGuide) > 6 {
						globalStyle["style_guide"] = analysis.StyleGuide[:6]
					} else {
						globalStyle["style_guide"] = analysis.StyleGuide
					}
				}
			}
			// 构造本批次使用的 glossary：命中的 enforce + 少量 auto hints。
			// strictGlossary: 不在 glossary 中暴露 placeholder，仅作为 hint；占位符严格依赖系统提示 + 后处理。
			glossaryForPrompt := make([]promptGlossaryEntry, 0, len(enforce)+16)
			for idx, e := range enforce {
				if e == nil || idx >= len(usedGloss) || !usedGloss[idx] {
					continue
				}
				origin := ""
				if idx < len(enforceOrigins) {
					origin = enforceOrigins[idx]
				}
				entry := promptGlossaryEntry{
					ID:             e.ID,
					SetID:          e.SetID,
					Source:         e.Source,
					DoNotTranslate: e.DoNotTranslate,
					CaseSensitive:  e.CaseSensitive,
					Translations:   e.Translations,
					Origin:         origin,
					CreatedAt:      e.CreatedAt,
					UpdatedAt:      e.UpdatedAt,
				}
				if !strictGlossary {
					entry.Placeholder = fmt.Sprintf("⟦G%03d⟧", idx)
				}
				glossaryForPrompt = append(glossaryForPrompt, entry)
			}
			// 2) auto hints（最多 100 条），不附带占位符
			const maxAutoGloss = 100
			autoCount := 0
			for _, ge := range promptOnly {
				if ge == nil {
					continue
				}
				entry := promptGlossaryEntry{
					ID:             ge.ID,
					SetID:          ge.SetID,
					Source:         ge.Source,
					DoNotTranslate: ge.DoNotTranslate,
					CaseSensitive:  ge.CaseSensitive,
					Translations:   ge.Translations,
					Origin:         "auto",
					CreatedAt:      ge.CreatedAt,
					UpdatedAt:      ge.UpdatedAt,
				}
				glossaryForPrompt = append(glossaryForPrompt, entry)
				autoCount++
				if autoCount >= maxAutoGloss {
					break
				}
			}

			// 本批中需要请求 LLM 的行（不使用翻译缓存：每次任务都请求最新结果）
			idsAsk := make([]string, 0, len(ids))
			srcsAsk := make([]string, 0, len(srcs))
			gloMaskedAsk := make([]string, 0)
			for idx := range ids {
				idsAsk = append(idsAsk, ids[idx])
				srcsAsk = append(srcsAsk, srcs[idx])
			}
			// 选择与 profile 的 JSONMode 策略
			var items []jsonlItem
			useJSONMode := false
			if prof != nil {
				if prof.JSONMode {
					useJSONMode = true
					jsonModeDetermined = true
					jsonModeOK = true
				} else {
					useJSONMode = false
					jsonModeDetermined = true
					jsonModeOK = false
				}
			} else if filter == nil {
				if !jsonModeDetermined && i == 0 {
					useJSONMode = true
				} else if jsonModeDetermined && jsonModeOK {
					useJSONMode = true
				}
			}

			if len(idsAsk) > 0 { // 仅当存在未命中缓存的行时才请求 LLM
				conv.Stage = "batch_sending"
				conv.StageDetail = fmt.Sprintf("batch %d/%d, size %d items", batchIndex, batchCount, len(idsAsk))
				s.pushLlmProgress(conv)
				s.persistTaskProgress(projectID, targetLang, conv)
				// 从 gloMasked 中抽取对应的 masked 源文
				if len(gloMasked) == len(srcs) {
					// map id->index for quick lookup
					idxmap := make(map[string]int, len(ids))
					for k := range ids {
						idxmap[ids[k]] = k
					}
					for _, id := range idsAsk {
						if j, ok := idxmap[id]; ok {
							gloMaskedAsk = append(gloMaskedAsk, gloMasked[j])
						}
					}
				} else {
					gloMaskedAsk = append(gloMaskedAsk, srcsAsk...)
				}
				skipFallback := false
				if useJSONMode {
					sys, usr := BuildBatchJSONModePrompts(func() string {
						if prof != nil {
							return prof.SysPromptTpl
						}
						return ""
					}(), promptSrcLabel, promptDstLabel, globalStyle, glossaryForPrompt, styleExamples, prevSrcTail, prevDstTail, idsAsk, gloMaskedAsk)
					messages := []provider.ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: usr}}
					opts := provider.ChatOptions{}
					if prof != nil {
						opts.Temperature = prof.Temperature
						opts.TopP = prof.TopP
						opts.MaxTokens = prof.MaxTokens
					}
					opts.JSONMode = true
					// 记录应用侧请求（聚合级别，只写一次）
					preview := fmt.Sprintf("Batch %d/%d (%d items) [JSON mode]\n\nSystem:\n%s\n\nUser:\n%s", batchIndex, batchCount, len(idsAsk), sys, usr)
					s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleApp), "request",
						preview, "batch_json", map[string]any{"batch_index": batchIndex, "batch_count": batchCount, "items": len(idsAsk), "json_mode": true})
					// Attach streaming hook so front-end chat modal can display LLM output progressively（仅通过 WS 推送，不持久化每个 delta）。
					streamCtx := provider.WithChatStreamCallback(s.ctx, func(delta string) error {
						if strings.TrimSpace(delta) == "" {
							return nil
						}
						s.pushLLMStreamDelta(
							projectID, targetLang, taskID,
							provRec.Name, provRec.ID, model,
							string(types.LLMChatRoleProvider), "response",
							delta,
							"batch_json_stream",
							map[string]any{
								"batch_index": batchIndex,
								"batch_count": batchCount,
								"items":       len(idsAsk),
								"json_mode":   true,
								"delta":       true,
								"append":      true,
							},
						)
						return nil
					})
					content, u1, perr := func() (string, provider.TokenUsage, error) {
						return s.providerService.ChatCompletionWithOptionsUsage(streamCtx, providerID, model, messages, opts)
					}()
					// retry once on error
					if perr != nil {
						content2, u2, perr2 := s.providerService.ChatCompletionWithOptionsUsage(s.ctx, providerID, model, messages, opts)
						if perr2 == nil {
							content = content2
							u1 = u2
							perr = nil
						} else {
							perr = perr2
							logErr(fmt.Sprintf("json-mode request failed (retry) batch %d/%d: %v", batchIndex, batchCount, perr2))
						}
					}
					if perr == nil {
						// 流结束后：持久化一次聚合后的完整回复到会话记录
						s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleProvider), "response",
							content, "batch_json", map[string]any{
								"batch_index":       batchIndex,
								"batch_count":       batchCount,
								"items":             len(idsAsk),
								"json_mode":         true,
								"prompt_tokens":     u1.PromptTokens,
								"completion_tokens": u1.CompletionTokens,
								"total_tokens":      u1.TotalTokens,
							})
						if its, e := parseBatchJSONItems(content); e == nil {
							items = its
							jsonModeDetermined = true
							jsonModeOK = true
							// accumulate usage
							conv.PromptTokens += u1.PromptTokens
							conv.CompletionTokens += u1.CompletionTokens
							conv.TotalTokens += u1.TotalTokens
							if u1.TotalTokens > 0 || u1.PromptTokens > 0 || u1.CompletionTokens > 0 {
								conv.RequestCount++
							}
							// persist per-batch usage to storage
							s.persistTaskProgress(projectID, targetLang, conv)
						} else {
							logger.Warn("parse json (json-mode) failed", zap.Error(e))
							logErr(fmt.Sprintf("parse json (json-mode) failed batch %d/%d: %v", batchIndex, batchCount, e))
							jsonModeDetermined = true
							jsonModeOK = false
						}
					} else {
						logger.Warn("json-mode request failed", zap.Error(perr))
						logErr(fmt.Sprintf("json-mode request failed batch %d/%d: %v", batchIndex, batchCount, perr))
						// 记录错误
						s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleProvider), "error",
							fmt.Sprintf("JSON mode request failed after retry: %v", perr),
							"batch_json_error", map[string]any{"batch_index": batchIndex, "batch_count": batchCount, "items": len(idsAsk)})
						// 标记 JSON 模式不可用，后续将自动回退到 JSONL
						jsonModeDetermined = true
						jsonModeOK = false
					}
				}

				if aborted {
					break
				}
				if !useJSONMode || (jsonModeDetermined && !jsonModeOK) {
					// 回退或默认：JSONL
					if !skipFallback && len(idsAsk) > 0 {
						sys, usr := BuildBatchJSONLPrompts(func() string {
							if prof != nil {
								return prof.SysPromptTpl
							}
							return ""
						}(), promptSrcLabel, promptDstLabel, globalStyle, glossaryForPrompt, styleExamples, prevSrcTail, prevDstTail, idsAsk, gloMaskedAsk)
						messages := []provider.ChatMessage{{Role: "system", Content: sys}, {Role: "user", Content: usr}}
						opts := provider.ChatOptions{}
						if prof != nil {
							opts.Temperature = prof.Temperature
							opts.TopP = prof.TopP
							opts.MaxTokens = prof.MaxTokens
						} else {
							opts.Temperature = 0.2
						}
						// 记录 JSONL 回退请求（聚合级别，只写一次）
						preview := fmt.Sprintf("Batch %d/%d (%d items) [JSONL]\n\nSystem:\n%s\n\nUser:\n%s", batchIndex, batchCount, len(idsAsk), sys, usr)
						s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleApp), "request",
							preview, "batch_jsonl", map[string]any{"batch_index": batchIndex, "batch_count": batchCount, "items": len(idsAsk), "json_mode": false})
						// 流式 JSONL：仅通过 WS 推送 delta，不持久化逐个片段
						streamCtxJSONL := provider.WithChatStreamCallback(s.ctx, func(delta string) error {
							if strings.TrimSpace(delta) == "" {
								return nil
							}
							s.pushLLMStreamDelta(
								projectID, targetLang, taskID,
								provRec.Name, provRec.ID, model,
								string(types.LLMChatRoleProvider), "response",
								delta,
								"batch_jsonl_stream",
								map[string]any{
									"batch_index": batchIndex,
									"batch_count": batchCount,
									"items":       len(idsAsk),
									"json_mode":   false,
									"delta":       true,
									"append":      true,
								},
							)
							return nil
						})
						content, u1, perr := s.providerService.ChatCompletionWithOptionsUsage(streamCtxJSONL, providerID, model, messages, opts)
						if perr != nil {
							// retry once
							content2, u2, perr2 := s.providerService.ChatCompletionWithOptionsUsage(s.ctx, providerID, model, messages, opts)
							if perr2 == nil {
								content = content2
								u1 = u2
								perr = nil
							} else {
								perr = perr2
								logErr(fmt.Sprintf("request failed (retry) batch %d/%d: %v", batchIndex, batchCount, perr2))
							}
						}
						if perr != nil {
							logger.Warn("batch translate request failed", zap.Error(perr))
							logErr(fmt.Sprintf("request failed batch %d/%d: %v", batchIndex, batchCount, perr))
							s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleProvider), "error",
								fmt.Sprintf("Batch request failed after retry: %v", perr),
								"batch_jsonl_error", map[string]any{"batch_index": batchIndex, "batch_count": batchCount, "items": len(idsAsk)})
						}
						its, jerr := parseJSONL(content)
						if jerr != nil {
							logger.Warn("parse jsonl failed", zap.Error(jerr))
							logErr(fmt.Sprintf("parse jsonl failed batch %d/%d: %v", batchIndex, batchCount, jerr))
						}
						if perr == nil {
							// 流结束后：持久化一次聚合后的完整 JSONL 回复
							s.appendLLMChatMessage(projectID, targetLang, taskID, provRec.Name, provRec.ID, model, string(types.LLMChatRoleProvider), "response",
								content, "batch_jsonl", map[string]any{
									"batch_index":       batchIndex,
									"batch_count":       batchCount,
									"items":             len(idsAsk),
									"json_mode":         false,
									"prompt_tokens":     u1.PromptTokens,
									"completion_tokens": u1.CompletionTokens,
									"total_tokens":      u1.TotalTokens,
								})
						}
						items = its
						// accumulate usage
						conv.PromptTokens += u1.PromptTokens
						conv.CompletionTokens += u1.CompletionTokens
						conv.TotalTokens += u1.TotalTokens
						if u1.TotalTokens > 0 || u1.PromptTokens > 0 || u1.CompletionTokens > 0 {
							conv.RequestCount++
						}
						// persist per-batch usage to storage
						s.persistTaskProgress(projectID, targetLang, conv)
					} else {
						// 跳过回退请求：保持 items 为空，稍后为本批所有行标记错误
					}
				}
			}
			// 若本批在重试后仍无任何可用输出且存在未命中缓存的请求，直接中止任务
			if len(items) == 0 && len(idsAsk) > 0 {
				aborted = true
				if abortReason == "" {
					abortReason = "request_failed_or_no_output_after_retry"
				}
				break
			}
			// 阶段：收到返回并解析（items/got）
			conv.Stage = "batch_received"
			conv.StageDetail = fmt.Sprintf("batch %d/%d, got %d items", batchIndex, batchCount, len(items))
			s.pushLlmProgress(conv)
			s.persistTaskProgress(projectID, targetLang, conv)
			// 索引
			got := make(map[string]jsonlItem)
			for _, it := range items {
				got[strings.TrimSpace(it.ID)] = it
			}
			conv.Stage = "batch_parsed"
			conv.StageDetail = fmt.Sprintf("batch %d/%d, parsed %d items", batchIndex, batchCount, len(got))
			s.pushLlmProgress(conv)
			s.persistTaskProgress(projectID, targetLang, conv)

			// 写回 & 进度
			for idx := range ids {
				id := ids[idx]
				seg := s.findSegmentByID(proj, id)
				if seg == nil {
					continue
				}
				src := srcs[idx]
				out := ""
				if it, ok := got[id]; ok && strings.TrimSpace(it.Final) != "" {
					out = it.Final
				} else if it, ok := got[id]; ok && strings.TrimSpace(it.Draft) != "" {
					out = it.Draft
				}
				// 组装/更新目标语言内容（不再用源文回退）
				s.ensureLangMaps(seg)
				lc, existed := seg.Languages[targetLang]
				proc := &types.LanguageProcess{Provider: provRec.Name, Model: model, TaskID: taskID, UpdatedAt: time.Now().Unix()}
				if strings.TrimSpace(out) == "" {
					failed++
					if len(failedIDsSample) < 10 {
						failedIDsSample = append(failedIDsSample, id)
					}
					proc.Status = "error"
					proc.Error = "no_output_for_id"
					// 不修改已有译文；且如果原本不存在该语言，则不要创建空语言项，避免写入空文本
					if existed {
						lc.Process = proc
						seg.Languages[targetLang] = lc
					}
				} else {
					// 恢复占位符并写入目标语言文本
					outRestored := s.restoreGlossaryPlaceholders(out, gloRestore)
					outRestored = s.restoreGlossaryPlaceholders(outRestored, genRestore)
					lc.Text = outRestored
					// 设置默认的 Guideline 标准，确保质量评估生效
					if seg.GuidelineStandard == nil {
						seg.GuidelineStandard = make(map[string]types.GuideLineStandard)
					}
					if _, ok := seg.GuidelineStandard[targetLang]; !ok || string(seg.GuidelineStandard[targetLang]) == "" {
						seg.GuidelineStandard[targetLang] = types.GuideLineStandardNetflix
					}
					proc.Status = "ok"
					lc.Process = proc
					seg.Languages[targetLang] = lc
				}
				*seg = *s.qualityAssessor.AssessSegmentQuality(seg)
				processed++

				// 用已译作为示例候选（限制 8 个）
				if len(styleExamples) < 8 && strings.TrimSpace(lc.Text) != "" {
					styleExamples = append(styleExamples, map[string]string{"src": src, "dst": lc.Text})
				}

				// 每 20 条刷新一次
				if processed%20 == 0 {
					conv.ProcessedSegments = processed
					conv.FailedSegments = failed
					if conv.TotalSegments > 0 {
						conv.Progress = float64(conv.ProcessedSegments) / float64(conv.TotalSegments) * 100
					}
					s.syncLLMConversationsFromStore(projectID, proj)
					if err := s.boltStorage.SaveSubtitle(proj); err != nil {
						logger.Warn("save subtitle (batched) mid", zap.Error(err))
					}
					s.pushLlmProgress(conv)
				}
			}

			// 更新边界上下文（取本批最后 5 条）
			n := len(ids)
			tail := 5
			if tail > n {
				tail = n
			}
			prevSrcTail = idsToSrc(srcs[n-tail:])
			prevDstTail = collectDst(proj, targetLang, ids[n-tail:])

			// 批末也刷新一次，照顾小批量任务
			conv.ProcessedSegments = processed
			conv.FailedSegments = failed
			if conv.TotalSegments > 0 {
				conv.Progress = float64(conv.ProcessedSegments) / float64(conv.TotalSegments) * 100
			}
			s.syncLLMConversationsFromStore(projectID, proj)
			if err := s.boltStorage.SaveSubtitle(proj); err != nil {
				logger.Warn("save subtitle (batched) batch-end", zap.Error(err))
			}
			conv.Stage = "batch_applied"
			conv.StageDetail = fmt.Sprintf("batch %d/%d, processed=%d failed=%d", batchIndex, batchCount, processed, failed)
			s.pushLlmProgress(conv)
			s.persistTaskProgress(projectID, targetLang, conv)
			// 在 LLM 返回后记录应用端对结果的处理过程（写入字幕/质量评估等）
			s.appendLLMChatMessage(
				projectID, targetLang, taskID, provRec.Name, provRec.ID, model,
				string(types.LLMChatRoleApp), "meta",
				fmt.Sprintf("Applied batch %d/%d to subtitles (processed=%d, failed=%d).", batchIndex, batchCount, processed, failed),
				conv.Stage,
				map[string]any{
					"batch_index": batchIndex,
					"batch_count": batchCount,
					"processed":   processed,
					"failed":      failed,
				},
			)
		}

		// 最终保存与收尾
		s.syncLLMConversationsFromStore(projectID, proj)
		if err := s.boltStorage.SaveSubtitle(proj); err != nil {
			logger.Warn("save subtitle (batched) final", zap.Error(err))
		}
		conv.ProcessedSegments = processed
		conv.FailedSegments = failed
		if aborted {
			conv.Status = types.ConversionStatusFailed
			if conv.ErrorMessage == "" {
				conv.ErrorMessage = abortReason
			}
			conv.Stage = "aborted"
			conv.StageDetail = abortReason
			// attach diagnostics
			if strings.TrimSpace(conv.ErrorMessage) != "" {
				conv.ErrorMessage = conv.ErrorMessage + "\n" + errBuf.String()
			} else {
				conv.ErrorMessage = errBuf.String()
			}
		} else {
			conv.Progress = 100
			if failed > 0 {
				conv.Status = types.ConversionStatusFailed
			} else {
				conv.Status = types.ConversionStatusCompleted
			}
			// Provide an aggregated reason when some segments failed but task finished (partial_failed)
			if failed > 0 && (conv.ErrorMessage == "" || len(strings.TrimSpace(conv.ErrorMessage)) == 0) {
				summary := fmt.Sprintf("partial_failed: %d segments did not produce output\nProcessed=%d Failed=%d\n", failed, processed, failed)
				samples := ""
				if len(failedIDsSample) > 0 {
					samples = fmt.Sprintf("FailedIDs(sample)=%s\n", strings.Join(failedIDsSample, ", "))
				}
				conv.ErrorMessage = summary + samples + errBuf.String()
			}
		}
		conv.EndTime = time.Now().Unix()
		// 写回元数据状态
		p2, _ := s.boltStorage.GetSubtitle(projectID)
		if p2 == nil {
			p2 = proj
		}
		m := p2.LanguageMetadata[targetLang]
		m.Revision++
		if aborted {
			m.SyncStatus = "failed"
		} else if failed > 0 {
			m.SyncStatus = "partial_failed"
		} else {
			m.SyncStatus = "done"
		}
		m.ActiveTaskID = ""
		if len(m.Status.ConversionTasks) > 0 {
			for idx := range m.Status.ConversionTasks {
				if m.Status.ConversionTasks[idx].ID == taskID {
					m.Status.ConversionTasks[idx] = conv
					break
				}
			}
		}
		m.Status.LastUpdated = time.Now().Unix()
		p2.LanguageMetadata[targetLang] = m
		_ = s.boltStorage.SaveSubtitle(p2)
		s.pushLlmProgress(conv)
		// 更新会话最终状态 + 任务总结
		if aborted {
			// 任务中止：记录一次 app 侧任务总结消息
			s.appendLLMChatMessage(
				projectID, targetLang, taskID, provRec.Name, provRec.ID, model,
				string(types.LLMChatRoleApp), "error",
				fmt.Sprintf("Translation aborted: %s (processed=%d, failed=%d).", abortReason, processed, failed),
				"aborted",
				map[string]any{
					"processed": processed,
					"failed":    failed,
					"reason":    abortReason,
				},
			)
			s.markLLMConversationFinished(projectID, targetLang, taskID, types.LLMConversationStatusFailed)
		} else {
			// 任务完成：记录一次 app 侧任务总结消息（包括部分失败的情况）
			summaryStatus := "completed"
			if failed > 0 {
				summaryStatus = "partial_failed"
			}
			s.appendLLMChatMessage(
				projectID, targetLang, taskID, provRec.Name, provRec.ID, model,
				string(types.LLMChatRoleApp), "meta",
				fmt.Sprintf("Translation %s. Segments: processed=%d, failed=%d.", summaryStatus, processed, failed),
				"completed",
				map[string]any{
					"processed": processed,
					"failed":    failed,
					"status":    summaryStatus,
				},
			)
			finalConvStatus := types.LLMConversationStatusFinished
			if conv.Status == types.ConversionStatusFailed {
				finalConvStatus = types.LLMConversationStatusFailed
			}
			s.markLLMConversationFinished(projectID, targetLang, taskID, finalConvStatus)
		}
		if aborted {
			logger.Warn("LLM translate (batched) aborted", zap.Int("processed", processed), zap.Int("failed", failed), zap.String("reason", abortReason))
		} else {
			logger.Info("LLM translate (batched) finished", zap.Int("processed", processed), zap.Int("failed", failed))
		}
	}()

	return nil
}

// 小工具
func uuidStr() string { return fmt.Sprintf("%d-%s", time.Now().UnixNano(), RandString(6)) }

// idsToSrc 保持简洁字符串（不带 id），用于 boundary prev_src
func idsToSrc(srcs []string) []string { out := make([]string, len(srcs)); copy(out, srcs); return out }

func collectDst(p *types.SubtitleProject, lang string, ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if seg := findByID(&p.Segments, id); seg != nil {
			out = append(out, strings.TrimSpace(seg.Languages[lang].Text))
		}
	}
	return out
}

func findByID(arr *[]types.SubtitleSegment, id string) *types.SubtitleSegment {
	if arr == nil {
		return nil
	}
	for i := range *arr {
		if (*arr)[i].ID == id {
			return &((*arr)[i])
		}
	}
	return nil
}

// RandString 生成随机字符串（仅用于 uuidStr 的尾缀，避免与现有 uuid 冲突）
func RandString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[int(time.Now().UnixNano()+int64(i))%len(letters)]
	}
	return string(b)
}
