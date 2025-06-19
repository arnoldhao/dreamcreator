package storage

import (
	"CanMe/backend/consts"
	"CanMe/backend/types"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"go.etcd.io/bbolt"
)

var (
	taskBucket       = []byte("tasks")
	imageBucket      = []byte("images")       // 用于存储图片的桶
	formatBucket     = []byte("formats")      // 用于存储格式的桶
	subtitleBucket   = []byte("subtitles")    // 用于存储字幕的桶
	dependencyBucket = []byte("dependencies") // 用于存储依赖信息的桶
	// other buckets...
)

type BoltStorage struct {
	path string
	db   *bbolt.DB
}

func NewBoltStorage() (*BoltStorage, error) {
	// 获取用户配置目录
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// 创建应用数据目录
	dbDir := filepath.Join(configDir, "CanMe")
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
