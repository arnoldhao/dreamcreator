package download

import (
	"CanMe/backend/consts"
	"CanMe/backend/models"
	"CanMe/backend/pkg/specials/cmdrun"
	"CanMe/backend/types"
	"CanMe/backend/utils/poolUtil"
	"context"
	"fmt"
	"math"
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
	errChan      chan *models.TaskError            // report to queue
	workerStatus chan map[string]models.TaskStatus // report to queue
	eventBus     events.EventBus                   // report to frontend
	// downloader
	videos sync.Map // downloader
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
	// speedSamples 用于计算移动平均速度
	speedSamples   []float64     // 保存最近的速度样本
	lastSampleTime time.Time     // 上次采样时间
	sampleInterval time.Duration // 采样间隔
	maxSampleCount int           // 最大样本数
}

// NewWorker
func NewWorker(ctx context.Context,
	task *models.DownloadTask,
	workerStatus chan map[string]models.TaskStatus,
	errChan chan *models.TaskError,
	eventBus events.EventBus,
	repo repository.DownloadRepository) *Worker {
	ctx, cancel := context.WithCancel(ctx)
	now := time.Now()
	var totalSize int64
	var streams []*models.StreamPart
	for _, stream := range task.StreamParts {
		stream.StartDownload = now // start time
		streams = append(streams, stream)
		totalSize += stream.TotalSize
	}
	task.StreamParts = streams
	task.StartTime = now

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
		videos:               sync.Map{},
		repo:                 repo,
		task:                 task,
		updateCounts:         0,
		lastSaved:            time.Now(),
		autoSaveInterval:     30 * time.Second,
		saveInterval:         1 * time.Second,
		speedSamples:         make([]float64, 0, 30), // 保存30个样本
		sampleInterval:       time.Second * 1,        // 每1秒采样一次
		maxSampleCount:       30,                     // 最多保存30个样本
		lastSampleTime:       time.Now(),
	}

	// save
	worker.SaveTask(true)

	// emit
	worker.eventBus.Publish(consts.TopicDownloadSingle, types.DownloadResponse{
		ID:         worker.task.TaskID,
		TaskStatus: models.TaskStatusCreated,
		DataType:   types.ExtractorDataTypeVideo,
		Total:      worker.task.TotalParts,
	})

	// period save
	go worker.startPeriodicSave()

	return worker
}

