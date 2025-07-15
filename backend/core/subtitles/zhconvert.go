package subtitles

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/events"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/pkg/zhconvert"
	"CanMe/backend/types"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *Service) GetSupportedConverters() []string {
	return s.zhConverter.GetSupportedConverters()
}

func (s *Service) ZHConvertSubtitle(id string, origin, converterString string) error {
	// params check
	if id == "" {
		return errors.New("subtitle id is empty")
	}

	if origin == "" {
		return errors.New("origin is empty")
	}

	if converterString == "" {
		return errors.New("converter is empty")
	}

	converter, err := zhconvert.ConverterTypeFromString(converterString)
	if err != nil {
		return err
	}

	// get subtitle
	sub, err := s.GetSubtitle(id)
	if err != nil {
		return err
	}

	if _, ok := sub.LanguageMetadata[origin]; !ok {
		return errors.New("origin language not found")
	}

	// origin subtitle
	originSubs := make([]string, len(sub.Segments))
	for i := 0; i < len(sub.Segments); i++ {
		if _, ok := sub.Segments[i].Languages[origin]; ok {
			originSubs[i] = sub.Segments[i].Languages[origin].Text
		} else {
			originSubs[i] = ""
		}
	}

	// 判断之前是否已经存在转换的任务
	taskID := uuid.NewString()
	conversionTask := types.ConversionTask{
		ID:                taskID,
		Type:              "zhconvert",
		Status:            types.ConversionStatusPending,
		Progress:          0,
		StartTime:         time.Now().Unix(),
		EndTime:           0,
		ErrorMessage:      "",
		SourceLang:        origin,
		TargetLang:        converter.String(),
		Converter:         int(converter),
		ConverterName:     converter.String(),
		Provider:          "zhconvert",
		TotalSegments:     len(originSubs),
		ProcessedSegments: 0,
		FailedSegments:    0,
	}
	var beforeConvertMetadata types.LanguageMetadata
	if metadata, ok := sub.LanguageMetadata[converter.String()]; ok {
		metadata.Revision++
		metadata.ActiveTaskID = taskID
		metadata.Status.ConversionTasks = append(metadata.Status.ConversionTasks, conversionTask)
		metadata.Status.LastUpdated = time.Now().Unix()
		beforeConvertMetadata = metadata
	} else {
		// update metadata
		beforeConvertMetadata = types.LanguageMetadata{
			Revision:     0,
			DetectedLang: int(converter),
			LanguageName: converter.String(),
			Translator:   "zhconvert",
			Notes:        "",
			Quality:      "zhconvert",
			SyncStatus:   "converting",
			CustomFields: nil,
			Status: types.LanguageContentStatus{
				IsOriginal:      false,
				ConversionTasks: []types.ConversionTask{conversionTask},
				LastUpdated:     time.Now().Unix(),
			},
			ActiveTaskID: taskID,
		}
	}

	// save initial metadata
	sub.LanguageMetadata[converter.String()] = beforeConvertMetadata
	// save subtitle
	err = s.boltStorage.SaveSubtitle(sub)
	if err != nil {
		return s.handleError("update export config", err)
	}

	// 这里直接返回，因为zhconvert是异步的，不会阻塞
	// 创建事件
	event := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      consts.TopicSubtitleProgress,
		Source:    "subtitles",
		Timestamp: time.Now(),
		Data:      conversionTask,
		Metadata: map[string]interface{}{
			"progress": conversionTask,
		},
	}

	s.eventBus.Publish(s.ctx, event)

	go func() {
		// zhconvert
		convertedSubs, err := s.zhConverter.ConvertMultiple(originSubs, converter)
		if err != nil {
			logger.GetLogger().Info("zhconvert failed", zap.Error(err))
			// 保存转换失败的元数据
			s.handleSubtitleChange(sub, converter, err)
			// return
			return
		}

		// 重新获取 subtitle
		sub, err = s.GetSubtitle(id)
		if err != nil {
			// 保存转换失败的元数据
			s.handleSubtitleChange(sub, converter, err)
			// return
			return
		}

		// 检查结果长度
		if len(convertedSubs) != len(sub.Segments) {
			err := fmt.Errorf("converted results count (%d) doesn't match segments count (%d)", len(convertedSubs), len(sub.Segments))
			s.handleSubtitleChange(sub, converter, err)
			// return
			return
		}

		// 检查 segments 是否为空
		if len(sub.Segments) == 0 {
			err := fmt.Errorf("no segments found in subtitle")
			s.handleSubtitleChange(sub, converter, err)
			return
		}

		// 获取标准
		var standard types.GuideLineStandard
		if standard = sub.Segments[0].GuidelineStandard[origin]; standard == "" {
			standard = types.GuideLineStandardNetflix
		}

		// 更新字幕，确保 map 已初始化
		for i := 0; i < len(sub.Segments); i++ {
			// 确保 map 已初始化
			if sub.Segments[i].Languages == nil {
				sub.Segments[i].Languages = make(map[string]types.LanguageContent)
			}
			if sub.Segments[i].GuidelineStandard == nil {
				sub.Segments[i].GuidelineStandard = make(map[string]types.GuideLineStandard)
			}

			content := types.LanguageContent{
				Text: convertedSubs[i],
			}
			sub.Segments[i].Languages[converter.String()] = content
			sub.Segments[i].GuidelineStandard[converter.String()] = standard
			sub.Segments[i] = *s.qualityAssessor.AssessSegmentQuality(&sub.Segments[i])
		}

		// validate
		err = s.validateProject(sub)
		if err != nil {
			s.handleSubtitleChange(sub, converter, err)
		}

		// update metadata
		s.handleSubtitleChange(sub, converter, nil)
	}()

	return nil
}

