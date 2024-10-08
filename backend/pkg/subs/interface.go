package subs

import (
	"context"

	"github.com/asticode/go-astisub"
)

type Sub struct{}

type Interface interface {
	Format(ctx context.Context, fileName, jsonData string) (captions *astisub.Subtitles, err error)
}
