package subtitles

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/subs"
	"CanMe/backend/pkg/subs/bcut"
	"CanMe/backend/pkg/subs/capcut"
	"CanMe/backend/pkg/subs/others"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"CanMe/backend/utils/sliceUtil"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	ctx context.Context
}

func New() *Service {
	return &Service{}
}

func (s *Service) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *Service) Convert(filePath string) (resp types.JSResp) {
	if filePath == "" {
		resp.Msg = "file path is empty"
		resp.Success = false
		return
	}

	// read file
	byte, err := os.ReadFile(filePath)
	if err != nil {
		resp.Msg = "open file failed: " + err.Error()
		resp.Success = false
		return
	}
	file := string(byte)

	// get file name, including the extension, e.g. thisfilename.txt
	fileName := filepath.Base(filePath)
	var subs subs.Interface
	switch filepath.Ext(strings.ToLower(fileName)) { // according to the extension
	case ".json":
		// judge capcut or bcut
		if isCapcut(file) {
			subs = &capcut.Capcut{}
		} else if isBcut(file) {
			subs = &bcut.BCut{}
		}
	default:
		subs = &others.Others{}
	}

	// convert file
	captions, err := subs.Format(s.ctx, fileName, file)
	if err != nil {
		resp.Msg = "convert failed: " + err.Error()
		resp.Success = false
		return
	}

	// convert captions to string
	captionsByte, err := json.Marshal(captions)
	if err != nil {
		resp.Msg = "marshal json failed: " + err.Error()
		resp.Success = false
		return
	}

	// brief
	brief, _ := BriefSubtitle(captions, 3)

	// 3. generate subtitle
	key := generateKey()
	fileNameTrimed := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	subtitles := storage.Subtitles{
		Key:                 key,                    // backend generate timestamp primary key
		FileName:            fileNameTrimed,         // original file name without extension
		Language:            "original",             // default
		Stream:              false,                  // not stream
		Models:              "original",             // not use model
		Brief:               brief,                  // subtitle brief
		Captions:            string(captionsByte),   // subtitle string
		TranslationStatus:   storage.StatusOriginal, // translation status
		TranslationProgress: 0,                      // translation progress
		ActionDescription:   "",                     // translation error
	}

	// 4. save to database
	err = subtitles.Create(s.ctx)
	if err != nil {
		resp.Msg = "save failed: " + err.Error()
		resp.Success = false
		return
	}

	// 5. return success and key
	captionsString, _ := CaptionsToString(captions, consts.SUBTITLE_FORMAT_SRT)
	resp.Success = true
	resp.Data = map[string]string{
		"key":       key,
		"fileName":  fileNameTrimed,
		"subtitles": captionsString,
	}

	return
}

func (s *Service) GetSubtitles(key string) (resp types.JSResp) {
	subtitles, err := s.getSubtitles(key, consts.SUBTITLE_FORMAT_SRT)
	if err != nil {
		resp.Msg = "get failed: " + err.Error()
		resp.Success = false
		return
	}

	subtitlesStr, err := json.Marshal(&subtitles)
	if err != nil {
		resp.Msg = "marshal failed: " + err.Error()
		resp.Success = false
		return
	}

	resp.Success = true
	resp.Data = string(subtitlesStr)
	return
}

func (s *Service) GetCaptions(key string) (resp types.JSResp) {
	captions, err := s.getCaptions(key, consts.SUBTITLE_FORMAT_SRT)
	if err != nil {
		resp.Msg = "get failed: " + err.Error()
		resp.Success = false
		return
	}

	resp.Success = true
	resp.Data = captions
	return
}

func (s *Service) ListSubtitles() (resp types.JSResp) {
	subtitles, err := storage.ListSubtitles(s.ctx)
	if err != nil {
		resp.Msg = "list failed: " + err.Error()
		resp.Success = false
		return
	}

	var result []storage.Subtitles
	for _, subtitle := range subtitles {
		if subtitle.Captions != "" {
			caption, err := unmarshalCaptions(subtitle.Captions, consts.SUBTITLE_FORMAT_SRT)
			if err == nil {
				subtitle.Captions = caption
			}

			result = append(result, subtitle)
		}
	}

	resp.Success = true
	resp.Data = result
	return
}

func (s *Service) DeleteSubtitles(key string) (resp types.JSResp) {
	if key == "" {
		resp.Msg = "key is empty"
		resp.Success = false
		return
	}

	subtitles := storage.Subtitles{}
	err := subtitles.Read(s.ctx, key)
	if err != nil {
		resp.Msg = "delete failed: " + err.Error()
		resp.Success = false
		return
	}

	err = subtitles.Delete(s.ctx)
	if err != nil {
		resp.Msg = "delete failed: " + err.Error()
		resp.Success = false
		return
	}

	resp.Success = true
	return
}

