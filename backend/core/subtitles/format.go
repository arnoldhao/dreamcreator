package subtitles

import (
	"CanMe/backend/types"
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FormatConverterImpl 格式转换器实现
type FormatConverterImpl struct {
	detector LanguageDetector
}

// NewFormatConverter 创建格式转换器
func NewFormatConverter() FormatConverter {
	return &FormatConverterImpl{
		detector: NewLanguageDetector(),
	}
}

func (fc *FormatConverterImpl) FromItt(filePath string, file []byte) (types.SubtitleProject, error) {
	return fc.fromItt(filePath, file)
}

func (fc *FormatConverterImpl) FromSRT(filePath string, file []byte) (types.SubtitleProject, error) {
	return fc.fromSrt(filePath, file)
}

func (fc *FormatConverterImpl) ToSRT(project *types.SubtitleProject, langCode string) ([]byte, error) {
	return fc.toSrt(project, langCode)
}

func (fc *FormatConverterImpl) ToVTT(project *types.SubtitleProject, langCode string) ([]byte, error) {
	return fc.toVtt(project, langCode)
}

func (fc *FormatConverterImpl) ToFCPXML(project *types.SubtitleProject, langCode string) ([]byte, error) {
	return fc.toFcpxml(project, langCode)
}

// 保持原有的 fromItt 方法
func (s *FormatConverterImpl) fromItt(filePath string, file []byte) (types.SubtitleProject, error) {
	now := time.Now().Unix()
	// 创建一个新的 SubtitleProject
	fileName, ext := separateFilePath(filePath)
	project := types.SubtitleProject{
		ID:               uuid.New().String(),
		ProjectName:      fileName,
		CreatedAt:        now,
		UpdatedAt:        now,
		GlobalStyles:     make(map[string]types.Style),
		LanguageMetadata: make(map[string]types.LanguageMetadata),
		Segments:         []types.SubtitleSegment{},
	}

	// 解析 ITT XML 内容
	var itt types.IttDocument
	if err := xml.Unmarshal(file, &itt); err != nil {
		return types.SubtitleProject{}, fmt.Errorf("failed to parse ITT file: %v", err)
	}

	// 设置项目元数据
	project.Metadata = types.ProjectMetadata{
		ID:          project.ID,
		Name:        itt.Head.Metadata.Title,
		Description: fmt.Sprintf("Imported from ITT file: %s", fileName),
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     "1.0",
		SourceInfo: &types.SourceFileInfo{
			FileName: fileName,
			FileExt:  ext,
			FilePath: filePath,
			FileSize: int64(len(file)),
		},
		ExportConfigs: types.ExportConfigs{
			FCPXML: &types.FCPXMLExportConfig{
				ProjectName: fileName,
			},
			SRT: &types.SRTExportConfig{},
			ASS: &types.ASSExportConfig{},
			VTT: &types.VTTExportConfig{},
		},
	}
	// fcpxml 4k 60fps sdr default
	project.Metadata.ExportConfigs.FCPXML.Default4K60FPSSDR()
	project.Metadata.ExportConfigs.FCPXML.AutoFill()

	// 处理全局样式
	for _, style := range itt.Head.Styling.Styles {
		project.GlobalStyles[style.ID] = types.Style{
			FontName:        style.FontFamily,
			FontSize:        style.FontSize,
			Color:           style.Color,
			Alignment:       style.TextAlign,
			BackgroundColor: style.BackgroundColor,
		}
	}

	// 处理字幕条目
	var allContents []types.LanguageContent
	var allText string
	for _, body := range itt.Body.Div {
		for _, p := range body.P {
			// 解析时间码
			startTime, err := parseITTTimecode(p.Begin)
			if err != nil {
				return types.SubtitleProject{}, fmt.Errorf("failed to parse begin timecode: %v", err)
			}
			endTime, err := parseITTTimecode(p.End)
			if err != nil {
				return types.SubtitleProject{}, fmt.Errorf("failed to parse end timecode: %v", err)
			}

			// 创建语言内容
			content := types.LanguageContent{
				Text: p.Content,
			}

			// 处理样式
			if p.Style != "" {
				if style, ok := project.GlobalStyles[p.Style]; ok {
					content.Style = &style
				}
			}

			allContents = append(allContents, content)
			allText = allText + content.Text + "\n"

			// 创建字幕片段
			segment := types.SubtitleSegment{
				ID:                uuid.New().String(),
				StartTime:         startTime,
				EndTime:           endTime,
				Speaker:           p.Speaker,
				Languages:         make(map[string]types.LanguageContent),
				GuidelineStandard: make(map[string]types.GuideLineStandard),
			}

			project.Segments = append(project.Segments, segment)
		}
	}

	// 检测语言 - 使用相对距离逻辑
	langInt, langCode := s.detector.DetectLanguageInt(allText)

	// 设置语言元数据
	project.LanguageMetadata[langCode] = types.LanguageMetadata{
		DetectedLang: int(langInt),
		LanguageName: langCode,
		Quality:      "imported",
		SyncStatus:   "synced",
	}

	// 将内容分配到对应的片段
	for i, content := range allContents {
		if i < len(project.Segments) {
			project.Segments[i].Languages[langCode] = content
		}
	}

	project.UpdatedAt = time.Now().Unix()
	return project, nil
}

// parseITTTimecode 将 ITT 时间码字符串转换为 Timecode 结构
func parseITTTimecode(tc string) (types.Timecode, error) {
	// ITT 时间码格式：HH:MM:SS:FF 或 HH:MM:SS.mmm
	var hours, minutes, seconds, frames int
	var milliseconds float64

	if strings.Contains(tc, ".") {
		// 处理毫秒格式
		_, err := fmt.Sscanf(tc, "%d:%d:%d.%f", &hours, &minutes, &seconds, &milliseconds)
		if err != nil {
			return types.Timecode{}, err
		}
	} else {
		// 处理帧格式
		_, err := fmt.Sscanf(tc, "%d:%d:%d:%d", &hours, &minutes, &seconds, &frames)
		if err != nil {
			return types.Timecode{}, err
		}
	}

	// 计算总时长
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds*float64(time.Millisecond))

	return types.Timecode{
		Time:   duration,
		Frames: int64(frames),
	}, nil
}

