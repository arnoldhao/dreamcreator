package subtitles

import (
    "dreamcreator/backend/pkg/events"
    "dreamcreator/backend/pkg/provider"
    "dreamcreator/backend/pkg/proxy"
    "dreamcreator/backend/pkg/zhconvert"
    "dreamcreator/backend/storage"
    "dreamcreator/backend/types"

    "context"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

type Service struct {
    ctx context.Context
    // 使用接口类型
    formatConverter FormatConverter
    textProcessor   TextProcessor
    qualityAssessor QualityAssessor

    // bolt storage
    boltStorage *storage.BoltStorage
    // proxy
    proxyManager proxy.ProxyManager
    // zhconvert
    zhConverter *zhconvert.Converter
    // 事件总线
    eventBus events.EventBus

    // LLM provider service (for AI translation)
    providerService ProviderService
}

// ProviderService abstracts LLM chat completion so subtitles module doesn't own HTTP details
type ProviderService interface{
    ChatCompletion(ctx context.Context, providerID, model string, messages []provider.ChatMessage, temperature float64) (string, error)
    ChatCompletionWithOptions(ctx context.Context, providerID, model string, messages []provider.ChatMessage, opts provider.ChatOptions) (string, error)
    ChatCompletionWithOptionsUsage(ctx context.Context, providerID, model string, messages []provider.ChatMessage, opts provider.ChatOptions) (string, provider.TokenUsage, error)
}

func NewService(boltStorage *storage.BoltStorage, proxyManager proxy.ProxyManager, eventBus events.EventBus, providerSvc ProviderService) *Service {
    return &Service{
        formatConverter: NewFormatConverter(),
        textProcessor:   NewTextProcessor(),
        qualityAssessor: NewQualityAssessor(),
        boltStorage:     boltStorage,
        proxyManager:    proxyManager,
        zhConverter:     zhconvert.New(zhconvert.DefaultConfig(), proxyManager),
        eventBus:        eventBus,
        providerService: providerSvc,
    }
}

func (s *Service) SetContext(ctx context.Context) {
    s.ctx = ctx
}

// persistTaskProgress updates the active task record for a target language in storage.
// Best-effort: logs are kept minimal to avoid noisy output during tight loops.
func (s *Service) persistTaskProgress(projectID, targetLang string, conv types.ConversionTask) {
    if s.boltStorage == nil { return }
    if strings.TrimSpace(projectID) == "" || strings.TrimSpace(targetLang) == "" { return }
    p, err := s.boltStorage.GetSubtitle(projectID)
    if err != nil || p == nil { return }
    m := p.LanguageMetadata[targetLang]
    if len(m.Status.ConversionTasks) > 0 {
        for i := range m.Status.ConversionTasks {
            if m.Status.ConversionTasks[i].ID == conv.ID {
                m.Status.ConversionTasks[i] = conv
                break
            }
        }
        m.Status.LastUpdated = time.Now().Unix()
        p.LanguageMetadata[targetLang] = m
        _ = s.boltStorage.SaveSubtitle(p)
    }
}

// -------- LLM Tasks history (aggregated from projects) --------

// ListLLMTasks lists all subtitle translation tasks (LLM and zhconvert) across all projects.
// If projectID is not empty, filters by project.
// Note: Although the name is kept for API compatibility, this now includes both
//       Type == "llm_translate" and Type == "zhconvert" tasks.
func (s *Service) ListLLMTasks(projectID string) ([]*types.LLMTaskView, error) {
    if s.boltStorage == nil { return nil, fmt.Errorf("bolt storage is nil") }
    subs, err := s.boltStorage.ListSubtitles()
    if err != nil { return nil, err }
    out := make([]*types.LLMTaskView, 0, 64)
    for _, p := range subs {
        if p == nil { continue }
        if strings.TrimSpace(projectID) != "" && p.ID != projectID { continue }
        pname := p.ProjectName
        if pname == "" { pname = p.Metadata.Name }
        for lang, meta := range p.LanguageMetadata {
            _ = lang // kept for future use; task contains Source/Target
            tasks := meta.Status.ConversionTasks
            for i := range tasks {
                t := tasks[i]
                // Include both AI translation and zhconvert tasks
                tt := strings.TrimSpace(strings.ToLower(t.Type))
                if tt != "llm_translate" && tt != "zhconvert" { continue }
                view := &types.LLMTaskView{ ConversionTask: t, ProjectID: p.ID, ProjectName: pname }
                out = append(out, view)
            }
        }
    }
    // sort by StartTime desc
    sort.Slice(out, func(i, j int) bool { return out[i].StartTime > out[j].StartTime })
    return out, nil
}

// GetLLMTask returns one task view by id
func (s *Service) GetLLMTask(taskID string) (*types.LLMTaskView, error) {
    if strings.TrimSpace(taskID) == "" { return nil, fmt.Errorf("task id is empty") }
    list, err := s.ListLLMTasks("")
    if err != nil { return nil, err }
    for _, v := range list { if v != nil && v.ID == taskID { return v, nil } }
    return nil, fmt.Errorf("llm task not found: %s", taskID)
}

// DeleteLLMTask deletes a historical task by id. Refuses to delete when the task is still processing.
func (s *Service) DeleteLLMTask(taskID string) error {
    if strings.TrimSpace(taskID) == "" { return fmt.Errorf("task id is empty") }
    subs, err := s.boltStorage.ListSubtitles()
    if err != nil { return err }
    for _, p := range subs {
        if p == nil { continue }
        changed := false
        for lang, meta := range p.LanguageMetadata {
            tasks := meta.Status.ConversionTasks
            idx := -1
            for i := range tasks {
                if tasks[i].ID == taskID {
                    if tasks[i].Status == types.ConversionStatusProcessing || meta.ActiveTaskID == taskID {
                        return fmt.Errorf("cannot delete processing task: %s", taskID)
                    }
                    idx = i
                    break
                }
            }
            if idx >= 0 {
                // remove idx
                meta.Status.ConversionTasks = append(tasks[:idx], tasks[idx+1:]...)
                p.LanguageMetadata[lang] = meta
                changed = true
                break
            }
        }
        if changed {
            return s.boltStorage.SaveSubtitle(p)
        }
    }
    return fmt.Errorf("llm task not found: %s", taskID)
}

// RetryLLMTaskFailedOnly restarts translation for failed/fallback segments using provided provider/model based on task id.
func (s *Service) RetryLLMTaskFailedOnly(taskID, providerID, model string) error {
    if strings.TrimSpace(taskID) == "" { return fmt.Errorf("task id is empty") }
    if strings.TrimSpace(providerID) == "" || strings.TrimSpace(model) == "" { return fmt.Errorf("providerID/model is empty") }
    // locate project/origin/target by task id
    subs, err := s.boltStorage.ListSubtitles()
    if err != nil { return err }
    for _, p := range subs {
        if p == nil { continue }
        for lang, meta := range p.LanguageMetadata {
            _ = lang
            for i := range meta.Status.ConversionTasks {
                t := meta.Status.ConversionTasks[i]
                if t.ID == taskID {
                    // Use failed-only variant with no glossary options (hint glossary mode by default)
                    return s.TranslateSubtitleLLMFailedOnlyWithOptions(p.ID, t.SourceLang, t.TargetLang, providerID, model, nil, nil, false)
                }
            }
        }
	}
	return fmt.Errorf("llm task not found: %s", taskID)
}

// GetLLMConversationForLanguage 返回指定项目 + 目标语言下最近一次 LLM 翻译任务的会话。
// 选择策略：
// - 若该语言存在 ActiveTaskID，则优先返回该任务的会话；
// - 否则在 ConversionTasks 中按时间/追加顺序选取最近一次 Type == "llm_translate" 的任务；
// - 若尚未有会话记录，则基于 ConversionTask 构造一个空会话骨架并持久化。
func (s *Service) GetLLMConversationForLanguage(projectID, lang string) (*types.LLMConversation, error) {
	if strings.TrimSpace(projectID) == "" || strings.TrimSpace(lang) == "" {
		return nil, fmt.Errorf("projectID or lang is empty")
	}
	proj, err := s.boltStorage.GetSubtitle(projectID)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}
	meta, ok := proj.LanguageMetadata[lang]
	if !ok {
		return nil, fmt.Errorf("language metadata not found: %s", lang)
	}

	// 决定使用哪个任务 ID（仅针对 LLM 翻译任务：Type == llm_translate）
	var taskID string
	active := strings.TrimSpace(meta.ActiveTaskID)
	if active != "" {
		// 仅当 ActiveTaskID 对应的任务为 llm_translate 时才使用
		for i := range meta.Status.ConversionTasks {
			t := meta.Status.ConversionTasks[i]
			if t.ID == active && strings.EqualFold(strings.TrimSpace(t.Type), "llm_translate") {
				taskID = active
				break
			}
		}
	}
	if taskID == "" {
		// 从历史任务中找最近的一次 llm_translate 任务
		for i := len(meta.Status.ConversionTasks) - 1; i >= 0; i-- {
			t := meta.Status.ConversionTasks[i]
			if strings.EqualFold(strings.TrimSpace(t.Type), "llm_translate") {
				taskID = t.ID
				break
			}
		}
	}
	if taskID == "" {
		return nil, fmt.Errorf("no llm_translate task for language: %s", lang)
	}

	if meta.Status.LLMConversations == nil {
		meta.Status.LLMConversations = make(map[string]types.LLMConversation)
	}
	if conv, ok := meta.Status.LLMConversations[taskID]; ok {
		// 尝试与任务状态对齐
		for i := range meta.Status.ConversionTasks {
			t := meta.Status.ConversionTasks[i]
			if t.ID != taskID {
				continue
			}
			switch t.Status {
			case types.ConversionStatusCompleted:
				conv.Status = types.LLMConversationStatusFinished
				if conv.EndedAt == 0 {
					conv.EndedAt = t.EndTime
				}
			case types.ConversionStatusFailed, types.ConversionStatusCancelled:
				conv.Status = types.LLMConversationStatusFailed
				if conv.EndedAt == 0 {
					conv.EndedAt = t.EndTime
				}
			default:
				conv.Status = types.LLMConversationStatusRunning
			}
			meta.Status.LLMConversations[taskID] = conv
			proj.LanguageMetadata[lang] = meta
			_ = s.boltStorage.SaveSubtitle(proj)
			break
		}
		convCopy := conv
		return &convCopy, nil
	}

	// 尚无会话记录：基于 ConversionTask 构造骨架
	var base *types.ConversionTask
	for i := range meta.Status.ConversionTasks {
		t := meta.Status.ConversionTasks[i]
		if t.ID == taskID {
			base = &t
			break
		}
	}
	if base == nil {
		return nil, fmt.Errorf("task not found for conversation: %s", taskID)
	}
	status := types.LLMConversationStatusRunning
	switch base.Status {
	case types.ConversionStatusCompleted:
		status = types.LLMConversationStatusFinished
	case types.ConversionStatusFailed, types.ConversionStatusCancelled:
		status = types.LLMConversationStatusFailed
	default:
		status = types.LLMConversationStatusRunning
	}
	conv := types.LLMConversation{
		ID:        taskID,
		ProjectID: projectID,
		Language:  lang,
		TaskID:    taskID,
		Provider:  base.Provider,
		Model:     base.Model,
		Status:    status,
		StartedAt: base.StartTime,
		EndedAt:   base.EndTime,
	}
	meta.Status.LLMConversations[taskID] = conv
	proj.LanguageMetadata[lang] = meta
	if err := s.boltStorage.SaveSubtitle(proj); err != nil {
		return nil, err
	}
	convCopy := conv
	return &convCopy, nil
}

