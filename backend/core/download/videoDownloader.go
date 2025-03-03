package download

import (
	"CanMe/backend/models"
	"CanMe/backend/pkg/request"
	"CanMe/backend/pkg/specials/chunk"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

const (
	DOWNLOAD_TEMP_EXT        = "canme"                // todo: temp config, update in config next version
	DOWNLOAD_UPDATE_INTERVAL = 100 * time.Millisecond // todo: temp config, update in config next version
	MAX_ROUTINES             = 10                     // todo: temp config, update in config next version
)

type VideoDownloader struct {
	ctx    context.Context
	cancel context.CancelFunc
	refer  string
	part   *models.StreamPart
	update chan *models.ProgressReciver
}

// 缺少构造函数
func NewVideoDownloader(ctx context.Context, refer string, part *models.StreamPart, update chan *models.ProgressReciver) *VideoDownloader {
	ctx, cancel := context.WithCancel(ctx)
	return &VideoDownloader{
		ctx:    ctx,
		refer:  refer,
		part:   part,
		cancel: cancel,
		update: update,
	}
}

func (d *VideoDownloader) Download() error {
	if d.part.Type != "stream" {
		return errors.New("part type is not stream")
	}

	err := d.downloadPart()
	update := &models.ProgressReciver{
		PartID: d.part.PartID,
		TaskID: d.part.TaskID,
	}

	if err != nil {
		update.Status = models.TaskStatusPartialFailed
		update.Error = err
		d.update <- update
		return err
	}

	update.Status = models.TaskStatusPartialSuccess
	d.update <- update

	return nil
}

func (d *VideoDownloader) Cancel() error {
	d.cancel()
	return nil
}

func (d *VideoDownloader) downloadPart() (err error) {
	// temp
	tempFileName := fmt.Sprintf("%s.%s", d.part.FileName, DOWNLOAD_TEMP_EXT)
	if _, err = os.Stat(tempFileName); err == nil {
		timeStamp := time.Now().Format("20060102150405")
		tempFileName = fmt.Sprintf("%s_%s.%s", d.part.FileName, timeStamp, DOWNLOAD_TEMP_EXT)
	}

	file, err := os.Create(tempFileName)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer func() {
		file.Close()
		if err == nil {
			os.Rename(tempFileName, d.part.FileName)
		}
	}()

	r, w := io.Pipe()
	d.downloadChunks(w)

	_, err = io.Copy(file, r)
	return err
}

func (d *VideoDownloader) downloadChunks(w *io.PipeWriter) {
	// chunk download
	chunks := chunk.GetChunk(d.part.TotalSize)
	maxRoutines := d.getMaxRoutines(len(chunks))

	cancelCtx, cancel := context.WithCancel(d.ctx)
	abort := func(err error) {
		w.CloseWithError(err)
		cancel()
	}

	currentChunk := atomic.Uint32{}
	for i := 0; i < maxRoutines; i++ {
		go func() {
			for {
				chunkIndex := int(currentChunk.Add(1)) - 1
				if chunkIndex >= len(chunks) {
					// no more chunks
					return
				}

				chunk := &chunks[chunkIndex]
				err := d.downloadChunk(d.part.URL, chunk)
				close(chunk.Data)

				if err != nil {
					abort(err)
					return
				}
			}
		}()
	}

	go func() {
		// copy chunks into the PipeWriter
		for i := 0; i < len(chunks); i++ {
			select {
			case <-cancelCtx.Done():
				abort(context.Canceled)
				return
			case data := <-chunks[i].Data:
				_, err := io.Copy(w, bytes.NewBuffer(data))

				if err != nil {
					abort(err)
				}
			}
		}

		// everything succeeded
		w.Close()
	}()
}

func (d *VideoDownloader) downloadChunk(url string, chunk *chunk.Chunk) (err error) {
	headers := make(map[string]string)
	headers["Referer"] = d.refer
	headers["Range"] = fmt.Sprintf("bytes=%d-%d", chunk.Start, chunk.End)

	resp, err := request.FastNew().Request(http.MethodGet, url, nil, headers)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK && resp.StatusCode >= 300 {
		return fmt.Errorf("failed to download chunk: %v", resp.StatusCode)
	}

	expected := chunk.End - chunk.Start + 1

	reader := NewProgressReader(resp.Body, d.part, d.update)
	data, err := io.ReadAll(reader)
	n := len(data)
	if err != nil {
		return err
	}

	if n != int(expected) {
		return fmt.Errorf("chunk at offset %d has invalid size: expected=%d actual=%d", chunk.Start, expected, n)
	}

	chunk.Data <- data
	return nil
}

func (d *VideoDownloader) getMaxRoutines(limit int) int {
	if limit > 0 && MAX_ROUTINES > limit {
		return limit
	}
	return MAX_ROUTINES
}
