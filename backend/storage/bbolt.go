package storage

import (
	"dreamcreator/backend/consts"
	"dreamcreator/backend/types"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

var (
	taskBucket          = []byte("tasks")
	imageBucket         = []byte("images")       // 用于存储图片的桶
	formatBucket        = []byte("formats")      // 用于存储格式的桶
	subtitleBucket      = []byte("subtitles")    // 用于存储字幕的桶
	dependencyBucket    = []byte("dependencies") // 用于存储依赖信息的桶
	cookiesBucketV2     = []byte("cookies_v2")   // 用于存储新的 Cookie 集合
	legacyCookiesBucket = []byte("cookies")      // 历史 Cookie 桶，启动时清理
	// LLM Providers & Profiles
	providersBucket      = []byte("providers")       // LLM Provider 管理
	globalProfilesBucket = []byte("global_profiles") // Global LLM Profiles (not bound to provider/model)
	modelsCacheBucket    = []byte("models_cache")    // Provider 模型缓存
	modelsMetaBucket     = []byte("models_meta")     // 模型元信息（可选，更丰富）
	glossaryBucket       = []byte("glossary")        // 翻译术语表（条目）
	glossarySetsBucket   = []byte("glossary_sets")   // 术语集合（全局）
	targetLangsBucket    = []byte("target_langs")    // 目标语言列表（AI 翻译使用）
	// other buckets...
)

type BoltStorage struct {
	path string
	db   *bbolt.DB // Keep DB private; expose required ops via methods
}

var NewBoltStorageForTest func(path string) (*BoltStorage, error)

func NewBoltStorage() (*BoltStorage, error) {
	// 获取用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// 创建应用数据目录
	dbDir := filepath.Join(configDir, consts.AppDataDirName())
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	// 打开数据库
	dbPath := filepath.Join(dbDir, consts.BBOLT_DB_NAME)
	db, err := bbolt.Open(dbPath, 0600, &bbolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	// 创建桶
	err = db.Update(func(tx *bbolt.Tx) error {
		// create tasks buckets
		if _, err := tx.CreateBucketIfNotExists(taskBucket); err != nil {
			return err
		}

		// create image buckets
		if _, err := tx.CreateBucketIfNotExists(imageBucket); err != nil {
			return err
		}

		// create format buckets
		if _, err := tx.CreateBucketIfNotExists(formatBucket); err != nil {
			return err
		}

		// create subtitle buckets
		if _, err := tx.CreateBucketIfNotExists(subtitleBucket); err != nil {
			return err
		}

		// create dependency buckets
		if _, err := tx.CreateBucketIfNotExists(dependencyBucket); err != nil {
			return err
		}

		// create llm provider buckets
		if _, err := tx.CreateBucketIfNotExists(providersBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(globalProfilesBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(modelsCacheBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(modelsMetaBucket); err != nil {
			return err
		}

		// glossary bucket
		if _, err := tx.CreateBucketIfNotExists(glossaryBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(glossarySetsBucket); err != nil {
			return err
		}

		// target languages bucket
		if _, err := tx.CreateBucketIfNotExists(targetLangsBucket); err != nil {
			return err
		}

		// drop legacy cookies bucket if still present
		if legacy := tx.Bucket(legacyCookiesBucket); legacy != nil {
			if err := tx.DeleteBucket(legacyCookiesBucket); err != nil {
				return err
			}
		}
		// create cookies_v2 bucket
		if _, err := tx.CreateBucketIfNotExists(cookiesBucketV2); err != nil {
			return err
		}
		// create other buckets...
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &BoltStorage{path: dbPath, db: db}, nil
}

func (s *BoltStorage) Path() string {
	return s.path
}

func (s *BoltStorage) SaveTask(task *types.DtTaskStatus) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(taskBucket)

		task.UpdatedAt = time.Now().Unix()
		encoded, err := json.Marshal(task)
		if err != nil {
			return err
		}

		return b.Put([]byte(task.ID), encoded)
	})
}

func (s *BoltStorage) GetTask(id string) (*types.DtTaskStatus, error) {
	var task types.DtTaskStatus

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(taskBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("task not found: %s", id)
		}

		return json.Unmarshal(data, &task)
	})

	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *BoltStorage) ListTasks() ([]*types.DtTaskStatus, error) {
	var tasks []*types.DtTaskStatus

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(taskBucket)

		return b.ForEach(func(k, v []byte) error {
			var task types.DtTaskStatus
			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}
			tasks = append(tasks, &task)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// DeleteTask 从存储中删除指定ID的任务
func (s *BoltStorage) DeleteTask(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(taskBucket)
		return b.Delete([]byte(id))
	})
}

// SaveImage 保存图片信息到存储
func (s *BoltStorage) SaveImage(image *types.ImageInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(imageBucket)

		encoded, err := json.Marshal(image)
		if err != nil {
			return err
		}

		return b.Put([]byte(image.URL), encoded)
	})
}

func (s *BoltStorage) GetImage(url string) (*types.ImageInfo, error) {
	var image types.ImageInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(imageBucket)
		data := b.Get([]byte(url))
		if data == nil {
			return fmt.Errorf("image not found: %s", url)
		}

		return json.Unmarshal(data, &image)
	})

	if err != nil {
		return nil, err
	}

	return &image, nil
}

