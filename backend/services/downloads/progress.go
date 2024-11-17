package downloads

import (
	"io"
	"sync/atomic"
	"time"

	"CanMe/backend/types"
)

type ProgressTracker struct {
	totalSize      int64
	downloadedSize atomic.Int64
	lastUpdate     time.Time
	id             string
	fileName       string
	report         chan<- *types.ProgressReport
}

type ProgressReader struct {
	Reader  io.Reader
	Tracker *ProgressTracker
}

func (r *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if n > 0 {
		downloaded := r.Tracker.downloadedSize.Add(int64(n))
		now := time.Now()

		// every DOWNLOAD_UPDATE_INTERVAL update progress
		if now.Sub(r.Tracker.lastUpdate) >= DOWNLOAD_UPDATE_INTERVAL {
			streamInfo := types.NewStreamInfo(r.Tracker.totalSize, "", false)
			streamInfo.SetCurrent(downloaded)
			r.Tracker.report <- &types.ProgressReport{
				Part: types.NewPart(r.Tracker.id,
					"",
					"",
					r.Tracker.fileName,
					"",
					"",
					nil,
					streamInfo,
				),
				DataType: types.ExtractorDataTypeVideo,
			}
			r.Tracker.lastUpdate = now
		}
	}
	return
}
