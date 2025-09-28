package types

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// 常量定义
const (
	// 支持的FCPXML版本
	FCPXMLVersion17  = "1.7"
	FCPXMLVersion19  = "1.9"
	FCPXMLVersion111 = "1.11"

	// 常用帧率
	FrameRate24   = 24.0
	FrameRate25   = 25.0
	FrameRate2997 = 29.97
	FrameRate30   = 30.0
	FrameRate50   = 50.0
	FrameRate5994 = 59.94
	FrameRate60   = 60.0

	// 常用分辨率
	Resolution720p  = "1280x720"
	Resolution1080p = "1920x1080"
	Resolution4K    = "3840x2160"

	// 颜色空间
	ColorSpaceRec709  = "1-1-1 (Rec. 709)"
	ColorSpaceRec2020 = "9-16-0 (Rec. 2020)"
)

type GuideLineStandard string

const (
	GuideLineStandardNetflix GuideLineStandard = "netflix"
	GuideLineStandardBBC     GuideLineStandard = "bbc"
	GuideLineStandardADE     GuideLineStandard = "ade"
)

// 接口定义

// Exporter 导出器接口
type Exporter interface {
	Export(project *SubtitleProject) ([]byte, error)
	GetFormat() string
	Validate(config interface{}) error
}

// Validator 验证器接口
type Validator interface {
	Validate() error
}

// 错误类型定义

// SubtitleError 字幕相关错误
type SubtitleError struct {
	Type    string
	Message string
	Cause   error
}

func (e *SubtitleError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *SubtitleError) Unwrap() error {
	return e.Cause
}

// 结构体定义
// SubtitleProject 字幕项目
type SubtitleProject struct {
	// 数值类型优先
	CreatedAt int64 `json:"created_at" yaml:"created_at"`
	UpdatedAt int64 `json:"updated_at" yaml:"updated_at"`

	// 字符串类型
	ID          string `json:"id" yaml:"id"`
	ProjectName string `json:"project_name" yaml:"project_name"`

	// 结构体类型
	Metadata   ProjectMetadata `json:"metadata" yaml:"metadata"`
	SourceFile *SourceFileInfo `json:"source_file,omitempty" yaml:"source_file,omitempty"`

	// 切片和map类型放在最后
	Segments         []SubtitleSegment           `json:"segments"`
	LanguageMetadata map[string]LanguageMetadata `json:"language_metadata"`
	GlobalStyles     map[string]Style            `json:"global_styles" yaml:"global_styles"`
}

// Validate 验证字幕项目的完整性
func (p *SubtitleProject) Validate() error {
	if p.ID == "" {
		return &SubtitleError{
			Type:    "ValidationError",
			Message: "project ID cannot be empty",
		}
	}
	if p.ProjectName == "" {
		return &SubtitleError{
			Type:    "ValidationError",
			Message: "project name cannot be empty",
		}
	}
	return nil
}

// GetLanguageCodes 获取项目中所有语言代码
func (p *SubtitleProject) GetLanguageCodes() []string {
	codes := make([]string, 0, len(p.LanguageMetadata))
	for code := range p.LanguageMetadata {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	return codes
}

// GetSegmentCount 获取字幕片段总数
func (p *SubtitleProject) GetSegmentCount() int {
	return len(p.Segments)
}

// GetTotalDuration 获取项目总时长
func (p *SubtitleProject) GetTotalDuration() time.Duration {
	if len(p.Segments) == 0 {
		return 0
	}
	lastSegment := p.Segments[len(p.Segments)-1]
	return lastSegment.EndTime.Time
}

// ProjectMetadata 表示字幕项目的元数据信息。
// 包含项目的基本信息、FCPXML视频配置和可选的源文件信息。
type ProjectMetadata struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	Version     string `json:"version"`

	// 源文件信息
	SourceInfo *SourceFileInfo `json:"source_info,omitempty"`
	// 导出信息
	ExportConfigs ExportConfigs `json:"export_configs"`

	// 关联来源（可选）：用于与下载任务建立双向关系
	OriginTaskID string `json:"origin_task_id,omitempty"`

	// 原始 ITT 信息（用于高保真还原导出）
	SourceITT *ITTSourceInfo `json:"source_itt,omitempty"`
}

