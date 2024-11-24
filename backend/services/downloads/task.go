package downloads

import (
	"CanMe/backend/consts"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"errors"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// setDownloadingTask set downloading task
func (wq *WorkQueue) setDownloadingTask(id string, t *types.DownloadTask) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()
	wq.downloading[id] = t
}

func (wq *WorkQueue) getDownloadingTask(id string) (*types.DownloadTask, bool) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()
	t, ok := wq.downloading[id]
	return t, ok
}

// parseToTask [INITIAL_FUNCTION] with WebSocket Logic
func (wq *WorkQueue) parseToTask(req types.DownloadRequest) (task *types.DownloadTask, cached *types.ExtractorData, err error) {
	// get cached data
	cached, ok := wq.cached.Get(req.ID)
	if !ok {
		return nil, nil, fmt.Errorf("not found")
	}

	sourceDir, err := wq.generateSourceDir(cached.Source)
	if err != nil {
		return nil, nil, err
	}

	// page info
	pageInfo := types.NewPageInfo(cached.Source, cached.Site, cached.Title)

	// define videos and captions
	ss := make(map[string]*types.Part)
	cs := make(map[string]*types.Part)
	var total, size int64
	var quality, format string

	// download caption
	if caption, ok := cached.Captions[req.Caption]; ok && caption != nil {
		if caption.Ext == "" {
			caption.Ext = "srt"
		}
		filePath, err := wq.captionFilePath(cached, caption)
		if err == nil {
			// total ++, caption not have size
			total++

			// new caption part
			cs[filePath] = types.NewPart(req.ID,
				req.Caption,
				sourceDir,
				filePath,
				caption.URL,
				caption.Ext,
				pageInfo,
				nil,
			)
		} else {
			runtime.LogInfof(wq.ctx, "caption filePath error: %v", err)
		}
	} else {
		runtime.LogInfof(wq.ctx, "caption not found")
	}

	// download stream
	if stream, ok := cached.Streams[req.Stream]; ok && stream != nil {
		partLen := len(stream.Parts)
		for idx, p := range stream.Parts {
			var partTitle string
			if partLen == 1 {
				partTitle = cached.Title
			} else {
				partTitle = fmt.Sprintf("%s[%d]", cached.Title, idx)
			}

			filePath, err := wq.generateOutputFile(cached.Source, partTitle, p.Ext)
			if err == nil {
				// total ++, stream have size
				total++
				size += p.Size

				// new stream part
				ss[filePath] = types.NewPart(req.ID,
					req.Stream,
					sourceDir,
					filePath,
					p.URL,
					p.Ext,
					pageInfo,
					types.NewStreamInfo(p.Size, stream.Quality, stream.NeedMux))
			} else {
				runtime.LogInfof(wq.ctx, "stream part filePath error: %v", err)
			}

			if quality == "" {
				quality = stream.Quality
			}

			if format == "" {
				format = p.Ext
			}
		}
	} else {
		runtime.LogInfof(wq.ctx, "stream not found")
	}

	// set downloading task
	task = &types.DownloadTask{
		ID:       req.ID,
		Streams:  ss,
		Captions: cs,
		Total:    total,
		Size:     size,
	}

	// set map
	wq.setDownloadingTask(req.ID, task)

	// set database
	savePath, err := wq.generateSourceDir(cached.Source)
	if err != nil {
		return nil, nil, err
	}

	record := &storage.Downloads{
		ID:        req.ID,
		Status:    consts.DownloadStatusDownloading,
		Source:    cached.Source,
		Site:      cached.Site,
		URL:       cached.URL,
		Title:     cached.Title,
		Quality:   quality,
		Format:    format,
		Total:     task.Total,
		Finished:  0,
		Size:      task.Size,
		Current:   0,
		Progress:  0,
		SavedPath: savePath,
		Error:     "",
	}

	err = record.ReadWithDeleted(wq.ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// create record
			err = record.Create(wq.ctx)
			if err != nil {
				runtime.LogInfof(wq.ctx, "create download record error: %v", err)
			} else {
				runtime.LogInfof(wq.ctx, "create download record success")
			}
		} else {
			runtime.LogInfof(wq.ctx, "read download record error: %v", err)
		}
	}

	// WebSocket Logic:report initial progress to client
	wq.handleReport <- types.DownloadResponse{
		ID:       req.ID,
		Status:   consts.DownloadStatusDownloading,
		DataType: types.ExtractorDataTypeAll,
		Total:    task.Total,
		Finished: 0,
		Progress: 0,
	}

	return task, cached, nil
}