// -------- Glossary helpers (CRUD via storage) --------

// ListGlossary returns all glossary entries
func (s *Service) ListGlossary() ([]*types.GlossaryEntry, error) {
    if s.boltStorage == nil { return nil, fmt.Errorf("bolt storage is nil") }
    return s.boltStorage.ListGlossaryEntries()
}

func (s *Service) ListGlossaryBySet(setID string) ([]*types.GlossaryEntry, error) {
    if s.boltStorage == nil { return nil, fmt.Errorf("bolt storage is nil") }
    return s.boltStorage.ListGlossaryEntriesBySet(setID)
}

// UpsertGlossaryEntry creates or updates a glossary entry
func (s *Service) UpsertGlossaryEntry(e *types.GlossaryEntry) (*types.GlossaryEntry, error) {
    if e == nil { return nil, fmt.Errorf("entry is nil") }
    if strings.TrimSpace(e.Source) == "" { return nil, fmt.Errorf("source is empty") }
    if e.Translations == nil { e.Translations = map[string]string{} }
    // Enforce single translation entry when provided (prefer 'all' if exists)
    if len(e.Translations) > 1 {
        if v, ok := e.Translations["all"]; ok {
            e.Translations = map[string]string{"all": v}
        } else if v, ok := e.Translations["*"]; ok {
            e.Translations = map[string]string{"*": v}
        } else {
            keys := make([]string, 0, len(e.Translations))
            for k := range e.Translations { keys = append(keys, k) }
            sort.Strings(keys)
            k := keys[0]
            e.Translations = map[string]string{k: e.Translations[k]}
        }
    }
    if e.DoNotTranslate {
        e.Translations = map[string]string{}
    }
    if strings.TrimSpace(e.ID) == "" {
        e.ID = fmt.Sprintf("glo_%d", time.Now().UnixNano())
        e.CreatedAt = time.Now().Unix()
    }
    if err := s.boltStorage.SaveGlossaryEntry(e); err != nil {
        return nil, err
    }
    return e, nil
}