func (s *BoltStorage) DeleteImage(url string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(imageBucket)
		return b.Delete([]byte(url))
	})
}

func (s *BoltStorage) Close() error {
	return s.db.Close()
}

// -------- Glossary CRUD --------

func (s *BoltStorage) SaveGlossaryEntry(e *types.GlossaryEntry) error {
	if e == nil || strings.TrimSpace(e.ID) == "" {
		return fmt.Errorf("glossary entry or id empty")
	}
	if strings.TrimSpace(e.SetID) == "" {
		e.SetID = "default"
	}
	e.UpdatedAt = time.Now().Unix()
	if e.CreatedAt == 0 {
		e.CreatedAt = e.UpdatedAt
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(glossaryBucket)
		buf, err := json.Marshal(e)
		if err != nil {
			return err
		}
		return b.Put([]byte(e.ID), buf)
	})
}

func (s *BoltStorage) GetGlossaryEntry(id string) (*types.GlossaryEntry, error) {
	var out types.GlossaryEntry
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(glossaryBucket)
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("glossary not found: %s", id)
		}
		return json.Unmarshal(v, &out)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *BoltStorage) ListGlossaryEntries() ([]*types.GlossaryEntry, error) {
	var list []*types.GlossaryEntry
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(glossaryBucket)
		return b.ForEach(func(k, v []byte) error {
			var e types.GlossaryEntry
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}
			list = append(list, &e)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	// stable sort by created time then source (optional)
	sort.Slice(list, func(i, j int) bool {
		if list[i].CreatedAt == list[j].CreatedAt {
			return list[i].Source < list[j].Source
		}
		return list[i].CreatedAt < list[j].CreatedAt
	})
	return list, nil
}

func (s *BoltStorage) ListGlossaryEntriesBySet(setID string) ([]*types.GlossaryEntry, error) {
	if strings.TrimSpace(setID) == "" {
		setID = "default"
	}
	list, err := s.ListGlossaryEntries()
	if err != nil {
		return nil, err
	}
	out := make([]*types.GlossaryEntry, 0, len(list))
	for _, e := range list {
		if e != nil && strings.TrimSpace(e.SetID) == setID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (s *BoltStorage) DeleteGlossaryEntry(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(glossaryBucket)
		return b.Delete([]byte(id))
	})
}

// -------- Glossary Sets CRUD --------

func (s *BoltStorage) SaveGlossarySet(gs *types.GlossarySet) error {
	if gs == nil || strings.TrimSpace(gs.ID) == "" {
		return fmt.Errorf("glossary set or id empty")
	}
	gs.UpdatedAt = time.Now().Unix()
	if gs.CreatedAt == 0 {
		gs.CreatedAt = gs.UpdatedAt
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(glossarySetsBucket)
		buf, err := json.Marshal(gs)
		if err != nil {
			return err
		}
		return b.Put([]byte(gs.ID), buf)
	})
}

// -------- Target Languages CRUD --------

// SaveTargetLanguage creates/updates a target language by its code (keyed by code)
func (s *BoltStorage) SaveTargetLanguage(l *types.TargetLanguage) error {
	if l == nil || strings.TrimSpace(l.Code) == "" {
		return fmt.Errorf("target language or code empty")
	}
	l.UpdatedAt = time.Now().Unix()
	if l.CreatedAt == 0 {
		l.CreatedAt = l.UpdatedAt
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(targetLangsBucket)
		buf, err := json.Marshal(l)
		if err != nil {
			return err
		}
		return b.Put([]byte(l.Code), buf)
	})
}

