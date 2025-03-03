package download

import (
	"CanMe/backend/consts"
	"CanMe/backend/core/events"
	"CanMe/backend/models"
	"CanMe/backend/pkg/extractors"
	"CanMe/backend/storage/repository"
	"CanMe/backend/types"
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/google/uuid"
)

type Service struct {
	repo      repository.DownloadRepository
	path      *Path
	queue     *Queue
	eventBus  events.EventBus
	content   []*types.ExtractorData
	contentMu sync.RWMutex
}

// NewService
func NewService(eventBus events.EventBus, repo repository.DownloadRepository) *Service {
	return &Service{
		path:     NewPath(),
		repo:     repo,
		queue:    NewQueue(eventBus, repo),
		eventBus: eventBus,
		content:  make([]*types.ExtractorData, 0),
	}
}

func (s *Service) Start(ctx context.Context) {
	// queue
	s.queue.Start(ctx)

	// cache
}

func (s *Service) GetContent(url string) (*types.ContentResponse, error) {
	if url == "" {
		return nil, ErrEmptyURL
	}

	videos, err := extractors.Extract(url, types.ExtractorOptions{})
	if err != nil {
		return nil, fmt.Errorf("extract content: %w", err)
	}

	if len(videos) < 1 {
		return nil, ErrNoContent
	}

	var total int
	for _, v := range videos {
		// calculate total
		total += len(v.Streams)
		total += len(v.Captions)
		total += len(v.Danmakus)

		// set id
		if v.ID == "" {
			v.ID = uuid.New().String()
		}
	}

	s.contentMu.Lock()
	s.content = videos
	s.contentMu.Unlock()

	return &types.ContentResponse{
		Videos: videos,
		Total:  total,
	}, nil
}

func (s *Service) IsDownloading(req *types.TaskRequest) bool {
	if req.ContentID == "" || req.Stream == "" {
		return false
	}

	return s.queue.IsQueue(req.ContentID, req.Stream)
}

func (s *Service) IsDownloadingByTaskID(taskID string) bool {
	_, ok := s.queue.Get(taskID)
	return ok
}

func (s *Service) CreateTask(ctx context.Context, req *types.TaskRequest) (*types.TaskResponse, error) {
	s.contentMu.RLock()
	if len(s.content) == 0 {
		s.contentMu.RUnlock()
		return nil, ErrContentNotReady
	}

	var targetContent *types.ExtractorData
	for _, c := range s.content {
		if c.ID == req.ContentID {
			targetContent = c
			break
		}
	}
	s.contentMu.RUnlock()

	if targetContent == nil {
		return nil, ErrContentNotFound
	}

	task, err := s.fillTaskDetails(req)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	if err := s.queue.Add(task); err != nil {
		return nil, err
	}

	s.eventBus.Publish(consts.TopicDownloadStatus, map[string]interface{}{
		"taskId": task.TaskID,
		"status": DownloadStatusQueued,
	})

	return &types.TaskResponse{
		TaskID: task.TaskID,
		Status: DownloadStatusQueued,
	}, nil
}

func (s *Service) CancelTask(taskID string) error {
	return s.queue.CancelTask(taskID)
}

func (s *Service) GetTaskStatus(ctx context.Context, taskID string) (*models.DownloadTask, error) {
	if task, exists := s.queue.Get(taskID); exists {
		return task, nil
	}

	return s.repo.FindByID(ctx, taskID)
}

func (s *Service) GetAllTasks(ctx context.Context) (tasks []*models.DownloadTask, err error) {
	return s.queue.GetAll()
}

func (s *Service) ListDownloaded(ctx context.Context) (tasks []*models.DownloadTask, err error) {
	return s.repo.ListTasks(ctx, 0, 100)
}

func (s *Service) GetFFMPEGVersion(ctx context.Context) (version []byte, err error) {
	return ffmpegVersion()
}

func (s *Service) DeleteRecord(ctx context.Context, id string) (err error) {
	return s.repo.Delete(ctx, id)
}

