package storage

import (
	"CanMe/backend/consts"
	"CanMe/backend/types"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

var (
	taskBucket  = []byte("tasks")
	imageBucket = []byte("images") // 用于存储图片的桶
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
