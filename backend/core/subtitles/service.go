package subtitles

import (
	"CanMe/backend/storage"
	"CanMe/backend/types"

	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	ctx context.Context
	// 使用接口类型
	formatConverter FormatConverter
	textProcessor   TextProcessor
	qualityAssessor QualityAssessor

	// bolt storage
	boltStorage *storage.BoltStorage
}

func NewService(boltStorage *storage.BoltStorage) *Service {
	return &Service{
		formatConverter: NewFormatConverter(),
		textProcessor:   NewTextProcessor(),
		qualityAssessor: NewQualityAssessor(),
		boltStorage:     boltStorage,
	}
}

func (s *Service) Subscribe(ctx context.Context) {
	s.ctx = ctx
}

// 统一错误处理
func (s *Service) handleError(operation string, err error) error {
	if err != nil {
		return fmt.Errorf("subtitle service %s failed: %w", operation, err)
	}
	return nil
}

// UpdateExportConfig 更新导出配置
func (s *Service) UpdateExportConfig(id string, config types.ExportConfigs) (*types.SubtitleProject, error) {
	// 1. validate input
	if id == "" {
		return nil, s.handleError("update export config", fmt.Errorf("id is empty"))
	}

	// 2. get project
	if s.boltStorage == nil {
		return nil, s.handleError("update export config", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update export config", err)
	}

	// 3. 验证和处理FCPXML配置
	if config.FCPXML != nil {
		// 如果没有提供项目名称，使用当前项目名称
		if config.FCPXML.ProjectName == "" {
			config.FCPXML.ProjectName = project.Metadata.Name
		}

		// 验证配置
		if err := config.FCPXML.Validate(); err != nil {
			return nil, s.handleError("update export config", fmt.Errorf("invalid FCPXML config: %w", err))
		}

		// 自动填充缺失字段
		config.FCPXML.AutoFill()
	}

	// 4. update export config
	project.Metadata.ExportConfigs = config

	// 5. save to database
	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update export config", err)
	}

	// 6. return updated project
	return project, nil
}

func (s *Service) ConvertSubtile(id, langCode, targetFormat string) ([]byte, error) {
	// 1. get file info
	if id == "" {
		return nil, s.handleError("convert subtitle", fmt.Errorf("id is empty"))
	}
	if targetFormat == "" {
		return nil, s.handleError("convert subtitle", fmt.Errorf("target format is empty"))
	}

	// 2. get file project
	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("convert subtitle", err)
	}

	// 3. convert to target format
	switch strings.ToLower(targetFormat) {
	case "srt":
		return s.formatConverter.ToSRT(project, langCode)
	case "vtt":
		return s.formatConverter.ToVTT(project, langCode)
	case "fcpxml":
		return s.formatConverter.ToFCPXML(project, langCode)
	default:
		return nil, s.handleError("convert subtitle", fmt.Errorf("unsupported format: %s", targetFormat))
	}
}

func (s *Service) ImportSubtitle(filePath string, options types.TextProcessingOptions) (*types.SubtitleProject, error) {
	if filePath == "" {
		return nil, s.handleError("import subtitle", fmt.Errorf("file path is empty"))
	}

	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}

	if len(file) == 0 {
		return nil, s.handleError("import subtitle", fmt.Errorf("file content is empty"))
	}

	// 根据文件扩展名确定格式
	var project types.SubtitleProject
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".itt":
		project, err = s.formatConverter.FromItt(filePath, file)
	case ".srt":
		project, err = s.formatConverter.FromSRT(filePath, file)
	// more formats...
	default:
		return nil, s.handleError("import subtitle", fmt.Errorf("unsupported file format: %s", ext))
	}

	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}

	// process
	err = s.processSubtitleText(&project, options)
	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}
	// validate
	err = s.validateProject(&project)
	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}

	// save to database
	err = s.boltStorage.SaveSubtitle(&project)
	if err != nil {
		return nil, s.handleError("import subtitle", err)
	}

	return &project, nil
}

