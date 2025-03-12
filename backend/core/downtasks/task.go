package downtasks

import (
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TaskManager 管理视频处理任务
type TaskManager struct {
	ctx       context.Context
	tasks     map[string]*types.DtTaskStatus
	taskMutex sync.RWMutex
	storage   *storage.BoltTaskStorage // 添加持久化存储
}

// NewTaskManager 创建新的任务管理器
func NewTaskManager(ctx context.Context) *TaskManager {
	// 初始化 BoltDB 存储
	boltStorage, err := storage.NewBoltTaskStorage()
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to initialize task storage: %v", err)
		// 即使存储初始化失败，我们仍然可以使用内存存储继续运行
	}

	tm := &TaskManager{
		ctx:     ctx,
		tasks:   make(map[string]*types.DtTaskStatus),
		storage: boltStorage,
	}

	// 如果存储初始化成功，从存储中加载任务
	if boltStorage != nil {
		tm.loadTasksFromStorage()
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
		runtime.LogErrorf(tm.ctx, "Failed to load tasks from storage: %v", err)
		return
	}

	tm.taskMutex.Lock()
	defer tm.taskMutex.Unlock()

	for _, task := range tasks {
		tm.tasks[task.ID] = task
	}

	runtime.LogDebugf(tm.ctx, "Loaded %d tasks from storage", len(tasks))
}

func (tm *TaskManager) Path() string {
	fileName := tm.storage.Path()
	return filepath.Dir(fileName)
}

// GetTask 获取任务状态
func (tm *TaskManager) GetTask(id string) *types.DtTaskStatus {
	tm.taskMutex.RLock()
	defer tm.taskMutex.RUnlock()

	// 首先尝试从内存中获取
	if task, ok := tm.tasks[id]; ok {
		return task
	}

	// 如果内存中没有，且存储可用，尝试从存储中获取
	if tm.storage != nil {
		task, err := tm.storage.GetTask(id)
		if err == nil {
			// 将任务添加到内存中
			tm.tasks[id] = task
			return task
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
			runtime.LogErrorf(tm.ctx, "Failed to save task %s to storage: %v", id, err)
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
			runtime.LogErrorf(tm.ctx, "Failed to update task %s in storage: %v", task.ID, err)
		}
	}
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
	if _, exists := tm.tasks[id]; !exists {
		return fmt.Errorf("task not found: %s", id)
	}

	// 从内存中删除任务
	delete(tm.tasks, id)

	// 如果存储可用，从存储中删除任务
	if tm.storage != nil {
		if err := tm.storage.DeleteTask(id); err != nil {
			runtime.LogErrorf(tm.ctx, "Failed to delete task %s from storage: %v", id, err)
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
