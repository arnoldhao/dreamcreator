package consts

// App Info
const APP_NAME = "CanMe"
const APP_DESC = "CanMe is a comprehensive multilingual video download manager with a fluid user experience and powerful content processing capabilities.\n\nCopyright 2025"
const APP_VERSION = "0.1.8"
const BBOLT_DB_NAME = "canme.db"

// App Config File
const PREFERENCES_FILE_NAME = "preferences.yaml"

// App Size
const DEFAULT_WINDOW_WIDTH = 1280
const DEFAULT_WINDOW_HEIGHT = 800
const MIN_WINDOW_WIDTH = 1280
const MIN_WINDOW_HEIGHT = 800

// App Upgrade URL
const CHECK_UPDATE_URL = "https://api.github.com/repos/arnoldhao/canme/releases/latest"

// YTDLP Version
const (
	YTDLP_VERSION          = "2025.03.31"
	YTDLP_CHECK_UPDATE_URL = "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest"
)

// Task
const (
	TASK_TYPE_CUSTOM = "custom"
	TASK_TYPE_QUICK  = "quick"
	TASK_TYPE_MCP    = "mcp"
)

// listend port
const (
	WS_PORT         = 34444
	MCP_SERVER_PORT = 34445
)

// HTTP
const (
	USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36"
)
