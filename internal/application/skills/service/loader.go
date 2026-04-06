package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"dreamcreator/internal/application/skills/dto"
	domainSkills "dreamcreator/internal/domain/skills"
)

const (
	defaultMaxCandidatesPerRoot     = 300
	defaultMaxSkillsLoadedPerSource = 200
	defaultMaxSkillsInPrompt        = 150
	defaultMaxSkillsPromptChars     = 30_000
	defaultMaxSkillFileBytes        = 256_000
	defaultSkillsWatchDebounceMs    = 250
)

type skillEntry struct {
	ID          string
	Name        string
	Description string
	Path        string
	Commands    []string
	Runtime     *dto.SkillRuntimeRequirements
	Source      string
	SourceID    string
	SourceName  string
	SourceKind  string
	SourceType  string
	SourcePath  string
}

type skillsLimits struct {
	MaxCandidatesPerRoot     int
	MaxSkillsLoadedPerSource int
	MaxSkillsInPrompt        int
	MaxSkillsPromptChars     int
	MaxSkillFileBytes        int
}

type skillsLoadConfig struct {
	AllowBundled    []string
	LocalSources    []skillsLocalSourceItem
	Watch           bool
	WatchDebounceMs int
}

type skillRoot struct {
	Path       string
	Source     string
	SourceID   string
	SourceName string
	SourceKind string
	SourceType string
}

func (service *SkillsService) resolveSkillsLimits(ctx context.Context) skillsLimits {
	_ = ctx
	limits := skillsLimits{
		MaxCandidatesPerRoot:     defaultMaxCandidatesPerRoot,
		MaxSkillsLoadedPerSource: defaultMaxSkillsLoadedPerSource,
		MaxSkillsInPrompt:        defaultMaxSkillsInPrompt,
		MaxSkillsPromptChars:     defaultMaxSkillsPromptChars,
		MaxSkillFileBytes:        defaultMaxSkillFileBytes,
	}
	return limits
}

func (service *SkillsService) resolveSkillsLoad(ctx context.Context) skillsLoadConfig {
	load := skillsLoadConfig{
		AllowBundled:    nil,
		LocalSources:    nil,
		Watch:           true,
		WatchDebounceMs: defaultSkillsWatchDebounceMs,
	}
	if service == nil || service.settings == nil {
		return load
	}
	current, err := service.settings.GetSettings(ctx)
	if err != nil {
		return load
	}
	_, skillsConfig := resolveSettingsToolsSkills(current)
	load.LocalSources = resolveSkillsLocalSources(skillsConfig)
	return load
}

func (service *SkillsService) loadWorkspaceSkillEntries(ctx context.Context, workspaceRoot string, limits *skillsLimits) []skillEntry {
	limit := skillsLimits{
		MaxCandidatesPerRoot:     defaultMaxCandidatesPerRoot,
		MaxSkillsLoadedPerSource: defaultMaxSkillsLoadedPerSource,
		MaxSkillsInPrompt:        defaultMaxSkillsInPrompt,
		MaxSkillsPromptChars:     defaultMaxSkillsPromptChars,
		MaxSkillFileBytes:        defaultMaxSkillFileBytes,
	}
	if limits != nil {
		limit = *limits
	} else {
		limit = service.resolveSkillsLimits(ctx)
	}
	load := service.resolveSkillsLoad(ctx)
	cacheKey := buildSkillsCacheKey(workspaceRoot, limit, load)
	cacheVersion := int64(0)
	if load.Watch {
		ensureSkillsWatcher(workspaceRoot, load, limit.MaxCandidatesPerRoot)
		cacheVersion = getSkillsSnapshotVersion(workspaceRoot)
		if entries, ok := getCachedSkills(cacheKey, cacheVersion); ok {
			return entries
		}
	}
	roots := resolveSkillRoots(workspaceRoot, load)
	if len(roots) == 0 {
		return nil
	}
	result := make([]skillEntry, 0)
	seen := make(map[string]struct{})
	for _, root := range roots {
		entries := loadSkillsFromRoot(root, limit.MaxCandidatesPerRoot, limit.MaxSkillsLoadedPerSource, limit.MaxSkillFileBytes)
		if root.Source == "bundled" {
			entries = filterBundledSkills(entries, load.AllowBundled)
		}
		for _, entry := range entries {
			key := strings.ToLower(strings.TrimSpace(entry.ID))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, entry)
		}
	}
	if load.Watch {
		setCachedSkills(cacheKey, cacheVersion, result)
	}
	return result
}

