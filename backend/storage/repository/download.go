package repository

import (
	"CanMe/backend/models"
	"context"

	"gorm.io/gorm"
)

// /storage/sqlite/download.go
type downloadRepository struct {
	db *gorm.DB
}

func NewDownloadRepository(db *gorm.DB) DownloadRepository {
	return &downloadRepository{
		db: db,
	}
}

func (r *downloadRepository) Create(ctx context.Context, task *models.DownloadTask) error {
	result := r.db.WithContext(ctx).Create(task)
	return result.Error
}

func (r *downloadRepository) FindByID(ctx context.Context, taskID string) (*models.DownloadTask, error) {
	var task models.DownloadTask
	err := r.db.WithContext(ctx).First(&task, "id = ?", taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *downloadRepository) Update(ctx context.Context, task *models.DownloadTask) error {
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Save(task).Error; err != nil {
		tx.Rollback()
		return err
	}

	for _, stream := range task.StreamParts {
		if err := tx.Model(&models.StreamPart{}).Where("part_id = ?", stream.PartID).Updates(stream).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	for _, common := range task.CommonParts {
		if err := tx.Model(&models.CommonPart{}).Where("part_id = ?", common.PartID).Updates(common).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func (r *downloadRepository) Delete(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).Select("StreamParts", "CommonParts").Delete(&models.DownloadTask{}, "task_id = ?", taskID).Error
}

func (r *downloadRepository) ListTasks(ctx context.Context, offset, limit int) ([]*models.DownloadTask, error) {
	var tasks []*models.DownloadTask
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Preload("StreamParts").Preload("CommonParts").Order("created_at DESC").Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *downloadRepository) CreateStreamPart(ctx context.Context, part *models.StreamPart) error {
	return r.db.WithContext(ctx).Create(part).Error
}

func (r *downloadRepository) CreateStreamParts(ctx context.Context, parts []*models.StreamPart) error {
	return r.db.WithContext(ctx).Create(parts).Error
}

func (r *downloadRepository) UpdateStreamPart(ctx context.Context, part *models.StreamPart) error {
	return r.db.WithContext(ctx).Save(part).Error
}

func (r *downloadRepository) FindStreamPartByID(ctx context.Context, partID string) (*models.StreamPart, error) {
	var part models.StreamPart
	err := r.db.WithContext(ctx).First(&part, "id = ?", partID).Error
	if err != nil {
		return nil, err
	}
	return &part, nil
}

func (r *downloadRepository) DeleteStreamPart(ctx context.Context, partID string) error {
	return r.db.WithContext(ctx).Delete(&models.StreamPart{}, "id = ?", partID).Error
}

func (r *downloadRepository) ListStreamParts(ctx context.Context, offset, limit int) ([]*models.StreamPart, error) {
	var parts []*models.StreamPart
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&parts).Error
	if err != nil {
		return nil, err
	}
	return parts, nil
}
