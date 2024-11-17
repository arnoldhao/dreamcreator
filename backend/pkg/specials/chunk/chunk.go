package chunk

import "fmt"

const (
	CHUNK_SIZE = 1024 * 1024 * 10 // 10MB
)

type Chunk struct {
	Start int64
	End   int64
	Data  chan []byte
}

func GetChunk(totalSize int64) []Chunk {
	var chunks []Chunk

	for start := int64(0); start < totalSize; start += CHUNK_SIZE {
		end := CHUNK_SIZE + start - 1
		if end > totalSize-1 {
			end = totalSize - 1
		}

		chunks = append(chunks, Chunk{start, end, make(chan []byte, 1)})
	}

	// debug log
	fmt.Printf("get chunks: %v\n", chunks)
	return chunks
}
