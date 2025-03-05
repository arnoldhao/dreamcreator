package consts

const DEFAULT_FONT_SIZE = 14
const DEFAULT_ASIDE_WIDTH = 300
const DEFAULT_WINDOW_WIDTH = 1280
const DEFAULT_WINDOW_HEIGHT = 800
const MIN_WINDOW_WIDTH = 1280
const MIN_WINDOW_HEIGHT = 800
const DEFAULT_LOAD_SIZE = 10000
const DEFAULT_SCAN_SIZE = 3000

const APP_NAME = "CanMe"
const APP_VERSION = "0.0.10"
const PREFERENCES_FILE_NAME = "preferences.yaml"
const LLMS_FILE_NAME = "llms.yaml"

const CHECK_UPDATE_URL = "https://api.github.com/repos/arnoldhao/canme/releases/latest"

const SUBTITLE_FORMAT_SRT = "srt"
const SUBTITLE_FORMAT_ASS = "ass"
const SUBTITLE_FORMAT_STL = "stl"
const SUBTITLE_FORMAT_TTML = "ttml"
const SUBTITLE_FORMAT_VTT = "vtt"

const TRANSLATION_WORK_QUEUE_MAX_SIZE = 100
const DOWNLOADS_WORK_QUEUE_MAX_SIZE = 100
const TEMP_EXTRACTOR_DATA_MAX_SIZE = 10

// Websocket Namespaces
type WSNamespace string

const NAMESPACE_TRANSLATION WSNamespace = "translation"
const NAMESPACE_DOWNLOAD WSNamespace = "download"
const NAMESPACE_OLLAMA WSNamespace = "ollama"
const NAMESPACE_CHAT WSNamespace = "chat"
const NAMESPACE_PROXY WSNamespace = "proxy"

// Websocket Request Events
type WSRequestEventType string

// Websocket Request Events:Translation
const EVENT_TRANSLATION_START WSRequestEventType = "request_translation_start"
const EVENT_TRANSLATION_CANCEL WSRequestEventType = "request_translation_cancel"

// Websocket Request Events:Ollama
const EVENT_OLLAMA_PULL WSRequestEventType = "request_ollama_pull"

// Websocket Request Events:Download
const EVENT_DOWNLOAD_START WSRequestEventType = "request_download_start"
const EVENT_DOWNLOAD_CANCEL WSRequestEventType = "request_download_cancel"

// Websocket Request Events:Chat
// todo

// Websocket Request Events:Proxy
const EVENT_PROXY_TEST WSRequestEventType = "request_proxy_test"

// Websocket Response Events
type WSResponseEventType string

// Websocket Response Events:Translation
const EVENT_TRANSLATION_PROGRESS WSResponseEventType = "response_translation_progress"
const EVENT_TRANSLATION_CANCELED WSResponseEventType = "response_translation_canceled"
const EVENT_TRANSLATION_COMPLETED WSResponseEventType = "response_translation_completed"
const EVENT_TRANSLATION_ERROR WSResponseEventType = "response_translation_error"

// Websocket Response Events:Ollama
const EVENT_OLLAMA_PULL_UPDATE WSResponseEventType = "response_ollama_pull_update"
const EVENT_OLLAMA_PULL_CANCELED WSResponseEventType = "response_ollama_pull_canceled"
const EVENT_OLLAMA_PULL_COMPLETED WSResponseEventType = "response_ollama_pull_completed"
const EVENT_OLLAMA_PULL_ERROR WSResponseEventType = "response_ollama_pull_error"

// Websocket Response Events:Download
const EVENT_DOWNLOAD_PROGRESS WSResponseEventType = "response_download_progress"
const EVENT_DOWNLOAD_COMPLETED WSResponseEventType = "response_download_completed"
const EVENT_DOWNLOAD_ERROR WSResponseEventType = "response_download_error"

// Websocket Response Events:Proxy
const EVENT_PROXY_TEST_RESULT WSResponseEventType = "response_proxy_test_result"
const EVENT_PROXY_TEST_RESULT_CANCEL WSResponseEventType = "response_proxy_test_cancel"
const EVENT_PROXY_TEST_RESULT_COMPLETED WSResponseEventType = "response_proxy_test_completed"
const EVENT_PROXY_TEST_RESULT_ERROR WSResponseEventType = "response_proxy_test_error"