func (s *FormatConverterImpl) fromSrt(filePath string, file []byte) (types.SubtitleProject, error) {
	now := time.Now().Unix()
	// 创建一个新的 SubtitleProject
	fileName, ext := separateFilePath(filePath)
	project := types.SubtitleProject{
		ID:               uuid.New().String(),
		ProjectName:      fileName,
		CreatedAt:        now,
		UpdatedAt:        now,
		GlobalStyles:     make(map[string]types.Style),
		LanguageMetadata: make(map[string]types.LanguageMetadata),
		Segments:         []types.SubtitleSegment{},
	}

	// 设置项目元数据
	project.Metadata = types.ProjectMetadata{
		ID:          project.ID,
		Name:        fileName,
		Description: fmt.Sprintf("Imported from SRT file: %s", fileName),
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     "1.0",
		SourceInfo: &types.SourceFileInfo{
			FileName: fileName,
			FileExt:  ext,
			FilePath: filePath,
			FileSize: int64(len(file)),
		},
		ExportConfigs: types.ExportConfigs{
			FCPXML: &types.FCPXMLExportConfig{
				ProjectName: fileName,
			},
			SRT: &types.SRTExportConfig{},
			ASS: &types.ASSExportConfig{},
			VTT: &types.VTTExportConfig{},
		},
	}
	// fcpxml 4k 60fps sdr default
	project.Metadata.ExportConfigs.FCPXML.Default4K60FPSSDR()
	project.Metadata.ExportConfigs.FCPXML.AutoFill()

	// 将文件内容转换为字符串并标准化换行符
	content := strings.ReplaceAll(string(file), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// 按空行分割字幕条目，支持多种分割方式
	var blocks []string
	if strings.Contains(content, "\n\n") {
		blocks = strings.Split(content, "\n\n")
	} else {
		// 如果没有双换行，尝试其他分割方式
		blocks = regexp.MustCompile(`\n\s*\n`).Split(content, -1)
	}

	// 收集所有内容用于语言检测
	var allContents []types.LanguageContent
	var allText string
	// 遍历每个字幕块
	for _, block := range blocks {
		// 跳过空块
		if strings.TrimSpace(block) == "" {
			continue
		}

		// 分割行并清理空行
		lines := strings.Split(strings.TrimSpace(block), "\n")
		var cleanLines []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				cleanLines = append(cleanLines, strings.TrimSpace(line))
			}
		}

		if len(cleanLines) < 3 {
			continue // 跳过无效块
		}

		// 查找时间码行（可能不在固定位置）
		var timecodeLineIndex = -1
		for i, line := range cleanLines {
			if strings.Contains(line, "-->") {
				timecodeLineIndex = i
				break
			}
		}

		if timecodeLineIndex == -1 {
			continue // 没找到时间码行
		}

		// 解析时间码行
		timecodes := regexp.MustCompile(`\s*-->\s*`).Split(cleanLines[timecodeLineIndex], 2)
		if len(timecodes) != 2 {
			continue // 跳过无效时间码
		}

		// 解析开始和结束时间
		startTime, err := parseSrtTimecode(timecodes[0])
		if err != nil {
			return types.SubtitleProject{}, fmt.Errorf("failed to parse start time: %v", err)
		}

		endTime, err := parseSrtTimecode(timecodes[1])
		if err != nil {
			return types.SubtitleProject{}, fmt.Errorf("failed to parse end time: %v", err)
		}

		// 收集文本行（时间码行之后的所有行）
		var textLines []string
		for i := timecodeLineIndex + 1; i < len(cleanLines); i++ {
			textLines = append(textLines, cleanLines[i])
		}

		if len(textLines) == 0 {
			continue // 跳过没有文本内容的块
		}

		// 创建语言内容
		// 将多行文本合并为一个内容，用换行符连接
		combinedText := strings.Join(textLines, "\n")
		content := types.LanguageContent{
			Text:  combinedText,
			Style: &types.Style{}, // srt无样式
		}

		allContents = append(allContents, content)
		allText = allText + content.Text + "\n"

		// 创建字幕片段
		segment := types.SubtitleSegment{
			ID:                uuid.New().String(),
			StartTime:         startTime,
			EndTime:           endTime,
			Speaker:           "", // srt file has no speaker
			Languages:         make(map[string]types.LanguageContent),
			GuidelineStandard: make(map[string]types.GuideLineStandard),
		}

		project.Segments = append(project.Segments, segment)
	}

	// 检测语言 - 使用相对距离逻辑
	langInt, langCode := s.detector.DetectLanguageInt(allText)

	// 设置语言元数据
	project.LanguageMetadata[langCode] = types.LanguageMetadata{
		DetectedLang: int(langInt),
		LanguageName: langCode,
		Quality:      "imported",
		SyncStatus:   "synced",
	}

	// 将内容分配到对应的片段
	for i, content := range allContents {
		if i < len(project.Segments) {
			project.Segments[i].Languages[langCode] = content
		}
	}

	return project, nil
}

