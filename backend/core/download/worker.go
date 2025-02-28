package download

import (
	"CanMe/backend/consts"
	"CanMe/backend/models"
	"CanMe/backend/pkg/specials/cmdrun"
	"CanMe/backend/types"
	"CanMe/backend/utils/poolUtil"
	"context"
	"sync"
	"sync/atomic"
	"time"

	"CanMe/backend/core/events"
	"CanMe/backend/storage/repository"

	"github.com/dustin/go-humanize"
	"github.com/mohae/deepcopy"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Worker struct {
	// ctx
	ctx    context.Context
	cancel context.CancelFunc
	// receiver
	update chan *models.ProgressReciver
	// progress calculator
	current              atomic.Int64
	totalSize            int64
	lastTime             time.Time
	lastReportedProgress float64
	speed                float64
	// update config
	reportSmoothFactor float64
	minSpeedThreshold  float64
	speedSmoothFactor  float64
	// reporter
	errChan      chan *models.TaskError                // report to queue
	workerStatus chan map[string]consts.DownloadStatus // report to queue
	eventBus     events.EventBus                       // report to frontend
	// downloader
	videos map[string]*VideoDownloader // downloader
	// task cache
	repo repository.DownloadRepository // local storage
	task *models.DownloadTask          // models
	// control
	updateCounts int
	lastSaved    time.Time
	// multiplex rw
	mu sync.RWMutex
	// task cache config
	autoSaveInterval time.Duration
	saveInterval     time.Duration
}

// NewWorker
func NewWorker(ctx context.Context,
	task *models.DownloadTask,
	workerStatus chan map[string]consts.DownloadStatus,
	errChan chan *models.TaskError,
	eventBus events.EventBus,
	repo repository.DownloadRepository) *Worker {
	ctx, cancel := context.WithCancel(ctx)
	var totalSize int64
	var streams []*models.StreamPart
	for _, stream := range task.StreamParts {
		stream.StartDownload = time.Now() // start time
		streams = append(streams, stream)
		totalSize += stream.TotalSize
	}
	task.StreamParts = streams

	worker := &Worker{
		ctx:                  ctx,
		cancel:               cancel,
		current:              atomic.Int64{},
		totalSize:            totalSize,
		lastTime:             time.Time{},
		lastReportedProgress: 0,
		speed:                0,
		reportSmoothFactor:   1.00,
		minSpeedThreshold:    1024,
		speedSmoothFactor:    0.3,
		workerStatus:         workerStatus,
		errChan:              errChan,
		eventBus:             eventBus,
		update:               make(chan *models.ProgressReciver, 100),
		videos:               make(map[string]*VideoDownloader),
		repo:                 repo,
		task:                 task,
		updateCounts:         0,
		lastSaved:            time.Now(),
		autoSaveInterval:     30 * time.Second,
		saveInterval:         1 * time.Second,
	}

	// save
	worker.SaveTask(true)

	// emit
	worker.eventBus.Publish(consts.TopicDownloadSingle, types.DownloadResponse{
		ID:       worker.task.TaskID,
		Status:   consts.DownloadingCacheSaved,
		DataType: types.ExtractorDataTypeVideo,
		Total:    worker.task.TotalParts,
	})

	// period save
	go worker.startPeriodicSave()

	return worker
}

// Cancel
func (w *Worker) Cancel(status consts.DownloadStatus) error {
	for _, v := range w.videos {
		err := v.Cancel()
		if err != nil {
			return err
		}
	}

	// save task
	w.report(status)

	w.cancel()
	return nil
}

func (w *Worker) Download() error {
	// start handle update
	go func() {
		for {
			select {
			case <-w.ctx.Done():
				return
			case update := <-w.update:
				err := w.handleUpdate(update)
				if err != nil {
					// todo
				}
			}
		}
	}()

	if len(w.task.StreamParts) == 0 {
		return nil
	}

	if len(w.task.StreamParts) == 1 {
		w.videos[w.task.StreamParts[0].PartID] = NewVideoDownloader(w.ctx, w.task.URL, w.task.StreamParts[0], w.update)
		go func() {
			err := w.videos[w.task.StreamParts[0].PartID].Download()
			w.task.FinishedParts++
			if err != nil {
				if w.task.FinishedParts == w.task.TotalParts {
					w.task.Status = DownloadStatusFailed
					// report to queue service
					w.errChan <- &models.TaskError{
						TaskID: w.task.TaskID,
						Err:    err,
					}
				}
			}
		}()
	}

	if len(w.task.StreamParts) > 1 {
		wg := poolUtil.NewWaitGroupPool(len(w.task.StreamParts)) // todo caculate max
		lock := sync.Mutex{}
		errs := make([]error, 0)
		fileNames := make([]string, 0)
		var finalFileName string
		for _, part := range w.task.StreamParts {
			if part.NeedMerge {
				fileNames = append(fileNames, part.FileName)
				finalFileName = part.FinalFileName
			}

			// set video
			w.videos[part.PartID] = NewVideoDownloader(w.ctx, w.task.URL, part, w.update)

			// goruntine download
			wg.Add() // Increment the WaitGroup counter
			go func(part *models.StreamPart) {
				defer wg.Done() // Decrement the WaitGroup counter
				if videoPart := w.videos[part.PartID]; videoPart != nil {
					err := w.videos[part.PartID].Download()
					if err != nil {
						lock.Lock()
						errs = append(errs, err)
						lock.Unlock()
					}
				}
			}(part)
		}

		wg.Wait()

		w.task.FinishedParts = int64(len(w.task.StreamParts))
		if len(errs) > 0 {
			for _, err := range errs {
				if err != nil {
					if w.task.FinishedParts == w.task.TotalParts {
						w.task.Status = DownloadStatusFailed
						// report to queue service
						w.errChan <- &models.TaskError{
							TaskID: w.task.TaskID,
							Err:    err,
						}
					}
				}
			}
		} else {
			w.updateTaskStatus(consts.DownloadStatusMuxing)
			w.report(consts.DownloadStatusMuxing)

			// merge
			err := w.mergeFiles(fileNames, finalFileName)
			if err != nil {
				// eventbus merge failed
				w.updateTaskStatus(consts.DownloadStatusMuxFailed)
				go func() {
					w.workerStatus <- map[string]consts.DownloadStatus{
						w.task.TaskID: consts.DownloadStatusMuxFailed,
					}
				}()

				// return
				return err
			} else {
				w.updateTaskStatus(consts.DownloadStatusMuxSuccess)
				w.report(consts.DownloadStatusMuxSuccess)
			}
		}
	}

	// save task
	w.updateTaskStatus(consts.DownloadStatusAllSuccess)
	w.SaveTask(true)

	// report to queue to cancel worker
	go func() {
		w.workerStatus <- map[string]consts.DownloadStatus{
			w.task.TaskID: consts.DownloadStatusAllSuccess,
		}
	}()

	return nil
}

// handleUpdate
func (w *Worker) handleUpdate(update *models.ProgressReciver) error {
	now := time.Now()
	if update.Added > 0 { // downloading progress calculate
		if !w.lastTime.IsZero() {
			timeDiff := now.Sub(w.lastTime).Milliseconds()
			if timeDiff > 0 {
				instantSpeed := float64(update.Added) / float64(timeDiff) * 1000 // millisecond
				// use smooth
				w.speed = w.speed*(1-w.speedSmoothFactor) +
					instantSpeed*w.speedSmoothFactor

			}
		}

		// set current
		w.current.Add(update.Added)
		w.lastTime = now

		// set task
		w.updateTask(update)

		// smooth report
		currentProgress := float64(w.current.Load()) / float64(w.totalSize) * 100
		if currentProgress-w.lastReportedProgress > w.reportSmoothFactor {
			// set
			w.lastReportedProgress = currentProgress

			// publish event
			w.report(consts.DownloadStatusDownloading)
		}
	} else { // download status
		// set task
		w.updateTask(update)

		// publish event
		w.report(update.Status)
	}

	// increase uodate
	w.updateCounts++
	if w.updateCounts >= CountThreshold {
		w.updateCounts = 0
		w.SaveTask(false)
	}

	return nil
}

// startPeriodicSave
func (w *Worker) startPeriodicSave() {
	ticker := time.NewTicker(w.autoSaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.SaveTask(false)
		}
	}
}

