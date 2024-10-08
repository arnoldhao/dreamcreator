package storage

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type LLM struct {
	gorm.Model
	Num         int     `gorm:"not null" json:"num" yaml:"num"`
	Name        string  `gorm:"size:50;not null;uniqueIndex" json:"name" yaml:"name"`
	Region      string  `gorm:"size:20;not null" json:"region" yaml:"region"`
	BaseURL     string  `gorm:"size:255;not null" json:"baseURL" yaml:"baseURL"`
	APIKey      string  `gorm:"size:255" json:"APIKey" yaml:"APIKey"`
	Available   bool    `gorm:"default:false" json:"available" yaml:"available"`
	Icon        string  `gorm:"size:50" json:"icon" yaml:"icon"`
	Show        bool    `gorm:"default:true" json:"show" yaml:"show"`
	Initialized bool    `gorm:"default:false" json:"initialized" yaml:"initialized"`
	Models      []Model `gorm:"foreignKey:LLMId" json:"models" yaml:"models"`
}

type Model struct {
	gorm.Model
	LLMId       uint   `gorm:"not null" json:"llm_id" yaml:"llm_id"`
	Num         int    `gorm:"not null" json:"num" yaml:"num"`
	Name        string `gorm:"size:50;not null" json:"name" yaml:"name"`
	Available   bool   `gorm:"default:false" json:"available" yaml:"available"`
	Description string `gorm:"size:255" json:"description" yaml:"description"`
}

type CurrentModel struct {
	gorm.Model
	Key       string `gorm:"primaryKey;type:varchar(36);not null;unique"`
	LLMName   string `gorm:"size:50;not null;uniqueIndex:idx_llm_model" json:"llmName" yaml:"llmName"`
	ModelName string `gorm:"size:50;not null;uniqueIndex:idx_llm_model" json:"modelName" yaml:"modelName"`
	LLM       LLM    `gorm:"foreignKey:LLMName;references:Name" json:"llm" yaml:"llm"`
}

func (l *LLM) Create(ctx context.Context) error {
	// check if llm exists but was soft deleted
	var existingLLM LLM
	err := GetGlobalPersistentStorage().DB(ctx).Unscoped().Where("name = ?", l.Name).First(&existingLLM).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if existingLLM.ID != 0 {
		// if soft deleted, restore it
		l.ID = existingLLM.ID
		return GetGlobalPersistentStorage().DB(ctx).Unscoped().Save(l).Error
	}

	// if not exists, create new record
	// search max num
	var maxNum int
	err = GetGlobalPersistentStorage().DB(ctx).Model(&LLM{}).Select("COALESCE(MAX(num), 0)").Scan(&maxNum).Error
	if err != nil {
		return err
	}

	// set new num
	l.Num = maxNum + 1

	return GetGlobalPersistentStorage().Create(ctx, l)
}

func (l *LLM) Read(ctx context.Context, name string) error {
	return GetGlobalPersistentStorage().First(ctx, l, "name = ?", name)
}

func (l *LLM) Update(ctx context.Context) error {
	return GetGlobalPersistentStorage().DB(ctx).Model(l).Where("name = ?", l.Name).Updates(l).Error
}

func (l *LLM) Delete(ctx context.Context) error {
	if l.Name == "" {
		return errors.New("llm name is empty")
	}

	err := l.Read(ctx, l.Name)
	if err != nil {
		return err
	}

	if l.Initialized {
		return errors.New("llm is initialized, cannot be deleted")
	}

	// delete LLM
	err = GetGlobalPersistentStorage().Delete(ctx, l)
	if err != nil {
		return err
	}

	// reorder LLMs
	return l.reorderLLMs(ctx)
}

