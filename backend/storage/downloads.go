package storage

import (
	"CanMe/backend/consts"
	"context"

	"gorm.io/gorm"
)

type Downloads struct {
	gorm.Model
	ID        string                `gorm:"primaryKey;type:varchar(36);not null;unique" json:"id"` // unique id
	Status    consts.DownloadStatus `gorm:"type:varchar(36);not null" json:"status"`               // download status
	Source    string                `gorm:"type:varchar(36);not null" json:"source"`               // website source name: youtube, bilibili, etc
	Site      string                `gorm:"type:varchar(36);not null" json:"site"`                 // website name: youtube.com, bilibili.com, etc
	URL       string                `gorm:"type:varchar(36);not null" json:"url"`                  // source video url
	Title     string                `gorm:"type:varchar(36);not null" json:"title"`                // source video title
	Quality   string                `gorm:"type:varchar(36);not null" json:"quality"`              // downloaded video quality
	Format    string                `gorm:"type:varchar(36);not null" json:"format"`               // downloaded video format
	Total     int64                 `gorm:"type:bigint;not null" json:"total"`                     // total tasks
	Finished  int64                 `gorm:"type:bigint;not null" json:"finished"`                  // finished tasks include error task
	Size      int64                 `gorm:"type:bigint;not null" json:"size"`                      // total size
	Current   int64                 `gorm:"type:bigint;not null" json:"current"`                   // current size
	Speed     string                `gorm:"type:varchar(36)" json:"speed"`                         // current speed
	Progress  float64               `gorm:"type:float;not null" json:"progress"`                   // total progress
	SavedPath string                `gorm:"type:varchar(36);not null" json:"savedPath"`            // downloaded directory path
	Error     string                `gorm:"type:varchar(36);not null" json:"error"`                // error message
}

// Create create a new download record
func (d *Downloads) Create(ctx context.Context) error {
	return GetGlobalPersistentStorage().Create(ctx, d)
}

// Read read a download record by id
func (d *Downloads) Read(ctx context.Context, id string) error {
	return GetGlobalPersistentStorage().First(ctx, d, "id = ?", id)
}

// Update update a download record
func (d *Downloads) Update(ctx context.Context) error {
	return GetGlobalPersistentStorage().Update(ctx, d, d)
}

// Delete delete a subtitle record
func (d *Downloads) Delete(ctx context.Context) error {
	return GetGlobalPersistentStorage().Delete(ctx, d)
}

// ListDownloads list last 50 download records, sorted by creation time in descending order
func ListDownloads(ctx context.Context) ([]Downloads, error) {
	var downloads []Downloads
	err := GetGlobalPersistentStorage().DB(ctx).
		Order("created_at DESC").
		Limit(consts.LIST_DOWNLOADS_MAX_SIZE).
		Find(&downloads).Error
	return downloads, err
}
