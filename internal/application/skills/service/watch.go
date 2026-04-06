package service

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type skillsCacheEntry struct {
	version int64
	entries []skillEntry
}

type skillsWatchState struct {
	stop        chan struct{}
	pathsKey    string
	debounce    time.Duration
	watcher     *fsnotify.Watcher
	watchedDirs map[string]struct{}
}

var (
	skillsCacheMu  sync.Mutex
	skillsCache    = map[string]skillsCacheEntry{}
	skillsWatchMu  sync.Mutex
	skillsWatchers = map[string]*skillsWatchState{}
	skillsVersions = map[string]int64{}
)

func buildSkillsCacheKey(workspaceRoot string, limits skillsLimits, load skillsLoadConfig) string {
	allowBundled := append([]string(nil), load.AllowBundled...)
	sort.Strings(allowBundled)
	rootKeys := make([]string, 0)
	for _, root := range resolveSkillRoots(workspaceRoot, load) {
		path := strings.TrimSpace(root.Path)
		if path == "" {
			continue
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		rootKeys = append(rootKeys, strings.Join([]string{
			path,
			strings.TrimSpace(root.SourceID),
			strings.TrimSpace(root.SourceName),
			strings.TrimSpace(root.SourceKind),
			strings.TrimSpace(root.SourceType),
		}, "#"))
	}
	parts := []string{
		strings.TrimSpace(workspaceRoot),
		strings.Join(allowBundled, ","),
		strings.Join(rootKeys, ","),
		strconv.Itoa(limits.MaxCandidatesPerRoot),
		strconv.Itoa(limits.MaxSkillsLoadedPerSource),
		strconv.Itoa(limits.MaxSkillsInPrompt),
		strconv.Itoa(limits.MaxSkillsPromptChars),
		strconv.Itoa(limits.MaxSkillFileBytes),
	}
	return strings.Join(parts, "|")
}

func getCachedSkills(key string, version int64) ([]skillEntry, bool) {
	if key == "" || version == 0 {
		return nil, false
	}
	skillsCacheMu.Lock()
	defer skillsCacheMu.Unlock()
	entry, ok := skillsCache[key]
	if !ok || entry.version != version {
		return nil, false
	}
	result := append([]skillEntry(nil), entry.entries...)
	return result, true
}

func setCachedSkills(key string, version int64, entries []skillEntry) {
	if key == "" || version == 0 {
		return
	}
	skillsCacheMu.Lock()
	defer skillsCacheMu.Unlock()
	skillsCache[key] = skillsCacheEntry{
		version: version,
		entries: append([]skillEntry(nil), entries...),
	}
}

func getSkillsSnapshotVersion(workspaceRoot string) int64 {
	normalized := strings.TrimSpace(workspaceRoot)
	skillsWatchMu.Lock()
	defer skillsWatchMu.Unlock()
	if normalized == "" {
		return 0
	}
	return skillsVersions[normalized]
}

func bumpSkillsSnapshotVersion(workspaceRoot string) int64 {
	normalized := strings.TrimSpace(workspaceRoot)
	if normalized == "" {
		return 0
	}
	now := time.Now().UnixNano()
	skillsWatchMu.Lock()
	defer skillsWatchMu.Unlock()
	if current := skillsVersions[normalized]; current >= now {
		now = current + 1
	}
	skillsVersions[normalized] = now
	return now
}

func ensureSkillsWatcher(workspaceRoot string, load skillsLoadConfig, maxCandidates int) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return
	}
	if !load.Watch {
		stopSkillsWatcher(root)
		return
	}
	debounce := time.Duration(load.WatchDebounceMs) * time.Millisecond
	if debounce <= 0 {
		debounce = time.Duration(defaultSkillsWatchDebounceMs) * time.Millisecond
	}
	pathsKey := buildSkillsWatchKey(root, load)

	skillsWatchMu.Lock()
	existing := skillsWatchers[root]
	if existing != nil && existing.pathsKey == pathsKey && existing.debounce == debounce {
		skillsWatchMu.Unlock()
		return
	}
	if existing != nil {
		close(existing.stop)
		delete(skillsWatchers, root)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		skillsWatchMu.Unlock()
		return
	}
	state := &skillsWatchState{
		stop:        make(chan struct{}),
		pathsKey:    pathsKey,
		debounce:    debounce,
		watcher:     watcher,
		watchedDirs: make(map[string]struct{}),
	}
	skillsWatchers[root] = state
	skillsWatchMu.Unlock()

	registerSkillWatchDirs(state, root, load, maxCandidates)
	bumpSkillsSnapshotVersion(root)
	go runSkillsWatcher(root, state, load, maxCandidates)
}