func resolveSkillRoots(workspaceRoot string, load skillsLoadConfig) []skillRoot {
	roots := make([]skillRoot, 0)
	localSources := load.LocalSources
	if len(localSources) == 0 {
		localSources = defaultSkillsLocalSources()
	}
	for _, source := range localSources {
		if !source.Enabled {
			continue
		}
		switch source.Type {
		case "workspace":
			if root := resolveDreamCreatorSkillsRoot(workspaceRoot); root != "" {
				roots = append(roots, buildSkillRoot(root, source))
			}
		case "extra":
			if path := resolveExtraSkillRootPath(workspaceRoot, source.Path); path != "" {
				roots = append(roots, buildSkillRoot(path, source))
			}
		}
	}
	return dedupeSkillRoots(roots)
}

type skillsLocalSourceItem struct {
	ID       string
	Name     string
	Kind     string
	Type     string
	Path     string
	Enabled  bool
	Priority int
	Index    int
}

func resolveSkillsLocalSources(skillsConfig map[string]any) []skillsLocalSourceItem {
	defaults := defaultSkillsLocalSources()
	if len(skillsConfig) == 0 {
		return defaults
	}
	sources, _ := skillsConfig["sources"].(map[string]any)
	if len(sources) == 0 {
		return defaults
	}
	localItems := parseStructuredLocalSourceItems(sources["local"])
	return mergeSkillsLocalSources(defaults, localItems)
}

func defaultSkillsLocalSources() []skillsLocalSourceItem {
	return []skillsLocalSourceItem{
		{
			ID:       "workspace",
			Name:     "DreamCreator",
			Kind:     "local",
			Type:     "workspace",
			Enabled:  true,
			Priority: 0,
			Index:    0,
		},
	}
}

func parseStructuredLocalSourceItems(raw any) []skillsLocalSourceItem {
	var values []any
	switch typed := raw.(type) {
	case []any:
		values = typed
	case []map[string]any:
		values = make([]any, 0, len(typed))
		for _, item := range typed {
			values = append(values, item)
		}
	default:
		return nil
	}
	items := make([]skillsLocalSourceItem, 0, len(values))
	for index, entry := range values {
		source, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		rawType := strings.TrimSpace(readMapString(source, "type"))
		sourceType := normalizeLocalSourceType(rawType)
		if sourceType == "" {
			if rawType != "" {
				continue
			}
			sourceType = "extra"
		}
		if !isSupportedConfiguredLocalSourceType(sourceType) {
			continue
		}
		path := strings.TrimSpace(readMapString(source, "path"))
		if sourceType == "extra" && path == "" {
			continue
		}
		enabled := true
		if value, ok := readMapBool(source, "enabled"); ok {
			enabled = value
		}
		priority := index
		if value, ok := readMapInt(source, "priority"); ok {
			priority = value
		}
		id := strings.TrimSpace(readMapString(source, "id"))
		if id == "" {
			id = defaultLocalSourceID(sourceType, index)
		}
		name := strings.TrimSpace(readMapString(source, "name"))
		if name == "" {
			name = defaultLocalSourceName(sourceType, path, index)
		}
		items = append(items, skillsLocalSourceItem{
			ID:       id,
			Name:     name,
			Kind:     "local",
			Type:     sourceType,
			Path:     path,
			Enabled:  enabled,
			Priority: priority,
			Index:    index,
		})
	}
	return items
}

