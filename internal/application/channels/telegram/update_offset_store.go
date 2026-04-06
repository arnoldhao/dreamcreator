package telegram

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type UpdateOffsetStore struct {
	path    string
	offsets map[string]int64
	mu      sync.Mutex
}

func NewUpdateOffsetStore(path string) *UpdateOffsetStore {
	if strings.TrimSpace(path) == "" {
		path = resolveTelegramOffsetPath()
	}
	store := &UpdateOffsetStore{
		path:    path,
		offsets: map[string]int64{},
	}
	if err := store.Load(); err != nil {
		zap.L().Warn("telegram: load update offset failed", zap.Error(err))
	}
	return store
}

func (store *UpdateOffsetStore) Load() error {
	if store == nil || strings.TrimSpace(store.path) == "" {
		return nil
	}
	data, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var offsets map[string]int64
	if err := json.Unmarshal(data, &offsets); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.offsets = offsets
	if store.offsets == nil {
		store.offsets = map[string]int64{}
	}
	return nil
}

func (store *UpdateOffsetStore) Get(accountID string) int64 {
	if store == nil {
		return 0
	}
	id := strings.TrimSpace(accountID)
	if id == "" {
		id = DefaultTelegramAccountID
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.offsets[id]
}

func (store *UpdateOffsetStore) Set(accountID string, offset int64) error {
	if store == nil || strings.TrimSpace(store.path) == "" {
		return nil
	}
	if offset <= 0 {
		return nil
	}
	id := strings.TrimSpace(accountID)
	if id == "" {
		id = DefaultTelegramAccountID
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if current, ok := store.offsets[id]; ok && current >= offset {
		return nil
	}
	store.offsets[id] = offset
	data, err := json.MarshalIndent(store.offsets, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(store.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(store.path, data, 0o644)
}

func (store *UpdateOffsetStore) Delete(accountID string) error {
	if store == nil || strings.TrimSpace(store.path) == "" {
		return nil
	}
	id := strings.TrimSpace(accountID)
	if id == "" {
		id = DefaultTelegramAccountID
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.offsets) == 0 {
		return removeOffsetFile(store.path)
	}
	if _, ok := store.offsets[id]; !ok {
		return nil
	}
	delete(store.offsets, id)
	if len(store.offsets) == 0 {
		return removeOffsetFile(store.path)
	}
	data, err := json.MarshalIndent(store.offsets, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(store.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(store.path, data, 0o644)
}

func (store *UpdateOffsetStore) ClearAll() error {
	if store == nil || strings.TrimSpace(store.path) == "" {
		return nil
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.offsets = map[string]int64{}
	return removeOffsetFile(store.path)
}

func resolveTelegramOffsetPath() string {
	targetDir := filepath.Join(os.TempDir(), "dreamcreator")
	if configDir, err := os.UserConfigDir(); err == nil {
		targetDir = filepath.Join(configDir, "dreamcreator")
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		targetDir = filepath.Join(os.TempDir(), "dreamcreator")
		_ = os.MkdirAll(targetDir, 0o755)
	}
	return filepath.Join(targetDir, "telegram_update_offset.json")
}

func removeOffsetFile(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
