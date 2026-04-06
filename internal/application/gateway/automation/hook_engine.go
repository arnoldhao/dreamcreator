package automation

import (
	"context"

	"dreamcreator/internal/domain/thread"
)

type HookEngine struct {
	engine *Engine
}

func NewHookEngine(engine *Engine) *HookEngine {
	return &HookEngine{engine: engine}
}

func (hook *HookEngine) OnRunStart(ctx context.Context, run thread.ThreadRun) {
	if hook == nil || hook.engine == nil {
		return
	}
	_, _ = hook.engine.Trigger(ctx, AutomationAction{
		Type:       "run.started",
		SessionKey: run.ThreadID,
		Payload: map[string]any{
			"runId":   run.ID,
			"threadId": run.ThreadID,
		},
	})
}

func (hook *HookEngine) OnRunEnd(ctx context.Context, run thread.ThreadRun, err error) {
	if hook == nil || hook.engine == nil {
		return
	}
	payload := map[string]any{
		"runId":   run.ID,
		"threadId": run.ThreadID,
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	_, _ = hook.engine.Trigger(ctx, AutomationAction{
		Type:       "run.completed",
		SessionKey: run.ThreadID,
		Payload:    payload,
	})
}