func (v *ProjectMetadata) Validate() error {
	// todo

	return nil
}

// SourceFileInfo 源文件信息（可选）
type SourceFileInfo struct {
	// 数值类型优先
	FileSize    int64   `json:"file_size,omitempty"`
	Duration    float64 `json:"duration,omitempty"`
	OriginalFPS float64 `json:"original_fps,omitempty"`

	// 字符串类型
	FileName string `json:"file_name,omitempty"`
	FilePath string `json:"file_path,omitempty"`
	FileExt  string `json:"file_ext,omitempty"`
	FileDir  string `json:"file_dir,omitempty"`
}

// ExportConfigs 导出配置集合
type ExportConfigs struct {
	FCPXML *FCPXMLExportConfig `json:"fcpxml,omitempty"`
	SRT    *SRTExportConfig    `json:"srt,omitempty"`
	ASS    *ASSExportConfig    `json:"ass,omitempty"`
	VTT    *VTTExportConfig    `json:"vtt,omitempty"`
	ITT    *ITTExportConfig    `json:"itt,omitempty"`
}

// FCPXMLExportConfig FCPXML导出配置（包含视频参数）
type FCPXMLExportConfig struct {
	// 视频核心参数（用户必须配置）
	FrameRate  float64 `json:"frame_rate"`  // 帧率
	Width      int     `json:"width"`       // 视频宽度
	Height     int     `json:"height"`      // 视频高度
	ColorSpace string  `json:"color_space"` // 颜色空间

	// 自动生成的参数
	FrameDuration string `json:"frame_duration,omitempty"` // 自动计算
	PixelAspect   string `json:"pixel_aspect,omitempty"`   // 自动设置
	Interlaced    bool   `json:"interlaced,omitempty"`     // 默认false

	// 项目管理参数（自动生成）
	Version       string   `json:"version,omitempty"`        // 默认"1.7"
	LibraryName   string   `json:"library_name,omitempty"`   // 基于项目名生成
	EventName     string   `json:"event_name,omitempty"`     // 基于项目名生成
	ProjectName   string   `json:"project_name,omitempty"`   // 基于项目名生成
	DefaultLane   int      `json:"default_lane,omitempty"`   // 默认1
	TitleEffect   string   `json:"title_effect,omitempty"`   // 可选
	StartTimecode Timecode `json:"start_timecode,omitempty"` // 默认1小时
}

// FCPXMLExportConfig.Validate() 需要验证合并后的所有字段
func (c *FCPXMLExportConfig) Validate() error {
	// 验证核心视频参数（用户必须提供）
	if c.Width <= 0 || c.Height <= 0 {
		return &SubtitleError{
			Type:    "ValidationError",
			Message: fmt.Sprintf("invalid video dimensions: %dx%d", c.Width, c.Height),
		}
	}
	if c.FrameRate <= 0 || c.FrameRate > 120 {
		return &SubtitleError{
			Type:    "ValidationError",
			Message: fmt.Sprintf("invalid frame rate: %f", c.FrameRate),
		}
	}
	if c.ColorSpace == "" {
		return &SubtitleError{
			Type:    "ValidationError",
			Message: "color space is required",
		}
	}

	// 项目名称是必需的（用于生成其他字段）
	if c.ProjectName == "" {
		return &SubtitleError{
			Type:    "ValidationError",
			Message: "project name is required",
		}
	}

	return nil
}

// Default4K60FPSSDR 设置为4K 60FPS的默认值
func (c *FCPXMLExportConfig) Default4K60FPSSDR() {
	c.Width = 3840
	c.Height = 2160
	c.FrameRate = 60.0
	c.ColorSpace = ColorSpaceRec709
}