// GetTargetLanguage returns one language by code
func (s *BoltStorage) GetTargetLanguage(code string) (*types.TargetLanguage, error) {
	var out types.TargetLanguage
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(targetLangsBucket)
		v := b.Get([]byte(code))
		if v == nil {
			return fmt.Errorf("target language not found: %s", code)
		}
		return json.Unmarshal(v, &out)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTargetLanguages lists all target languages
func (s *BoltStorage) ListTargetLanguages() ([]*types.TargetLanguage, error) {
	var list []*types.TargetLanguage
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(targetLangsBucket)
		return b.ForEach(func(k, v []byte) error {
			var l types.TargetLanguage
			if err := json.Unmarshal(v, &l); err != nil {
				return err
			}
			list = append(list, &l)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Name == list[j].Name {
			return list[i].Code < list[j].Code
		}
		return list[i].Name < list[j].Name
	})
	return list, nil
}

// DeleteTargetLanguage removes one language by code
func (s *BoltStorage) DeleteTargetLanguage(code string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(targetLangsBucket)
		return b.Delete([]byte(code))
	})
}

func (s *BoltStorage) GetGlossarySet(id string) (*types.GlossarySet, error) {
	var out types.GlossarySet
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(glossarySetsBucket)
		v := b.Get([]byte(id))
		if v == nil {
			return fmt.Errorf("glossary set not found: %s", id)
		}
		return json.Unmarshal(v, &out)
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *BoltStorage) ListGlossarySets() ([]*types.GlossarySet, error) {
	var list []*types.GlossarySet
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(glossarySetsBucket)
		return b.ForEach(func(k, v []byte) error {
			var gs types.GlossarySet
			if err := json.Unmarshal(v, &gs); err != nil {
				return err
			}
			list = append(list, &gs)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt < list[j].CreatedAt })
	return list, nil
}

func (s *BoltStorage) DeleteGlossarySet(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		// 1) delete the set record
		if err := tx.Bucket(glossarySetsBucket).Delete([]byte(id)); err != nil {
			return err
		}
		// 2) cascade: remove all glossary entries that belong to this set
		gb := tx.Bucket(glossaryBucket)
		// collect keys to delete to avoid cursor invalidation during iteration
		keys := make([][]byte, 0, 16)
		if err := gb.ForEach(func(k, v []byte) error {
			var e types.GlossaryEntry
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}
			if strings.TrimSpace(e.SetID) == strings.TrimSpace(id) {
				kk := make([]byte, len(k))
				copy(kk, k)
				keys = append(keys, kk)
			}
			return nil
		}); err != nil {
			return err
		}
		for _, k := range keys {
			if err := gb.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

// formatIDToKey 将整数 ID 转换为字节切片键
func formatIDToKey(id int) []byte {
	return []byte(strconv.Itoa(id))
}

// SaveConversionFormat 保存单个转换格式
func (s *BoltStorage) SaveConversionFormat(format *types.ConversionFormat) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(formatBucket)
		key := formatIDToKey(format.ID)

		encoded, err := json.Marshal(format)
		if err != nil {
			return fmt.Errorf("failed to marshal conversion format (ID: %d): %w", format.ID, err)
		}
		return b.Put(key, encoded)
	})
}

// GetConversionFormat 根据 ID 获取单个转换格式
func (s *BoltStorage) GetConversionFormat(id int) (*types.ConversionFormat, error) {
	var format types.ConversionFormat
	key := formatIDToKey(id)

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(formatBucket)
		data := b.Get(key)
		if data == nil {
			return fmt.Errorf("conversion format with ID %d not found", id)
		}
		return json.Unmarshal(data, &format)
	})

	if err != nil {
		return nil, err
	}
	return &format, nil
}

