package ollama

import (
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
				s.wsService.SendToClient(key, types.OllamaResponse{
					ID:        key,
					Status:    resp.Status,
					Digest:    resp.Digest,
					Total:     resp.Total,
					Completed: resp.Completed,
				}.WSResponseMessage())
			}
			return nil
		})

		if err != nil {
			log.Println("Error during pull:", err)
			// send error message to frontend
			s.wsService.SendToClient(key, types.OllamaResponse{
				ID:      key,
				Status:  "error",
				Error:   true,
				Message: err.Error(),
			}.WSResponseMessage())
		}
	}()

	return nil
}