func (s *Service) DeleteSubtitleByKeepDays(keepDays int) (resp types.JSResp) {
	if keepDays <= 0 {
		resp.Msg = "keepDays must be greater than 0"
		resp.Success = false
		return
	}

	subtitles, err := storage.ListSubtitles(s.ctx)
	if err != nil {
		resp.Msg = "get subtitles failed: " + err.Error()
		resp.Success = false
		return
	}

	cutoffTime := time.Now().AddDate(0, 0, -keepDays)
	var deletedCount int

	for _, subtitle := range subtitles {
		if subtitle.CreatedAt.Before(cutoffTime) {
			err = subtitle.Delete(s.ctx)
			if err != nil {
				log.Printf("delete subtitle %s failed: %v", subtitle.Key, err)
				continue
			}
			deletedCount++
		}
	}

	resp.Success = true
	resp.Msg = fmt.Sprintf("delete %d subtitles successfully", deletedCount)
	return
}

func (s *Service) DeleteAllSubtitles() (resp types.JSResp) {
	subtitles, err := storage.ListSubtitles(s.ctx)
	if err != nil {
		resp.Msg = "delete failed: " + err.Error()
		resp.Success = false
		return
	}

	for _, subtitle := range subtitles {
		err = subtitle.Delete(s.ctx)
		if err != nil {
			resp.Msg = "delete failed: " + err.Error()
			resp.Success = false
			return
		}
	}

	resp.Success = true
	return
}

func (s *Service) SaveSubtitles(key, fileName string, extensions []string) (resp types.JSResp) {
	filters := sliceUtil.Map(extensions, func(i int) runtime.FileFilter {
		return runtime.FileFilter{
			Pattern: "*." + extensions[i],
		}
	})

	filePath, err := runtime.SaveFileDialog(s.ctx, runtime.SaveDialogOptions{
		Title:           fileName,
		ShowHiddenFiles: true,
		DefaultFilename: fileName,
		Filters:         filters,
	})

	if err != nil {
		resp.Msg = "save failed: " + err.Error()
		resp.Success = false
		return
	}

	// file path is empty may also mean user canceled the operation
	if filePath == "" {
		resp.Msg = "user canceled the operation"
		resp.Success = false
		resp.Data = map[string]bool{"canceled": true}
		return
	}

	// input check
	if key == "" {
		resp.Msg = "key is empty"
		resp.Success = false
		return
	}

	// get subtitles
	captions, err := s.getCaptions(key, strings.TrimPrefix(filepath.Ext(filePath), "."))
	if err != nil {
		resp.Msg = "get subtitles failed: " + err.Error()
		resp.Success = false
		return
	}

	// write to file
	err = os.WriteFile(filePath, []byte(captions), 0644)
	if err != nil {
		resp.Msg = "write file failed: " + err.Error()
		resp.Success = false
		return
	}

	resp.Success = true
	resp.Data = map[string]string{"path": filePath}
	return
}

func (s *Service) UpdateTitle(key, title string) (resp types.JSResp) {
	subtitles := storage.Subtitles{}
	err := subtitles.Read(s.ctx, key)
	if err != nil {
		resp.Msg = "update failed: " + err.Error()
		resp.Success = false
		return
	}
	subtitles.FileName = title
	err = subtitles.Update(s.ctx)
	if err != nil {
		resp.Msg = "update failed: " + err.Error()
		resp.Success = false
		return
	}

	resp.Success = true
	return
}

func (s *Service) UpdateCaptions(key, captions string) (resp types.JSResp) {
	if key == "" {
		resp.Msg = "key is empty"
		resp.Success = false
		return
	}

	if captions == "" {
		resp.Msg = "captions is empty"
		resp.Success = false
		return
	}

	formatSRT, err := formatSRT(captions)
	if err != nil {
		resp.Msg = "update failed: " + err.Error()
		resp.Success = false
		return
	}

	subtitles := storage.Subtitles{}
	err = subtitles.Read(s.ctx, key)
	if err != nil {
		resp.Msg = "update failed: " + err.Error()
		resp.Success = false
		return
	}

	// convert srt to astisub
	subs := &others.Others{}
	astisubCaptions, err := subs.Format(s.ctx, subtitles.FileName+".srt", formatSRT)
	if err != nil {
		resp.Msg = "update failed: " + err.Error()
		resp.Success = false
		return
	}

	// convert astisub to string
	astisubCaptionsByte, err := json.Marshal(astisubCaptions)
	if err != nil {
		resp.Msg = "update failed: " + err.Error()
		resp.Success = false
		return
	}

	subtitles.Captions = string(astisubCaptionsByte)
	err = subtitles.Update(s.ctx)
	if err != nil {
		resp.Msg = "update failed: " + err.Error()
		resp.Success = false
		return
	}

	returnCaptions, err := unmarshalCaptions(string(astisubCaptionsByte), consts.SUBTITLE_FORMAT_SRT)
	if err != nil {
		runtime.LogError(s.ctx, "unmarshal failed: "+err.Error())
		returnCaptions = formatSRT
	}

	resp.Success = true
	resp.Data = returnCaptions
	return
}