func (w *Worker) SaveTask(force bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	safeTask := deepcopy.Copy(w.task).(*models.DownloadTask)
	if time.Since(w.lastSaved) >= w.saveInterval || force {
		if err := w.repo.Update(w.ctx, safeTask); err != nil {
			runtime.LogErrorf(w.ctx, "Failed to save task %s: %v", w.task.TaskID, err)
		}

		w.lastSaved = time.Now()
	}

}

func (w *Worker) mergeFiles(fileNames []string, finalFileName string) (err error) {
	cmd := []string{"-y"}

	for _, fileName := range fileNames {
		cmd = append(cmd, "-i", fileName)
	}

	cmd = append(cmd, "-c:v", "copy", "-c:a", "copy", finalFileName)
	return runMuxParts(cmdrun.RunCommand(findFFmpeg(), cmd...), fileNames)
}

func (w *Worker) updateTaskStatus(status consts.DownloadStatus) {
	// lock
	w.mu.Lock()
	defer w.mu.Unlock()

	// update progress
	if status == consts.DownloadStatusMuxing {
		w.lastReportedProgress = 100.00
		w.speed = 0
	}

	w.task.Status = string(status)
}

func (w *Worker) updateTask(update *models.ProgressReciver) {
	// lock
	w.mu.Lock()
	defer w.mu.Unlock()

	// set task
	newTask := w.task
	var progress float64
	if update.Added > 0 {
		var streams []*models.StreamPart
		for _, stream := range w.task.StreamParts {
			if stream.PartID == update.PartID {
				stream.CurrentSize += update.Added
				stream.Progress = float64(stream.CurrentSize) / float64(stream.TotalSize) * 100
				stream.Status = string(update.Status)
			}

			progress += stream.Progress
			streams = append(streams, stream)
		}
		newTask.StreamParts = streams
		newTask.Status = string(update.Status)
	} else {
		var streams []*models.StreamPart
		for _, stream := range w.task.StreamParts {
			if stream.PartID == update.PartID {
				stream.Status = string(update.Status)
				stream.EndDownload = time.Now()
				stream.Duration = int64(stream.EndDownload.Sub(stream.StartDownload).Microseconds())
				stream.AverageSpeed = humanize.Bytes(uint64(stream.TotalSize/stream.Duration)*1000*1000) + "/s"
				stream.FinalStatus = true
				if update.Error != nil {
					stream.Message = update.Error.Error()
				}
			}

			progress += stream.Progress
			streams = append(streams, stream)
		}

		newTask.StreamParts = streams
		newTask.Status = string(update.Status)
	}

	if len(w.task.StreamParts) == 0 {
		newTask.Progress = 0
	} else {
		newTask.Progress = progress / float64(len(w.task.StreamParts))
	}
	w.task = newTask
}

func (w *Worker) report(status consts.DownloadStatus) {
	// save
	w.SaveTask(false)

	// publish event
	w.eventBus.Publish(consts.TopicDownloadProgress, types.DownloadResponse{
		ID:       w.task.TaskID,
		Status:   status,
		DataType: types.ExtractorDataTypeVideo,
		Total:    w.task.TotalParts,
		Finished: w.task.FinishedParts,
		Speed:    w.getSpeedString(),
		Progress: w.lastReportedProgress,
	})
}

func (w *Worker) getSpeedString() string {
	// speed string
	if w.speed < 0 {
		return "0 B/s"
	} else {
		return humanize.Bytes(uint64(w.speed)) + "/s"
	}
}