// parseSrtTimecode 将 SRT 时间码字符串转换为 Timecode 结构
// 格式：00:00:00,000 -> 时:分:秒,毫秒
func parseSrtTimecode(tc string) (types.Timecode, error) {
	// 移除所有空白字符
	tc = strings.TrimSpace(tc)

	// 分离时间和毫秒
	parts := strings.Split(tc, ",")
	if len(parts) != 2 {
		return types.Timecode{}, fmt.Errorf("invalid timecode format: %s", tc)
	}

	// 解析时、分、秒
	timeParts := strings.Split(parts[0], ":")
	if len(timeParts) != 3 {
		return types.Timecode{}, fmt.Errorf("invalid time format: %s", parts[0])
	}

	// 转换为数字
	hours, err := strconv.Atoi(timeParts[0])
	if err != nil {
		return types.Timecode{}, fmt.Errorf("invalid hours: %v", err)
	}

	minutes, err := strconv.Atoi(timeParts[1])
	if err != nil {
		return types.Timecode{}, fmt.Errorf("invalid minutes: %v", err)
	}

	seconds, err := strconv.Atoi(timeParts[2])
	if err != nil {
		return types.Timecode{}, fmt.Errorf("invalid seconds: %v", err)
	}

	milliseconds, err := strconv.Atoi(parts[1])
	if err != nil {
		return types.Timecode{}, fmt.Errorf("invalid milliseconds: %v", err)
	}

	// 计算总时长
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds)*time.Millisecond

	return types.Timecode{
		Time: duration,
	}, nil
}

