package downloads

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/request"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"CanMe/backend/utils/poolUtil"
	"errors"
	"fmt"
	"sync"

	"github.com/asticode/go-astisub"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (wq *WorkQueue) processCaptions(parts map[string]*types.Part, transform func([]byte) (*astisub.Subtitles, error)) (err error) {
	var id, fileName string
	defer func() {
		if err != nil {
			wq.setPartFinished(id, fileName, err)
		} else {
			wq.setPartFinished(id, fileName, nil)
		}
	}()

	if len(parts) != 1 {
		// get first part id
		for _, part := range parts {
			id = part.GetID()
			fileName = part.GetFileName()
			break
		}
		return fmt.Errorf("captions count error: need 1, but got: %v", len(parts))
	}

	for _, part := range parts {
		if id == "" {
			id = part.GetID()
		}

		if fileName == "" {
			fileName = part.GetFileName()
		}

		srtBytes, err := request.FastNew().GetByte(part.GetURL(), "", nil)
		if err != nil {
			return err
		}

		sub, err := transform(srtBytes)
		if err != nil {
			return err
		}

		err = sub.Write(fileName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (wq *WorkQueue) processStreams(parts map[string]*types.Part) (err error) {
	if len(parts) == 0 {
		return fmt.Errorf("no stream parts")
	}

	var stream *types.Part
	for _, p := range parts {
		stream = p
		break
	}

	if stream == nil {
		return fmt.Errorf("no parts available")
	}

	mergedFilePath, err := wq.streamFilePath(stream.GetSource(),
		stream.GetTitle(),
		stream.GetQuality(),
		stream.GetExt())
	if err != nil {
		return err
	}

	// if only one part, no need merge, download directly
	if len(parts) == 1 {
		// no need merge
		err = wq.downloadPart(&types.ExtractorPart{
			ID:       stream.GetID(),
			FileName: stream.GetFileName(),
			URL:      stream.GetURL(),
			Size:     stream.GetSize(),
			Ext:      stream.GetExt(),
		})
		if err != nil {
			wq.setPartFinished(stream.GetID(), stream.GetFileName(), err)

			// save record error
			wq.setFinished(stream.GetID(), consts.DownloadStatusDownloadFailed, err)
			return err
		} else {
			// fix current < size ,but download success
			_, _ = wq.fixDownloadSuccess(stream.GetID())
			wq.setPartFinished(stream.GetID(), stream.GetFileName(), nil)

			// save record finished
			wq.setFinished(stream.GetID(), consts.DownloadStatusAllSuccess, nil)
			return nil
		}
	}

	// if more than one part, need merge
	wg := poolUtil.NewWaitGroupPool(len(parts)) // todo caculate max
	lock := sync.Mutex{}
	errs := make([]error, 0)
	partsList := make([]string, 0)
	for _, p := range parts {
		partsList = append(partsList, p.GetFileName())
		wg.Add()
		go func(part *types.Part) {
			defer wg.Done()
			err := wq.downloadPart(&types.ExtractorPart{
				ID:       part.GetID(),
				FileName: part.GetFileName(),
				URL:      part.GetURL(),
				Size:     part.GetSize(),
				Ext:      part.GetExt(),
			})
			if err != nil {
				lock.Lock()
				wq.setPartFinished(part.GetID(), part.GetFileName(), err)
				errs = append(errs, err)
				lock.Unlock()
			} else {
				wq.setPartFinished(part.GetID(), part.GetFileName(), nil)
			}

		}(p)
	}

	wg.Wait()
	if len(errs) > 0 {
		errMessage := fmt.Sprintf("some parts download failed: %v", errs)
		// save record error
		wq.setFinished(stream.GetID(), consts.DownloadStatusDownloadFailed, errors.New(errMessage))
		return errors.New(errMessage)
	}

	// fix current < size ,but download success
	total, finished := wq.fixDownloadSuccess(stream.GetID())

	// merge parts
	if stream.GetNeedMux() {
		// WebSocket Logic:report current part final status to client
		wq.handleReport <- types.DownloadResponse{
			ID:       stream.GetID(),
			Status:   consts.DownloadStatusMuxing,
			DataType: types.ExtractorDataTypeVideo,
			Total:    total,
			Finished: finished,
			Progress: 100.00,
		}

		err = wq.muxParts(partsList, mergedFilePath)
		if err != nil {
			// save record error
			wq.setFinished(stream.GetID(), consts.DownloadStatusMuxFailed, err)
			return err
		}
	}

	// save record finished
	wq.setFinished(stream.GetID(), consts.DownloadStatusAllSuccess, nil)
	return nil
}

// processProgress process progress report
func (wq *WorkQueue) processProgress(progress *types.ProgressReport) {
	// validate id
	if progress.GetID() == "" {
		runtime.LogInfof(wq.ctx, "progress report id is empty")
		return
	}

	// update process map
	switch progress.GetDataType() {
	case types.ExtractorDataTypeVideo:
		wq.setStreamCurrent(progress.GetID(), progress.GetFileName(), progress.GetCurrent())
	case types.ExtractorDataTypeCaption:
		runtime.LogInfof(wq.ctx, fmt.Sprintf("caption progress: %v\n", progress))
	default:
		runtime.LogInfof(wq.ctx, fmt.Sprintf("current data type:%v is not support for now", progress.GetDataType()))
	}
}

func (wq *WorkQueue) processFinished(task *types.DownloadTask) {
	// 1. save database
	// 1.1 read record
	record := &storage.Downloads{}
	err := record.Read(wq.ctx, task.ID)
	if err != nil {
		runtime.LogInfof(wq.ctx, fmt.Sprintf("processFinished: %v\n", err))
		return
	}

	errMessage := func() string {
		if task.Error != nil {
			return task.Error.Error()
		}
		return ""
	}()

	// 1.2update record items
	record.Status = task.Status
	record.Total = task.Total
	record.Finished = task.Finished
	record.Size = task.Size
	record.Current = task.Current
	record.Progress = task.Progress
	record.Error = errMessage

	// 1.3update record
	err = record.Update(wq.ctx)
	if err != nil {
		runtime.LogInfof(wq.ctx, fmt.Sprintf("processFinished: %v\n", err))
	}

	// 2. send to client
	wq.handleReport <- types.DownloadResponse{
		ID:       task.ID,
		Status:   task.Status,
		Total:    task.Total,
		Finished: task.Finished,
		DataType: types.ExtractorDataTypeAll,
		Progress: task.Progress,
		Error:    errMessage,
	}
}

func (wq *WorkQueue) processReport(info types.DownloadResponse) {
	// send to client
	wq.wsService.CommonSendToClient(info.ID, info.WSResponseMessage())
}
