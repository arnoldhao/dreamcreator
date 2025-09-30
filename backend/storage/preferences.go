package storage

import (
	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/types"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type PreferencesStorage struct {
	storage *LocalStorage
	mutex   sync.Mutex
}

func NewPreferences() *PreferencesStorage {
	storage := NewLocalStore(consts.PREFERENCES_FILE_NAME)
	logger.Debug("preferences path", zap.String("path", storage.ConfPath))
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
		ensureTelemetryDefaults(&ret)
		if os.IsNotExist(err) {
			// Persist defaults so subsequent loads use the same client ID.
			_ = p.savePreferences(&ret)
		}
		return
	}

	if err = yaml.Unmarshal(b, &ret); err != nil {
		ret = p.DefaultPreferences()
	}

	// 如果 logger 配置为空，使用默认配置
	if reflect.DeepEqual(ret.Logger, logger.Config{}) {
		ret.Logger = *logger.DefaultConfig()
	}

	// 迁移：将旧的 General.Theme (auto/light/dark) 映射到 General.Appearance
	migrated := false
	if ret.General.Appearance == "" {
		// if theme carried old appearance value
		switch strings.ToLower(ret.General.Theme) {
		case "auto", "light", "dark":
			ret.General.Appearance = ret.General.Theme
			ret.General.Theme = "blue"
			migrated = true
		default:
			// no appearance defined, set defaults
			ret.General.Appearance = "auto"
			if ret.General.Theme == "" {
				ret.General.Theme = "blue"
			}
		}
	} else {
		// ensure theme has a default if missing
		if ret.General.Theme == "" {
			ret.General.Theme = "blue"
			migrated = true
		}
	}

	if ensureTelemetryDefaults(&ret) {
		migrated = true
	}

	if migrated {
		// best-effort persist migration so future loads are clean
		_ = p.savePreferences(&ret)
	}

	return
}

// GetPreferences Get preferences from local
func (p *PreferencesStorage) GetPreferences() (ret types.Preferences) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ret = p.getPreferences()
	ret.Behavior.WindowWidth = max(ret.Behavior.WindowWidth, consts.MIN_WINDOW_WIDTH)
	ret.Behavior.WindowHeight = max(ret.Behavior.WindowHeight, consts.MIN_WINDOW_HEIGHT)

	if reflect.DeepEqual(ret.ListendInfo, types.ListendInfo{}) {
		ret.ListendInfo = types.DefaultListendInfo()
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

	return p.savePreferences(&pf)
}

func (p *PreferencesStorage) RestoreDefault() types.Preferences {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	pf := p.DefaultPreferences()
	ensureTelemetryDefaults(&pf)
	p.savePreferences(&pf)
	return pf
}

func (p *PreferencesStorage) ConfigPath() string {
	return p.storage.ConfPath
}

func ensureTelemetryDefaults(pref *types.Preferences) bool {
	if strings.TrimSpace(pref.Telemetry.ClientID) == "" {
		pref.Telemetry.ClientID = uuid.NewString()
		if !pref.Telemetry.Enabled {
			pref.Telemetry.Enabled = true
		}
		return true
	}
	return false
}
