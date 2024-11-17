package config

import (
	"path/filepath"
	"sync"
)

var (
	downloadOnce     sync.Once
	downloadInstance *DownloadManager
)

type DownloadManager struct {
	downloadURL string
	mu          sync.RWMutex
}

func GetDownloadInstance() *DownloadManager {
	downloadOnce.Do(func() {
		downloadInstance = &DownloadManager{}
	})
	return downloadInstance
}

func (dm *DownloadManager) SetDownloadURL(url string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.downloadURL = url
}

func (dm *DownloadManager) GetDownloadURL() string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.downloadURL
}

func (dm *DownloadManager) GetDownloadURLWithCanMe() string {
	return filepath.Join(dm.GetDownloadURL(), "canme")
}
