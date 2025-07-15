package consts

// Websocket Namespaces
type WSNamespace string

const NAMESPACE_DOWNTASKS WSNamespace = "downtasks"
const NAMESPACE_SUBTITLES WSNamespace = "subtitles"

// Websocket Request Events
type WSRequestEventType string

// Websocket Response Events
type WSResponseEventType string

const (
	// DOWNTASKS
	EVENT_DOWNTASKS_PROGRESS    WSResponseEventType = "response_downtasks_progress"
	EVENT_DOWNTASKS_SIGNAL      WSResponseEventType = "response_downtasks_signal"
	EVENT_DOWNTASKS_INSTALLING  WSResponseEventType = "response_downtasks_installing"
	EVENT_DOWNTASKS_COOKIE_SYNC WSResponseEventType = "response_downtasks_cookie_sync"
	// SUBTITLE
	EVENT_SUBTITLE_PROGRESS WSResponseEventType = "response_subtitle_progress"
)

// Topics
const (
	// DOWNTASKS
	TopicDowntasksProgress   = "downtasks.progress"
	TopicDowntasksSignal     = "downtasks.signal"
	TopicDowntasksInstalling = "downtasks.installing"
	TopicDowntasksCookieSync = "downtasks.cookie_sync"
	// SUBTITLE
	TopicSubtitleProgress = "subtitle.progress"
)
