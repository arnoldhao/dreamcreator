package timeUtil

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func ParseCapcut(c int) (time.Duration, error) {
	// capcut timestamp is microsecond
	d := time.Duration(c) * time.Microsecond

	// format to srt string
	srtFormat := fmt.Sprintf("%02d:%02d:%02d,%03d",
		int(d.Hours()),
		int(d.Minutes())%60,
		int(d.Seconds())%60,
		int(d.Milliseconds())%1000)

	return formatDurationCapcut(srtFormat)
}

func ParseBCut(d time.Duration) (time.Duration, error) {
	// todo

	return d, nil
}

func ParseBilibiliSubtitle(i float32) (d time.Duration, err error) {
	return parseDuration(fmt.Sprintf("%f", i), ".", 3)
}

func ParseYoutubeTranscript(i string) (d time.Duration, err error) {
	return parseDuration(i, ".", 3)
}

func formatDurationCapcut(i string) (d time.Duration, err error) {
	return parseDuration(i, ",", 3)
}

// parseDuration parses a duration in "00:00:00.000", "00:00:00,000" or "0:00:00:00" format
func parseDuration(i string, millisecondSep string, numberOfMillisecondDigits int) (o time.Duration, err error) {
	// split milliseconds
	parts := strings.Split(i, millisecondSep)
	timeStr := parts[0]
	var milliseconds int

	if len(parts) >= 2 {
		msStr := strings.TrimSpace(parts[1])
		if len(msStr) > 3 {
			return 0, fmt.Errorf("invalid number of millisecond digits detected in %s", i)
		}
		if milliseconds, err = strconv.Atoi(msStr); err != nil {
			return 0, fmt.Errorf("atoi of %s failed: %w", msStr, err)
		}
		milliseconds *= int(math.Pow10(numberOfMillisecondDigits - len(msStr)))
	}

	// split hours, minutes, seconds
	timeParts := strings.Split(strings.TrimSpace(timeStr), ":")
	if len(timeParts) < 2 || len(timeParts) > 3 {
		return 0, fmt.Errorf("invalid time format in %s", i)
	}

	var hours, minutes, seconds int
	if len(timeParts) == 3 {
		if hours, err = strconv.Atoi(timeParts[0]); err != nil {
			return 0, fmt.Errorf("invalid hours in %s: %w", i, err)
		}
		timeParts = timeParts[1:]
	}
	if minutes, err = strconv.Atoi(timeParts[0]); err != nil {
		return 0, fmt.Errorf("invalid minutes in %s: %w", i, err)
	}
	if seconds, err = strconv.Atoi(timeParts[1]); err != nil {
		return 0, fmt.Errorf("invalid seconds in %s: %w", i, err)
	}

	// generate output
	o = time.Duration(milliseconds)*time.Millisecond +
		time.Duration(seconds)*time.Second +
		time.Duration(minutes)*time.Minute +
		time.Duration(hours)*time.Hour
	return
}

func FormatDurationSRT(i time.Duration) string {
	return formatDuration(i, ",", 3)
}

func formatDuration(i time.Duration, millisecondSep string, numberOfMillisecondDigits int) (s string) {
	// Parse hours
	var hours = int(i / time.Hour)
	var n = i % time.Hour
	if hours < 10 {
		s += "0"
	}
	s += strconv.Itoa(hours) + ":"

	// Parse minutes
	var minutes = int(n / time.Minute)
	n = i % time.Minute
	if minutes < 10 {
		s += "0"
	}
	s += strconv.Itoa(minutes) + ":"

	// Parse seconds
	var seconds = int(n / time.Second)
	n = i % time.Second
	if seconds < 10 {
		s += "0"
	}
	s += strconv.Itoa(seconds) + millisecondSep

	// Parse milliseconds
	var milliseconds = float64(n/time.Millisecond) / float64(1000)
	s += fmt.Sprintf("%."+strconv.Itoa(numberOfMillisecondDigits)+"f", milliseconds)[2:]
	return
}