// DeleteGlossaryEntry deletes one glossary entry by id
func (s *Service) DeleteGlossaryEntry(id string) error {
    if strings.TrimSpace(id) == "" { return fmt.Errorf("id is empty") }
    return s.boltStorage.DeleteGlossaryEntry(id)
}

// Glossary sets
func (s *Service) ListGlossarySets() ([]*types.GlossarySet, error) {
    if s.boltStorage == nil { return nil, fmt.Errorf("bolt storage is nil") }
    return s.boltStorage.ListGlossarySets()
}

func (s *Service) UpsertGlossarySet(gs *types.GlossarySet) (*types.GlossarySet, error) {
    if gs == nil { return nil, fmt.Errorf("glossary set nil") }
    if strings.TrimSpace(gs.ID) == "" {
        gs.ID = fmt.Sprintf("gls_%d", time.Now().UnixNano())
        gs.CreatedAt = time.Now().Unix()
    }
    if strings.TrimSpace(gs.Name) == "" { return nil, fmt.Errorf("name is empty") }
    if err := s.boltStorage.SaveGlossarySet(gs); err != nil { return nil, err }
    return gs, nil
}

func (s *Service) DeleteGlossarySet(id string) error {
    if strings.TrimSpace(id) == "" { return fmt.Errorf("id is empty") }
    return s.boltStorage.DeleteGlossarySet(id)
}

