package download

import (
	"CanMe/backend/pkg/specials/config"
	stringutil "CanMe/backend/utils/stringUtil"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Path struct{}

func NewPath() *Path {
	return &Path{}
}

func (p *Path) SavedPath(source string) (string, error) {
	return p.generateSourceDir(source)
}

func (p *Path) CaptionFilePath(source, title, ext, code string) (string, error) {
	filePath, err := p.generateOutputFile(source, title, ext, code)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func (p *Path) StreamFilePath(source, title, ext, quality string) (string, error) {
	resolution := ""
	if p := strings.Split(quality, " ")[0]; p != "" {
		resolution = p
	}
	return p.generateOutputFile(source, title, ext, resolution)
}

func (p *Path) generateOutputFile(source, title string, ext string, adds ...string) (string, error) {
	sourceDir, err := p.generateSourceDir(source)
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

func (p *Path) generateSourceDir(source string) (string, error) {
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