// 添加自动填充方法
func (c *FCPXMLExportConfig) AutoFill() {
	// 自动计算 FrameDuration
	if c.FrameDuration == "" {
		// 计算帧时长，例如：30fps = "100/3000s"
		denominator := int(c.FrameRate * 100)
		c.FrameDuration = fmt.Sprintf("100/%ds", denominator)
	}

	// 设置默认 PixelAspect
	if c.PixelAspect == "" {
		c.PixelAspect = "1"
	}

	// 设置默认版本
	if c.Version == "" {
		c.Version = FCPXMLVersion17
	}

	// 基于项目名称生成其他字段
	if c.LibraryName == "" {
		c.LibraryName = c.ProjectName + "_Library"
	}
	if c.EventName == "" {
		c.EventName = c.ProjectName + "_Event"
	}
	if c.ProjectName == "" {
		c.ProjectName = c.ProjectName + "_Project"
	}

	// 设置默认值
	if c.DefaultLane == 0 {
		c.DefaultLane = 1
	}
	if c.StartTimecode.Time == 0 {
		// 设置默认为1小时
		c.StartTimecode = NewTimecode(time.Hour, c.FrameRate)
	}
}

// SRTExportConfig SRT导出配置
type SRTExportConfig struct {
	Encoding string `json:"encoding,omitempty"` // 字符编码
}

// ASSExportConfig ASS导出配置
type ASSExportConfig struct {
	PlayResX int    `json:"play_res_x,omitempty"` // 播放分辨率X
	PlayResY int    `json:"play_res_y,omitempty"` // 播放分辨率Y
	Title    string `json:"title,omitempty"`      // 标题
}

// VTTExportConfig VTT导出配置
type VTTExportConfig struct {
	Kind     string `json:"kind,omitempty"`     // 字幕类型
	Language string `json:"language,omitempty"` // 语言代码
}

// ITTExportConfig iTunes Timed Text (TTML subset) 导出配置
type ITTExportConfig struct {
	FrameRate float64 `json:"frame_rate,omitempty"`
	Language  string  `json:"language,omitempty"`
	// 可选：覆盖/指定 ITT 头部参数（若不设置，导出时回退到 SourceITT 或默认）
	TimeBase   string        `json:"time_base,omitempty"`
	DropMode   string        `json:"drop_mode,omitempty"`
	Multiplier IttMultiplier `json:"frame_rate_multiplier,omitempty"`
}

// IttMultiplier 表示 ttp:frameRateMultiplier 中的分子与分母
type IttMultiplier struct {
	Num int `json:"num,omitempty"`
	Den int `json:"den,omitempty"`
}

// ITTSourceInfo 保存导入 ITT 的关键信息，便于高保真回写
type ITTSourceInfo struct {
	XmlLang    string        `json:"xml_lang,omitempty"`
	TimeBase   string        `json:"time_base,omitempty"`
	FrameRate  float64       `json:"frame_rate,omitempty"`
	Multiplier IttMultiplier `json:"frame_rate_multiplier,omitempty"`
	DropMode   string        `json:"drop_mode,omitempty"`

	Regions      []ITTRegion     `json:"regions,omitempty"`
	BodyDefaults ITTBodyDefaults `json:"body_defaults,omitempty"`
}

type ITTRegion struct {
	ID           string `json:"id"`
	Origin       string `json:"origin,omitempty"`
	Extent       string `json:"extent,omitempty"`
	DisplayAlign string `json:"display_align,omitempty"`
	WritingMode  string `json:"writing_mode,omitempty"`
}

type ITTBodyDefaults struct {
	Region string `json:"region,omitempty"`
	Style  string `json:"style,omitempty"`
	Color  string `json:"color,omitempty"`
}

