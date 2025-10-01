package downtasks

import (
	"context"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/storage"
	"dreamcreator/backend/types"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TaskManager 管理视频处理任务
type TaskManager struct {
	ctx       context.Context
	tasks     map[string]*types.DtTaskStatus
	taskMutex sync.RWMutex
	storage   *storage.BoltStorage // 添加持久化存储
}

// NewTaskManager 创建新的任务管理器
func NewTaskManager(ctx context.Context, boltStorage *storage.BoltStorage) *TaskManager {
	tm := &TaskManager{
		ctx:     ctx,
		tasks:   make(map[string]*types.DtTaskStatus),
		storage: boltStorage,
	}

	// 如果存储初始化成功，从存储中加载任务
	if boltStorage != nil {
		tm.loadTasksFromStorage()

		// if formats is null, initialize it
		formats := tm.ListAllConversionFormats()
		if formats == nil || len(formats) == 0 {
			err := tm.storage.InitializeOrRestoreDefaultConversionFormats(true)
			if err != nil {
				logger.Error("Failed to initialize default conversion formats", zap.Error(err))
			} else {
				refreshed := tm.ListAllConversionFormats()
				total := 0
				for _, group := range refreshed {
					total += len(group)
				}
				logger.Info("Default conversion formats initialized", zap.Int("groups", len(refreshed)), zap.Int("total", total))
			}
		}
	}

	return tm
}

// 从存储中加载任务到内存
func (tm *TaskManager) loadTasksFromStorage() {
	if tm.storage == nil {
		return
	}

	tasks, err := tm.storage.ListTasks()
	if err != nil {
		logger.Error("Failed to load tasks from storage", zap.Error(err))
		return
	}

	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()

	for _, task := range tasks {
		tm.tasks[task.ID] = task
	}

	logger.Debug("Tasks loaded from storage", zap.Int("count", len(tasks)))
}

func (tm *TaskManager) Path() string {
	fileName := tm.storage.Path()
	return filepath.Dir(fileName)
}

// GetTask 获取任务状态
func (tm *TaskManager) GetTask(id string) *types.DtTaskStatus {
	// Fast path: read lock to check existing
	tm.taskMutex.RLock()
	task, ok := tm.tasks[id]
	tm.taskMutex.RUnlock()
	if ok {
		return task
	}

	// Slow path: fetch from storage (no lock held while doing I/O)
	if tm.storage != nil {
		if loaded, err := tm.storage.GetTask(id); err == nil && loaded != nil {
			// Upgrade to write lock to cache into memory
			tm.taskMutex.Lock()
			tm.tasks[id] = loaded
			tm.taskMutex.Unlock()
			return loaded
		}
	}

	return nil
}

// CreateTask 创建新任务
func (tm *TaskManager) CreateTask(id string) *types.DtTaskStatus {
	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()

	now := time.Now().Unix()
	task := &types.DtTaskStatus{
		ID:        id,
		Stage:     types.DtStageInitializing,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tm.tasks[id] = task

	// 如果存储可用，保存到存储中
	if tm.storage != nil {
		if err := tm.storage.SaveTask(task); err != nil {
			logger.Error("Failed to save task",
				zap.String("id", id),
				zap.Error(err))
		}
	}

	return task
}

// UpdateTask 更新任务状态
func (tm *TaskManager) UpdateTask(task *types.DtTaskStatus) {
	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()

	task.UpdatedAt = time.Now().Unix()
	tm.tasks[task.ID] = task

	// 如果存储可用，保存到存储中
	if tm.storage != nil {
		if err := tm.storage.SaveTask(task); err != nil {
			logger.Error("Failed to update task",
				zap.String("id", task.ID),
				zap.Error(err))
		}
	}
}

// UpdateTaskWith applies a mutation function to a task under write lock,
// persists it when storage is available, and returns the updated task.
func (tm *TaskManager) UpdateTaskWith(id string, fn func(*types.DtTaskStatus)) *types.DtTaskStatus {
	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()

	t, ok := tm.tasks[id]
	if !ok && tm.storage != nil {
		if loaded, err := tm.storage.GetTask(id); err == nil && loaded != nil {
			tm.tasks[id] = loaded
			t = loaded
		}
	}
	if t == nil {
		return nil
	}
	if fn != nil {
		fn(t)
	}
	t.UpdatedAt = time.Now().Unix()
	if tm.storage != nil {
		if err := tm.storage.SaveTask(t); err != nil {
			logger.Error("Failed to update task via UpdateTaskWith",
				zap.String("id", id),
				zap.Error(err))
		}
	}
	return t
}

// ListTasks 列出所有任务
func (tm *TaskManager) ListTasks() []*types.DtTaskStatus {
	tm.taskMutex.RLock()
	defer tm.taskMutex.RUnlock()

	tasks := make([]*types.DtTaskStatus, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}

	// 按照CreatedAt降序排序，最新的任务排在前面
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt > tasks[j].CreatedAt
	})

	return tasks
}

