package downloads

import (
	"fmt"
	"sync"
	"time"

	"CanMe/backend/consts"
	"CanMe/backend/types"
)

type cachedExtractorData struct {
	data    map[string]*types.ExtractorData
	keys    []string
	maxSize int
	mutex   sync.RWMutex
}

func newCachedExtractorData() *cachedExtractorData {
	return &cachedExtractorData{
		data:    make(map[string]*types.ExtractorData),
		keys:    make([]string, 0, consts.TEMP_EXTRACTOR_DATA_MAX_SIZE),
		maxSize: consts.TEMP_EXTRACTOR_DATA_MAX_SIZE,
	}
}

func (cached *cachedExtractorData) Cache(value *types.ExtractorData) (data *types.ExtractorData) {
	uniqId := fmt.Sprintf("%s-%d", value.Source, time.Now().UnixNano())
	cached.Add(uniqId, value)
	// add id to value
	value.ID = uniqId
	return value
}

func (cached *cachedExtractorData) Add(key string, value *types.ExtractorData) {
	cached.mutex.Lock()
	defer cached.mutex.Unlock()

	// if the key already exists, update the value and return
	if _, exists := cached.data[key]; exists {
		cached.data[key] = value
		return
	}

	// if reach the max size, delete the oldest element
	if len(cached.data) >= cached.maxSize {
		oldestKey := cached.keys[0]
		delete(cached.data, oldestKey)
		cached.keys = cached.keys[1:]
	}

	// add new element
	cached.data[key] = value
	cached.keys = append(cached.keys, key)
}

// Get get the value by key
func (cached *cachedExtractorData) Get(key string) (*types.ExtractorData, bool) {
	cached.mutex.Lock()
	defer cached.mutex.Unlock()

	value, exists := cached.data[key]
	return value, exists
}