// Style 表示字幕文本的样式
type Style struct {
	// 数值类型优先
	FontSize     float64 `json:"font_size" yaml:"font_size"`         // 字体大小
	LineHeight   float64 `json:"line_height" yaml:"line_height"`     // 行高
	OutlineWidth float64 `json:"outline_width" yaml:"outline_width"` // 描边宽度

	// 布尔类型
	Bold      bool `json:"bold" yaml:"bold"`           // 是否加粗
	Italic    bool `json:"italic" yaml:"italic"`       // 是否斜体
	Underline bool `json:"underline" yaml:"underline"` // 是否有下划线

	// 字符串类型
	FontName        string `json:"font_name" yaml:"font_name"`               // 字体名称
	Color           string `json:"color" yaml:"color"`                       // 文本颜色
	Alignment       string `json:"alignment" yaml:"alignment"`               // 对齐方式
	VerticalAlign   string `json:"vertical_align" yaml:"vertical_align"`     // 垂直对齐
	PositionX       string `json:"position_x" yaml:"position_x"`             // 水平位置
	PositionY       string `json:"position_y" yaml:"position_y"`             // 垂直位置
	BackgroundColor string `json:"background_color" yaml:"background_color"` // 背景颜色
	OutlineColor    string `json:"outline_color" yaml:"outline_color"`       // 描边颜色

	// map类型放在最后
	FcpXMLSpecificStyle map[string]string `json:"fcpxml_specific_style" yaml:"fcpxml_specific_style"` // FCPXML特定样式
}

// LanguageMetadata 保持原有的LanguageMetadata结构
type LanguageMetadata struct {
	// 数值类型优先
	Revision int `json:"revision" yaml:"revision"`

	// Detected
	DetectedLang int    `json:"detected_lang" yaml:"detected_lang"`
	LanguageName string `json:"language_name" yaml:"language_name"`

	Translator string `json:"translator" yaml:"translator"`
	Notes      string `json:"notes" yaml:"notes"`
	Quality    string `json:"quality" yaml:"quality"`
	SyncStatus string `json:"sync_status" yaml:"sync_status"`

	// map类型放在最后
	CustomFields map[string]string `json:"custom_fields" yaml:"custom_fields"`

	Status       LanguageContentStatus `json:"status" yaml:"status"`
	ActiveTaskID string                `json:"active_task_id,omitempty" yaml:"active_task_id,omitempty"` // 当前活跃的转换任务ID
}

func (v *LanguageMetadata) Validate() error {
	// todo
	return nil
}

// SubtitleSegment 字幕片段
type SubtitleSegment struct {
	ID        string   `json:"id"`
	StartTime Timecode `json:"start_time"`
	EndTime   Timecode `json:"end_time"`
	Speaker   string   `json:"speaker"`
	Notes     string   `json:"notes"`

	// 多语言内容直接包含在片段中
	Languages         map[string]LanguageContent   `json:"languages"`
	GuidelineStandard map[string]GuideLineStandard `json:"guideline_standard"`
	IsKidsContent     bool                         `json:"is_kids_content"`
}

// Validate 验证字幕片段的完整性
func (s *SubtitleSegment) Validate() error {
	// 验证 ID
	if s.ID == "" {
		return fmt.Errorf("segment ID cannot be empty")
	}

	// 验证时间码
	if s.StartTime.Time < 0 {
		return fmt.Errorf("start time cannot be negative")
	}

	if s.EndTime.Time < 0 {
		return fmt.Errorf("end time cannot be negative")
	}

	// 不验证时间逻辑，在代码中对开始结束时间相同的字幕进行过滤
	// if s.EndTime.Time <= s.StartTime.Time {
	// 	return fmt.Errorf("end time (%v) must be after start time (%v)", s.EndTime.Time, s.StartTime.Time)
	// }

	// 验证持续时间不能过短（至少100毫秒）
	duration := s.Duration()
	if duration < 100*time.Millisecond {
		return fmt.Errorf("segment duration (%v) is too short, minimum 100ms required", duration)
	}

	// 验证持续时间不能过长（最多30秒）
	// if duration > 30*time.Second {
	// 	return fmt.Errorf("segment duration (%v) is too long, maximum 30s allowed", duration)
	// }

	// 验证语言内容
	if len(s.Languages) == 0 {
		return fmt.Errorf("segment must contain at least one language")
	}

	// 验证每个语言内容
	for langCode, content := range s.Languages {
		if langCode == "" {
			return fmt.Errorf("language code cannot be empty")
		}

		if err := content.Validate(); err != nil {
			return fmt.Errorf("invalid content for language '%s': %w", langCode, err)
		}
	}

	return nil
}