// DeleteTask 删除指定ID的任务
func (tm *TaskManager) DeleteTask(id string) error {
	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()

	// 检查任务是否存在
	task, exists := tm.tasks[id]
	if !exists {
		return fmt.Errorf("task not found: %s", id)
	}

	// 从内存中删除任务
	delete(tm.tasks, id)

	// 如果存储可用，从存储中删除任务
	if tm.storage != nil {
		if err := tm.storage.DeleteTask(id); err != nil {
			logger.Error("Failed to delete task",
				zap.String("id", id),
				zap.Error(err))
			return err
		}

		// delete task image cache
		if err := tm.storage.DeleteImage(task.Thumbnail); err != nil {
			logger.Error("Failed to delete task image",
				zap.String("id", id),
				zap.Error(err))
			return err
		}
	}

	return nil
}

// Close 关闭任务管理器，清理资源
func (tm *TaskManager) Close() error {
	if tm.storage != nil {
		return tm.storage.Close()
	}
	return nil
}

func (tm *TaskManager) ListAllConversionFormats() map[string][]*types.ConversionFormat {
	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()

	sortedFormats := make(map[string][]*types.ConversionFormat)
	if tm.storage != nil {
		formats, err := tm.storage.ListAllConversionFormats()
		if err != nil {
			logger.Error("Failed to list all conversion formats", zap.Error(err))
			return nil
		}

		if formats != nil {
			videoFormats := make([]*types.ConversionFormat, 0)
			audioFormats := make([]*types.ConversionFormat, 0)
			for _, format := range formats {
				if format.Type == "video" {
					videoFormats = append(videoFormats, format)
				} else if format.Type == "audio" {
					audioFormats = append(audioFormats, format)
				}
			}
			if len(videoFormats) > 0 {
				sortedFormats["video"] = videoFormats
			}
			if len(audioFormats) > 0 {
				sortedFormats["audio"] = audioFormats
			}
		}

	}

	return sortedFormats
}

func (tm *TaskManager) ListAvalibleConversionFormats() map[string][]*types.ConversionFormat {
	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()

	sortedFormats := make(map[string][]*types.ConversionFormat)
	if tm.storage != nil {
		formats, err := tm.storage.ListAllConversionFormats()
		if err != nil {
			logger.Error("Failed to list all conversion formats", zap.Error(err))
			return nil
		}

		if formats != nil {
			videoFormats := make([]*types.ConversionFormat, 0)
			audioFormats := make([]*types.ConversionFormat, 0)
			for _, format := range formats {
				if format.Type == "video" && format.Available {
					videoFormats = append(videoFormats, format)
				} else if format.Type == "audio" && format.Available {
					audioFormats = append(audioFormats, format)
				}
			}
			if len(videoFormats) > 0 {
				sortedFormats["video"] = videoFormats
			}
			if len(audioFormats) > 0 {
				sortedFormats["audio"] = audioFormats
			}
		}

	}

	return sortedFormats
}

func (tm *TaskManager) GetConversionFormatExtension(id int) (string, error) {
	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()
	if tm.storage != nil {
		format, err := tm.storage.GetConversionFormat(id)
		if err != nil {
			logger.Error("Failed to get conversion format", zap.Error(err))
			return "", err
		}
		if format != nil {
			return format.Extension, nil
		}
	}

	return "", fmt.Errorf("format not found")
}
