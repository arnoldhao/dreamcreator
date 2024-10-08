package subtitles

import (
	"CanMe/backend/storage"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
)

// check if the content is capcut
func isCapcut(content string) bool {
	return strings.Contains(content, "\"tracks\"") && strings.Contains(content, "\"materials\"")
}

func isBcut(content string) bool {
	return strings.Contains(content, "\"video_track\"") && strings.Contains(content, "\"audio_track\"")
}

func generateKey() string {
	// generate unique name by timestamp
	return fmt.Sprintf("subtitle_%d", time.Now().UnixNano())
}

// getSubtitles
func (s *Service) getSubtitles(key string, format string) (subtitlesInfo storage.SubtitlesInfo, err error) {
	subtitles := storage.Subtitles{}
	err = subtitles.Read(s.ctx, key)
	if err != nil {
		return
	}

	if subtitles.Captions == "" {
		return storage.SubtitlesInfo{}, errors.New("subtitles not found")
	}

	captions, err := unmarshalCaptions(subtitles.Captions, format)
	if err != nil {
		return storage.SubtitlesInfo{}, fmt.Errorf("unmarshal captions failed: %v", err)
	}

	subtitlesInfo = storage.SubtitlesInfo{
		Key:                 subtitles.Key,
		FileName:            subtitles.FileName,
		Language:            subtitles.Language,
		Stream:              subtitles.Stream,
		Models:              subtitles.Models,
		Brief:               subtitles.Brief,
		Captions:            captions,
		TranslationStatus:   subtitles.TranslationStatus,
		TranslationProgress: subtitles.TranslationProgress,
		ActionDescription:   subtitles.ActionDescription,
	}

	return subtitlesInfo, nil
}

// getCaptions
func (s *Service) getCaptions(key string, format string) (captions string, err error) {
	subtitles := storage.Subtitles{}
	err = subtitles.Read(s.ctx, key)
	if err != nil {
		return "", err
	}

	if subtitles.Captions == "" {
		return "", errors.New("subtitles not found")
	}

	captions, err = unmarshalCaptions(subtitles.Captions, format)
	if err != nil {
		return "", err
	}

	return captions, nil
}

// unmarshalCaptions
func unmarshalCaptions(content string, format string) (captions string, err error) {
	// quick set
	if content == "" {
		return "", errors.New("content is empty")
	}

	// default format is srt
	if format == "" {
		format = "srt"
	}

	var subtitle *astisub.Subtitles
	if err = json.Unmarshal([]byte(content), &subtitle); err != nil {
		return "", err
	}

	return CaptionsToString(subtitle, format)
}

// convertCaptions
func CaptionsToString(sub *astisub.Subtitles, format string) (captions string, err error) {
	var buffer bytes.Buffer
	switch format {
	case "srt":
		if err = sub.WriteToSRT(&buffer); err != nil {
			return "", err
		}
	case "ass":
		if err = sub.WriteToSSA(&buffer); err != nil {
			return "", err
		}
	case "stl":
		if err = sub.WriteToSTL(&buffer); err != nil {
			return "", err
		}
	case "ttml":
		if err = sub.WriteToTTML(&buffer); err != nil {
			return "", err
		}
	case "vtt":
		if err = sub.WriteToWebVTT(&buffer); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}

	return buffer.String(), nil
}

// BriefSubtitle
func BriefSubtitle(subtitle *astisub.Subtitles, lines int) (brief string, err error) {
	// set default lines
	if lines == 0 {
		lines = 3
	}

	var buffer bytes.Buffer
	if items := subtitle.Items; len(items) > 0 {
		if len(items) < 3 {
			lines = len(items)
		}
		for i := 0; i < lines; i++ {
			buffer.WriteString(items[i].String())
		}
	} else {
		buffer.WriteString("No subtitles found")
	}

	return buffer.String(), nil
}

func formatSRT(input string) (string, error) {
	lines := strings.Split(input, "\n")
	var formatted []string
	index := 1
	timeRegex := regexp.MustCompile(`^(\d{2}:\d{2}:\d{2})[,.](\d{3}) --> (\d{2}:\d{2}:\d{2})[,.](\d{3})$`)

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// 检查索引
		if _, err := strconv.Atoi(line); err != nil && index == 1 {
			return "", errors.New("invalid subtitle index format")
		}

		// 格式化索引
		formatted = append(formatted, fmt.Sprintf("%d", index))
		index++

		// 检查并格式化时间戳
		if i+1 >= len(lines) {
			return "", errors.New("missing timestamp")
		}
		if !timeRegex.MatchString(strings.TrimSpace(lines[i+1])) {
			return "", fmt.Errorf("invalid timestamp format: %s", lines[i+1])
		}
		timestamp, err := formatTimestamp(lines[i+1])
		if err != nil {
			return "", err
		}
		formatted = append(formatted, timestamp)
		i++

		textFound := false
		for i+1 < len(lines) && strings.TrimSpace(lines[i+1]) != "" {
			formatted = append(formatted, strings.TrimSpace(lines[i+1]))
			i++
			textFound = true
		}
		if !textFound {
			return "", errors.New("missing subtitle text")
		}

		formatted = append(formatted, "")
	}

	if len(formatted) == 0 {
		return "", errors.New("empty SRT file")
	}

	return strings.Join(formatted, "\n"), nil
}

func formatTimestamp(timestamp string) (string, error) {
	parts := strings.Split(strings.TrimSpace(timestamp), " --> ")
	if len(parts) != 2 {
		return "", errors.New("invalid timestamp format")
	}
	start, err := formatTime(parts[0])
	if err != nil {
		return "", err
	}
	end, err := formatTime(parts[1])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s --> %s", start, end), nil
}

func formatTime(t string) (string, error) {
	parsed, err := time.Parse("15:04:05,999", strings.Replace(t, ".", ",", 1))
	if err != nil {
		return "", fmt.Errorf("invalid time format: %s", t)
	}
	return parsed.Format("15:04:05,000"), nil
}