// -------- Target Languages (CRUD via storage) --------

// default list used to bootstrap storage when empty
var defaultTargetLanguages = []types.TargetLanguage{
	{ Code: "en", Name: "English" },
	{ Code: "zh-CN", Name: "简体中文" },
	{ Code: "zh-TW", Name: "繁體中文" },
	{ Code: "ja", Name: "日本語" },
	{ Code: "ko", Name: "한국어" },
	{ Code: "fr", Name: "Français" },
	{ Code: "de", Name: "Deutsch" },
	{ Code: "es", Name: "Español" },
	{ Code: "ru", Name: "Русский" },
	{ Code: "vi", Name: "Tiếng Việt" },
	{ Code: "pt-BR", Name: "Português (Brasil)" },
	{ Code: "pt-PT", Name: "Português (Portugal)" },
	{ Code: "id", Name: "Bahasa Indonesia" },
	{ Code: "hi", Name: "हिन्दी" },
	{ Code: "ar", Name: "العربية" },
	{ Code: "it", Name: "Italiano" },
	{ Code: "tr", Name: "Türkçe" },
	{ Code: "th", Name: "ไทย" },
	{ Code: "nl", Name: "Nederlands" },
	{ Code: "pl", Name: "Polski" },
}

func (s *Service) ListTargetLanguages() ([]*types.TargetLanguage, error) {
    if s.boltStorage == nil { return nil, fmt.Errorf("bolt storage is nil") }
    list, err := s.boltStorage.ListTargetLanguages()
    if err != nil { return nil, err }
    if len(list) == 0 {
        // bootstrap with defaults
        for i := range defaultTargetLanguages {
            tl := defaultTargetLanguages[i]
            now := time.Now().Unix()
            tl.CreatedAt, tl.UpdatedAt = now, now
            _ = s.boltStorage.SaveTargetLanguage(&tl)
        }
        return s.boltStorage.ListTargetLanguages()
    }
    return list, nil
}

func (s *Service) UpsertTargetLanguage(l *types.TargetLanguage) (*types.TargetLanguage, error) {
    if l == nil { return nil, fmt.Errorf("target language is nil") }
    l.Code = strings.TrimSpace(l.Code)
    l.Name = strings.TrimSpace(l.Name)
    if l.Code == "" { return nil, fmt.Errorf("code is empty") }
    if l.Name == "" { l.Name = l.Code }
    if s.boltStorage == nil { return nil, fmt.Errorf("bolt storage is nil") }
    if err := s.boltStorage.SaveTargetLanguage(l); err != nil { return nil, err }
    return l, nil
}

func (s *Service) DeleteTargetLanguage(code string) error {
    code = strings.TrimSpace(code)
    if code == "" { return fmt.Errorf("code is empty") }
    if s.boltStorage == nil { return fmt.Errorf("bolt storage is nil") }
    return s.boltStorage.DeleteTargetLanguage(code)
}

