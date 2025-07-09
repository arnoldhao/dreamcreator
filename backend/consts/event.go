package consts

// Websocket Namespaces
type WSNamespace string

const NAMESPACE_DOWNTASKS WSNamespace = "downtasks"

// Websocket Request Events
type WSRequestEventType string

// Websocket Response Events
type WSResponseEventType string

const (
	EVENT_DOWNTASKS_PROGRESS    WSResponseEventType = "response_downtasks_progress"
	EVENT_DOWNTASKS_SIGNAL      WSResponseEventType = "response_downtasks_signal"
	EVENT_DOWNTASKS_INSTALLING  WSResponseEventType = "response_downtasks_installing"
	EVENT_DOWNTASKS_COOKIE_SYNC WSResponseEventType = "response_downtasks_cookie_sync"
)

// Topics
const (
	TopicDowntasksProgress   = "downtasks.progress"
	TopicDowntasksSignal     = "downtasks.signal"
	TopicDowntasksInstalling = "downtasks.installing"
	TopicDowntasksCookieSync = "downtasks.cookie_sync"
)
