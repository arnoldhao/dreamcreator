package download

var (
	// start status
	DownloadStatusQueued        = "queued"
	DownloadStatusPedding       = "pedding"
	DownloadStatusUnknown       = "unknown"
	DownloadStatusDownloading   = "downloading"
	DownloadStatusMerging       = "merging"
	DownloadStatusMergingFailed = "mergeFailed"
	DownloadStatusCancelled     = "cancelled"
	DownloadStatusCompleted     = "completed"
	DownloadStatusFailed        = "failed"
	DownloadStatusMerged        = "merged"
	DownloadStatusSuccess       = "success"

	// Update
	CountThreshold = 10

	// caption
	DefaultCaptionExt = "srt"
)