func mergeSkillsLocalSources(
	defaults []skillsLocalSourceItem,
	structured []skillsLocalSourceItem,
) []skillsLocalSourceItem {
	merged := make([]skillsLocalSourceItem, 0, len(defaults))
	builtinByType := make(map[string]int, len(defaults))
	for index, item := range defaults {
		cloned := item
		cloned.Type = normalizeLocalSourceType(cloned.Type)
		if cloned.Kind == "" {
			cloned.Kind = "local"
		}
		merged = append(merged, cloned)
		builtinByType[cloned.Type] = index
	}
	appendExtra := func(item skillsLocalSourceItem) {
		if item.Kind == "" {
			item.Kind = "local"
		}
		item.Type = normalizeLocalSourceType(item.Type)
		if item.Type == "" {
			item.Type = "extra"
		}
		if !isSupportedConfiguredLocalSourceType(item.Type) {
			return
		}
		if item.ID == "" {
			item.ID = defaultLocalSourceID(item.Type, len(merged))
		}
		if item.Name == "" {
			item.Name = defaultLocalSourceName(item.Type, item.Path, len(merged))
		}
		merged = append(merged, item)
	}
	for _, item := range structured {
		item.Type = normalizeLocalSourceType(item.Type)
		if builtinIndex, ok := builtinByType[item.Type]; ok {
			if item.ID == "" {
				item.ID = merged[builtinIndex].ID
			}
			if item.Name == "" {
				item.Name = merged[builtinIndex].Name
			}
			if item.Kind == "" {
				item.Kind = "local"
			}
			merged[builtinIndex] = item
			continue
		}
		appendExtra(item)
	}
	sort.SliceStable(merged, func(i, j int) bool {
		if merged[i].Priority != merged[j].Priority {
			return merged[i].Priority < merged[j].Priority
		}
		return merged[i].Index < merged[j].Index
	})
	for index := range merged {
		if merged[index].Kind == "" {
			merged[index].Kind = "local"
		}
		if merged[index].ID == "" {
			merged[index].ID = defaultLocalSourceID(merged[index].Type, index)
		}
		if merged[index].Name == "" {
			merged[index].Name = defaultLocalSourceName(merged[index].Type, merged[index].Path, index)
		}
		merged[index].Priority = index
		merged[index].Index = index
	}
	return merged
}

func normalizeLocalSourceType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "workspace", "extra":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func isSupportedConfiguredLocalSourceType(value string) bool {
	switch normalizeLocalSourceType(value) {
	case "workspace", "extra":
		return true
	default:
		return false
	}
}

func defaultLocalSourceID(sourceType string, index int) string {
	sourceType = normalizeLocalSourceType(sourceType)
	if sourceType == "" {
		sourceType = "extra"
	}
	if sourceType != "extra" {
		return sourceType
	}
	return fmt.Sprintf("extra-%d", index+1)
}

func defaultLocalSourceName(sourceType string, path string, index int) string {
	if normalizeLocalSourceType(sourceType) == "workspace" {
		return "DreamCreator"
	}
	return localSourceDisplayName(path, index)
}

func localSourceDisplayName(path string, index int) string {
	trimmed := strings.TrimSpace(path)
	if trimmed != "" {
		base := strings.TrimSpace(filepath.Base(trimmed))
		if base != "" && base != "." && base != string(filepath.Separator) {
			return base
		}
		return trimmed
	}
	return fmt.Sprintf("Extra %d", index+1)
}

func buildSkillRoot(path string, source skillsLocalSourceItem) skillRoot {
	return skillRoot{
		Path:       path,
		Source:     firstNonEmpty(source.Type, source.ID, source.Kind),
		SourceID:   strings.TrimSpace(source.ID),
		SourceName: strings.TrimSpace(source.Name),
		SourceKind: firstNonEmpty(source.Kind, "local"),
		SourceType: firstNonEmpty(source.Type, "extra"),
	}
}

func resolveExtraSkillRootPath(workspaceRoot string, rawPath string) string {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return ""
	}
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	resolvedPath := rawPath
	wasRelative := !filepath.IsAbs(rawPath)
	if wasRelative {
		if workspaceRoot == "" {
			return ""
		}
		resolvedPath = filepath.Join(workspaceRoot, rawPath)
	}
	if absolute, err := filepath.Abs(resolvedPath); err == nil {
		resolvedPath = absolute
	}
	if realPath, err := filepath.EvalSymlinks(resolvedPath); err == nil {
		resolvedPath = realPath
	}
	if wasRelative && workspaceRoot != "" && !isPathWithinRoot(workspaceRoot, resolvedPath) {
		return ""
	}
	return resolvedPath
}

func isPathWithinRoot(root string, target string) bool {
	root = strings.TrimSpace(root)
	target = strings.TrimSpace(target)
	if root == "" || target == "" {
		return false
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}

func readMapString(source map[string]any, key string) string {
	if len(source) == 0 || key == "" {
		return ""
	}
	raw, ok := source[key]
	if !ok {
		return ""
	}
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(raw))
	}
}

func readMapBool(source map[string]any, key string) (bool, bool) {
	if len(source) == 0 || key == "" {
		return false, false
	}
	raw, ok := source[key]
	if !ok {
		return false, false
	}
	switch typed := raw.(type) {
	case bool:
		return typed, true
	case string:
		value := strings.TrimSpace(strings.ToLower(typed))
		if value == "true" || value == "1" || value == "yes" {
			return true, true
		}
		if value == "false" || value == "0" || value == "no" {
			return false, true
		}
	}
	return false, false
}

