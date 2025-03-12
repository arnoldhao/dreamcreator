package consts

// Websocket Namespaces
type WSNamespace string

const NAMESPACE_DOWNTASKS WSNamespace = "downtasks"

// Websocket Request Events
type WSRequestEventType string

// Websocket Response Events
type WSResponseEventType string

const EVENT_DOWNTASKS_PROGRESS WSResponseEventType = "response_downtasks_progress"
const EVENT_DOWNTASKS_SINGLE WSResponseEventType = "response_downtasks_single"

// Topics
const (
	TopicDowntasksProgress = "downtasks.progress"
	TopicDowntasksSingle   = "downtasks.single"
)