func (s *Service) fillTaskDetails(req *types.TaskRequest) (task *models.DownloadTask, err error) {
	// check params
	if len(s.content) == 0 {
		return nil, errors.New("no content found")
	}

	if req.ContentID != s.content[0].ID {
		return nil, errors.New("content ID mismatch")
	}

	// generate params
	savedPath, err := s.path.SavedPath(s.content[0].Source)
	if err != nil {
		return nil, err
	}

	var totalSize, totalParts int64

	streamParts := []*models.StreamPart{}
	commonParts := []*models.CommonPart{}
	// stream path
	if stream, ok := s.content[0].Streams[req.Stream]; ok {
		// stream file name
		fileName, err := s.path.StreamFilePath(s.content[0].Source,
			s.content[0].Title,
			stream.Ext,
			stream.Quality)
		if err != nil {
			return nil, err
		}

		// stream part
		if len(stream.Parts) > 1 {
			for num, part := range stream.Parts {
				streamParts = append(streamParts, &models.StreamPart{
					PartID:        uuid.New().String(),
					TaskID:        req.TaskID,
					ContentID:     req.ContentID,
					Name:          "stream",
					Type:          "stream",
					FileName:      fileName + "_" + strconv.Itoa(num) + part.FileName,
					FinalFileName: fileName,
					URL:           part.URL,
					Quality:       stream.Quality,
					Ext:           stream.Ext,
					Progress:      0,
					Duration:      0,
					AverageSpeed:  "",
					CurrentSize:   0,
					TotalSize:     part.Size, // total size
					Status:        models.TaskStatusPending.String(),
					FinalStatus:   false,
					Message:       "",
					NeedMerge:     true,
				})

				totalSize += part.Size
				totalParts++
			}
		} else if len(stream.Parts) == 1 {
			streamParts = append(streamParts, &models.StreamPart{
				PartID:        uuid.New().String(),
				TaskID:        req.TaskID,
				ContentID:     req.ContentID,
				Name:          "stream",
				Type:          "stream",
				FileName:      fileName,
				FinalFileName: fileName,
				URL:           stream.Parts[0].URL,
				Quality:       stream.Quality,
				Ext:           stream.Ext,
				Progress:      0,
				Duration:      0,
				AverageSpeed:  "",
				CurrentSize:   0,
				TotalSize:     stream.Parts[0].Size, // total size
				Status:        models.TaskStatusPending.String(),
				FinalStatus:   false,
				Message:       "",
				NeedMerge:     false,
			})

			totalSize = stream.Parts[0].Size
			totalParts = 1
		} else {
			// do nothing
		}

	}

	// captions path
	if len(req.Captions) > 0 {
		for _, lang := range req.Captions {
			if caption, ok := s.content[0].Captions[lang]; ok {
				// caption file name
				fileName, err := s.path.CaptionFilePath(s.content[0].Source,
					s.content[0].Title,
					DefaultCaptionExt,
					caption.LanguageCode)
				if err != nil {
					return nil, err
				}

				// part
				commonParts = append(commonParts, &models.CommonPart{
					PartID:      caption.LanguageCode,
					TaskID:      req.TaskID,
					ContentID:   req.ContentID,
					Name:        "caption",
					Type:        "caption",
					FileName:    fileName,
					URL:         caption.URL,
					Ext:         DefaultCaptionExt,
					Status:      "pending",
					FinalStatus: false,
					Message:     "",
				})

				totalParts++
			}
		}
	}

	// danmaku path
	if req.Danmaku {
		if danmaku := s.content[0].Danmakus[req.Stream]; danmaku != nil {
			commonParts = append(commonParts, &models.CommonPart{
				PartID:      danmaku.ID,
				TaskID:      req.TaskID,
				ContentID:   req.ContentID,
				Name:        "danmaku",
				Type:        "danmaku",
				FileName:    danmaku.FileName,
				URL:         danmaku.URL,
				Ext:         danmaku.Ext,
				Status:      "pending",
				FinalStatus: false,
				Message:     "",
			})

			totalParts++
		}

	}

	// initial task
	task = &models.DownloadTask{
		TaskID:           uuid.NewString(),
		ContentID:        req.ContentID,
		TotalSize:        totalSize,
		TotalCurrentSize: 0,
		TotalParts:       totalParts,
		FinishedParts:    0,
		Stream:           req.Stream,
		Captions:         req.Captions,
		Danmaku:          req.Danmaku,
		Source:           s.content[0].Source,
		URL:              s.content[0].URL,
		Title:            s.content[0].Title,
		SavedPath:        savedPath,
		StreamParts:      streamParts,
		CommonParts:      commonParts,
		TaskStatus:       models.TaskStatusPending,
	}

	// return
	return task, nil
}
