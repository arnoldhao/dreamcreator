package models

import (
	"CanMe/backend/consts"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/asticode/go-astisub"
	"github.com/dustin/go-humanize"
	"gorm.io/gorm"
)

// StringArray 是一个自定义类型，用于处理字符串数组和数据库 TEXT 类型之间的转换
type StringArray []string

// Scan 实现 sql.Scanner 接口，用于从数据库读取数据
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = StringArray{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	// 如果是空字符串，返回空数组
	if len(bytes) == 0 {
		*a = StringArray{}
		return nil
	}

	// 尝试解析为 JSON 数组
	return json.Unmarshal(bytes, a)
}

// Value 实现 driver.Valuer 接口，用于将数据写入数据库
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return nil, nil
	}
	return json.Marshal(a)
}

// DownloadTask
type DownloadTask struct {
	gorm.Model    `json:"-"`
	TaskID        string        `gorm:"type:varchar(36);not null;unique" json:"taskId"`
	ContentID     string        `gorm:"type:varchar(36);not null" json:"contentId"`
	TotalSize     int64         `gorm:"type:bigint;not null" json:"totalSize"`
	TotalParts    int64         `gorm:"type:bigint;not null" json:"totalParts"`
	FinishedParts int64         `gorm:"type:bigint;not null" json:"finishedParts"`
	Stream        string        `gorm:"type:text;not null" json:"stream"`
	Captions      StringArray   `gorm:"type:text" json:"captions"`
	Danmaku       bool          `gorm:"type:boolean" json:"danmaku"`
	Source        string        `gorm:"type:varchar(36);not null" json:"source"`
	URL           string        `gorm:"type:varchar(255);not null" json:"url"`
	Title         string        `gorm:"type:varchar(255);not null" json:"title"`
	SavedPath     string        `gorm:"type:varchar(255);not null" json:"savedPath"`
	StreamParts   []*StreamPart `gorm:"foreignKey:TaskID;references:TaskID" json:"streams"`
	CommonParts   []*CommonPart `gorm:"foreignKey:TaskID;references:TaskID" json:"commonParts"`
	Status        string        `gorm:"type:varchar(36);not null" json:"status"` // 已废弃，将在未来版本移除
	Progress      float64       `gorm:"type:float;not null" json:"progress"`
	// 0.0.9 new add, must set default or it will cause problem
	TaskStatus       TaskStatus `gorm:"type:int;not null;default:0" json:"taskStatus"`          // 新的状态字段
	TotalCurrentSize int64      `gorm:"type:bigint;not null;default:0" json:"totalCurrentSize"` // 当前下载的大小
	StartTime        time.Time  `gorm:"type:datetime" json:"startTime"`                         // 任务开始时间
	EstimatedEndTime time.Time  `gorm:"type:datetime" json:"estimatedEndTime"`                  // 预计完成时间
	EndTime          time.Time  `gorm:"type:datetime" json:"endTime"`                           // 任务结束时间
	DurationSeconds  int64      `gorm:"type:bigint;default:0" json:"durationSeconds"`           // 总耗时（秒）
	CurrentSpeed     float64    `gorm:"type:float;default:0" json:"currentSpeed"`               // 当前速度(bytes/s)
	SpeedString      string     `gorm:"type:varchar(36);default:''" json:"speedString"`         // 格式化的速度字符串
	AverageSpeed     string     `gorm:"type:varchar(36);default:''" json:"averageSpeed"`        // 平均速度字符串
	TimeRemaining    string     `gorm:"type:varchar(36);default:''" json:"timeRemaining"`       // 剩余时间字符串
	IsProcessing     bool       `gorm:"-" json:"isProcessing"`
	// 0.0.10 new add
	CaptionsTransform func([]byte) (*astisub.Subtitles, error) `gorm:"-" json:"-"` // 字幕转换函数
}

// TaskStatus 表示下载任务的状态
type TaskStatus int

const (
	TaskStatusCreated        TaskStatus = iota // 0: 任务已创建
	TaskStatusPending                          // 1: 等待下载
	TaskStatusDownloading                      // 2: 正在下载
	TaskStatusPaused                           // 3: 已暂停
	TaskStatusMuxing                           // 4: 正在合并分片
	TaskStatusMuxingSuccess                    // 5: 合并成功
	TaskStatusMuxingFailed                     // 6: 合并失败
	TaskStatusCompleted                        // 7: 下载完成
	TaskStatusFailed                           // 8: 下载失败
	TaskStatusPartialSuccess                   // 9: 部分分下下载成功
	TaskStatusPartialFailed                    // 10: 部分分片下载失败
	TaskStatusCancelled                        // 11: 已取消
	TaskStatusUnknown                          // 12: 未知状态
)

// String 实现 Stringer 接口
func (s TaskStatus) String() string {
	switch s {
	case TaskStatusCreated:
		return "created"
	case TaskStatusPending:
		return "pending"
	case TaskStatusDownloading:
		return "downloading"
	case TaskStatusPaused:
		return "paused"
	case TaskStatusMuxing:
		return "muxing"
	case TaskStatusMuxingSuccess:
		return "muxing_success"
	case TaskStatusMuxingFailed:
		return "muxing_failed"
	case TaskStatusCompleted:
		return "completed"
	case TaskStatusFailed:
		return "failed"
	case TaskStatusPartialSuccess:
		return "partial_success"
	case TaskStatusPartialFailed:
		return "partial_failed"
	case TaskStatusCancelled:
		return "cancelled"
	case TaskStatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// IsProcessing 计算任务是否在进行中
func (s TaskStatus) IsProcessing() bool {
	switch s {
	case TaskStatusPending,
		TaskStatusDownloading,
		TaskStatusPaused,
		TaskStatusMuxing,
		TaskStatusMuxingSuccess,
		TaskStatusMuxingFailed,
		TaskStatusPartialSuccess:
		return true
	default:
		return false
	}
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

// BeforeSave GORM 钩子，同步状态字段
func (t *DownloadTask) BeforeSave(tx *gorm.DB) error {
	// 保持向后兼容，同时更新旧的 status 字段
	t.Status = t.TaskStatus.String()
	return nil
}

// AfterFind GORM 钩子，设置 IsProcessing
func (t *DownloadTask) AfterFind(tx *gorm.DB) error {
	t.calculateIsProcessing()
	return nil
}

// calculateIsProcessing 计算任务是否在进行中
func (t *DownloadTask) calculateIsProcessing() {
	if t.TaskStatus.IsProcessing() {
		t.IsProcessing = true
	} else {
		t.IsProcessing = false
	}
}

// IsFinished 判断任务是否已结束
func (t *DownloadTask) IsFinished() bool {
	switch t.TaskStatus {
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusPartialFailed, TaskStatusMuxingFailed, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

// IsSuccess 判断任务是否成功完成
func (t *DownloadTask) IsSuccess() bool {
	return t.TaskStatus == TaskStatusCompleted
}

// BeforeCreate GORM 钩子，验证任务
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

// BeforeCreate GORM 钩子，验证分片
func (s *StreamPart) BeforeCreate(tx *gorm.DB) error {
	if err := s.Validate(); err != nil {
		return err
	}
	return nil
}

type ProgressReciver struct {
	PartID string     `json:"partId"`
	TaskID string     `json:"taskId"`
	Status TaskStatus `json:"status"`
	Added  int64      `json:"added"`
	Error  error      `json:"err"`
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
