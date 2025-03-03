package repository

import (
	"CanMe/backend/models"
	"context"
	"fmt"

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
	tx := r.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 先检查记录是否存在
	var task models.DownloadTask
	if err := tx.First(&task, "task_id = ?", taskID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("Task not found: %s", taskID)
		}
		return fmt.Errorf("Failed to get Task: %v", err)
	}

	// 删除关联的StreamParts
	if err := tx.Where("task_id = ?", taskID).Delete(&models.StreamPart{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("Delete StreamParts failed: %v", err)
	}

	// 删除关联的CommonParts
	if err := tx.Where("task_id = ?", taskID).Delete(&models.CommonPart{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("Delete CommonParts failed: %v", err)
	}

	// 删除主任务
	if err := tx.Delete(&task).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("Fail to delete Task: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("Commit tx failed: %v", err)
	}

	return nil
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