func readMapInt(source map[string]any, key string) (int, bool) {
	if len(source) == 0 || key == "" {
		return 0, false
	}
	raw, ok := source[key]
	if !ok {
		return 0, false
	}
	switch typed := raw.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case string:
		value := strings.TrimSpace(typed)
		if value == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func dedupeSkillRoots(roots []skillRoot) []skillRoot {
	if len(roots) == 0 {
		return nil
	}
	result := make([]skillRoot, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		trimmed := strings.TrimSpace(root.Path)
		if trimmed == "" {
			continue
		}
		abs := trimmed
		if resolved, err := filepath.Abs(trimmed); err == nil {
			abs = resolved
		}
		key := strings.ToLower(abs)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, root)
	}
	return result
}

func defaultWorkspaceRoots() []string {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	root := filepath.Join(base, "dreamcreator", "workspaces")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		result = append(result, filepath.Join(root, name))
	}
	return result
}

func loadSkillsFromRoot(root skillRoot, maxCandidates int, maxSkills int, maxBytes int) []skillEntry {
	base := resolveNestedSkillsRoot(strings.TrimSpace(root.Path))
	if base == "" {
		return nil
	}
	info, err := os.Stat(base)
	if err != nil || !info.IsDir() {
		return nil
	}
	skillFile := filepath.Join(base, "SKILL.md")
	if fileInfo, err := os.Stat(skillFile); err == nil && !fileInfo.IsDir() {
		if entry, ok := parseSkillFile(skillFile, filepath.Base(base), root, maxBytes); ok {
			return []skillEntry{entry}
		}
		return nil
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	result := make([]skillEntry, 0)
	candidates := 0
	for _, entry := range entries {
		if maxSkills > 0 && len(result) >= maxSkills {
			break
		}
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") || name == "node_modules" {
			continue
		}
		candidates++
		if maxCandidates > 0 && candidates > maxCandidates {
			return result
		}
		skillPath := filepath.Join(base, name, "SKILL.md")
		if fileInfo, err := os.Stat(skillPath); err == nil && !fileInfo.IsDir() {
			if parsed, ok := parseSkillFile(skillPath, name, root, maxBytes); ok {
				result = append(result, parsed)
			}
		}
	}
	return result
}

func resolveNestedSkillsRoot(root string) string {
	if root == "" {
		return ""
	}
	nested := filepath.Join(root, "skills")
	info, err := os.Stat(nested)
	if err != nil || !info.IsDir() {
		return root
	}
	entries, err := os.ReadDir(nested)
	if err != nil {
		return root
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		skillPath := filepath.Join(nested, name, "SKILL.md")
		if info, err := os.Stat(skillPath); err == nil && !info.IsDir() {
			return nested
		}
	}
	return root
}

func parseSkillFile(path string, fallbackID string, root skillRoot, maxBytes int) (skillEntry, bool) {
	data, err := readSkillFile(path, maxBytes)
	if err != nil {
		return skillEntry{}, false
	}
	content := string(data)
	frontmatter, body := parseFrontmatter(content)
	name := firstNonEmpty(frontmatterString(frontmatter, "name", "title"), extractHeading(body))
	id := strings.TrimSpace(frontmatterString(frontmatter, "id"))
	if id == "" {
		id = strings.TrimSpace(name)
	}
	if id == "" {
		id = strings.TrimSpace(fallbackID)
	}
	description := strings.TrimSpace(frontmatterString(frontmatter, "description", "summary"))
	if description == "" {
		description = extractDescription(body)
	}
	commands := frontmatterStringSlice(frontmatter, "commands")
	if len(commands) == 0 {
		if cmd := strings.TrimSpace(frontmatterString(frontmatter, "command")); cmd != "" {
			commands = []string{cmd}
		}
	}
	if len(commands) == 0 {
		commands = frontmatterStringSlice(frontmatter, "aliases")
	}
	if len(commands) == 0 {
		if name != "" {
			commands = []string{name}
		} else if id != "" {
			commands = []string{id}
		}
	}
	entry := skillEntry{
		ID:          id,
		Name:        firstNonEmpty(name, id),
		Description: description,
		Path:        filepath.ToSlash(path),
		Commands:    normalizeStringList(commands),
		Runtime:     parseSkillRuntimeRequirementsFromMarkdown(content),
		Source:      firstNonEmpty(root.Source, root.SourceType, root.SourceID, root.SourceKind),
		SourceID:    strings.TrimSpace(root.SourceID),
		SourceName:  strings.TrimSpace(root.SourceName),
		SourceKind:  firstNonEmpty(root.SourceKind, "local"),
		SourceType:  firstNonEmpty(root.SourceType, root.Source),
		SourcePath:  filepath.ToSlash(strings.TrimSpace(root.Path)),
	}
	return normalizeSkillEntry(entry), entry.ID != ""
}

