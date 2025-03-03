package download

import (
	"io"

	"CanMe/backend/models"
)

type ProgressReader struct {
	Reader io.Reader
	part   *models.StreamPart
	update chan *models.ProgressReciver
}

func NewProgressReader(reader io.Reader, part *models.StreamPart, update chan *models.ProgressReciver) *ProgressReader {
	return &ProgressReader{
		Reader: reader,
		part:   part,
		update: update,
	}
}

func (r *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if n > 0 {
		// update part calculate
		r.update <- &models.ProgressReciver{
			PartID: r.part.PartID,
			TaskID: r.part.TaskID,
			Status: models.TaskStatusDownloading,
			Added:  int64(n),
		}
	}
	return
}