func (s *FormatConverterImpl) toSrt(project *types.SubtitleProject, languageCode string) ([]byte, error) {
	var buffer bytes.Buffer

	// 检查语言是否存在
	if _, exists := project.LanguageMetadata[languageCode]; !exists {
		return nil, fmt.Errorf("language '%s' not found in project", languageCode)
	}

	// 遍历所有字幕片段
	for i, segment := range project.Segments {
		// 检查该片段是否包含指定语言的内容
		if !segment.HasLanguage(languageCode) {
			continue // 跳过没有该语言内容的片段
		}

		// 写入序号
		buffer.WriteString(fmt.Sprintf("%d\n", i+1))

		// 写入时间码
		start := segment.StartTime.ToSRTFormat()
		end := segment.EndTime.ToSRTFormat()
		buffer.WriteString(fmt.Sprintf("%s --> %s\n", start, end))

		// 写入文本内容
		text := segment.GetText(languageCode)
		buffer.WriteString(text + "\n")

		// 添加空行分隔符
		buffer.WriteString("\n")
	}

	return buffer.Bytes(), nil
}

func (s *FormatConverterImpl) toFcpxml(project *types.SubtitleProject, languageCode string) ([]byte, error) {
	// 获取FCPXML配置
	fcpXMLConfig := project.Metadata.ExportConfigs.FCPXML
	if fcpXMLConfig == nil {
		return nil, fmt.Errorf("FCPXML export config is not set")
	}

	// 验证配置
	if err := fcpXMLConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid FCPXML config: %w", err)
	}

	// 自动填充缺失的字段
	fcpXMLConfig.AutoFill()

	// 动态生成Format名称
	formatName := generateFormatName(fcpXMLConfig.Width, fcpXMLConfig.Height, fcpXMLConfig.FrameRate)

	// 构建FCPXML结构
	fcpxml := &types.FCPXML{
		Version: fcpXMLConfig.Version,
		Resources: &types.FCPXMLResources{
			Formats: []types.FCPXMLFormat{
				{
					ID:            "r1",
					Name:          formatName,
					FrameDuration: fcpXMLConfig.FrameDuration,
					Width:         fcpXMLConfig.Width,
					Height:        fcpXMLConfig.Height,
					ColorSpace:    fcpXMLConfig.ColorSpace,
				},
			},
			// 添加Effects资源
			Effects: []types.FCPXMLEffect{
				{
					ID:   "r2",
					Name: "Basic Title",
					UID:  ".../Titles.localized/Bumper:Opener.localized/Basic Title.localized/Basic Title.moti",
				},
			},
		},
		Library: &types.FCPXMLLibrary{
			Location: fmt.Sprintf("file:///root/Movies/%s.fcpbundle", fcpXMLConfig.LibraryName),
			Events: []types.FCPXMLEvent{
				{
					Name: fcpXMLConfig.EventName,
					UID:  uuid.NewString(),
					Projects: []types.FCPXMLProject{
						{
							Name:     fcpXMLConfig.ProjectName,
							UID:      uuid.NewString(),
							ModDate:  time.Now().Format("2006-01-02 15:04:05 -0700"),
							Sequence: createSequence(project, languageCode, fcpXMLConfig),
						},
					},
				},
			},
		},
	}

	// 生成XML
	xmlData, err := xml.MarshalIndent(fcpxml, "", "  ")
	if err != nil {
		return nil, err
	}

	// 添加XML声明和DOCTYPE
	result := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE fcpxml>

