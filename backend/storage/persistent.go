package storage

import (
	"CanMe/backend/consts"
	"context"
	"log"
	"path"

	"github.com/vrischmann/userdir"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type PersistentStorage struct {
	db *gorm.DB
}

// global persistent storage
var globalPersistentStorage *PersistentStorage

func SetGlobalPersistentStorage(ps *PersistentStorage) {
	globalPersistentStorage = ps
}

func GetGlobalPersistentStorage() *PersistentStorage {
	return globalPersistentStorage
}

// db auto migrate
func (p *PersistentStorage) AutoMigrate(ctx context.Context) error {
	// migrate subtitles
	err := p.db.WithContext(ctx).AutoMigrate(&Subtitles{})
	if err != nil {
		return err
	}

	// migrate language data
	err = p.db.WithContext(ctx).AutoMigrate(&LanguageData{})
	if err != nil {
		return err
	}

	// migrate ollama
	err = p.db.WithContext(ctx).AutoMigrate(&Ollama{})
	if err != nil {
		return err
	}

	// migrate llm
	err = p.db.WithContext(ctx).AutoMigrate(&LLM{})
	if err != nil {
		return err
	}

	// migrate model
	err = p.db.WithContext(ctx).AutoMigrate(&Model{})
	if err != nil {
		return err
	}

	// migrate current model
	err = p.db.WithContext(ctx).AutoMigrate(&CurrentModel{})
	if err != nil {
		return err
	}

	return nil
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
