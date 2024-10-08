package others

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asticode/go-astisub"
)

type Others struct{}

func (p *Others) Format(ctx context.Context, fileName, jsonData string) (captions *astisub.Subtitles, err error) {
	switch filepath.Ext(strings.ToLower(fileName)) {
	case ".srt":
		captions, err = astisub.ReadFromSRT(strings.NewReader(jsonData))
	case ".ssa", ".ass":
		captions, err = astisub.ReadFromSSA(strings.NewReader(jsonData))
	case ".stl":
		captions, err = astisub.ReadFromSTL(strings.NewReader(jsonData), astisub.STLOptions{})
	case ".ts":
		captions, err = astisub.ReadFromTeletext(strings.NewReader(jsonData), astisub.TeletextOptions{})
	case ".ttml":
		captions, err = astisub.ReadFromTTML(strings.NewReader(jsonData))
	case ".vtt":
		captions, err = astisub.ReadFromWebVTT(strings.NewReader(jsonData))
	default:
		err = fmt.Errorf("request file type:%v is not supported", filepath.Ext(strings.ToLower(fileName)))
	}

	return
}
