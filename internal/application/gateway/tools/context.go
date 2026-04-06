package tools

import "context"

type runtimeContextKey struct{}

type runtimeContext struct {
	sessionKey string
	runID      string
}

func WithRuntimeContext(ctx context.Context, sessionKey string, runID string) context.Context {
	if ctx == nil {
		return ctx
	}
	value := runtimeContext{
		sessionKey: sessionKey,
		runID:      runID,
	}
	return context.WithValue(ctx, runtimeContextKey{}, value)
}

func RuntimeContextFromContext(ctx context.Context) (string, string) {
	if ctx == nil {
		return "", ""
	}
	value, ok := ctx.Value(runtimeContextKey{}).(runtimeContext)
	if !ok {
		return "", ""
	}
	return value.sessionKey, value.runID
}