// Duration 计算字幕片段的持续时间
func (s *SubtitleSegment) Duration() time.Duration {
	return s.EndTime.Time - s.StartTime.Time
}

// HasLanguage 检查是否包含指定语言
func (s *SubtitleSegment) HasLanguage(langCode string) bool {
	_, exists := s.Languages[langCode]
	return exists
}

// GetText 获取指定语言的文本内容
func (s *SubtitleSegment) GetText(langCode string) string {
	if content, exists := s.Languages[langCode]; exists {
		return content.Text
	}
	return ""
}

// LanguageContent 语言内容
type LanguageContent struct {
	Text              string             `json:"text"`
	SubtitleGuideline *SubtitleGuideline `json:"subtitle_guideline"` // 字幕指南
	Style             *Style             `json:"style"`
	StyleID           string             `json:"style_id,omitempty"`
	RegionID          string             `json:"region_id,omitempty"`
}

type SubtitleGuideline struct {
	CPS *Guideline `json:"cps"` // Characters-per-second
	WPM *Guideline `json:"wpm"` // Words-per-minute
	CPL *Guideline `json:"cpl"` // Characters-per-line

}

type Guideline struct {
	Current int `json:"current"` // 当前数值
	Level   int `json:"level"`   // 0.正常值 1.超出标准 2.大量超出标准
}

func (l *LanguageContent) Validate() error {
	// 验证文本内容
	if strings.TrimSpace(l.Text) == "" {
		return fmt.Errorf("text content cannot be empty")
	}

	return nil
}

// Timecode 表示时间码
type Timecode struct {
	Time   time.Duration `json:"time" yaml:"time"`     // 基于时间的表示，易于计算
	Frames int64         `json:"frames" yaml:"frames"` // 基于帧的表示 (可选)
}

// ToTimecodeString 转换为标准时间码字符串格式 "HH:MM:SS:FF"
func (t *Timecode) ToTimecodeString(frameRate float64) string {
	total := int64(t.Time.Seconds())
	hours := total / 3600
	minutes := (total % 3600) / 60
	seconds := total % 60

	// 计算帧数
	framesPart := t.Time.Seconds() - float64(total)
	frames := int64(framesPart * frameRate)

	return fmt.Sprintf("%02d:%02d:%02d:%02d", hours, minutes, seconds, frames)
}

// ToSecondsString 转换为秒数字符串
func (t *Timecode) ToSecondsString() string {
	return fmt.Sprintf("%.0f", t.Time.Seconds())
}

// 自定义 JSON 序列化，支持字符串格式输入
func (t *Timecode) UnmarshalJSON(data []byte) error {
	// 尝试解析为字符串格式
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		// 假设默认帧率为 25fps，或者从上下文获取
		tc, err := NewTimecodeFromString(str, 25.0)
		if err != nil {
			return err
		}
		*t = tc
		return nil
	}

	// 尝试解析为对象格式
	type Alias Timecode
	return json.Unmarshal(data, (*Alias)(t))
}

func (t Timecode) MarshalJSON() ([]byte, error) {
	// 可以选择输出格式：对象格式或字符串格式
	type Alias Timecode
	return json.Marshal((Alias)(t))
}