// setStreamCurrent [UPDATAT_STATAUS_FUNCTION] with WebSocket Logic
func (wq *WorkQueue) setStreamCurrent(id, fileName string, current int64) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()

	ac := int64(0)
	var find *types.Part
	if task, ok := wq.downloading[id]; ok {
		for k, v := range task.Streams {
			ac += v.GetCurrent()
			if k == fileName {
				find = v
			}
		}

		if find != nil {
			// calculate increase
			if increase := current - find.GetCurrent(); increase > 0 {
				ac += increase
			}

			// update speed
			task.UpdateSpeed(ac)

			// update current
			find.SetCurrent(current)

			// update task
			task.Streams[fileName] = find

			// update downloading map
			wq.downloading[id] = task

			// WebSocket Logic:report current progress to client
			report := types.DownloadResponse{
				ID:       id,
				Status:   consts.DownloadStatusDownloading,
				DataType: types.ExtractorDataTypeVideo,
				Total:    task.Total,
				Finished: task.Finished,
				Speed:    task.Speed,
				Progress: task.Progress,
			}

			runtime.LogInfof(wq.ctx, "setStreamCurrent report: %v", report)
			wq.handleReport <- report
		} else {
			runtime.LogInfof(wq.ctx, "setStreamCurrent error: fileName:%s not found", fileName)
		}
	} else {
		runtime.LogInfof(wq.ctx, "setStreamCurrent error: id:%s not found", id)
	}
}

// setPartFinished [FINAL_STATUS_FUNCTION] with WebSocket Logic
func (wq *WorkQueue) setPartFinished(id, fileName string, err error) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()

	if task, ok := wq.downloading[id]; ok {
		done := false
		// find stream first
		for k, v := range task.Streams {
			if k == fileName {
				var status consts.DownloadStatus
				// set finished
				v.SetFinished(err)

				// update finished
				task.Finished++
				if err != nil {
					task.Error = err
					status = consts.DownloadStatusDownloadFailed
				} else {
					status = consts.DownloadStatusDownloadSuccess
				}

				// update task
				task.Streams[k] = v
				done = true
				// WebSocket Logic:report current part final status to client
				wq.handleReport <- types.DownloadResponse{
					ID:       id,
					Status:   status,
					DataType: types.ExtractorDataTypeVideo,
					Total:    task.Total,
					Finished: task.Finished,
					Progress: task.Progress,
				}
				break
			}
		}

		// if not found, find caption
		if !done {
			for k, v := range task.Captions {
				if k == fileName {
					var status consts.DownloadStatus
					// set finished
					v.SetFinished(err)

					// update finished
					task.Finished++
					if err != nil {
						task.Error = err
						status = consts.DownloadStatusCaptionsFailed
					} else {
						status = consts.DownloadStatusCaptionsSuccess
					}

					// update task
					task.Captions[k] = v
					done = true
					// WebSocket Logic:report current part final status to client
					wq.handleReport <- types.DownloadResponse{
						ID:       id,
						Status:   status,
						DataType: types.ExtractorDataTypeCaption,
						Total:    task.Total,
						Finished: task.Finished,
						Progress: task.Progress,
					}
					break
				}
			}
		}

		// if not found, return
		if !done {
			runtime.LogInfof(wq.ctx, "setPartFinished error: id:%s found, fileName:%s not found", id, fileName)
			return
		}

		// update task
		wq.downloading[id] = task
		runtime.LogInfof(wq.ctx, "setPartFinished: updated task for id:%s", id)
	} else {
		runtime.LogInfof(wq.ctx, "setPartFinished error: id:%s not found", id)
	}
}

func (wq *WorkQueue) fixDownloadSuccess(id string) (total, finished int64) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()

	if task, ok := wq.downloading[id]; ok {
		task.Status = consts.DownloadStatusDownloadSuccess
		task.Progress = 100.00
		total = task.Total
		finished = task.Finished
		wq.downloading[id] = task
	} else {
		runtime.LogInfof(wq.ctx, "fixDownloadSuccess error: id:%s not found", id)
	}

	return
}

func (wq *WorkQueue) setFinished(id string, status consts.DownloadStatus, err error) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()

	if task, ok := wq.downloading[id]; ok {
		if err != nil {
			task.Error = err
		}

		task.Status = status
		// send to finished
		wq.finished <- task
		delete(wq.downloading, id)
		runtime.LogInfof(wq.ctx, "setFinished: deleted id:%s from downloading map", id)
	} else {
		runtime.LogInfof(wq.ctx, "setFinished error: id:%s not found", id)
	}
}
