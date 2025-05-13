package types

// ConversionFormat 定义了视频/音频转换格式的结构
type ConversionFormat struct {
	ID        int    `json:"id"`        // 唯一标识符，例如 1, 2, 3...
	Name      string `json:"name"`      // 展示给用户的名称，例如 "MP4"
	Type      string `json:"type"`      // 格式类别，例如 "video", "audio"
	Extension string `json:"extension"` // 文件扩展名，例如 "mp4", "mp3"
	Available bool   `json:"available"` // 此格式是否对用户可见并可用
}

// DefaultConversionFormats 是预定义的转换格式列表
var DefaultConversionFormats = []ConversionFormat{
	{ID: 1, Name: "MP4", Type: "video", Extension: "mp4", Available: true},
	{ID: 2, Name: "WebM", Type: "video", Extension: "webm", Available: true},
	{ID: 101, Name: "MP3", Type: "audio", Extension: "mp3", Available: true},
	{ID: 102, Name: "M4A", Type: "audio", Extension: "m4a", Available: true},
}