// ResetTargetLanguagesToDefault clears current list and restores defaults
func (s *Service) ResetTargetLanguagesToDefault() error {
    if s.boltStorage == nil { return fmt.Errorf("bolt storage is nil") }
    // delete all existing
    list, err := s.boltStorage.ListTargetLanguages()
    if err != nil { return err }
    for _, l := range list {
        if l != nil && strings.TrimSpace(l.Code) != "" {
            _ = s.boltStorage.DeleteTargetLanguage(l.Code)
        }
    }
    // reinsert defaults
    now := time.Now().Unix()
    for i := range defaultTargetLanguages {
        tl := defaultTargetLanguages[i]
        tl.CreatedAt, tl.UpdatedAt = now, now
        if err := s.boltStorage.SaveTargetLanguage(&tl); err != nil { return err }
    }
    return nil
}

// 统一错误处理
func (s *Service) handleError(operation string, err error) error {
	if err != nil {
		return fmt.Errorf("subtitle service %s failed: %w", operation, err)
	}
	return nil
}

// UpdateExportConfig 更新导出配置
func (s *Service) UpdateExportConfig(id string, config types.ExportConfigs) (*types.SubtitleProject, error) {
	// 1. validate input
	if id == "" {
		return nil, s.handleError("update export config", fmt.Errorf("id is empty"))
	}

	// 2. get project
	if s.boltStorage == nil {
		return nil, s.handleError("update export config", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update export config", err)
	}

	// 3. 验证和处理FCPXML配置
	if config.FCPXML != nil {
		// 如果没有提供项目名称，使用当前项目名称
		if config.FCPXML.ProjectName == "" {
			config.FCPXML.ProjectName = project.Metadata.Name
		}

		// 验证配置
		if err := config.FCPXML.Validate(); err != nil {
			return nil, s.handleError("update export config", fmt.Errorf("invalid FCPXML config: %w", err))
		}

		// 自动填充缺失字段
		config.FCPXML.AutoFill()
	}

	// 4. update export config
	project.Metadata.ExportConfigs = config

	// 5. save to database
	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update export config", err)
	}

	// 6. return updated project
	return project, nil
}

func (s *Service) ConvertSubtitle(id, langCode, targetFormat string) ([]byte, error) {
	// 1. get file info
	if id == "" {
		return nil, s.handleError("convert subtitle", fmt.Errorf("id is empty"))
	}
	if targetFormat == "" {
		return nil, s.handleError("convert subtitle", fmt.Errorf("target format is empty"))
	}

	// 2. get file project
	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("convert subtitle", err)
	}

	// 3. convert to target format
	switch strings.ToLower(targetFormat) {
	case "srt":
		return s.formatConverter.ToSRT(project, langCode)
	case "vtt":
		return s.formatConverter.ToVTT(project, langCode)
	case "ass", "ssa":
		return s.formatConverter.ToASS(project, langCode)
	case "itt":
		return s.formatConverter.ToITT(project, langCode)
	case "fcpxml":
		return s.formatConverter.ToFCPXML(project, langCode)
	default:
		return nil, s.handleError("convert subtitle", fmt.Errorf("unsupported format: %s", targetFormat))
	}
}

func (s *Service) ImportSubtitle(filePath string, options types.TextProcessingOptions) (*types.SubtitleProject, error) {
	if filePath == "" {
		return nil, s.handleError("import subtitle", fmt.Errorf("file path is empty"))
	}

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}

	if len(file) == 0 {
		return nil, s.handleError("import subtitle", fmt.Errorf("file content is empty"))
	}

	// 根据文件扩展名确定格式
	var project types.SubtitleProject
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".itt":
		project, err = s.formatConverter.FromItt(filePath, file)
	case ".srt":
		project, err = s.formatConverter.FromSRT(filePath, file)
	case ".vtt", ".webvtt":
		project, err = s.formatConverter.FromVTT(filePath, file)
	case ".ass", ".ssa":
		project, err = s.formatConverter.FromASS(filePath, file)
	// more formats...
	default:
		return nil, s.handleError("import subtitle", fmt.Errorf("unsupported file format: %s", ext))
	}

	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}

	// process
	err = s.processSubtitleText(&project, options)
	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}
	// validate
	err = s.validateProject(&project)
	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}

	// save to database
	err = s.boltStorage.SaveSubtitle(&project)
	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}

	return &project, nil
}