// Cancel
func (w *Worker) Cancel(status models.TaskStatus) error {
	var cancelErr error
	w.videos.Range(func(key, value interface{}) bool {
		v := value.(*VideoDownloader)
		err := v.Cancel()
		if err != nil {
			cancelErr = err
			return false // 停止遍历
		}
		return true // 继续遍历
	})

	if cancelErr != nil {
		return cancelErr
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
		w.videos.Store(w.task.StreamParts[0].PartID, NewVideoDownloader(w.ctx, w.task.URL, w.task.StreamParts[0], w.update))
		go func() {
			videoPart, ok := w.videos.Load(w.task.StreamParts[0].PartID)
			if ok && videoPart != nil {
				err := videoPart.(*VideoDownloader).Download()
				if err != nil {
					w.task.FinishedParts++
					if w.task.FinishedParts == w.task.TotalParts {
						w.task.TaskStatus = models.TaskStatusFailed
						// report to queue service
						w.errChan <- &models.TaskError{
							TaskID: w.task.TaskID,
							Err:    err,
						}
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
			w.videos.Store(part.PartID, NewVideoDownloader(w.ctx, w.task.URL, part, w.update))
			// goruntine download
			wg.Add() // Increment the WaitGroup counter
			go func(part *models.StreamPart) {
				defer wg.Done() // Decrement the WaitGroup counter
				if videoPart, ok := w.videos.Load(part.PartID); ok && videoPart != nil {
					err := videoPart.(*VideoDownloader).Download()
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
						w.task.TaskStatus = models.TaskStatusFailed
						// report to queue service
						w.errChan <- &models.TaskError{
							TaskID: w.task.TaskID,
							Err:    err,
						}
					}
				}
			}
		} else {
			w.updateTaskStatus(models.TaskStatusMuxing)
			w.report(models.TaskStatusMuxing)

			// merge
			err := w.mergeFiles(fileNames, finalFileName)
			if err != nil {
				// eventbus merge failed
				w.updateTaskStatus(models.TaskStatusMuxingFailed)
				go func() {
					w.workerStatus <- map[string]models.TaskStatus{
						w.task.TaskID: models.TaskStatusMuxingFailed,
					}
				}()

				// return
				return err
			} else {
				w.updateTaskStatus(models.TaskStatusMuxingSuccess)
				w.report(models.TaskStatusMuxingSuccess)
			}
		}
	}

	// save task
	w.updateTaskStatus(models.TaskStatusCompleted)
	w.SaveTask(true)

	// report to queue to cancel worker
	go func() {
		w.workerStatus <- map[string]models.TaskStatus{
			w.task.TaskID: models.TaskStatusCompleted,
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
			w.report(models.TaskStatusDownloading)
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

func (w *Worker) updateTaskStatus(status models.TaskStatus) {
	// lock
	w.mu.Lock()
	defer w.mu.Unlock()

	// update progress
	if status == models.TaskStatusMuxing {
		w.lastReportedProgress = 100.00
		w.speed = 0
	}

	w.task.TaskStatus = status
}

func (w *Worker) updateTask(update *models.ProgressReciver) {
	// lock
	w.mu.Lock()
	defer w.mu.Unlock()

	// set task
	newTask := w.task
	now := time.Now()

	var progress float64
	if update.Added > 0 {
		var streams []*models.StreamPart
		for _, stream := range w.task.StreamParts {
			if stream.PartID == update.PartID {
				stream.CurrentSize += update.Added
				stream.Progress = float64(stream.CurrentSize) / float64(stream.TotalSize) * 100
				stream.Status = update.Status.String()
			}

			progress += stream.Progress
			streams = append(streams, stream)
		}
		newTask.StreamParts = streams
		newTask.TaskStatus = update.Status
	} else {
		var streams []*models.StreamPart
		for _, stream := range w.task.StreamParts {
			if stream.PartID == update.PartID {
				stream.Status = update.Status.String()
				stream.EndDownload = now
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
		newTask.TaskStatus = update.Status
	}

	if len(w.task.StreamParts) == 0 {
		newTask.Progress = 0
	} else {
		newTask.Progress = progress / float64(len(w.task.StreamParts))
	}

	// update task params
	newTask.TotalCurrentSize = w.current.Load()
	newTask.CurrentSpeed = w.speed
	newTask.SpeedString = w.getSpeedString()

	// 计算剩余时间
	w.calculateTimeRemaining(newTask)

	// calculate task final info
	allDone := true
	for _, stream := range newTask.StreamParts {
		if !stream.FinalStatus {
			allDone = false
			break
		}
	}

	if allDone {
		newTask.EndTime = now
		newTask.DurationSeconds = int64(newTask.EndTime.Sub(newTask.StartTime).Microseconds())
		newTask.AverageSpeed = humanize.Bytes(uint64(newTask.TotalSize/newTask.DurationSeconds)*1000*1000) + "/s"
	}

	// update task
	w.task = newTask
}

func (w *Worker) report(status models.TaskStatus) {
	// save
	w.SaveTask(false)

	// publish event
	w.eventBus.Publish(consts.TopicDownloadProgress, types.DownloadResponse{
		ID:            w.task.TaskID,
		TaskStatus:    status,
		DataType:      types.ExtractorDataTypeVideo,
		Total:         w.task.TotalParts,
		Finished:      w.task.FinishedParts,
		Progress:      w.lastReportedProgress,
		SpeedString:   w.getSpeedString(),
		TimeRemaining: w.task.TimeRemaining,
		IsProcessing:  status.IsProcessing(),
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

func (w *Worker) calculateMovingAverageSpeed(currentSpeed float64) float64 {
	now := time.Now()

	// 每隔sampleInterval采样一次
	if now.Sub(w.lastSampleTime) >= w.sampleInterval {
		w.speedSamples = append(w.speedSamples, currentSpeed)
		if len(w.speedSamples) > w.maxSampleCount {
			// 移除最旧的样本
			w.speedSamples = w.speedSamples[1:]
		}
		w.lastSampleTime = now
	}

	// 计算移动平均速度
	if len(w.speedSamples) == 0 {
		return currentSpeed
	}

	var sum float64
	for _, speed := range w.speedSamples {
		sum += speed
	}
	return sum / float64(len(w.speedSamples))
}

func (w *Worker) calculateTimeRemaining(newTask *models.DownloadTask) {
	if newTask.CurrentSpeed <= 0 {
		return
	}

	// 计算移动平均速度
	avgSpeed := w.calculateMovingAverageSpeed(newTask.CurrentSpeed)

	// 如果有开始时间，使用基于进度的预估
	if !newTask.StartTime.IsZero() {
		elapsedTime := time.Since(newTask.StartTime).Seconds()
		progress := float64(newTask.TotalCurrentSize) / float64(newTask.TotalSize)
		if progress > 0 {
			// 基于已完成进度预估总时间
			estimatedTotalTime := elapsedTime / progress
			remainingSeconds := estimatedTotalTime - elapsedTime

			// 结合移动平均速度计算的剩余时间
			remainingBytesBySpeed := float64(newTask.TotalSize-newTask.TotalCurrentSize) / avgSpeed

			// 取两种方法的加权平均，随着进度增加，进度预估的权重增加
			weightProgress := math.Min(progress*2, 0.8) // 进度预估的权重最高到0.8
			weightSpeed := 1 - weightProgress

			finalRemainingSeconds := remainingSeconds*weightProgress + remainingBytesBySpeed*weightSpeed

			newTask.TimeRemaining = formatDuration(time.Duration(finalRemainingSeconds) * time.Second)
			newTask.EstimatedEndTime = time.Now().Add(time.Duration(finalRemainingSeconds) * time.Second)
		}
	} else {
		// 如果没有开始时间，仅使用移动平均速度
		remainingBytes := newTask.TotalSize - newTask.TotalCurrentSize
		remainingSeconds := float64(remainingBytes) / avgSpeed
		newTask.TimeRemaining = formatDuration(time.Duration(remainingSeconds) * time.Second)
		newTask.EstimatedEndTime = time.Now().Add(time.Duration(remainingSeconds) * time.Second)
	}
}

// 格式化持续时间
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