func (l *LLM) reorderLLMs(ctx context.Context) error {
	var llms []LLM
	err := GetGlobalPersistentStorage().DB(ctx).Order("num").Find(&llms).Error
	if err != nil {
		return err
	}

	for i, llm := range llms {
		llm.Num = i + 1
		err = llm.Update(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (l *LLM) List(ctx context.Context) ([]LLM, error) {
	var llms []LLM
	err := GetGlobalPersistentStorage().DB(ctx).Find(&llms).Error

	if len(llms) == 0 {
		err = l.Initialize(ctx)
		if err != nil {
			return nil, err
		}

		_ = GetGlobalPersistentStorage().DB(ctx).Find(&llms).Error
	}

	return llms, err
}

func (l *LLM) ListLLMsAndModels(ctx context.Context) ([]LLM, error) {
	var llms []LLM
	err := GetGlobalPersistentStorage().DB(ctx).Preload("Models").Find(&llms).Error

	if len(llms) == 0 {
		err = l.Initialize(ctx)
		if err != nil {
			return nil, err
		}

		_ = GetGlobalPersistentStorage().DB(ctx).Preload("Models").Find(&llms).Error
	}

	return llms, err
}

func (l *LLM) Initialize(ctx context.Context) error {
	ollama := LLM{
		Name:        "ollama",
		Region:      "localhost",
		BaseURL:     "http://localhost:11434",
		APIKey:      "ollama",
		Available:   true,
		Icon:        "ollama",
		Show:        true,
		Initialized: true,
	}
	// if not exists, create new record
	return ollama.Create(ctx)
}

func (m *Model) Create(ctx context.Context) error {
	// check if model exists but was soft deleted
	var existingModel Model
	err := GetGlobalPersistentStorage().DB(ctx).Unscoped().Where("llm_id = ? AND name = ?", m.LLMId, m.Name).First(&existingModel).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if existingModel.ID != 0 {
		// if soft deleted, restore it
		m.ID = existingModel.ID
		return GetGlobalPersistentStorage().DB(ctx).Unscoped().Save(m).Error
	}

	// if not exists, create new record
	// search max num
	var maxNum int
	err = GetGlobalPersistentStorage().DB(ctx).Model(&Model{}).Where("llm_id = ?", m.LLMId).Select("COALESCE(MAX(num), 0)").Scan(&maxNum).Error
	if err != nil {
		return err
	}

	// set new num
	m.Num = maxNum + 1

	return GetGlobalPersistentStorage().Create(ctx, m)
}

func (m *Model) Read(ctx context.Context, name string) error {
	return GetGlobalPersistentStorage().First(ctx, m, "llm_id = ? AND name = ?", m.LLMId, name)
}

func (m *Model) Update(ctx context.Context) error {
	return GetGlobalPersistentStorage().DB(ctx).Model(m).Where("llm_id = ? AND name = ?", m.LLMId, m.Name).Updates(m).Error
}

func (m *Model) Delete(ctx context.Context) error {
	return GetGlobalPersistentStorage().DB(ctx).Model(m).Where("llm_id = ? AND name = ?", m.LLMId, m.Name).Delete(m).Error
}

func (m *Model) List(ctx context.Context) ([]Model, error) {
	var models []Model
	err := GetGlobalPersistentStorage().DB(ctx).Find(&models).Error
	return models, err
}

func (m *Model) ListByLLM(ctx context.Context, llmId uint) ([]Model, error) {
	var models []Model
	err := GetGlobalPersistentStorage().DB(ctx).Where("llm_id = ?", llmId).Find(&models).Error
	return models, err
}

func (ct *CurrentModel) Create(ctx context.Context) error {
	ct.Key = "default"
	return GetGlobalPersistentStorage().Create(ctx, ct)
}

func (ct *CurrentModel) Read(ctx context.Context) error {
	ct.Key = "default"
	return GetGlobalPersistentStorage().First(ctx, ct)
}

func (ct *CurrentModel) Update(ctx context.Context) error {
	ct.Key = "default"
	return GetGlobalPersistentStorage().Update(ctx, ct, ct)
}

func (ct *CurrentModel) Delete(ctx context.Context) error {
	ct.Key = "default"
	return GetGlobalPersistentStorage().Delete(ctx, ct)
}

func (ct *CurrentModel) Clear(ctx context.Context) error {
	ct.Key = "default"
	return GetGlobalPersistentStorage().DB(ctx).Where("1 = 1").Delete(&CurrentModel{}).Error
}

func DefaultLLMSAndModels(ctx context.Context) error {
	// delete all llms
	err := GetGlobalPersistentStorage().DB(ctx).Where("1 = 1").Delete(&LLM{}).Error
	if err != nil {
		return err
	}

	// delete all models
	err = GetGlobalPersistentStorage().DB(ctx).Where("1 = 1").Delete(&Model{}).Error
	if err != nil {
		return err
	}

	// delete current model
	err = GetGlobalPersistentStorage().DB(ctx).Where("1 = 1").Delete(&CurrentModel{}).Error
	if err != nil {
		return err
	}

	// initialize default llms
	llm := LLM{}
	err = llm.Initialize(ctx)
	if err != nil {
		return err
	}

	return nil
}