func (s *Service) GetSubtitle(id string) (*types.SubtitleProject, error) {
	if id == "" {
		return nil, s.handleError("get subtitle", fmt.Errorf("id is empty"))
	}

	if s.boltStorage == nil {
		return nil, s.handleError("get subtitle", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("get subtitle", err)
	}

	return project, nil
}

// RemoveProjectLanguage 删除项目中的一种翻译语言（不可删除原始语言）。
// 依据约定：原始语言的 LanguageMetadata.Translator 为空字符串。
func (s *Service) RemoveProjectLanguage(projectID, langCode string) (*types.SubtitleProject, error) {
    if strings.TrimSpace(projectID) == "" || strings.TrimSpace(langCode) == "" {
        return nil, fmt.Errorf("id or langCode is empty")
    }
    if s.boltStorage == nil { return nil, fmt.Errorf("bolt storage is nil") }
    p, err := s.boltStorage.GetSubtitle(projectID)
    if err != nil { return nil, err }
    if p == nil { return nil, fmt.Errorf("project not found") }

    // 不允许删除原始语言：以 IsOriginal 权威标记判断
    meta, ok := p.LanguageMetadata[langCode]
    if !ok {
        return nil, fmt.Errorf("language not found: %s", langCode)
    }
    if meta.Status.IsOriginal {
        return nil, fmt.Errorf("cannot delete original language")
    }

    // 删除每个片段的该语言内容
    for i := range p.Segments {
        if p.Segments[i].Languages != nil {
            delete(p.Segments[i].Languages, langCode)
        }
        if p.Segments[i].GuidelineStandard != nil {
            delete(p.Segments[i].GuidelineStandard, langCode)
        }
    }
    // 删除语言元数据
    delete(p.LanguageMetadata, langCode)
    p.UpdatedAt = time.Now().Unix()
    if err := s.boltStorage.SaveSubtitle(p); err != nil { return nil, err }
    return p, nil
}

// FindSubtitleBySourcePath scans existing projects and returns the one whose
// metadata.source_info.file_path matches the given absolute path.
func (s *Service) FindSubtitleBySourcePath(filePath string) (*types.SubtitleProject, error) {
	if s.boltStorage == nil {
		return nil, s.handleError("find subtitle by path", fmt.Errorf("bolt storage is nil"))
	}
	if strings.TrimSpace(filePath) == "" {
		return nil, s.handleError("find subtitle by path", fmt.Errorf("file path is empty"))
	}
	subs, err := s.boltStorage.ListSubtitles()
	if err != nil {
		return nil, s.handleError("find subtitle by path", err)
	}
	// normalize compare
	norm := func(p string) string { return strings.ReplaceAll(strings.TrimSpace(p), "\\", "/") }
	target := norm(filePath)
	for _, p := range subs {
		if p != nil && p.Metadata.SourceInfo != nil {
			if norm(p.Metadata.SourceInfo.FilePath) == target {
				return p, nil
			}
		}
	}
	return nil, fmt.Errorf("not found")
}

func (s *Service) DeleteSubtitle(id string) error {
	if id == "" {
		return s.handleError("delete subtitle", fmt.Errorf("id is empty"))
	}

	if s.boltStorage == nil {
		return s.handleError("delete subtitle", fmt.Errorf("bolt storage is nil"))
	}

	err := s.boltStorage.DeleteSubtitle(id)
	if err != nil {
		return s.handleError("delete subtitle", err)
	}

	return nil
}

func (s *Service) DeleteAllSubtitle() error {
	if s.boltStorage == nil {
		return s.handleError("delete all subtitle", fmt.Errorf("bolt storage is nil"))
	}
	err := s.boltStorage.DeleteAllSubtitle()
	if err != nil {
		return s.handleError("delete all subtitle", err)
	}
	return nil
}

func (s *Service) ListSubtitles() ([]*types.SubtitleProject, error) {
	if s.boltStorage == nil {
		return nil, s.handleError("get all subtitles", fmt.Errorf("bolt storage is nil"))
	}

	projects, err := s.boltStorage.ListSubtitles()
	if err != nil {
		return nil, s.handleError("get all subtitles", err)
	}

	return projects, nil
}

func (s *Service) UpdateSubtitleProject(project *types.SubtitleProject) (*types.SubtitleProject, error) {
	if project == nil {
		return nil, s.handleError("update subtitle project", fmt.Errorf("project is nil"))
	}

	if err := project.Validate(); err != nil {
		return nil, s.handleError("update subtitle project", fmt.Errorf("invalid project: %w", err))
	}

	project.UpdatedAt = time.Now().Unix()

	err := s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update subtitle project", err)
	}

	return project, nil
}