func (s *Service) GetSubtitle(id string) (*types.SubtitleProject, error) {
	if id == "" {
		return nil, s.handleError("get subtitle", fmt.Errorf("id is empty"))
	}

	if s.boltStorage == nil {
		return nil, s.handleError("get subtitle", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("get subtitle", err)
	}

	return project, nil
}

func (s *Service) DeleteSubtitle(id string) error {
	if id == "" {
		return s.handleError("delete subtitle", fmt.Errorf("id is empty"))
	}

	if s.boltStorage == nil {
		return s.handleError("delete subtitle", fmt.Errorf("bolt storage is nil"))
	}

	err := s.boltStorage.DeleteSubtitle(id)
	if err != nil {
		return s.handleError("delete subtitle", err)
	}

	return nil
}

func (s *Service) DeleteAllSubtitle() error {
	if s.boltStorage == nil {
		return s.handleError("delete all subtitle", fmt.Errorf("bolt storage is nil"))
	}
	err := s.boltStorage.DeleteAllSubtitle()
	if err != nil {
		return s.handleError("delete all subtitle", err)
	}
	return nil
}

func (s *Service) ListSubtitles() ([]*types.SubtitleProject, error) {
	if s.boltStorage == nil {
		return nil, s.handleError("get all subtitles", fmt.Errorf("bolt storage is nil"))
	}

	projects, err := s.boltStorage.ListSubtitles()
	if err != nil {
		return nil, s.handleError("get all subtitles", err)
	}

	return projects, nil
}

func (s *Service) UpdateSubtitleProject(project *types.SubtitleProject) (*types.SubtitleProject, error) {
	if project == nil {
		return nil, s.handleError("update subtitle project", fmt.Errorf("project is nil"))
	}

	if err := project.Validate(); err != nil {
		return nil, s.handleError("update subtitle project", fmt.Errorf("invalid project: %w", err))
	}

	project.UpdatedAt = time.Now().Unix()

	err := s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update subtitle project", err)
	}

	return project, nil
}

func (s *Service) UpdateProjectName(id, name string) (*types.SubtitleProject, error) {
	if id == "" {
		return nil, s.handleError("update project name", fmt.Errorf("id is empty"))
	}
	if name == "" {
		return nil, s.handleError("update project name", fmt.Errorf("name is empty"))
	}
	if s.boltStorage == nil {
		return nil, s.handleError("update project name", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update project name", err)
	}
	project.ProjectName = name
	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update project name", err)
	}
	return project, nil
}

func (s *Service) UpdateProjectMetadata(id string, metadata types.ProjectMetadata) (*types.SubtitleProject, error) {
	if id == "" {
		return nil, s.handleError("update project metadata", fmt.Errorf("id is empty"))
	}

	if s.boltStorage == nil {
		return nil, s.handleError("update project metadata", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update project metadata", err)
	}
	project.Metadata = metadata
	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update project metadata", err)
	}
	return project, nil
}

// UpdateSubtitleSegment 更新单个字幕片段 - 修改为使用指针类型
func (s *Service) UpdateSubtitleSegment(id string, segmentID string, segment *types.SubtitleSegment) (*types.SubtitleProject, error) {
	if id == "" || segmentID == "" {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("id or segmentID is empty"))
	}

	if segment == nil {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("segment is nil"))
	}

	if s.boltStorage == nil {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update subtitle segment", err)
	}

	// 验证片段数据
	if err := segment.Validate(); err != nil {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("invalid segment: %w", err))
	}

	// recalculate guideline - 直接使用指针
	updatedSegment := s.qualityAssessor.AssessSegmentQuality(segment)

	// 查找并更新片段
	found := false
	for i, seg := range project.Segments {
		if seg.ID == segmentID {
			project.Segments[i] = *updatedSegment
			found = true
			break
		}
	}

	if !found {
		return nil, s.handleError("update subtitle segment", fmt.Errorf("segment with ID %s not found", segmentID))
	}

	project.UpdatedAt = time.Now().Unix()

	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update subtitle segment", err)
	}

	return project, nil
}