func readSkillFile(path string, maxBytes int) ([]byte, error) {
	if maxBytes <= 0 {
		return os.ReadFile(path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if info, err := file.Stat(); err == nil && info.Size() > int64(maxBytes) {
		return nil, fmt.Errorf("skill file too large: %s", path)
	}
	limited := &io.LimitedReader{
		R: file,
		N: int64(maxBytes) + 1,
	}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("skill file too large: %s", path)
	}
	return data, nil
}

func normalizeSkillEntry(entry skillEntry) skillEntry {
	entry.ID = strings.TrimSpace(entry.ID)
	entry.Name = strings.TrimSpace(entry.Name)
	entry.Description = strings.TrimSpace(entry.Description)
	entry.Path = strings.TrimSpace(entry.Path)
	entry.Source = strings.TrimSpace(entry.Source)
	entry.SourceID = strings.TrimSpace(entry.SourceID)
	entry.SourceName = strings.TrimSpace(entry.SourceName)
	entry.SourceKind = strings.TrimSpace(entry.SourceKind)
	entry.SourceType = strings.TrimSpace(entry.SourceType)
	entry.SourcePath = strings.TrimSpace(entry.SourcePath)
	entry.Commands = normalizeStringList(entry.Commands)
	if entry.Name == "" {
		entry.Name = entry.ID
	}
	if entry.ID == "" && entry.Name != "" {
		entry.ID = entry.Name
	}
	if entry.Description == "" && entry.Name != "" {
		entry.Description = entry.Name
	}
	if entry.SourceKind == "" {
		entry.SourceKind = "local"
	}
	if entry.SourceType == "" {
		entry.SourceType = entry.Source
	}
	if entry.Source == "" {
		entry.Source = firstNonEmpty(entry.SourceType, entry.SourceID, entry.SourceKind)
	}
	if entry.SourceID == "" {
		entry.SourceID = firstNonEmpty(entry.SourceType, entry.Source, "workspace")
	}
	if entry.SourceName == "" {
		entry.SourceName = defaultLocalSourceName(entry.SourceType, entry.SourcePath, 0)
	}
	return entry
}

func parseFrontmatter(content string) (map[string]any, string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return nil, content
	}
	if strings.TrimSpace(lines[0]) != "---" {
		return nil, content
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" || line == "..." {
			end = i
			break
		}
	}
	if end == -1 {
		return nil, content
	}
	raw := strings.Join(lines[1:end], "\n")
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, strings.Join(lines[end+1:], "\n")
	}
	return parsed, strings.Join(lines[end+1:], "\n")
}

