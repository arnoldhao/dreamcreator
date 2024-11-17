package innerinterfaces

import "CanMe/backend/types"

type DownloadServiceInterface interface {
	Download(req types.DownloadRequest) (err error)
}
