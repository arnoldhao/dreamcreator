package tools

import (
	"context"
	"errors"
)

func runStubTool(name string) func(ctx context.Context, args string) (string, error) {
	return func(_ context.Context, _ string) (string, error) {
		if name == "" {
			return "", errors.New("tool not implemented")
		}
		return "", errors.New(name + " not implemented")
	}
}
