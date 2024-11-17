package extractors

import (
	"CanMe/backend/types"
)

type Extractor interface {
	Extract(url string, option types.ExtractorOptions) ([]*types.ExtractorData, error)
}
