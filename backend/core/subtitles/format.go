package subtitles

import (
	"bytes"
	"dreamcreator/backend/types"
	"encoding/xml"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
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

func (fc *FormatConverterImpl) FromVTT(filePath string, file []byte) (types.SubtitleProject, error) {
	return fc.fromVtt(filePath, file)
}

func (fc *FormatConverterImpl) FromASS(filePath string, file []byte) (types.SubtitleProject, error) {
	return fc.fromAss(filePath, file)
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

func (fc *FormatConverterImpl) ToASS(project *types.SubtitleProject, langCode string) ([]byte, error) {
	return fc.toAss(project, langCode)
}

func (fc *FormatConverterImpl) ToITT(project *types.SubtitleProject, langCode string) ([]byte, error) {
	return fc.toItt(project, langCode)
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

	// 解析帧率（优先 ttp:frameRate），并考虑 ttp:frameRateMultiplier
	frameRateNominal := 25.0 // 名义帧率
	// 1) 先尝试从解出来的结构体读取（若命名空间映射导致未命中，则为空）
	if itt.TtpFrameRate != "" {
		if rate, err := strconv.ParseFloat(itt.TtpFrameRate, 64); err == nil {
			frameRateNominal = rate
		}
	}
	// 2) 若仍为空，使用正则直接从原始 XML 中提取，增强兼容性
	if itt.TtpFrameRate == "" {
		if m := regexp.MustCompile(`ttp:frameRate\s*=\s*"([0-9.]+)"`).FindSubmatch(file); len(m) == 2 {
			if rate, err := strconv.ParseFloat(string(m[1]), 64); err == nil {
				frameRateNominal = rate
			}
		} else if m2 := regexp.MustCompile(`frameRate\s*=\s*"([0-9.]+)"`).FindSubmatch(file); len(m2) == 2 {
			// 兜底：无前缀的 frameRate
			if rate, err := strconv.ParseFloat(string(m2[1]), 64); err == nil {
				frameRateNominal = rate
			}
		}
	}
	// multiplier: "num den"
	mulNum, mulDen := 1, 1
	if strings.TrimSpace(itt.TtpFrameRateMultiplier) != "" {
		parts := strings.Fields(strings.TrimSpace(itt.TtpFrameRateMultiplier))
		if len(parts) == 2 {
			if v, err := strconv.Atoi(parts[0]); err == nil && v > 0 {
				mulNum = v
			}
			if v, err := strconv.Atoi(parts[1]); err == nil && v > 0 {
				mulDen = v
			}
		}
	} else {
		// 正则提取 multiplier
		if m := regexp.MustCompile(`ttp:frameRateMultiplier\s*=\s*"([0-9]+)\s+([0-9]+)"`).FindSubmatch(file); len(m) == 3 {
			if v, err := strconv.Atoi(string(m[1])); err == nil && v > 0 {
				mulNum = v
			}
			if v, err := strconv.Atoi(string(m[2])); err == nil && v > 0 {
				mulDen = v
			}
		}
	}
	// 额外提取 dropMode 与 timeBase（用于高保真回写）
	dropMode := itt.TtpDropMode
	if strings.TrimSpace(dropMode) == "" {
		if m := regexp.MustCompile(`ttp:dropMode\s*=\s*"([^"]+)"`).FindSubmatch(file); len(m) == 2 {
			dropMode = string(m[1])
		}
	}
	timeBase := itt.TtpTimeBase
	if strings.TrimSpace(timeBase) == "" {
		if m := regexp.MustCompile(`ttp:timeBase\s*=\s*"([^"]+)"`).FindSubmatch(file); len(m) == 2 {
			timeBase = string(m[1])
		}
	}
	// 有效帧率（用于将帧换算为时间）：frameRateNominal * num/den
	frameRate := frameRateNominal * (float64(mulNum) / float64(mulDen))

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
	// 记录源帧率，便于导出时回退
	if project.Metadata.SourceInfo != nil {
		project.Metadata.SourceInfo.OriginalFPS = frameRate
	}
	// 预填 ITT 导出配置（语言在检测后设置）
	project.Metadata.ExportConfigs.ITT = &types.ITTExportConfig{FrameRate: frameRateNominal, Multiplier: types.IttMultiplier{Num: mulNum, Den: mulDen}}

	// 保存原始 ITT 头与布局信息用于高保真回写
	project.Metadata.SourceITT = &types.ITTSourceInfo{
		XmlLang:    itt.XmlLang,
		TimeBase:   timeBase,
		FrameRate:  frameRateNominal,
		Multiplier: types.IttMultiplier{Num: mulNum, Den: mulDen},
		DropMode:   dropMode,
	}
	// 复制 regions 布局
	for _, r := range itt.Head.Layout.Regions {
		project.Metadata.SourceITT.Regions = append(project.Metadata.SourceITT.Regions, types.ITTRegion{
			ID:           r.XMLID,
			Origin:       r.TtsOrigin,
			Extent:       r.TtsExtent,
			DisplayAlign: r.TtsDisplayAlign,
			WritingMode:  r.TtsWritingMode,
		})
	}
	// 复制 Body 默认属性
	project.Metadata.SourceITT.BodyDefaults = types.ITTBodyDefaults{
		Region: itt.Body.Region,
		Style:  itt.Body.Style,
		Color:  itt.Body.TtsColor,
	}

	// 处理全局样式（兼容 tts:* 属性与 xml:id）
	for _, style := range itt.Head.Styling.Styles {
		id := style.XMLID
		if strings.TrimSpace(id) == "" {
			id = uuid.NewString()
		}
		// 尝试从 tts:fontSize 提取数字
		var fsz float64
		if s := regexp.MustCompile(`[0-9.]+`).FindString(style.TtsFontSize); s != "" {
			if v, err := strconv.ParseFloat(s, 64); err == nil {
				fsz = v
			}
		}
		project.GlobalStyles[id] = types.Style{
			FontName:        style.TtsFontFamily,
			FontSize:        fsz,
			Color:           style.TtsColor,
			Alignment:       style.TtsTextAlign,
			BackgroundColor: style.TtsBackground,
		}
	}

	// 处理字幕条目
	var allContents []types.LanguageContent
	var allText string
	for _, body := range itt.Body.Div {
		for _, p := range body.P {
			// 解析时间码
			startTime, err := parseITTTimecode(p.Begin, frameRate)
			if err != nil {
				return types.SubtitleProject{}, fmt.Errorf("failed to parse begin timecode: %v", err)
			}
			endTime, err := parseITTTimecode(p.End, frameRate)
			if err != nil {
				return types.SubtitleProject{}, fmt.Errorf("failed to parse end timecode: %v", err)
			}

			// 创建语言内容
			// region 优先 p.region，否则 body 默认
			regionID := p.Region
			if strings.TrimSpace(regionID) == "" {
				regionID = itt.Body.Region
			}
			content := types.LanguageContent{Text: p.Content, RegionID: regionID}

			// 处理样式
			if p.Style != "" {
				if style, ok := project.GlobalStyles[p.Style]; ok {
					content.Style = &style
				}
				content.StyleID = p.Style
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
	// 完善 ITT 导出配置的语言
	if project.Metadata.ExportConfigs.ITT != nil {
		project.Metadata.ExportConfigs.ITT.Language = langCode
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
func parseITTTimecode(tc string, frameRate float64) (types.Timecode, error) {
	// ITT 时间码格式：HH:MM:SS:FF 或 HH:MM:SS.mmm
	var hours, minutes, seconds, frames int
	var milliseconds float64

	if strings.Contains(tc, ".") {
		// 处理毫秒格式
		_, err := fmt.Sscanf(tc, "%d:%d:%d.%f", &hours, &minutes, &seconds, &milliseconds)
		if err != nil {
			return types.Timecode{}, err
		}
		// 修复：将小数部分转换为毫秒
		milliseconds = milliseconds * 1000
	} else {
		// 处理帧格式
		_, err := fmt.Sscanf(tc, "%d:%d:%d:%d", &hours, &minutes, &seconds, &frames)
		if err != nil {
			return types.Timecode{}, err
		}
		// 将帧转换为毫秒，使用四舍五入提高精度
		if frameRate > 0 {
			milliseconds = math.Round(float64(frames) / frameRate * 1000)
		}
	}

	// 计算总时长
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(milliseconds)*time.Millisecond

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

		// 过滤掉开始时间与结束时间相同的字幕
		if startTime.Time == endTime.Time {
			continue
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

	// 遍历所有字幕片段（连续编号）
	idx := 1
	for _, segment := range project.Segments {
		if !segment.HasLanguage(languageCode) {
			continue
		}
		buffer.WriteString(fmt.Sprintf("%d\n", idx))
		idx++
		start := segment.StartTime.ToSRTFormat()
		end := segment.EndTime.ToSRTFormat()
		buffer.WriteString(fmt.Sprintf("%s --> %s\n", start, end))
		text := segment.GetText(languageCode)
		buffer.WriteString(text + "\n\n")
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
				Name:     content.Text,
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
	cueIndex := 1
	for _, segment := range project.Segments {
		// 检查该片段是否包含指定语言的内容
		if !segment.HasLanguage(languageCode) {
			continue // 跳过没有该语言内容的片段
		}

		// 写入序号（Cue Identifier）
		buffer.WriteString(fmt.Sprintf("%d\n", cueIndex))
		cueIndex++

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

// fromAss 解析 ASS/SSA
func (s *FormatConverterImpl) fromAss(filePath string, file []byte) (types.SubtitleProject, error) {
	now := time.Now().Unix()
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
	project.Metadata = types.ProjectMetadata{
		ID:          project.ID,
		Name:        fileName,
		Description: fmt.Sprintf("Imported from %s file: %s", strings.ToUpper(ext), fileName),
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
			FCPXML: &types.FCPXMLExportConfig{ProjectName: fileName},
			SRT:    &types.SRTExportConfig{},
			ASS:    &types.ASSExportConfig{Title: fileName},
			VTT:    &types.VTTExportConfig{},
		},
	}
	project.Metadata.ExportConfigs.FCPXML.Default4K60FPSSDR()
	project.Metadata.ExportConfigs.FCPXML.AutoFill()

	text := string(file)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")

	// parse [V4+ Styles] and [Events]
	inEvents := false
	inStyles := false
	var allContents []types.LanguageContent
	var allText string
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "[") {
			// section header
			sec := strings.ToLower(strings.TrimSpace(t))
			inEvents = (sec == "[events]")
			inStyles = (sec == "[v4+ styles]" || sec == "[v4 styles]")
			continue
		}
		if inStyles {
			if strings.HasPrefix(t, "Style:") {
				if name, style := parseAssStyleLine(strings.TrimSpace(strings.TrimPrefix(t, "Style:"))); name != "" {
					project.GlobalStyles[name] = style
				}
			}
			continue
		}
		if !inEvents {
			continue
		}
		if strings.HasPrefix(t, "Dialogue:") {
			payload := strings.TrimSpace(strings.TrimPrefix(t, "Dialogue:"))
			parts := splitAssDialogue(payload, 9) // split into 10 fields total (9 + text)
			if len(parts) < 10 {
				continue
			}
			start, err := parseAssTimecode(strings.TrimSpace(parts[1]))
			if err != nil {
				continue
			}
			end, err := parseAssTimecode(strings.TrimSpace(parts[2]))
			if err != nil {
				continue
			}
			if start.Time == end.Time {
				continue
			}
			txt := strings.Join(parts[9:], ",") // join remaining as text
			// normalize line breaks and optionally strip simple override tags
			txt = strings.ReplaceAll(txt, "\\N", "\n")
			// create segment shell
			seg := types.SubtitleSegment{
				ID:                uuid.New().String(),
				StartTime:         start,
				EndTime:           end,
				Languages:         make(map[string]types.LanguageContent),
				GuidelineStandard: make(map[string]types.GuideLineStandard),
			}
			project.Segments = append(project.Segments, seg)
			styleName := strings.TrimSpace(parts[3])
			lc := types.LanguageContent{Text: txt, StyleID: styleName}
			allContents = append(allContents, lc)
			if allText != "" {
				allText += "\n"
			}
			allText += txt
		}
	}

	// detect language and assign
	langInt, langCode := s.detector.DetectLanguageInt(allText)
	if langCode == "" {
		langCode = "Chinese"
	}
	project.LanguageMetadata[langCode] = types.LanguageMetadata{DetectedLang: int(langInt), LanguageName: langCode, Quality: "imported", SyncStatus: "synced"}
	for i, c := range allContents {
		if i < len(project.Segments) {
			project.Segments[i].Languages[langCode] = c
		}
	}

	return project, nil
}

// parseAssStyleLine parses a single "Style:" line into style name and Style
func parseAssStyleLine(payload string) (string, types.Style) {
	// Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour,
	// Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle,
	// Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
	fields := splitAssDialogue(payload, 22)
	if len(fields) < 23 {
		return "", types.Style{}
	}
	name := strings.TrimSpace(fields[0])
	font := strings.TrimSpace(fields[1])
	sizeStr := strings.TrimSpace(fields[2])
	var size float64
	if v, err := strconv.ParseFloat(sizeStr, 64); err == nil {
		size = v
	}
	// primary colour like &HAA BB GG RR (hex); convert to #RRGGBB best-effort
	color := assColorToHex(strings.TrimSpace(fields[3]))
	align := ""
	// Alignment numeric: 1..9 per ASS; map center=2/5/8, left/right to approximate
	// Keep as raw number string to avoid wrong mapping; leave empty
	bold := strings.TrimSpace(fields[7]) == "-1"
	italic := strings.TrimSpace(fields[8]) == "-1"
	style := types.Style{
		FontName:  font,
		FontSize:  size,
		Color:     color,
		Bold:      bold,
		Italic:    italic,
		Alignment: align,
	}
	return name, style
}

func assColorToHex(s string) string {
	// ASS color: &HAABBGGRR or &HBBGGRR
	s = strings.TrimSpace(s)
	if strings.HasPrefix(strings.ToUpper(s), "&H") {
		hex := strings.TrimPrefix(strings.ToUpper(s), "&H")
		if len(hex) == 8 {
			// AABBGGRR -> RRGGBB
			rr := hex[6:8]
			gg := hex[4:6]
			bb := hex[2:4]
			return "#" + rr + gg + bb
		}
		if len(hex) == 6 {
			rr := hex[4:6]
			gg := hex[2:4]
			bb := hex[0:2]
			return "#" + rr + gg + bb
		}
	}
	return ""
}

// splitAssDialogue splits by comma but only the first n commas, keeping the rest joined
func splitAssDialogue(s string, commas int) []string {
	out := make([]string, 0, commas+1)
	cur := s
	for i := 0; i < commas; i++ {
		idx := strings.Index(cur, ",")
		if idx < 0 {
			break
		}
		out = append(out, cur[:idx])
		cur = cur[idx+1:]
	}
	out = append(out, cur)
	return out
}

// parseAssTimecode H:MM:SS.CS -> Timecode
func parseAssTimecode(tc string) (types.Timecode, error) {
	tc = strings.TrimSpace(tc)
	parts := strings.Split(tc, ":")
	if len(parts) != 3 {
		return types.Timecode{}, fmt.Errorf("invalid ASS timecode: %s", tc)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return types.Timecode{}, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return types.Timecode{}, err
	}
	secParts := strings.Split(parts[2], ".")
	if len(secParts) != 2 {
		return types.Timecode{}, fmt.Errorf("invalid ASS time sec: %s", parts[2])
	}
	s, err := strconv.Atoi(secParts[0])
	if err != nil {
		return types.Timecode{}, err
	}
	csStr := secParts[1]
	// centiseconds -> milliseconds
	if len(csStr) > 2 {
		csStr = csStr[:2]
	}
	for len(csStr) < 2 {
		csStr += "0"
	}
	cs, err := strconv.Atoi(csStr)
	if err != nil {
		return types.Timecode{}, err
	}
	d := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second + time.Duration(cs)*10*time.Millisecond
	return types.Timecode{Time: d}, nil
}

// toAss 导出为 ASS（最小可用）
func (s *FormatConverterImpl) toAss(project *types.SubtitleProject, languageCode string) ([]byte, error) {
	var buf bytes.Buffer
	// basic header
	title := project.ProjectName
	if project.Metadata.ExportConfigs.ASS != nil && project.Metadata.ExportConfigs.ASS.Title != "" {
		title = project.Metadata.ExportConfigs.ASS.Title
	}
	// resolution
	resX, resY := 1920, 1080
	if project.Metadata.ExportConfigs.ASS != nil {
		if project.Metadata.ExportConfigs.ASS.PlayResX > 0 {
			resX = project.Metadata.ExportConfigs.ASS.PlayResX
		}
		if project.Metadata.ExportConfigs.ASS.PlayResY > 0 {
			resY = project.Metadata.ExportConfigs.ASS.PlayResY
		}
	} else if project.Metadata.ExportConfigs.FCPXML != nil {
		if project.Metadata.ExportConfigs.FCPXML.Width > 0 {
			resX = project.Metadata.ExportConfigs.FCPXML.Width
		}
		if project.Metadata.ExportConfigs.FCPXML.Height > 0 {
			resY = project.Metadata.ExportConfigs.FCPXML.Height
		}
	}

	buf.WriteString("[Script Info]\n")
	buf.WriteString("; Script generated by dreamcreator\n")
	buf.WriteString("ScriptType: v4.00+\n")
	buf.WriteString(fmt.Sprintf("Title: %s\n", title))
	buf.WriteString("WrapStyle: 0\nScaledBorderAndShadow: yes\n")
	buf.WriteString(fmt.Sprintf("PlayResX: %d\nPlayResY: %d\n\n", resX, resY))

	buf.WriteString("[V4+ Styles]\n")
	buf.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	// Write styles from GlobalStyles; include Default if none
	wroteAny := false
	hexToAss := func(hex string) string {
		// #RRGGBB -> &H00BBGGRR
		if len(hex) == 7 && hex[0] == '#' {
			rr := strings.ToUpper(hex[1:3])
			gg := strings.ToUpper(hex[3:5])
			bb := strings.ToUpper(hex[5:7])
			return "&H00" + bb + gg + rr
		}
		return "&H00FFFFFF"
	}
	for name, st := range project.GlobalStyles {
		if strings.TrimSpace(name) == "" {
			continue
		}
		font := st.FontName
		if font == "" {
			font = "Arial"
		}
		size := st.FontSize
		if size <= 0 {
			size = 48
		}
		prim := hexToAss(st.Color)
		bold := -0
		if st.Bold {
			bold = -1
		}
		italic := -0
		if st.Italic {
			italic = -1
		}
		// Alignment: map center to 2
		align := 2
		if strings.Contains(strings.ToLower(st.Alignment), "left") {
			align = 1
		}
		if strings.Contains(strings.ToLower(st.Alignment), "right") {
			align = 3
		}
		line := fmt.Sprintf("Style: %s,%s,%.0f,%s,&H000000FF,&H00000000,&H64000000,%d,%d,0,0,100,100,0,0,1,2,0,%d,10,10,10,1\n",
			name, font, size, prim, bold, italic, align)
		buf.WriteString(line)
		wroteAny = true
	}
	if !wroteAny {
		buf.WriteString("Style: Default,Arial,48,&H00FFFFFF,&H000000FF,&H00000000,&H64000000,-1,0,0,0,100,100,0,0,1,2,0,2,10,10,10,1\n")
	}
	buf.WriteString("\n")

	buf.WriteString("[Events]\n")
	buf.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")

	// ensure language exists
	if _, exists := project.LanguageMetadata[languageCode]; !exists {
		return nil, fmt.Errorf("language '%s' not found in project", languageCode)
	}
	for _, seg := range project.Segments {
		if !seg.HasLanguage(languageCode) {
			continue
		}
		start := assTime(seg.StartTime.Time)
		end := assTime(seg.EndTime.Time)
		lc := seg.Languages[languageCode]
		styleName := strings.TrimSpace(lc.StyleID)
		if styleName == "" || project.GlobalStyles[styleName].FontName == "" {
			styleName = "Default"
		}
		text := strings.ReplaceAll(lc.Text, "\n", "\\N")
		buf.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,%s,,0,0,0,,%s\n", start, end, styleName, text))
	}
	return buf.Bytes(), nil
}

func assTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSeconds := int(d / time.Second)
	cs := int((d % time.Second) / (10 * time.Millisecond))
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

// toItt 导出为 iTT (TTML subset)
func (s *FormatConverterImpl) toItt(project *types.SubtitleProject, languageCode string) ([]byte, error) {
	// 检查语言
	if _, ok := project.LanguageMetadata[languageCode]; !ok {
		return nil, fmt.Errorf("language '%s' not found in project", languageCode)
	}

	// 构建 ITT 文档
	var doc types.IttDocument
	// Use TTML ns (as in sample)
	doc.Xmlns = "http://www.w3.org/ns/ttml"
	doc.XmlnsTtp = "http://www.w3.org/ns/ttml#parameter"
	doc.XmlnsTts = "http://www.w3.org/ns/ttml#styling"
	doc.XmlnsTtm = "http://www.w3.org/ns/ttml#metadata"
	// SMPTE time base with explicit frame rate (preserve when possible)
	// Prefer export config; fallback to imported source ITT; then defaults
	fpsNominal := 0.0
	mulNum, mulDen := 0, 0
	timeBase := ""
	dropMode := ""
	if project.Metadata.ExportConfigs.ITT != nil {
		if project.Metadata.ExportConfigs.ITT.FrameRate > 0 {
			fpsNominal = project.Metadata.ExportConfigs.ITT.FrameRate
		}
		if project.Metadata.ExportConfigs.ITT.Multiplier.Num > 0 && project.Metadata.ExportConfigs.ITT.Multiplier.Den > 0 {
			mulNum = project.Metadata.ExportConfigs.ITT.Multiplier.Num
			mulDen = project.Metadata.ExportConfigs.ITT.Multiplier.Den
		}
		if strings.TrimSpace(project.Metadata.ExportConfigs.ITT.TimeBase) != "" {
			timeBase = project.Metadata.ExportConfigs.ITT.TimeBase
		}
		if strings.TrimSpace(project.Metadata.ExportConfigs.ITT.DropMode) != "" {
			dropMode = project.Metadata.ExportConfigs.ITT.DropMode
		}
	}
	if fpsNominal <= 0 && project.Metadata.SourceITT != nil && project.Metadata.SourceITT.FrameRate > 0 {
		fpsNominal = project.Metadata.SourceITT.FrameRate
	}
	if (mulNum == 0 || mulDen == 0) && project.Metadata.SourceITT != nil && project.Metadata.SourceITT.Multiplier.Num > 0 && project.Metadata.SourceITT.Multiplier.Den > 0 {
		mulNum = project.Metadata.SourceITT.Multiplier.Num
		mulDen = project.Metadata.SourceITT.Multiplier.Den
	}
	if strings.TrimSpace(timeBase) == "" {
		if project.Metadata.SourceITT != nil && strings.TrimSpace(project.Metadata.SourceITT.TimeBase) != "" {
			timeBase = project.Metadata.SourceITT.TimeBase
		} else {
			timeBase = "smpte"
		}
	}
	if strings.TrimSpace(dropMode) == "" {
		if project.Metadata.SourceITT != nil && strings.TrimSpace(project.Metadata.SourceITT.DropMode) != "" {
			dropMode = project.Metadata.SourceITT.DropMode
		} else {
			dropMode = "nonDrop"
		}
	}
	if fpsNominal <= 0 {
		// fallback chain to fcpxml / source original fps / default 60
		if project.Metadata.ExportConfigs.FCPXML != nil && project.Metadata.ExportConfigs.FCPXML.FrameRate > 0 {
			fpsNominal = project.Metadata.ExportConfigs.FCPXML.FrameRate
		} else if project.Metadata.SourceInfo != nil && project.Metadata.SourceInfo.OriginalFPS > 0 {
			// original fps 可能为有效帧率，这里仍作为名义帧率使用
			fpsNominal = project.Metadata.SourceInfo.OriginalFPS
		} else {
			fpsNominal = 60.0
		}
	}

	doc.TtpTimeBase = timeBase
	doc.TtpFrameRate = strconv.FormatFloat(fpsNominal, 'f', -1, 64)
	if mulNum > 0 && mulDen > 0 {
		doc.TtpFrameRateMultiplier = fmt.Sprintf("%d %d", mulNum, mulDen)
	} else {
		doc.TtpFrameRateMultiplier = "1 1"
	}
	doc.TtpDropMode = dropMode
	// Metadata
	title := project.ProjectName
	if project.Metadata.Name != "" {
		title = project.Metadata.Name
	}
	doc.Head.Metadata.Title = title
	doc.Head.Metadata.Tool = "dreamcreator"
	// language code (best-effort): ITT config > source xml:lang > arg languageCode
	langOut := languageCode
	if project.Metadata.ExportConfigs.ITT != nil && strings.TrimSpace(project.Metadata.ExportConfigs.ITT.Language) != "" {
		langOut = project.Metadata.ExportConfigs.ITT.Language
	} else if project.Metadata.SourceITT != nil && strings.TrimSpace(project.Metadata.SourceITT.XmlLang) != "" {
		langOut = project.Metadata.SourceITT.XmlLang
	}
	doc.XmlLang = langOut
	// Styling: 还原项目的全局样式，至少保留 normal
	// 将 GlobalStyles 转为 TTML style（匹配 IttDocument 中的匿名结构体定义，包括 fontStyle/fontWeight）
	styles := make([]struct {
		XMLID         string `xml:"xml:id,attr,omitempty"`
		TtsFontFamily string `xml:"fontFamily,attr,omitempty"`
		TtsFontSize   string `xml:"fontSize,attr,omitempty"`
		TtsTextAlign  string `xml:"textAlign,attr,omitempty"`
		TtsColor      string `xml:"color,attr,omitempty"`
		TtsBackground string `xml:"backgroundColor,attr,omitempty"`
		TtsFontStyle  string `xml:"fontStyle,attr,omitempty"`
		TtsFontWeight string `xml:"fontWeight,attr,omitempty"`
	}, 0, len(project.GlobalStyles)+1)
	// add styles from project
	for id, st := range project.GlobalStyles {
		styles = append(styles, struct {
			XMLID         string `xml:"xml:id,attr,omitempty"`
			TtsFontFamily string `xml:"fontFamily,attr,omitempty"`
			TtsFontSize   string `xml:"fontSize,attr,omitempty"`
			TtsTextAlign  string `xml:"textAlign,attr,omitempty"`
			TtsColor      string `xml:"color,attr,omitempty"`
			TtsBackground string `xml:"backgroundColor,attr,omitempty"`
			TtsFontStyle  string `xml:"fontStyle,attr,omitempty"`
			TtsFontWeight string `xml:"fontWeight,attr,omitempty"`
		}{XMLID: id, TtsFontFamily: st.FontName, TtsFontSize: fmt.Sprintf("%.0f%%", st.FontSize), TtsTextAlign: st.Alignment, TtsColor: st.Color, TtsBackground: st.BackgroundColor})
	}
	// ensure a fallback normal style exists
	hasNormal := false
	for _, st := range styles {
		if st.XMLID == "normal" {
			hasNormal = true
			break
		}
	}
	if !hasNormal {
		styles = append(styles, struct {
			XMLID         string `xml:"xml:id,attr,omitempty"`
			TtsFontFamily string `xml:"fontFamily,attr,omitempty"`
			TtsFontSize   string `xml:"fontSize,attr,omitempty"`
			TtsTextAlign  string `xml:"textAlign,attr,omitempty"`
			TtsColor      string `xml:"color,attr,omitempty"`
			TtsBackground string `xml:"backgroundColor,attr,omitempty"`
			TtsFontStyle  string `xml:"fontStyle,attr,omitempty"`
			TtsFontWeight string `xml:"fontWeight,attr,omitempty"`
		}{XMLID: "normal", TtsFontFamily: "sansSerif", TtsFontSize: "100%", TtsTextAlign: "center", TtsColor: "white"})
	}
	doc.Head.Styling.Styles = styles

	// Layout regions: 尽量用导入的 regions，否则提供一个 bottom
	if project.Metadata.SourceITT != nil && len(project.Metadata.SourceITT.Regions) > 0 {
		regs := make([]struct {
			XMLID           string `xml:"xml:id,attr"`
			TtsOrigin       string `xml:"origin,attr,omitempty"`
			TtsExtent       string `xml:"extent,attr,omitempty"`
			TtsDisplayAlign string `xml:"displayAlign,attr,omitempty"`
			TtsWritingMode  string `xml:"writingMode,attr,omitempty"`
		}, 0, len(project.Metadata.SourceITT.Regions))
		for _, r := range project.Metadata.SourceITT.Regions {
			regs = append(regs, struct {
				XMLID           string `xml:"xml:id,attr"`
				TtsOrigin       string `xml:"origin,attr,omitempty"`
				TtsExtent       string `xml:"extent,attr,omitempty"`
				TtsDisplayAlign string `xml:"displayAlign,attr,omitempty"`
				TtsWritingMode  string `xml:"writingMode,attr,omitempty"`
			}{XMLID: r.ID, TtsOrigin: r.Origin, TtsExtent: r.Extent, TtsDisplayAlign: r.DisplayAlign, TtsWritingMode: r.WritingMode})
		}
		doc.Head.Layout.Regions = regs
	} else {
		doc.Head.Layout.Regions = []struct {
			XMLID           string `xml:"xml:id,attr"`
			TtsOrigin       string `xml:"origin,attr,omitempty"`
			TtsExtent       string `xml:"extent,attr,omitempty"`
			TtsDisplayAlign string `xml:"displayAlign,attr,omitempty"`
			TtsWritingMode  string `xml:"writingMode,attr,omitempty"`
		}{
			{XMLID: "bottom", TtsOrigin: "0% 85%", TtsExtent: "100% 15%"},
		}
	}
	// body defaults
	if project.Metadata.SourceITT != nil {
		if v := strings.TrimSpace(project.Metadata.SourceITT.BodyDefaults.Style); v != "" {
			doc.Body.Style = v
		} else {
			doc.Body.Style = "normal"
		}
		if v := strings.TrimSpace(project.Metadata.SourceITT.BodyDefaults.Region); v != "" {
			doc.Body.Region = v
		} else {
			doc.Body.Region = "bottom"
		}
		if v := strings.TrimSpace(project.Metadata.SourceITT.BodyDefaults.Color); v != "" {
			doc.Body.TtsColor = v
		} else {
			doc.Body.TtsColor = "white"
		}
	} else {
		doc.Body.Style = "normal"
		doc.Body.Region = "bottom"
		doc.Body.TtsColor = "white"
	}

	// Build body with exact field tags matching types.IttDocument
	ps := []struct {
		Begin   string `xml:"begin,attr"`
		End     string `xml:"end,attr"`
		Region  string `xml:"region,attr,omitempty"`
		Style   string `xml:"style,attr,omitempty"`
		Speaker string `xml:"speaker,attr,omitempty"`
		Content string `xml:",chardata"`
	}{}
	// sort segments by start time
	segs := append([]types.SubtitleSegment{}, project.Segments...)
	sort.Slice(segs, func(i, j int) bool { return segs[i].StartTime.Time < segs[j].StartTime.Time })
	for _, seg := range segs {
		if !seg.HasLanguage(languageCode) {
			continue
		}
		// Convert newlines to spaces (basic)
		txt := strings.ReplaceAll(seg.GetText(languageCode), "\r\n", "\n")
		txt = strings.ReplaceAll(txt, "\r", "\n")
		txt = strings.ReplaceAll(txt, "\n", " ")
		p2 := struct {
			Begin   string `xml:"begin,attr"`
			End     string `xml:"end,attr"`
			Region  string `xml:"region,attr,omitempty"`
			Style   string `xml:"style,attr,omitempty"`
			Speaker string `xml:"speaker,attr,omitempty"`
			Content string `xml:",chardata"`
		}{
			Begin: ittSmpteTime(seg.StartTime.Time, fpsNominal),
			End:   ittSmpteTime(seg.EndTime.Time, fpsNominal),
			Region: func() string {
				c := seg.Languages[languageCode]
				if strings.TrimSpace(c.RegionID) != "" {
					return c.RegionID
				}
				if project.Metadata.SourceITT != nil && strings.TrimSpace(project.Metadata.SourceITT.BodyDefaults.Region) != "" {
					return project.Metadata.SourceITT.BodyDefaults.Region
				}
				return "bottom"
			}(),
			Style: func() string {
				c := seg.Languages[languageCode]
				if strings.TrimSpace(c.StyleID) != "" {
					return c.StyleID
				}
				if project.Metadata.SourceITT != nil && strings.TrimSpace(project.Metadata.SourceITT.BodyDefaults.Style) != "" {
					return project.Metadata.SourceITT.BodyDefaults.Style
				}
				return "normal"
			}(),
			Speaker: seg.Speaker,
			Content: txt,
		}
		ps = append(ps, p2)
	}
	doc.Body.Div = []struct {
		P []struct {
			Begin   string `xml:"begin,attr"`
			End     string `xml:"end,attr"`
			Region  string `xml:"region,attr,omitempty"`
			Style   string `xml:"style,attr,omitempty"`
			Speaker string `xml:"speaker,attr,omitempty"`
			Content string `xml:",chardata"`
		} `xml:"p"`
	}{
		{P: ps},
	}

	// Marshal
	b, err := xml.MarshalIndent(&doc, "", "  ")
	if err != nil {
		return nil, err
	}
	out := append([]byte(xml.Header), b...)
	return out, nil
}

func ittTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalMillis := d.Milliseconds()
	hours := totalMillis / 3600000
	minutes := (totalMillis % 3600000) / 60000
	seconds := (totalMillis % 60000) / 1000
	millis := totalMillis % 1000
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}

// ittSmpteTime converts duration to SMPTE time HH:MM:SS:FF using fps
func ittSmpteTime(d time.Duration, fps float64) string {
	if d < 0 {
		d = 0
	}
	totalSec := d.Seconds()
	h := int(totalSec) / 3600
	m := (int(totalSec) % 3600) / 60
	s := int(totalSec) % 60
	frac := totalSec - float64(int(totalSec))
	frames := int(math.Round(frac * fps))
	if frames >= int(fps) {
		frames = 0
		s++
		if s >= 60 {
			s = 0
			m++
			if m >= 60 {
				m = 0
				h++
			}
		}
	}
	// two-digit frames
	return fmt.Sprintf("%02d:%02d:%02d:%02d", h, m, s, frames)
}

// fromVtt 解析 WebVTT 文件为项目
func (s *FormatConverterImpl) fromVtt(filePath string, file []byte) (types.SubtitleProject, error) {
	now := time.Now().Unix()
	// 新项目
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

	// 元数据
	project.Metadata = types.ProjectMetadata{
		ID:          project.ID,
		Name:        fileName,
		Description: fmt.Sprintf("Imported from VTT file: %s", fileName),
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
			FCPXML: &types.FCPXMLExportConfig{ProjectName: fileName},
			SRT:    &types.SRTExportConfig{},
			ASS:    &types.ASSExportConfig{},
			VTT:    &types.VTTExportConfig{},
		},
	}
	project.Metadata.ExportConfigs.FCPXML.Default4K60FPSSDR()
	project.Metadata.ExportConfigs.FCPXML.AutoFill()

	// 清理内容：去除 BOM，标准化换行
	content := string(file)
	if strings.HasPrefix(content, "\uFEFF") {
		content = strings.TrimPrefix(content, "\uFEFF")
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	// 按空行分块
	blocks := regexp.MustCompile(`\n\s*\n`).Split(strings.TrimSpace(content), -1)

	var allContents []types.LanguageContent
	var allText string

	for _, block := range blocks {
		b := strings.TrimSpace(block)
		if b == "" {
			continue
		}
		// 跳过 WEBVTT/STYLE/REGION/NOTE 等非 cue 区块
		upper := strings.ToUpper(strings.SplitN(b, "\n", 2)[0])
		if strings.HasPrefix(upper, "WEBVTT") || strings.HasPrefix(upper, "STYLE") || strings.HasPrefix(upper, "REGION") || strings.HasPrefix(upper, "NOTE") {
			continue
		}

		// 查找时间码行（含 -->）
		lines := strings.Split(b, "\n")
		timeIdx := -1
		for i, ln := range lines {
			if strings.Contains(ln, "-->") {
				timeIdx = i
				break
			}
		}
		if timeIdx == -1 {
			continue
		}

		// 解析时间码（去除后续对齐等设置）
		tcParts := strings.SplitN(lines[timeIdx], "-->", 2)
		if len(tcParts) != 2 {
			continue
		}
		left := strings.TrimSpace(tcParts[0])
		right := strings.TrimSpace(tcParts[1])
		// remove trail settings after a space
		if sp := strings.IndexAny(right, " \t"); sp >= 0 {
			right = strings.TrimSpace(right[:sp])
		}
		// 解析时间
		start, err := parseVttTimecode(left)
		if err != nil {
			return types.SubtitleProject{}, fmt.Errorf("failed to parse start time: %v", err)
		}
		end, err := parseVttTimecode(right)
		if err != nil {
			return types.SubtitleProject{}, fmt.Errorf("failed to parse end time: %v", err)
		}
		if start.Time == end.Time {
			continue
		}

		// 收集文本（时间行之后）
		var textLines []string
		for i := timeIdx + 1; i < len(lines); i++ {
			t := strings.TrimRight(lines[i], "\n\r")
			if t != "" {
				textLines = append(textLines, t)
			}
		}
		if len(textLines) == 0 {
			continue
		}

		combined := strings.Join(textLines, "\n")
		lc := types.LanguageContent{Text: combined}
		allContents = append(allContents, lc)
		if allText != "" {
			allText += "\n"
		}
		allText += combined

		seg := types.SubtitleSegment{
			ID:                uuid.New().String(),
			StartTime:         start,
			EndTime:           end,
			Speaker:           "",
			Languages:         make(map[string]types.LanguageContent),
			GuidelineStandard: make(map[string]types.GuideLineStandard),
		}
		project.Segments = append(project.Segments, seg)
	}

	// 语言检测 & 分配
	langInt, langCode := s.detector.DetectLanguageInt(allText)
	project.LanguageMetadata[langCode] = types.LanguageMetadata{
		DetectedLang: int(langInt),
		LanguageName: langCode,
		Quality:      "imported",
		SyncStatus:   "synced",
	}
	for i, c := range allContents {
		if i < len(project.Segments) {
			project.Segments[i].Languages[langCode] = c
		}
	}

	return project, nil
}

// parseVttTimecode 解析 WebVTT 时间格式（HH:MM:SS.mmm 或 MM:SS.mmm）
func parseVttTimecode(tc string) (types.Timecode, error) {
	tc = strings.TrimSpace(tc)
	// 拆分毫秒
	parts := strings.Split(tc, ".")
	if len(parts) != 2 {
		return types.Timecode{}, fmt.Errorf("invalid VTT timecode format: %s", tc)
	}
	msStr := parts[1]
	if len(msStr) > 3 { // 规范到毫秒
		msStr = msStr[:3]
	}
	for len(msStr) < 3 {
		msStr += "0"
	}
	ms, err := strconv.Atoi(msStr)
	if err != nil {
		return types.Timecode{}, fmt.Errorf("invalid milliseconds: %v", err)
	}

	// 解析 H(:)MM:SS
	hms := strings.Split(parts[0], ":")
	var hours, minutes, seconds int
	switch len(hms) {
	case 3:
		hours, err = strconv.Atoi(hms[0])
		if err != nil {
			return types.Timecode{}, err
		}
		minutes, err = strconv.Atoi(hms[1])
		if err != nil {
			return types.Timecode{}, err
		}
		seconds, err = strconv.Atoi(hms[2])
		if err != nil {
			return types.Timecode{}, err
		}
	case 2:
		hours = 0
		minutes, err = strconv.Atoi(hms[0])
		if err != nil {
			return types.Timecode{}, err
		}
		seconds, err = strconv.Atoi(hms[1])
		if err != nil {
			return types.Timecode{}, err
		}
	default:
		return types.Timecode{}, fmt.Errorf("invalid VTT time part: %s", parts[0])
	}

	d := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second + time.Duration(ms)*time.Millisecond
	return types.Timecode{Time: d}, nil
}