// ListAllConversionFormats 获取所有存储的转换格式
func (s *BoltStorage) ListAllConversionFormats() ([]*types.ConversionFormat, error) {
	var formats []*types.ConversionFormat
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(formatBucket)
		return b.ForEach(func(k, v []byte) error {
			var format types.ConversionFormat
			if err := json.Unmarshal(v, &format); err != nil {
				// 可以选择记录错误并跳过，或者直接返回错误
				// logger.Error("Failed to unmarshal conversion format during list", zap.ByteString("key", k), zap.Error(err))
				// return nil // 跳过这个损坏的条目
				return fmt.Errorf("failed to unmarshal conversion format (key: %s): %w", string(k), err)
			}
			formats = append(formats, &format)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	// 按 ID 排序 (可选，但通常更好)
	sort.Slice(formats, func(i, j int) bool {
		return formats[i].ID < formats[j].ID
	})

	return formats, nil
}

// InitializeOrRestoreDefaultConversionFormats 初始化或恢复默认转换格式
// overwrite: true 会覆盖现有所有格式；false 只在数据库为空时添加默认格式
func (s *BoltStorage) InitializeOrRestoreDefaultConversionFormats(overwrite bool) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(formatBucket)

		if overwrite {
			// 如果覆盖，先删除所有现有格式
			// 注意：b.ForEach + b.Delete 在同一个事务中可能导致迭代器问题
			// 更安全的方式是先收集所有key，然后删除
			var keysToClear [][]byte
			err := b.ForEach(func(k, v []byte) error {
				keysToClear = append(keysToClear, k)
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to list keys for clearing formats: %w", err)
			}
			for _, key := range keysToClear {
				if err := b.Delete(key); err != nil {
					return fmt.Errorf("failed to delete format (key: %s) during restore: %w", string(key), err)
				}
			}
		} else {
			// 如果不覆盖，检查是否已经有数据
			cursor := b.Cursor()
			k, _ := cursor.First()
			if k != nil {
				// 已经有数据，并且不要求覆盖，则不执行任何操作
				return nil
			}
		}

		// 写入默认格式
		for _, defaultFormat := range types.DefaultConversionFormats {
			formatCopy := defaultFormat // 必须复制，否则后续迭代会修改已保存的指针
			key := formatIDToKey(formatCopy.ID)
			encoded, err := json.Marshal(&formatCopy)
			if err != nil {
				return fmt.Errorf("failed to marshal default format (ID: %d): %w", formatCopy.ID, err)
			}
			if err := b.Put(key, encoded); err != nil {
				return fmt.Errorf("failed to save default format (ID: %d): %w", formatCopy.ID, err)
			}
		}
		return nil
	})
}

/* SubtitlesBucket */
// SaveSubtitle saves a subtitle to the storage
func (s *BoltStorage) SaveSubtitle(sub *types.SubtitleProject) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subtitleBucket)

		sub.UpdatedAt = time.Now().Unix()
		encoded, err := json.Marshal(sub)
		if err != nil {
			return err
		}

		return b.Put([]byte(sub.ID), encoded)
	})
}

// GetSubtitle retrieves a subtitle from the storage
func (s *BoltStorage) GetSubtitle(id string) (*types.SubtitleProject, error) {
	var sub types.SubtitleProject

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subtitleBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("subtitle not found: %s", id)
		}

		return json.Unmarshal(data, &sub)
	})

	if err != nil {
		return nil, err
	}

	return &sub, nil
}

// ListSubtitles lists all subtitles in the storage
func (s *BoltStorage) ListSubtitles() ([]*types.SubtitleProject, error) {
	var subs []*types.SubtitleProject

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subtitleBucket)

		return b.ForEach(func(k, v []byte) error {
			var sub types.SubtitleProject
			if err := json.Unmarshal(v, &sub); err != nil {
				return err
			}
			subs = append(subs, &sub)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return subs, nil
}

// DeleteSubtitle deletes a subtitle from the storage
func (s *BoltStorage) DeleteSubtitle(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subtitleBucket)
		return b.Delete([]byte(id))
	})
}

// DeleteAllSubtitle deletes all subtitles from the storage
func (s *BoltStorage) DeleteAllSubtitle() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(subtitleBucket)
		return b.ForEach(func(k, v []byte) error {
			return b.Delete(k)
		})
	})
}

// SaveDependency 保存依赖信息到存储
func (s *BoltStorage) SaveDependency(dep *types.DependencyInfo) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(dependencyBucket)

		// 更新最后检查时间
		dep.LastCheck = time.Now()
		encoded, err := json.Marshal(dep)
		if err != nil {
			return fmt.Errorf("failed to marshal dependency %s: %w", dep.Type, err)
		}

		return b.Put([]byte(dep.Type), encoded)
	})
}