// FromTimecodeString 从时间码字符串创建 Timecode
func NewTimecodeFromString(timecodeStr string, frameRate float64) (Timecode, error) {
	parts := strings.Split(timecodeStr, ":")
	if len(parts) != 4 {
		return Timecode{}, fmt.Errorf("invalid timecode format: %s", timecodeStr)
	}

	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	seconds, _ := strconv.Atoi(parts[2])
	frames, _ := strconv.Atoi(parts[3])

	// 转换为总时间
	totalSeconds := float64(hours*3600+minutes*60+seconds) + float64(frames)/frameRate
	duration := time.Duration(totalSeconds * float64(time.Second))

	return NewTimecode(duration, frameRate), nil
}

// NewTimecode 创建新的时间码
func NewTimecode(duration time.Duration, frameRate float64) Timecode {
	frames := int64(duration.Seconds() * frameRate)
	return Timecode{
		Time:   duration,
		Frames: frames,
	}
}

// ToSRTFormat 转换为SRT格式的时间字符串
func (t *Timecode) ToSRTFormat() string {
	total := int64(t.Time.Milliseconds())
	hours := total / 3600000
	minutes := (total % 3600000) / 60000
	seconds := (total % 60000) / 1000
	milliseconds := total % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, milliseconds)
}

// ToFCPXMLFormat 转换为FCPXML格式的时间字符串
func (t *Timecode) ToFCPXMLFormat(frameRate float64) string {
	totalFrames := int64(t.Time.Seconds() * frameRate)
	if frameRate == FrameRate2997 {
		return fmt.Sprintf("%d/30000s", totalFrames*1001)
	}
	return fmt.Sprintf("%d/%ds", totalFrames*1000, int64(frameRate*1000))
}

// ToASSFormat 转换为ASS格式的时间字符串
func (t *Timecode) ToASSFormat() string {
	total := int64(t.Time.Milliseconds())
	hours := total / 3600000
	minutes := (total % 3600000) / 60000
	seconds := (total % 60000) / 1000
	centiseconds := (total % 1000) / 10
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, centiseconds)
}

// ToVTTFormat 转换为VTT格式的时间字符串
func (t *Timecode) ToVTTFormat() string {
	total := int64(t.Time.Milliseconds())
	hours := total / 3600000
	minutes := (total % 3600000) / 60000
	seconds := (total % 60000) / 1000
	milliseconds := total % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
}

// 转换状态枚举
type ConversionStatus string

const (
	ConversionStatusPending    ConversionStatus = "pending"    // 等待转换
	ConversionStatusProcessing ConversionStatus = "processing" // 转换中
	ConversionStatusCompleted  ConversionStatus = "completed"  // 转换完成
	ConversionStatusFailed     ConversionStatus = "failed"     // 转换失败
	ConversionStatusCancelled  ConversionStatus = "cancelled"  // 转换取消
)

// 转换任务信息
type ConversionTask struct {
	ID           string           `json:"id"`
	Type         string           `json:"type"` // "zhconvert" 或 "llm_translate"
	Status       ConversionStatus `json:"status"`
	Progress     float64          `json:"progress"` // 0-100
	StartTime    int64            `json:"start_time"`
	EndTime      int64            `json:"end_time,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`

	// 转换参数
	SourceLang    string `json:"source_lang"`
	TargetLang    string `json:"target_lang"`
	Converter     int    `json:"converter,omitempty"` // zhconvert 转换器类型
	ConverterName string `json:"converter_name,omitempty"`
	Provider      string `json:"provider,omitempty"` // LLM 提供商

	// 进度详情
	TotalSegments     int `json:"total_segments"`
	ProcessedSegments int `json:"processed_segments"`
	FailedSegments    int `json:"failed_segments"`
}

// 语言内容状态
type LanguageContentStatus struct {
	IsOriginal      bool             `json:"is_original"`      // 是否为原始语言
	ConversionTasks []ConversionTask `json:"conversion_tasks"` // 转换任务历史
	LastUpdated     int64            `json:"last_updated"`
}