func (s *Service) UpdateProjectName(id, name string) (*types.SubtitleProject, error) {
	if id == "" {
		return nil, s.handleError("update project name", fmt.Errorf("id is empty"))
	}
	if name == "" {
		return nil, s.handleError("update project name", fmt.Errorf("name is empty"))
	}
	if s.boltStorage == nil {
		return nil, s.handleError("update project name", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update project name", err)
	}
	project.ProjectName = name
	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update project name", err)
	}
	return project, nil
}

func (s *Service) UpdateProjectMetadata(id string, metadata types.ProjectMetadata) (*types.SubtitleProject, error) {
	if id == "" {
		return nil, s.handleError("update project metadata", fmt.Errorf("id is empty"))
	}

	if s.boltStorage == nil {
		return nil, s.handleError("update project metadata", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update project metadata", err)
	}
	// Normalize/validate export configs similar to FCPXML path
	if metadata.ExportConfigs.FCPXML != nil {
		if metadata.ExportConfigs.FCPXML.ProjectName == "" {
			pn := project.Metadata.Name
			if pn == "" {
				pn = project.ProjectName
			}
			metadata.ExportConfigs.FCPXML.ProjectName = pn
		}
		if err := metadata.ExportConfigs.FCPXML.Validate(); err != nil {
			return nil, s.handleError("update project metadata", fmt.Errorf("invalid FCPXML config: %w", err))
		}
		metadata.ExportConfigs.FCPXML.AutoFill()
	}
	if metadata.ExportConfigs.ASS != nil {
		if metadata.ExportConfigs.ASS.Title == "" {
			tn := project.Metadata.Name
			if tn == "" {
				tn = project.ProjectName
			}
			metadata.ExportConfigs.ASS.Title = tn
		}
		if (metadata.ExportConfigs.ASS.PlayResX == 0 || metadata.ExportConfigs.ASS.PlayResY == 0) && metadata.ExportConfigs.FCPXML != nil {
			if metadata.ExportConfigs.FCPXML.Width > 0 {
				metadata.ExportConfigs.ASS.PlayResX = metadata.ExportConfigs.FCPXML.Width
			}
			if metadata.ExportConfigs.FCPXML.Height > 0 {
				metadata.ExportConfigs.ASS.PlayResY = metadata.ExportConfigs.FCPXML.Height
			}
		}
		if metadata.ExportConfigs.ASS.PlayResX == 0 {
			metadata.ExportConfigs.ASS.PlayResX = 1920
		}
		if metadata.ExportConfigs.ASS.PlayResY == 0 {
			metadata.ExportConfigs.ASS.PlayResY = 1080
		}
	}
	if metadata.ExportConfigs.VTT != nil {
		if metadata.ExportConfigs.VTT.Kind == "" {
			metadata.ExportConfigs.VTT.Kind = "subtitles"
		}
		if metadata.ExportConfigs.VTT.Language == "" {
			codes := project.GetLanguageCodes()
			if len(codes) > 0 {
				metadata.ExportConfigs.VTT.Language = codes[0]
			} else {
				metadata.ExportConfigs.VTT.Language = "en-US"
			}
		}
	}
	if metadata.ExportConfigs.ITT != nil {
		if metadata.ExportConfigs.ITT.FrameRate <= 0 {
			if metadata.ExportConfigs.FCPXML != nil && metadata.ExportConfigs.FCPXML.FrameRate > 0 {
				metadata.ExportConfigs.ITT.FrameRate = metadata.ExportConfigs.FCPXML.FrameRate
			} else {
				metadata.ExportConfigs.ITT.FrameRate = 25
			}
		}
		if strings.TrimSpace(metadata.ExportConfigs.ITT.Language) == "" {
			codes := project.GetLanguageCodes()
			if len(codes) > 0 {
				metadata.ExportConfigs.ITT.Language = codes[0]
			} else {
				metadata.ExportConfigs.ITT.Language = "en-US"
			}
		}
	}

	project.Metadata = metadata
	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update project metadata", err)
	}
	return project, nil
}