// UpdateLanguageContent 更新特定语言的内容
func (s *Service) UpdateLanguageContent(id string, segmentID string, langCode string, content types.LanguageContent) (*types.SubtitleProject, error) {
	if id == "" || segmentID == "" || langCode == "" {
		return nil, s.handleError("update language content", fmt.Errorf("id, segmentID or langCode is empty"))
	}

	if s.boltStorage == nil {
		return nil, s.handleError("update language content", fmt.Errorf("bolt storage is nil"))
	}

	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update language content", err)
	}

	// 验证内容
	if err := content.Validate(); err != nil {
		return nil, s.handleError("update language content", fmt.Errorf("invalid content: %w", err))
	}

	// 查找并更新语言内容
	found := false
	for i, seg := range project.Segments {
		if seg.ID == segmentID {
			if project.Segments[i].Languages == nil {
				project.Segments[i].Languages = make(map[string]types.LanguageContent)
			}
			project.Segments[i].Languages[langCode] = content
			found = true
			break
		}
	}

	if !found {
		return nil, s.handleError("update language content", fmt.Errorf("segment with ID %s not found", segmentID))
	}

	project.UpdatedAt = time.Now().Unix()

	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update language content", err)
	}

	return project, nil
}

func (s *Service) UpdateLanguageMetadata(id string, langCode string, metadata types.LanguageMetadata) (*types.SubtitleProject, error) {
	if id == "" || langCode == "" {
		return nil, s.handleError("update language metadata", fmt.Errorf("id or langCode is empty"))
	}
	if s.boltStorage == nil {
		return nil, s.handleError("update language metadata", fmt.Errorf("bolt storage is nil"))
	}
	project, err := s.boltStorage.GetSubtitle(id)
	if err != nil {
		return nil, s.handleError("update language metadata", err)
	}
	if project.LanguageMetadata == nil {
		project.LanguageMetadata = make(map[string]types.LanguageMetadata)
	}
	project.LanguageMetadata[langCode] = metadata
	err = s.boltStorage.SaveSubtitle(project)
	if err != nil {
		return nil, s.handleError("update language metadata", err)
	}
	return project, nil
}

// processSubtitleText 批量处理字幕文本
func (s *Service) processSubtitleText(project *types.SubtitleProject, options types.TextProcessingOptions) error {
	if project == nil {
		return s.handleError("process subtitle text", fmt.Errorf("project is nil"))
	}

	for i := range project.Segments {
		segment := &project.Segments[i]

		for langCode, content := range segment.Languages {
			processedContent := content

			// 文本清理
			if options.RemoveEmptyLines {
				processedContent.Text = s.textProcessor.RemoveEmptyLines(processedContent.Text)
			}

			if options.TrimWhitespace {
				processedContent.Text = s.textProcessor.TrimWhitespace(processedContent.Text)
			}

			if options.NormalizeLineBreaks {
				processedContent.Text = s.textProcessor.NormalizeLineBreaks(processedContent.Text)
			}

			if options.FixEncoding {
				processedContent.Text = s.textProcessor.FixEncoding(processedContent.Text)
			}

			if options.FixCommonErrors {
				processedContent.Text = s.textProcessor.FixCommonTextErrors(processedContent.Text)
			}

			// 重新计算guideline
			segment.Languages[langCode] = processedContent

			// set guideline standard
			segment.GuidelineStandard[langCode] = options.GuidelineStandard
		}

		// 重新计算该片段的guideline指标
		if options.ValidateGuidelines {
			segment.IsKidsContent = options.IsKidsContent
			s.qualityAssessor.AssessSegmentQuality(segment)
		}
	}

	return nil
}

// validateProject 验证项目内容
func (s *Service) validateProject(project *types.SubtitleProject) error {
	var validationErrors []error

	// 验证项目级别
	if err := project.Validate(); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// 验证每个片段
	for i, segment := range project.Segments {
		if err := segment.Validate(); err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("segment %d: %w", i+1, err))
		}
	}

	if len(validationErrors) > 0 {
		return s.handleError("validate project", fmt.Errorf("validation errors: %v", validationErrors))
	}

	return nil
}