// GetDependency 根据类型获取依赖信息
func (s *BoltStorage) GetDependency(depType types.DependencyType) (*types.DependencyInfo, error) {
	var dep types.DependencyInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(dependencyBucket)
		data := b.Get([]byte(depType))
		if data == nil {
			return fmt.Errorf("dependency not found: %s", depType)
		}

		return json.Unmarshal(data, &dep)
	})

	if err != nil {
		return nil, err
	}

	return &dep, nil
}

// ListAllDependencies 获取所有依赖信息
func (s *BoltStorage) ListAllDependencies() ([]*types.DependencyInfo, error) {
	var dependencies []*types.DependencyInfo

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(dependencyBucket)

		return b.ForEach(func(k, v []byte) error {
			var dep types.DependencyInfo
			if err := json.Unmarshal(v, &dep); err != nil {
				return err
			}
			dependencies = append(dependencies, &dep)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return dependencies, nil
}

// DeleteDependency 删除指定类型的依赖信息
func (s *BoltStorage) DeleteDependency(depType types.DependencyType) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(dependencyBucket)
		return b.Delete([]byte(depType))
	})
}

// UpdateDependencyVersion 更新依赖版本信息
func (s *BoltStorage) UpdateDependencyVersion(depType types.DependencyType, version, latestVersion string, needUpdate bool) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(dependencyBucket)

		// 先获取现有数据
		data := b.Get([]byte(depType))
		if data == nil {
			return fmt.Errorf("dependency not found: %s", depType)
		}

		var dep types.DependencyInfo
		if err := json.Unmarshal(data, &dep); err != nil {
			return fmt.Errorf("failed to unmarshal dependency %s: %w", depType, err)
		}

		// 更新版本信息
		dep.Version = version
		dep.LatestVersion = latestVersion
		dep.NeedUpdate = needUpdate
		dep.LastCheck = time.Now()

		// 重新保存
		encoded, err := json.Marshal(&dep)
		if err != nil {
			return fmt.Errorf("failed to marshal updated dependency %s: %w", depType, err)
		}

		return b.Put([]byte(depType), encoded)
	})
}

// SaveCookieCollection 写入或更新 Cookie 集合
func (s *BoltStorage) SaveCookieCollection(collection *types.CookieCollection) error {
	if collection == nil || collection.ID == "" {
		return fmt.Errorf("collection or collection id is empty")
	}

	collection.UpdatedAt = time.Now()
	if collection.CreatedAt.IsZero() {
		collection.CreatedAt = collection.UpdatedAt
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cookiesBucketV2)
		encoded, err := json.Marshal(collection)
		if err != nil {
			return fmt.Errorf("failed to marshal collection %s: %w", collection.ID, err)
		}
		return b.Put([]byte(collection.ID), encoded)
	})
}

// DeleteCookieCollection 删除指定集合
func (s *BoltStorage) DeleteCookieCollection(id string) error {
	if id == "" {
		return fmt.Errorf("collection id is empty")
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cookiesBucketV2)
		return b.Delete([]byte(id))
	})
}

// GetCookieCollection 返回指定 ID 的集合
func (s *BoltStorage) GetCookieCollection(id string) (*types.CookieCollection, error) {
	if id == "" {
		return nil, fmt.Errorf("collection id is empty")
	}

	var collection *types.CookieCollection
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cookiesBucketV2)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("cookies collection not found: %s", id)
		}
		return json.Unmarshal(data, &collection)
	})
	if err != nil {
		return nil, err
	}
	return collection, nil
}

// ListCookieCollections 返回所有存储的集合
func (s *BoltStorage) ListCookieCollections() ([]*types.CookieCollection, error) {
	var collections []*types.CookieCollection

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(cookiesBucketV2)
		return b.ForEach(func(k, v []byte) error {
			var collection *types.CookieCollection
			if err := json.Unmarshal(v, &collection); err != nil {
				return err
			}
			collections = append(collections, collection)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return collections, nil
}

// FindCookieCollectionByBrowser 按浏览器名称查找集合（source 匹配 yt-dlp）
func (s *BoltStorage) FindCookieCollectionByBrowser(browser string) (*types.CookieCollection, error) {
	all, err := s.ListCookieCollections()
	if err != nil {
		return nil, err
	}
	for _, c := range all {
		if c != nil && c.Source == types.CookieSourceYTDLP && c.Browser == browser {
			return c, nil
		}
	}
	return nil, fmt.Errorf("cookies collection not found for browser: %s", browser)
}
