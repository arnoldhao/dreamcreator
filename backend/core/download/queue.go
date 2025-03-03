package download

import (
	"CanMe/backend/core/events"
	"CanMe/backend/models"
	"CanMe/backend/storage/repository"
	"context"
	"errors"
	"sync"

	"github.com/mohae/deepcopy"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	MaxWorkers = 5
)

type Queue struct {
	mu           sync.RWMutex
	ctx          context.Context
	workers      map[string]*Worker
	workerStatus chan map[string]models.TaskStatus
	errChan      chan *models.TaskError
	eventBus     events.EventBus
	repo         repository.DownloadRepository
	maxWorkers   int
}

// NewQueue
func NewQueue(eventBus events.EventBus, repo repository.DownloadRepository) *Queue {
	return &Queue{
		workers:      make(map[string]*Worker),
		maxWorkers:   MaxWorkers,
		workerStatus: make(chan map[string]models.TaskStatus, MaxWorkers),
		errChan:      make(chan *models.TaskError, MaxWorkers),
		eventBus:     eventBus,
		repo:         repo,
	}
}

func (q *Queue) Start(ctx context.Context) {
	// assign context
	q.ctx = ctx

	// start workers
	go func() {
		for {
			select {
			case <-q.ctx.Done():
				return
			case workerStatus := <-q.workerStatus:
				err := q.handleWorkerUpdate(workerStatus)
				if err != nil {
					runtime.LogErrorf(q.ctx, "queue handle worker update error: %v", err)
				}
			case errChan := <-q.errChan:
				err := q.handleError(errChan)
				if err != nil {
					runtime.LogErrorf(q.ctx, "queue handle error: %v", err)
				}
			}
		}
	}()
}

func (q *Queue) Add(task *models.DownloadTask) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.workers) >= q.maxWorkers {
		return ErrQueueFull
	}

	if _, exists := q.workers[task.TaskID]; exists {
		return ErrTaskExists
	}

	if task.TaskStatus != models.TaskStatusPending {
		return ErrInvalidStatus
	}

	safeTask := deepcopy.Copy(task).(*models.DownloadTask)
	q.workers[task.TaskID] = NewWorker(q.ctx, safeTask, q.workerStatus, q.errChan, q.eventBus, q.repo)

	// start download
	q.workers[task.TaskID].Download()
	return nil
}

func (q *Queue) Get(taskID string) (*models.DownloadTask, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if worker, exists := q.workers[taskID]; exists {
		return worker.task, true
	}

	return nil, false
}

func (q *Queue) IsQueue(contentID, streamID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, work := range q.workers {
		if work.task.ContentID == contentID && work.task.Stream == streamID {
			return true
		}
	}

	return false
}

func (q *Queue) GetAll() ([]*models.DownloadTask, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.workers) == 0 {
		return nil, errors.New("no tasks in queue")
	}

	tasks := make([]*models.DownloadTask, 0)

	for _, worker := range q.workers {
		tasks = append(tasks, worker.task)
	}

	return tasks, nil
}

func (q *Queue) Remove(taskID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.workers, taskID)
}

func (q *Queue) handleWorkerUpdate(status map[string]models.TaskStatus) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for taskID, status := range status {
		if worker, exists := q.workers[taskID]; exists {
			err := worker.Cancel(status)
			if err != nil {
				return err
			} else {
				delete(q.workers, taskID)
			}
		}
	}

	return nil
}

// CancelTask
func (q *Queue) CancelTask(taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if worker, exists := q.workers[taskID]; exists {
		err := worker.Cancel(models.TaskStatusCancelled)
		if err != nil {
			return err
		} else {
			delete(q.workers, taskID)
		}
	}

	return nil
}

func (q *Queue) handleError(err *models.TaskError) error {
	return q.CancelTask(err.TaskID)
}
