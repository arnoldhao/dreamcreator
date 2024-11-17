package downloads

import (
	"CanMe/backend/pkg/request"
	"CanMe/backend/pkg/specials/chunk"
	"CanMe/backend/types"
	"bytes"
	"context"
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

func (wq *WorkQueue) downloadChunk(url string, chunk *chunk.Chunk, tracker *ProgressTracker) (err error) {
	headers := make(map[string]string)
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

	reader := &ProgressReader{
		Reader:  resp.Body,
		Tracker: tracker,
	}
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

func (wq *WorkQueue) downloadChunks(part *types.ExtractorPart, w *io.PipeWriter) {
	// chunk download
	chunks := chunk.GetChunk(part.Size)
	maxRoutines := wq.getMaxRoutines(len(chunks))

	cancelCtx, cancel := context.WithCancel(wq.ctx)
	abort := func(err error) {
		w.CloseWithError(err)
		cancel()
	}

	progressTracker := &ProgressTracker{
		totalSize: part.Size,
		id:        part.ID,
		fileName:  part.FileName,
		report:    wq.report,
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
				err := wq.downloadChunk(part.URL, chunk, progressTracker)
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

func (wq *WorkQueue) downloadPart(part *types.ExtractorPart) (err error) {
	// temp
	tempFileName := fmt.Sprintf("%s.%s", part.FileName, DOWNLOAD_TEMP_EXT)
	if _, err = os.Stat(tempFileName); err == nil {
		timeStamp := time.Now().Format("20060102150405")
		tempFileName = fmt.Sprintf("%s_%s.%s", part.FileName, timeStamp, DOWNLOAD_TEMP_EXT)
	}

	file, err := os.Create(tempFileName)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer func() {
		file.Close()
		if err == nil {
			os.Rename(tempFileName, part.FileName)
		}
	}()

	r, w := io.Pipe()
	wq.downloadChunks(part, w)

	_, err = io.Copy(file, r)
	return err
}

func (wq *WorkQueue) getMaxRoutines(limit int) int {
	if limit > 0 && MAX_ROUTINES > limit {
		return limit
	}
	return MAX_ROUTINES
}