type WSRequestType string

const REQUEST_TRANSLATION_START WSRequestType = "request_translation_start"   // backend watched ai translation start
const REQUEST_TRANSLATION_CANCEL WSRequestType = "request_translation_cancel" // backend watched ai translation cancel
const REQUEST_OLLAMA_PULL WSRequestType = "request_ollama_pull"               // backend watched ollama pull
const REQUEST_TEST_PROXY WSRequestType = "request_test_proxy"                 // backend watched test proxy
const REQUEST_DOWNLOAD WSRequestType = "request_download"                     // backend watched download

type WSResponseType string

const TRANSLATION_UPDATE WSResponseType = "translation_update" // frontend watched
const OLLAMA_PULL_UPDATE WSResponseType = "ollama_pull_update" // frontend watched

const WS_EVENT_REQUEST_TRANSLATION_START = "request_translation_start"   // backend watched ai translation start
const WS_EVENT_REQUEST_TRANSLATION_CANCEL = "request_translation_cancel" // backend watched ai translation cancel

const WS_EVENT_TRANSLATION_UPDATE = "translation_update"       // frontend watched
const WS_EVENT_TRANSLATION_PROGRESS = "translation_progress"   // frontend watched
const WS_EVENT_TRANSLATION_CANCELED = "translation_canceled"   // frontend watched
const WS_EVENT_TRANSLATION_COMPLETED = "translation_completed" // frontend watched
const WS_EVENT_TRANSLATION_ERROR = "translation_error"         // frontend watched

const WS_EVENT_TEST_PROXY_RESULT = "test_proxy_result" // frontend watched
const WS_EVENT_DOWNLOAD_UPDATE = "download_update"     // frontend watched
type LanguageGroupType string

const LANGUAGE_GROUP_TYPE_COMMON LanguageGroupType = "common"
const LANGUAGE_GROUP_TYPE_EXTRA LanguageGroupType = "extra"

// 翻译提示词
func TranslatePrompt(lang string) string {
	return "Translate the given sentence into simple and natural " + lang
}

// EmitKey
func EmitKey(key string, lang string) string {
	return key + "_" + lang
}

// EmitKeyError
func EmitKeyError(key string, lang string) string {
	return key + "_" + lang + "_error"
}

// EmitKeyDonekey
func EmitKeyDone(key string, lang string) string {
	return key + "_" + lang + "_done"
}

type DownloadStatus string

const (
	// single task status
	DownloadingCacheSaved         DownloadStatus = "Downloading Cache Saved"
	DownloadStatusDownloading     DownloadStatus = "Downloading"
	DownloadStatusDownloadSuccess DownloadStatus = "Video Download Success"
	DownloadStatusDownloadFailed  DownloadStatus = "Video Download Failed"
	DownloadStatusCaptionsSuccess DownloadStatus = "Caption Download Success"
	DownloadStatusCaptionsFailed  DownloadStatus = "Caption Download Failed"

	// entire download status
	DownloadStatusMuxing     DownloadStatus = "Muxing"
	DownloadStatusMuxSuccess DownloadStatus = "Mux Success"
	DownloadStatusMuxFailed  DownloadStatus = "Mux Failed"
	DownloadStatusAllSuccess DownloadStatus = "All Success"
	DownloadStatusAllFailed  DownloadStatus = "All Failed"
	DownloadStatusPartFailed DownloadStatus = "Part Failed"
	DownloadStatusCanceled   DownloadStatus = "Canceled"

	// unknown error
	DownloadStatusUnknownError DownloadStatus = "Unknown Error"
)

const LIST_DOWNLOADS_MAX_SIZE = 50

type PartStatus string

const (
	PartStatusDownloading PartStatus = "Downloading"
	PartStatusMerging     PartStatus = "Merging"
	PartStatusFailed      PartStatus = "Failed"
	PartStatusCompleted   PartStatus = "Completed"
)