`)
	result = append(result, xmlData...)

	return result, nil
}

// createSequence 创建FCPXML序列
func createSequence(project *types.SubtitleProject, languageCode string, fcpXMLConfig *types.FCPXMLExportConfig) *types.FCPXMLSequence {
	// 计算总时长
	totalDuration := project.GetTotalDuration()
	if totalDuration == 0 && len(project.Segments) > 0 {
		lastSegment := project.Segments[len(project.Segments)-1]
		totalDuration = lastSegment.EndTime.Time
	}

	// 修改时间格式化函数 - 使用秒数格式
	formatTimecode := func(duration time.Duration) string {
		if duration == 0 {
			return "0s"
		}
		seconds := duration.Seconds()
		// 如果是整数秒，直接返回
		if seconds == float64(int(seconds)) {
			return fmt.Sprintf("%.0fs", seconds)
		}
		// 对于非整数秒，使用帧数格式
		totalFrames := int64(seconds * fcpXMLConfig.FrameRate)
		frameRate := int64(fcpXMLConfig.FrameRate)
		return fmt.Sprintf("%d/%ds", totalFrames*frameRate, frameRate)
	}

	// 创建主轨道项目
	var spineItems []types.FCPXMLSpineItem

	// 如果有字幕片段，创建一个包含所有字幕的gap
	if len(project.Segments) > 0 {
		// 创建字幕标题列表
		var titles []types.FCPXMLTitle

		for i, segment := range project.Segments {
			// 检查是否有指定语言的内容
			content, hasLang := segment.Languages[languageCode]
			if !hasLang || content.Text == "" {
				continue
			}

			// 直接使用字幕开始时间作为偏移量（已经包含了1小时的项目开始时间）
			offsetFromStart := types.NewTimecode(fcpXMLConfig.StartTimecode.Time+segment.StartTime.Time, fcpXMLConfig.FrameRate)

			// 计算持续时间
			duration := types.NewTimecode(segment.EndTime.Time-segment.StartTime.Time, fcpXMLConfig.FrameRate)

			// 创建标题元素
			title := types.FCPXMLTitle{
				Name:     fmt.Sprintf("%s", content.Text),
				Lane:     fcpXMLConfig.DefaultLane,
				Offset:   offsetFromStart.ToFCPXMLFormat(fcpXMLConfig.FrameRate), // 直接使用 Timecode 的格式化方法
				Duration: duration.ToFCPXMLFormat(fcpXMLConfig.FrameRate),        // 使用 Timecode 的格式化方法
				Start:    fcpXMLConfig.StartTimecode.ToSecondsString(),           // 默认为3600秒（1小时）
				Ref:      "r2",
				Params: []types.FCPXMLParam{
					{
						Name:  "Position",
						Key:   "9999/999166631/999166633/1/100/101",
						Value: "0 -450",
					},
					{
						Name:  "Alignment",
						Key:   "9999/999166631/999166633/2/354/999169573/401",
						Value: "1 (Center)",
					},
					{
						Name:  "Flatten",
						Key:   "9999/999166631/999166633/2/351",
						Value: "1",
					},
				},
				Text: &types.FCPXMLText{
					TextStyle: []types.FCPXMLTextStyle{
						{
							Ref:     "ts1",
							Content: content.Text,
						},
					},
				},
			}

			// 只在第一个title中定义TextStyleDef
			if i == 0 {
				title.TextStyleDef = []types.FCPXMLTextStyleDef{
					{
						ID: "ts1",
						TextStyle: &types.FCPXMLTextStyleAttr{
							Font:         "PingFang SC",
							FontSize:     "52",
							FontFace:     "Semibold",
							FontColor:    "0.999993 1 1 1",
							Bold:         "1",
							ShadowColor:  "0 0 0 0.75",
							ShadowOffset: "5 315",
							Alignment:    "center",
						},
					},
				}
			}

			titles = append(titles, title)
		}

		// 创建包含所有字幕的gap
		gap := types.FCPXMLGap{
			Name:     "Gap", // 保持英文
			Offset:   formatTimecode(0),
			Duration: formatTimecode(totalDuration),
			Start:    formatTimecode(time.Hour), // 修改为3600s格式
			Titles:   titles,
		}
		spineItems = append(spineItems, gap)
	} else {
		// 如果没有字幕片段，创建一个空的gap
		gap := types.FCPXMLGap{
			Name:     "Gap",
			Offset:   formatTimecode(0),
			Duration: formatTimecode(10 * time.Second), // 默认10秒
			Start:    formatTimecode(time.Hour),
			Titles:   []types.FCPXMLTitle{},
		}
		spineItems = append(spineItems, gap)
		totalDuration = 10 * time.Second
	}

	// 创建序列
	sequence := &types.FCPXMLSequence{
		Duration:    formatTimecode(totalDuration),
		Format:      "r1",  // 引用format资源
		TCStart:     "0s",  // 修改为秒数格式
		TCFormat:    "NDF", // Non-Drop Frame
		AudioLayout: "stereo",
		AudioRate:   "48k",
		Spine: &types.FCPXMLSpine{
			Items: spineItems,
		},
	}

	return sequence
}

func separateFilePath(path string) (fileName, ext string) {
	base := filepath.Base(path)
	ext = filepath.Ext(base)
	return strings.TrimSuffix(base, ext), strings.TrimPrefix(ext, ".")
}

// 添加这个辅助函数
func generateFormatName(width, height int, frameRate float64) string {
	// 确定分辨率标识
	var resolutionName string
	switch {
	case width == 1920 && height == 1080:
		resolutionName = "1080p"
	case width == 3840 && height == 2160:
		resolutionName = "4K"
	case width == 1280 && height == 720:
		resolutionName = "720p"
	case width == 2560 && height == 1440:
		resolutionName = "1440p"
	default:
		resolutionName = fmt.Sprintf("%dx%d", width, height)
	}

	// 确定帧率标识
	var frameRateName string
	switch frameRate {
	case 23.976:
		frameRateName = "2398"
	case 24.0:
		frameRateName = "24"
	case 25.0:
		frameRateName = "25"
	case 29.97:
		frameRateName = "2997"
	case 30.0:
		frameRateName = "30"
	case 50.0:
		frameRateName = "50"
	case 59.94:
		frameRateName = "5994"
	case 60.0:
		frameRateName = "60"
	default:
		frameRateName = fmt.Sprintf("%.0f", frameRate)
	}

	return fmt.Sprintf("%s %s", resolutionName, frameRateName)
}

// 添加 VTT 导出功能
func (s *FormatConverterImpl) toVtt(project *types.SubtitleProject, languageCode string) ([]byte, error) {
	var buffer bytes.Buffer

	// VTT 文件头
	buffer.WriteString("WEBVTT\n\n")

	// 检查语言是否存在
	if _, exists := project.LanguageMetadata[languageCode]; !exists {
		return nil, fmt.Errorf("language '%s' not found in project", languageCode)
	}

	// 遍历所有字幕片段
	for _, segment := range project.Segments {
		// 检查该片段是否包含指定语言的内容
		if !segment.HasLanguage(languageCode) {
			continue // 跳过没有该语言内容的片段
		}

		// 写入时间码
		start := segment.StartTime.ToVTTFormat()
		end := segment.EndTime.ToVTTFormat()
		buffer.WriteString(fmt.Sprintf("%s --> %s\n", start, end))

		// 写入文本内容
		text := segment.GetText(languageCode)
		buffer.WriteString(text + "\n")

		// 添加空行分隔符
		buffer.WriteString("\n")
	}

	return buffer.Bytes(), nil
}