func stopSkillsWatcher(workspaceRoot string) {
	root := strings.TrimSpace(workspaceRoot)
	if root == "" {
		return
	}
	skillsWatchMu.Lock()
	defer skillsWatchMu.Unlock()
	if existing := skillsWatchers[root]; existing != nil {
		close(existing.stop)
		delete(skillsWatchers, root)
	}
}

func buildSkillsWatchKey(workspaceRoot string, load skillsLoadConfig) string {
	roots := resolveSkillRoots(workspaceRoot, load)
	paths := make([]string, 0, len(roots))
	for _, root := range roots {
		path := strings.TrimSpace(root.Path)
		if path == "" {
			continue
		}
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		paths = append(paths, strings.ToLower(path))
	}
	sort.Strings(paths)
	return strings.Join(paths, "|")
}

func runSkillsWatcher(workspaceRoot string, state *skillsWatchState, load skillsLoadConfig, maxCandidates int) {
	defer func() {
		if state.watcher != nil {
			_ = state.watcher.Close()
		}
	}()

	var (
		timer     *time.Timer
		timerChan <-chan time.Time
	)
	trigger := func() {
		if timer == nil {
			timer = time.NewTimer(state.debounce)
			timerChan = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(state.debounce)
	}

	for {
		select {
		case <-state.stop:
			if timer != nil {
				timer.Stop()
			}
			return
		case event, ok := <-state.watcher.Events:
			if !ok {
				return
			}
			path := filepath.Clean(event.Name)
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(path); err == nil && info.IsDir() {
					_ = addWatchDir(state, path)
				}
			}
			if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
				removeWatchDir(state, path)
			}
			if strings.EqualFold(filepath.Base(path), "SKILL.md") {
				trigger()
			}
		case <-timerChan:
			registerSkillWatchDirs(state, workspaceRoot, load, maxCandidates)
			bumpSkillsSnapshotVersion(workspaceRoot)
			timer = nil
			timerChan = nil
		case <-state.watcher.Errors:
			// Ignore watcher errors and keep best-effort tracking alive.
		}
	}
}

func registerSkillWatchDirs(state *skillsWatchState, workspaceRoot string, load skillsLoadConfig, maxCandidates int) {
	if state == nil || state.watcher == nil {
		return
	}
	dirs := collectSkillWatchDirs(workspaceRoot, load, maxCandidates)
	next := make(map[string]struct{}, len(dirs))
	for _, dir := range dirs {
		normalized := filepath.Clean(dir)
		if normalized == "" {
			continue
		}
		next[normalized] = struct{}{}
		_ = addWatchDir(state, normalized)
	}
	for watched := range state.watchedDirs {
		if _, ok := next[watched]; ok {
			continue
		}
		_ = state.watcher.Remove(watched)
		delete(state.watchedDirs, watched)
	}
}

func addWatchDir(state *skillsWatchState, path string) error {
	if state == nil || state.watcher == nil {
		return nil
	}
	cleaned := filepath.Clean(path)
	if cleaned == "" {
		return nil
	}
	if _, ok := state.watchedDirs[cleaned]; ok {
		return nil
	}
	if err := state.watcher.Add(cleaned); err != nil {
		return err
	}
	state.watchedDirs[cleaned] = struct{}{}
	return nil
}

func removeWatchDir(state *skillsWatchState, path string) {
	if state == nil || state.watcher == nil {
		return
	}
	cleaned := filepath.Clean(path)
	if cleaned == "" {
		return
	}
	if _, ok := state.watchedDirs[cleaned]; !ok {
		return
	}
	_ = state.watcher.Remove(cleaned)
	delete(state.watchedDirs, cleaned)
}

func collectSkillWatchDirs(workspaceRoot string, load skillsLoadConfig, maxCandidates int) []string {
	roots := resolveSkillRoots(workspaceRoot, load)
	limit := maxCandidates
	if limit <= 0 {
		limit = defaultMaxCandidatesPerRoot
	}
	dirs := make(map[string]struct{})
	for _, root := range roots {
		base := resolveNestedSkillsRoot(strings.TrimSpace(root.Path))
		if base == "" {
			continue
		}
		info, err := os.Stat(base)
		if err != nil || !info.IsDir() {
			continue
		}
		dirs[filepath.Clean(base)] = struct{}{}
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		candidates := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := strings.TrimSpace(entry.Name())
			if name == "" || strings.HasPrefix(name, ".") || name == "node_modules" {
				continue
			}
			candidates++
			if limit > 0 && candidates > limit {
				break
			}
			dirs[filepath.Clean(filepath.Join(base, name))] = struct{}{}
		}
	}
	if len(dirs) == 0 {
		return nil
	}
	result := make([]string, 0, len(dirs))
	for path := range dirs {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}