func (s *Service) handleSubtitleChange(sub *types.SubtitleProject, converter zhconvert.ConverterType, err error) {
	var status types.ConversionStatus
	var errorMessage string
	if err != nil {
		status = types.ConversionStatusFailed
		errorMessage = err.Error()
	} else {
		status = types.ConversionStatusCompleted
	}
	if metadata, ok := sub.LanguageMetadata[converter.String()]; ok {
		metadata.Revision++
		// 处理转换任务失败
		var tasks []types.ConversionTask
		var conversionTask types.ConversionTask
		for _, task := range metadata.Status.ConversionTasks {
			if task.ID == metadata.ActiveTaskID {
				conversionTask = task
			} else {
				tasks = append(tasks, task)
			}
		}

		conversionTask.Status = status
		conversionTask.EndTime = time.Now().Unix()
		conversionTask.ErrorMessage = errorMessage

		// 添加到所有任务
		tasks = append(tasks, conversionTask)

		// 保存metadata
		metadata.Status.ConversionTasks = tasks
		metadata.Status.LastUpdated = time.Now().Unix()
		metadata.SyncStatus = "done"
		sub.LanguageMetadata[converter.String()] = metadata

		// 返回
		err = s.boltStorage.SaveSubtitle(sub)
		if err != nil {
			conversionTask.ErrorMessage = err.Error()
			conversionTask.Status = types.ConversionStatusFailed
			// 创建事件
			event := &events.BaseEvent{
				ID:        uuid.New().String(),
				Type:      consts.TopicSubtitleProgress,
				Source:    "subtitles",
				Timestamp: time.Now(),
				Data:      conversionTask,
				Metadata: map[string]interface{}{
					"progress": conversionTask,
				},
			}

			s.eventBus.Publish(s.ctx, event)
			// end
		} else {
			// 创建事件
			event := &events.BaseEvent{
				ID:        uuid.New().String(),
				Type:      consts.TopicSubtitleProgress,
				Source:    "subtitles",
				Timestamp: time.Now(),
				Data:      conversionTask,
				Metadata: map[string]interface{}{
					"progress": conversionTask,
				},
			}

			s.eventBus.Publish(s.ctx, event)
			// end
		}
	} else {
		err = fmt.Errorf("subtitle %s handle error failed: %w", sub.ID, err)
		// 创建事件
		conversionTask := types.ConversionTask{
			ID:           uuid.New().String(),
			Status:       types.ConversionStatusFailed,
			ErrorMessage: err.Error(),
			EndTime:      time.Now().Unix(),
		}

		event := &events.BaseEvent{
			ID:        uuid.New().String(),
			Type:      consts.TopicSubtitleProgress,
			Source:    "subtitles",
			Timestamp: time.Now(),
			Data:      conversionTask,
			Metadata: map[string]interface{}{
				"progress": conversionTask,
			},
		}

		s.eventBus.Publish(s.ctx, event)
		// end
	}
}
