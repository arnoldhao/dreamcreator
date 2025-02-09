package ollama

import (
	"CanMe/backend/consts"
	"CanMe/backend/types"
	"errors"
	"log"
	"strings"

	"github.com/ollama/ollama/api"
)

func (s *Service) Pull(key string, model string) error {
	if key == "" {
		return errors.New("key is empty")
	}

	if model == "" {
		return errors.New("model is empty")
	}

	if !strings.Contains(model, ":") {
		model = model + ":latest"
	}

	stream := true
	req := api.PullRequest{
		Model:  strings.ToLower(model),
		Stream: &stream,
	}

	go func() {
		err := s.ollama.Pull(s.ctx, &req, func(resp api.ProgressResponse) error {
			select {
			case <-s.ctx.Done():
				return s.ctx.Err()
			default:
				s.wsService.SendToClient(types.WSResponse{
					Namespace: consts.NAMESPACE_OLLAMA,
					Event:     consts.EVENT_OLLAMA_PULL_UPDATE,
					Data: types.OllamaResponse{
						ID:        key,
						Status:    resp.Status,
						Digest:    resp.Digest,
						Total:     resp.Total,
						Completed: resp.Completed,
					},
				})
			}
			return nil
		})

		if err != nil {
			log.Println("Error during pull:", err)
			// send error message to frontend
			s.wsService.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_OLLAMA,
				Event:     consts.EVENT_OLLAMA_PULL_ERROR,
				Data: types.OllamaResponse{
					ID:      key,
					Status:  "error",
					Error:   true,
					Message: err.Error(),
				},
			})
		} else {
			s.wsService.SendToClient(types.WSResponse{
				Namespace: consts.NAMESPACE_OLLAMA,
				Event:     consts.EVENT_OLLAMA_PULL_COMPLETED,
				Data: types.OllamaResponse{
					ID:      key,
					Status:  "done",
					Error:   false,
					Message: "success",
				},
			})
		}
	}()

	return nil
}