func frontmatterString(frontmatter map[string]any, keys ...string) string {
	if len(frontmatter) == 0 {
		return ""
	}
	for _, key := range keys {
		value, ok := frontmatter[key]
		if !ok {
			continue
		}
		if str, ok := value.(string); ok {
			if trimmed := strings.TrimSpace(str); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func frontmatterStringSlice(frontmatter map[string]any, key string) []string {
	if len(frontmatter) == 0 || key == "" {
		return nil
	}
	value, ok := frontmatter[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			switch raw := item.(type) {
			case string:
				result = append(result, raw)
			case map[string]any:
				if name, ok := raw["name"].(string); ok {
					result = append(result, name)
				} else if name, ok := raw["command"].(string); ok {
					result = append(result, name)
				}
			}
		}
		return normalizeStringList(result)
	case []string:
		return normalizeStringList(typed)
	case string:
		return normalizeStringList(strings.Split(typed, ","))
	default:
		return nil
	}
}

var headingPattern = regexp.MustCompile(`^#+\s+`)

func extractHeading(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if headingPattern.MatchString(trimmed) {
			return strings.TrimSpace(headingPattern.ReplaceAllString(trimmed, ""))
		}
	}
	return ""
}

func extractDescription(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if headingPattern.MatchString(trimmed) {
			continue
		}
		return trimmed
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func mergeProviderSkills(settingsItems []domainSkills.ProviderSkillSpec, workspaceEntries []skillEntry) []dto.ProviderSkillSpec {
	lookup := make(map[string]domainSkills.ProviderSkillSpec, len(settingsItems))
	for _, item := range settingsItems {
		key := strings.ToLower(strings.TrimSpace(item.ID))
		if key == "" {
			continue
		}
		lookup[key] = item
	}
	result := make([]dto.ProviderSkillSpec, 0, len(settingsItems)+len(workspaceEntries))
	seen := make(map[string]struct{})
	for _, entry := range workspaceEntries {
		id := strings.TrimSpace(entry.ID)
		if id == "" {
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			continue
		}
		enabled := true
		if setting, ok := lookup[key]; ok {
			enabled = setting.Enabled
		}
		result = append(result, dto.ProviderSkillSpec{
			ID:          id,
			ProviderID:  firstNonEmpty(entry.SourceID, entry.Source, "workspace"),
			Name:        firstNonEmpty(entry.Name, id),
			Description: entry.Description,
			Version:     "",
			Enabled:     enabled,
			SourceID:    firstNonEmpty(entry.SourceID, entry.Source, "workspace"),
			SourceName:  firstNonEmpty(entry.SourceName, defaultLocalSourceName(entry.SourceType, entry.SourcePath, 0)),
			SourceKind:  firstNonEmpty(entry.SourceKind, "local"),
			SourceType:  firstNonEmpty(entry.SourceType, entry.Source, "extra"),
			SourcePath:  entry.SourcePath,
		})
		seen[key] = struct{}{}
	}
	for _, item := range settingsItems {
		key := strings.ToLower(strings.TrimSpace(item.ID))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		result = append(result, toDTO(item))
	}
	return result
}

func buildSkillPromptItems(entries []skillEntry, settingsItems []domainSkills.ProviderSkillSpec, workspaceRoot string, maxItems int) []dto.SkillPromptItem {
	result := make([]dto.SkillPromptItem, 0)
	seen := make(map[string]struct{})
	addItem := func(item dto.SkillPromptItem) {
		key := strings.ToLower(strings.TrimSpace(item.Name))
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}

	for _, entry := range entries {
		name := firstNonEmpty(entry.Name, entry.ID)
		if name == "" {
			continue
		}
		commands := entry.Commands
		if len(commands) == 0 {
			commands = []string{name}
		}
		for _, command := range commands {
			command = strings.TrimSpace(command)
			if command == "" {
				continue
			}
			addItem(dto.SkillPromptItem{
				ID:          entry.ID,
				Name:        command,
				Description: entry.Description,
				Path:        entry.Path,
			})
			if maxItems > 0 && len(result) >= maxItems {
				return result
			}
		}
	}

	for _, item := range settingsItems {
		if !item.Enabled {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = strings.TrimSpace(item.ID)
		}
		if name == "" {
			continue
		}
		path := buildSkillPath(item.ID, workspaceRoot)
		addItem(dto.SkillPromptItem{
			ID:          item.ID,
			Name:        name,
			Description: item.Description,
			Path:        path,
		})
		if maxItems > 0 && len(result) >= maxItems {
			return result
		}
	}
	return result
}

func buildSkillPath(id string, workspaceRoot string) string {
	return resolveSkillDocumentPath(id, workspaceRoot)
}

func entryExists(entries []skillEntry, id string) bool {
	if id == "" {
		return false
	}
	target := strings.ToLower(strings.TrimSpace(id))
	for _, entry := range entries {
		if strings.ToLower(strings.TrimSpace(entry.ID)) == target {
			return true
		}
	}
	return false
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func filterBundledSkills(entries []skillEntry, allowlist []string) []skillEntry {
	if len(entries) == 0 {
		return nil
	}
	if len(allowlist) == 0 {
		return entries
	}
	allowed := make(map[string]struct{}, len(allowlist))
	for _, entry := range allowlist {
		trimmed := strings.ToLower(strings.TrimSpace(entry))
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	if len(allowed) == 0 {
		return entries
	}
	filtered := make([]skillEntry, 0, len(entries))
	for _, entry := range entries {
		id := strings.ToLower(strings.TrimSpace(entry.ID))
		name := strings.ToLower(strings.TrimSpace(entry.Name))
		if _, ok := allowed[id]; ok {
			filtered = append(filtered, entry)
			continue
		}
		if _, ok := allowed[name]; ok {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