// UpdateSubtitleSegment 更新单个字幕片段 - 修改为使用指针类型
func (s *Service) UpdateSubtitleSegment(id string, segmentID string, segment *types.SubtitleSegment) (*types.SubtitleProject, error) {
	if id == "" || segmentID == "" {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("id or segmentID is empty"))
	}

	if segment == nil {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("segment is nil"))
	}

	if s.boltStorage == nil {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update subtitle segment", err)
	}

	// 验证片段数据
	if err := segment.Validate(); err != nil {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("invalid segment: %w", err))
	}

	// recalculate guideline - 直接使用指针
	updatedSegment := s.qualityAssessor.AssessSegmentQuality(segment)

	// 查找并更新片段
	found := false
	for i, seg := range project.Segments {
		if seg.ID == segmentID {
			project.Segments[i] = *updatedSegment
			found = true
			break
		}
	}

	if !found {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("segment with ID %s not found", segmentID))
	}

	project.UpdatedAt = time.Now().Unix()

	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update subtitle segment", err)
	}

	return project, nil
}

// UpdateLanguageContent 更新特定语言的内容
func (s *Service) UpdateLanguageContent(id string, segmentID string, langCode string, content types.LanguageContent) (*types.SubtitleProject, error) {
	if id == "" || segmentID == "" || langCode == "" {
		return nil, s.handleError("update language content", fmt.Errorf("id, segmentID or langCode is empty"))
	}

	if s.boltStorage == nil {
		return nil, s.handleError("update language content", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update language content", err)
	}

	// 验证内容
	if err := content.Validate(); err != nil {
		return nil, s.handleError("update language content", fmt.Errorf("invalid content: %w", err))
	}

	// 查找并更新语言内容
	found := false
	for i, seg := range project.Segments {
		if seg.ID == segmentID {
			if project.Segments[i].Languages == nil {
				project.Segments[i].Languages = make(map[string]types.LanguageContent)
			}
			project.Segments[i].Languages[langCode] = content
			found = true
			break
		}
	}

	if !found {
		return nil, s.handleError("update language content", fmt.Errorf("segment with ID %s not found", segmentID))
	}

	project.UpdatedAt = time.Now().Unix()

	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update language content", err)
	}

	return project, nil
}

func (s *Service) UpdateLanguageMetadata(id string, langCode string, metadata types.LanguageMetadata) (*types.SubtitleProject, error) {
	if id == "" || langCode == "" {
		return nil, s.handleError("update language metadata", fmt.Errorf("id or langCode is empty"))
	}
	if s.boltStorage == nil {
		return nil, s.handleError("update language metadata", fmt.Errorf("bolt storage is nil"))
	}
	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update language metadata", err)
	}
	if project.LanguageMetadata == nil {
		project.LanguageMetadata = make(map[string]types.LanguageMetadata)
	}
	project.LanguageMetadata[langCode] = metadata
	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update language metadata", err)
	}
	return project, nil
}

// processSubtitleText 批量处理字幕文本
func (s *Service) processSubtitleText(project *types.SubtitleProject, options types.TextProcessingOptions) error {
	if project == nil {
		return s.handleError("process subtitle text", fmt.Errorf("project is nil"))
	}

	for i := range project.Segments {
		segment := &project.Segments[i]

		for langCode, content := range segment.Languages {
			processedContent := content

			// 文本清理
			if options.RemoveEmptyLines {
				processedContent.Text = s.textProcessor.RemoveEmptyLines(processedContent.Text)
			}

			if options.TrimWhitespace {
				processedContent.Text = s.textProcessor.TrimWhitespace(processedContent.Text)
			}

			if options.NormalizeLineBreaks {
				processedContent.Text = s.textProcessor.NormalizeLineBreaks(processedContent.Text)
			}

			if options.FixEncoding {
				processedContent.Text = s.textProcessor.FixEncoding(processedContent.Text)
			}

			if options.FixCommonErrors {
				processedContent.Text = s.textProcessor.FixCommonTextErrors(processedContent.Text)
			}

			// 重新计算guideline
			segment.Languages[langCode] = processedContent

			// set guideline standard
			segment.GuidelineStandard[langCode] = options.GuidelineStandard
		}

		// 重新计算该片段的guideline指标
		if options.ValidateGuidelines {
			segment.IsKidsContent = options.IsKidsContent
			s.qualityAssessor.AssessSegmentQuality(segment)
		}
	}

	return nil
}

// validateProject 验证项目内容
func (s *Service) validateProject(project *types.SubtitleProject) error {
	var validationErrors []error

	// 验证项目级别
	if err := project.Validate(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// 验证每个片段
	for i, segment := range project.Segments {
		if err := segment.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("segment %d: %w", i+1, err))
		}
	}

	if len(validationErrors) > 0 {
		return s.handleError("validate project", fmt.Errorf("validation errors: %v", validationErrors))
	}

	return nil
}
