package downloads

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"

	"CanMe/backend/pkg/extractors"
	innerInterfaces "CanMe/backend/services/innerInterfaces"
	"CanMe/backend/storage"
	"CanMe/backend/types"

	"gorm.io/gorm"
)

type WorkQueue struct {
	mutex        sync.Mutex
	ctx          context.Context
	wsService    innerInterfaces.WebSocketServiceInterface
	cached       *cachedExtractorData
	downloading  map[string]*types.DownloadTask
	finished     chan *types.DownloadTask
	report       chan *types.ProgressReport
	handleReport chan types.DownloadResponse
}

func New() *WorkQueue {
	return &WorkQueue{
		cached:       newCachedExtractorData(),
		downloading:  make(map[string]*types.DownloadTask),
		finished:     make(chan *types.DownloadTask, 100),
		report:       make(chan *types.ProgressReport, 100),
		handleReport: make(chan types.DownloadResponse, 100),
	}
}

func (wq *WorkQueue) Process(ctx context.Context, wsService innerInterfaces.WebSocketServiceInterface) {
	wq.ctx = ctx
	wq.wsService = wsService
	go func() {
		for {
			select {
			case <-wq.ctx.Done():
				return
			case report := <-wq.handleReport:
				// process send to client
				wq.processReport(report)
			case final := <-wq.finished:
				// process finished
				wq.processFinished(final)
			case progress := <-wq.report:
				// process progress report
				wq.processProgress(progress)
			}
		}
	}()
}

func (wq *WorkQueue) Get(url string) (resp types.JSResp) {
	if url == "" {
		resp.Msg = "url is empty"
		resp.Success = false
		return
	}

	video, err := extractors.Extract(url, types.ExtractorOptions{})
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		return
	}

	// cache
	var cachedVideo []*types.ExtractorData
	for _, v := range video {
		cachedVideo = append(cachedVideo, wq.cached.Cache(v))
	}

	videoBytes, err := json.Marshal(cachedVideo)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		return
	}

	resp.Data = string(videoBytes)
	resp.Success = true
	return
}

func (wq *WorkQueue) GetFFMPEGVersion() (resp types.JSResp) {
	out, err := exec.Command(findFFmpeg(), "-version").Output()
	if err != nil {
		resp.Msg = fmt.Errorf("failed to get ffmpeg: %v", err).Error()
		resp.Success = false
		return
	}

	resp.Data = string(out)
	resp.Success = true
	return
}

// CheckTask check if the task is downloading, if so, return a new temp id, otherwise return old task id
func (wq *WorkQueue) CheckTask(id, streamID, captionsID string) (resp types.JSResp) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()

	if task, ok := wq.downloading[id]; ok {
		if streamID != "" {
			for _, stream := range task.Streams {
				if stream.GetRequestCode() == streamID {
					resp.Msg = fmt.Sprintf("requested stream:%v is downloading", streamID)
					resp.Success = false
					return
				}
			}
		}
		if captionsID != "" {
			for _, caption := range task.Captions {
				if caption.GetRequestCode() == captionsID {
					resp.Msg = fmt.Sprintf("requested caption:%v is downloading", captionsID)
					resp.Success = false
					return
				}
			}
		}
	}

	record := storage.Downloads{}
	err := record.Read(wq.ctx, id)
	if err != nil && err == gorm.ErrRecordNotFound {
		resp.Data = id // return old task id
		resp.Success = true
		return
	}

	cached, ok := wq.cached.Get(id)
	if cached != nil && ok {
		// save new temp
		newTemp := wq.cached.Cache(cached)
		resp.Data = newTemp.ID // return new temp id
		resp.Success = true
		return
	}
	return
}

func (wq *WorkQueue) ListDownloaded() (resp types.JSResp) {
	records, err := storage.ListDownloads(wq.ctx)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		return
	}

	recordsBytes, err := json.Marshal(records)
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
		return
	}

	resp.Data = string(recordsBytes)
	resp.Success = true
	return
}

// Download download video and caption for websocket request
func (wq *WorkQueue) Download(req types.DownloadRequest) (err error) {
	// parse to task
	t, cached, err := wq.parseToTask(req)
	if err != nil {
		return err
	}

	// download caption
	if caption := t.Captions; caption != nil {
		_ = wq.processCaptions(caption, cached.CaptionsTransform)
	}

	// download stream
	if stream := t.Streams; stream != nil {
		_ = wq.processStreams(stream)
	}

	return nil
}
