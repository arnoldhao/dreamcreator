package downloads

import (
	"CanMe/backend/pkg/specials/config"
	"CanMe/backend/types"
	stringutil "CanMe/backend/utils/stringUtil"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (wq *WorkQueue) captionFilePath(cached *types.ExtractorData, caption *types.ExtractorCaption) (string, error) {
	filePath, err := wq.generateOutputFile(cached.Source, cached.Title, caption.Ext, caption.LanguageCode)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func (wq *WorkQueue) streamFilePath(source, title, quality, ext string) (string, error) {
	resolution := ""
	if p := strings.Split(quality, " ")[0]; p != "" {
		resolution = p
	}
	return wq.generateOutputFile(source, title, ext, resolution)
}

func (wq *WorkQueue) generateOutputFile(source, title string, ext string, adds ...string) (string, error) {
	sourceDir, err := wq.generateSourceDir(source)
	if err != nil {
		return "", err
	}

	if len(adds) > 0 {
		for _, add := range adds {
			if add != "" {
				title = fmt.Sprintf("%s_%s", title, add)
			}
		}
	}

	title = stringutil.SanitizeFileName(title)
	// generate base file name
	baseFileName := title
	if ext != "" {
		baseFileName = fmt.Sprintf("%s.%s", title, ext)
	}

	fileName := filepath.Join(sourceDir, baseFileName)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return fileName, nil
	}

	// if file exsited, add timestamp to file name
	timeStamp := time.Now().Format("20060102150405")
	if ext != "" {
		return filepath.Join(sourceDir, fmt.Sprintf("%s_%s.%s", title, timeStamp, ext)), nil
	}
	return filepath.Join(sourceDir, fmt.Sprintf("%s_%s", title, timeStamp)), nil
}

func (wq *WorkQueue) generateSourceDir(source string) (string, error) {
	downloadDir := config.GetDownloadInstance().GetDownloadURLWithCanMe()
	if downloadDir == "" {
		return "", fmt.Errorf("download dir is empty")
	}

	sourceDir := filepath.Join(downloadDir, source)
	// check if source dir is exsited
	if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
		// if source dir is not exsited, create it
		if err := os.MkdirAll(sourceDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create source directory: %w", err)
		}
	}

	return sourceDir, nil
}
