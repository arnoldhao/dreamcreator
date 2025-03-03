package storage

import (
	"CanMe/backend/consts"
	"CanMe/backend/models"
	"context"
	"log"
	"path"

	"github.com/vrischmann/userdir"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// PersistentStorage
type PersistentStorage struct {
	db *gorm.DB
}

var globalPersistentStorage *PersistentStorage

// SetGlobalPersistentStorage
func SetGlobalPersistentStorage(ps *PersistentStorage) {
	globalPersistentStorage = ps
}

// GetGlobalPersistentStorage
func GetGlobalPersistentStorage() *PersistentStorage {
	return globalPersistentStorage
}

// NewPersistentStorage create a new persistent storage
func NewPersistentStorage() (*PersistentStorage, error) {
	cachePath := path.Join(userdir.GetConfigHome(), consts.APP_NAME, "cache")
	dir := path.Dir(cachePath)
	if err := ensureDirExists(dir); err != nil {
		return nil, err
	}

	db, err := gorm.Open(
		sqlite.Open(cachePath),
		&gorm.Config{})
	if err != nil {
		return nil, err
	}

	log.Printf("cache path: %s\n", cachePath)
	return &PersistentStorage{
		db: db,
	}, nil
}

// db auto migrate
func (p *PersistentStorage) AutoMigrate(ctx context.Context) error {
	// 自动迁移所有模型
	if err := p.db.WithContext(ctx).AutoMigrate(
		&models.DownloadTask{},
		&models.StreamPart{},
		&models.CommonPart{},
		&Subtitles{},
		&LanguageData{},
		&Ollama{},
		&LLM{},
		&Model{},
		&CurrentModel{},
		&Downloads{},
	); err != nil {
		return err
	}
	return nil
}

// DB return a new query builder
func (p *PersistentStorage) DBWithoutContext() *gorm.DB {
	return p.db
}

// DB return a new query builder
func (p *PersistentStorage) DB(ctx context.Context) *gorm.DB {
	return p.db.WithContext(ctx)
}

// Create create a record
func (p *PersistentStorage) Create(ctx context.Context, value interface{}) error {
	return p.db.WithContext(ctx).Create(value).Error
}

// First get a record by condition
func (p *PersistentStorage) First(ctx context.Context, dest interface{}, conds ...interface{}) error {
	return p.db.WithContext(ctx).First(dest, conds...).Error
}

// Find get multiple records by condition
func (p *PersistentStorage) Find(ctx context.Context, dest interface{}, conds ...interface{}) error {
	return p.db.WithContext(ctx).Find(dest, conds...).Error
}

// Update update a record
func (p *PersistentStorage) Update(ctx context.Context, model interface{}, updates interface{}) error {
	return p.db.WithContext(ctx).Model(model).Updates(updates).Error
}

// Delete delete a record
func (p *PersistentStorage) Delete(ctx context.Context, value interface{}, conds ...interface{}) error {
	return p.db.WithContext(ctx).Delete(value, conds...).Error
}

// Raw execute a raw SQL query
func (p *PersistentStorage) Raw(ctx context.Context, sql string, values ...interface{}) *gorm.DB {
	return p.db.WithContext(ctx).Raw(sql, values...)
}
