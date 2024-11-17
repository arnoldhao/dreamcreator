package storage

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/specials/proxy"
	"CanMe/backend/types"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"CanMe/backend/pkg/specials/config"

	"gopkg.in/yaml.v3"
)

type PreferencesStorage struct {
	storage *LocalStorage
	mutex   sync.Mutex
}

func NewPreferences() *PreferencesStorage {
	storage := NewLocalStore(consts.PREFERENCES_FILE_NAME)
	log.Printf("preferences path: %s\n", storage.ConfPath)
	return &PreferencesStorage{
		storage: storage,
	}
}

func (p *PreferencesStorage) DefaultPreferences() types.Preferences {
	return types.NewPreferences()
}

func (p *PreferencesStorage) getPreferences() (ret types.Preferences) {
	ret = p.DefaultPreferences()
	b, err := p.storage.Load()
	if err != nil {
		return
	}

	if err = yaml.Unmarshal(b, &ret); err != nil {
		ret = p.DefaultPreferences()
		return
	}
	return
}

// GetPreferences Get preferences from local
func (p *PreferencesStorage) GetPreferences() (ret types.Preferences) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ret = p.getPreferences()
	if ret.General.ScanSize <= 0 {
		ret.General.ScanSize = consts.DEFAULT_SCAN_SIZE
	}
	ret.Behavior.AsideWidth = max(ret.Behavior.AsideWidth, consts.DEFAULT_ASIDE_WIDTH)
	ret.Behavior.WindowWidth = max(ret.Behavior.WindowWidth, consts.MIN_WINDOW_WIDTH)
	ret.Behavior.WindowHeight = max(ret.Behavior.WindowHeight, consts.MIN_WINDOW_HEIGHT)

	// set proxy
	if ret.Proxy.Enabled {
		if ret.Proxy.Addr != "" && ret.Proxy.Protocal != "" && ret.Proxy.Port != 0 {
			url := fmt.Sprintf("%s://%s:%d", ret.Proxy.Protocal, ret.Proxy.Addr, ret.Proxy.Port)
			err := proxy.GetInstance().SetProxy(url)
			if err != nil {
				log.Println("set proxy error:", err)
			} else {
				log.Println("set proxy success:", url)
			}
		}
	}

	// set download dir
	if ret.Download.Dir == "" {
		ret.Download.Dir = p.getDefaultDownloadDir()
	} else {
		config.GetDownloadInstance().SetDownloadURL(ret.Download.Dir)
	}
	return
}

func (p *PreferencesStorage) setPreferences(pf *types.Preferences, key string, value any) error {
	parts := strings.Split(key, ".")
	if len(parts) > 0 {
		var reflectValue reflect.Value
		if reflect.TypeOf(pf).Kind() == reflect.Ptr {
			reflectValue = reflect.ValueOf(pf).Elem()
		} else {
			reflectValue = reflect.ValueOf(pf)
		}
		for i, part := range parts {
			part = strings.ToUpper(part[:1]) + part[1:]
			reflectValue = reflectValue.FieldByName(part)
			if reflectValue.IsValid() {
				if i == len(parts)-1 {
					reflectValue.Set(reflect.ValueOf(value))
					return nil
				}
			} else {
				break
			}
		}
	}

	return fmt.Errorf("invalid key path(%s)", key)
}

func (p *PreferencesStorage) savePreferences(pf *types.Preferences) error {
	b, err := yaml.Marshal(pf)
	if err != nil {
		return err
	}

	if err = p.storage.Store(b); err != nil {
		return err
	}
	return nil
}

// SetPreferences replace preferences
func (p *PreferencesStorage) SetPreferences(pf *types.Preferences) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return p.savePreferences(pf)
}

// UpdatePreferences update values by key paths, the key path use "." to indicate multiple level
func (p *PreferencesStorage) UpdatePreferences(values map[string]any) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	pf := p.getPreferences()
	for path, v := range values {
		if err := p.setPreferences(&pf, path, v); err != nil {
			return err
		}
	}
	log.Println("after save", pf)

	return p.savePreferences(&pf)
}

func (p *PreferencesStorage) RestoreDefault() types.Preferences {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	pf := p.DefaultPreferences()
	p.savePreferences(&pf)
	return pf
}

func (p *PreferencesStorage) getDefaultDownloadDir() string {
	var downloadDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: %USERPROFILE%\Downloads
		downloadDir = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	case "darwin":
		// macOS: ~/Downloads
		homeDir, _ := os.UserHomeDir()
		downloadDir = filepath.Join(homeDir, "Downloads")
	case "linux":
		// Linux: ~/Downloads
		homeDir, _ := os.UserHomeDir()
		downloadDir = filepath.Join(homeDir, "Downloads")
	default:
		// default
		downloadDir = "Downloads"
	}

	// make sure dir exists
	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		os.MkdirAll(downloadDir, 0755)
	}

	return downloadDir
}
