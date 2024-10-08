package bcut

import (
	"context"

	"github.com/asticode/go-astisub"
)

type BCut struct{}

func (p *BCut) Format(ctx context.Context, fileName, jsonData string) (captions *astisub.Subtitles, err error) {
	return
}
