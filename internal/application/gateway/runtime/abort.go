package runtime

import (
	"context"
	"errors"
	"strings"

	"dreamcreator/internal/domain/thread"
)

type AbortRequest struct {
	RunID     string `json:"runId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type AbortResponse struct {
	Aborted bool   `json:"aborted"`
	RunID   string `json:"runId,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

func (service *Service) Abort(ctx context.Context, request AbortRequest) (AbortResponse, error) {
	if service == nil {
		return AbortResponse{}, errors.New("runtime unavailable")
	}
	runID := strings.TrimSpace(request.RunID)
	if runID == "" {
		sessionID := strings.TrimSpace(request.SessionID)
		if sessionID == "" {
			return AbortResponse{}, errors.New("run id or session id is required")
		}
		if service.runs == nil {
			return AbortResponse{}, errors.New("run repository unavailable")
		}
		activeRuns, err := service.runs.ListActiveByThread(ctx, sessionID)
		if err != nil {
			if errors.Is(err, thread.ErrRunNotFound) {
				return AbortResponse{Aborted: false}, nil
			}
			return AbortResponse{}, err
		}
		if len(activeRuns) == 0 {
			return AbortResponse{Aborted: false}, nil
		}
		runID = activeRuns[0].ID
	}
	if service.aborts == nil {
		return AbortResponse{Aborted: false, RunID: runID}, nil
	}
	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		reason = "user cancelled"
	}
	ok := service.aborts.Abort(runID, reason)
	return AbortResponse{Aborted: ok, RunID: runID, Reason: reason}, nil
}
