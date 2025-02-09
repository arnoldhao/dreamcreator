package repository

import (
	"CanMe/backend/models"
	"context"
)

type DownloadRepository interface {
	Create(ctx context.Context, task *models.DownloadTask) error
	FindByID(ctx context.Context, taskID string) (*models.DownloadTask, error)
	Update(ctx context.Context, task *models.DownloadTask) error
	Delete(ctx context.Context, taskID string) error
	ListTasks(ctx context.Context, offset, limit int) ([]*models.DownloadTask, error)

	// Part 相关操作
	CreateStreamPart(ctx context.Context, part *models.StreamPart) error
	CreateStreamParts(ctx context.Context, parts []*models.StreamPart) error
	UpdateStreamPart(ctx context.Context, part *models.StreamPart) error
	FindStreamPartByID(ctx context.Context, partID string) (*models.StreamPart, error)
	DeleteStreamPart(ctx context.Context, partID string) error
	ListStreamParts(ctx context.Context, offset, limit int) ([]*models.StreamPart, error)
}
