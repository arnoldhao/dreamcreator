package models

import (
	"CanMe/backend/consts"
	"errors"
	"time"

	"github.com/dustin/go-humanize"
	"gorm.io/gorm"
)

// DownloadTask
type DownloadTask struct {
	gorm.Model    `json:"-"`
	TaskID        string        `gorm:"type:varchar(36);not null;unique" json:"taskId"`
	ContentID     string        `gorm:"type:varchar(36);not null" json:"contentId"`
	TotalSize     int64         `gorm:"type:bigint;not null" json:"totalSize"`
	TotalParts    int64         `gorm:"type:bigint;not null" json:"totalParts"`
	FinishedParts int64         `gorm:"type:bigint;not null" json:"finishedParts"`
	Stream        string        `gorm:"type:text;not null" json:"stream"`
	Captions      []string      `gorm:"type:text" json:"captions"`
	Danmaku       bool          `gorm:"type:boolean" json:"danmaku"`
	Source        string        `gorm:"type:varchar(36);not null" json:"source"`
	URL           string        `gorm:"type:varchar(255);not null" json:"url"`
	Title         string        `gorm:"type:varchar(255);not null" json:"title"`
	SavedPath     string        `gorm:"type:varchar(255);not null" json:"savedPath"`
	StreamParts   []*StreamPart `gorm:"foreignKey:TaskID;references:TaskID" json:"streams"`
	CommonParts   []*CommonPart `gorm:"foreignKey:TaskID;references:TaskID" json:"commonParts"`
	Status        string        `gorm:"type:varchar(36);not null" json:"status"`
	Progress      float64       `gorm:"type:float;not null" json:"progress"`
}

// Validate
func (t *DownloadTask) Validate() error {
	if t.TaskID == "" {
		return errors.New("task ID is required")
	}
	if t.ContentID == "" {
		return errors.New("content ID is required")
	}
	if t.URL == "" {
		return errors.New("URL is required")
	}
	if t.Title == "" {
		return errors.New("title is required")
	}
	if t.TotalParts <= 0 {
		return errors.New("total parts must be greater than 0")
	}
	return nil
}

// GORM
func (t *DownloadTask) BeforeCreate(tx *gorm.DB) error {
	if err := t.Validate(); err != nil {
		return err
	}
	return nil
}

// DownloadPart
type StreamPart struct {
	gorm.Model    `json:"-"`
	PartID        string  `gorm:"type:varchar(36);not null;unique" json:"partId"`
	TaskID        string  `gorm:"type:varchar(36);not null;index" json:"taskId"`
	ContentID     string  `gorm:"type:varchar(36);not null" json:"contentId"`
	Name          string  `gorm:"type:varchar(255);not null" json:"name"`
	Type          string  `gorm:"type:varchar(36);not null" json:"type"`
	FileName      string  `gorm:"type:varchar(255);not null" json:"fileName"`
	FinalFileName string  `gorm:"type:varchar(255);not null" json:"finalFileName"`
	URL           string  `gorm:"type:varchar(255);not null" json:"url"`
	Quality       string  `gorm:"type:varchar(36);not null" json:"quality"`
	Ext           string  `gorm:"type:varchar(36);not null" json:"ext"`
	NeedMerge     bool    `gorm:"type:boolean;not null" json:"needMerge"`
	Progress      float64 `gorm:"type:float;not null" json:"progress"`
	Duration      int64   `gorm:"type:bigint;not null" json:"duration"`
	AverageSpeed  string  `gorm:"type:varchar(36);not null" json:"averageSpeed"`
	CurrentSize   int64   `gorm:"type:bigint;not null" json:"currentSize"`
	TotalSize     int64   `gorm:"type:bigint;not null" json:"totalSize"`
	Status        string  `gorm:"type:varchar(36);not null" json:"status"`
	FinalStatus   bool    `gorm:"type:boolean;not null" json:"finalStatus"`
	Message       string  `gorm:"type:text;not null" json:"message"`

	// calculate speed
	StartDownload time.Time `gorm:"type:datetime" json:"startDownload"`
	EndDownload   time.Time `gorm:"type:datetime" json:"endDownload"`
}

// Validate
func (s *StreamPart) Validate() error {
	if s.PartID == "" {
		return errors.New("part ID is required")
	}
	if s.TaskID == "" {
		return errors.New("task ID is required")
	}
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.TotalSize <= 0 {
		return errors.New("total size must be greater than 0")
	}
	return nil
}

// GORM
func (s *StreamPart) BeforeCreate(tx *gorm.DB) error {
	if err := s.Validate(); err != nil {
		return err
	}
	return nil
}

type ProgressReciver struct {
	PartID string                `json:"partId"`
	TaskID string                `json:"taskId"`
	Status consts.DownloadStatus `json:"status"`
	Added  int64                 `json:"added"`
	Error  error                 `json:"err"`
}

// UpdateTask
type UpdateTask struct {
	PartID      string            `json:"partId" validate:"required"`
	TaskID      string            `json:"taskId" validate:"required"`
	Status      consts.PartStatus `json:"status" validate:"required"`
	CurrentSize int64             `json:"currentSize"`
	TotalSize   int64             `json:"totalSize"`
	SpeedFloat  float64           `json:"speedFloat"`
	CurrentTime time.Time         `json:"currentTime"`
	Message     string            `json:"message"`
}

func (u *UpdateTask) GetProgress() float64 {
	var progress float64
	if u.TotalSize > 0 && u.CurrentSize > 0 {
		progress = float64(u.CurrentSize) / float64(u.TotalSize) * 100.00
	}
	if progress > 100 {
		progress = 100
	}

	return progress
}

func (u *UpdateTask) GetSpeedString() string {
	// speed string
	if u.SpeedFloat < 0 {
		return "0 B/s"
	} else {
		return humanize.Bytes(uint64(u.SpeedFloat)) + "/s"
	}
}

// CommonPart
type CommonPart struct {
	gorm.Model  `json:"-"`
	PartID      string `gorm:"type:varchar(36);not null;unique" json:"partId"`
	TaskID      string `gorm:"type:varchar(36);not null;index" json:"taskId"`
	ContentID   string `gorm:"type:varchar(36);not null" json:"contentId"`
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Type        string `gorm:"type:varchar(36);not null" json:"type"`
	FileName    string `gorm:"type:varchar(255);not null" json:"fileName"`
	URL         string `gorm:"type:varchar(255);not null" json:"url"`
	Ext         string `gorm:"type:varchar(36);not null" json:"ext"`
	Status      string `gorm:"type:varchar(36);not null" json:"status"`
	FinalStatus bool   `gorm:"type:boolean;not null" json:"finalStatus"`
	Message     string `gorm:"type:text;not null" json:"message"`
}

type TaskError struct {
	TaskID string `json:"taskId" yaml:"taskId"`
	Err    error  `json:"error" yaml:"error"`
}

// TableName
func (DownloadTask) TableName() string {
	return "download_tasks"
}

func (StreamPart) TableName() string {
	return "stream_parts"
}
