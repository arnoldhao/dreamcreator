package ollama

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	ii "CanMe/backend/services/innerinterfaces"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	stringutil "CanMe/backend/utils/stringUtil"

	"github.com/ollama/ollama/api"
)

type Service struct {
	ctx       context.Context
	ollama    *api.Client
	wsService ii.WebSocketServiceInterface
}

func New() *Service {
	return &Service{}
}

func (s *Service) RegisterServices(
	ctx context.Context,
	wsService ii.WebSocketServiceInterface,
) {
	s.ctx = ctx
	s.wsService = wsService
	host := storage.Ollama{}
	err := host.Read(ctx)
	if err != nil {
		log.Println("Error reading ollama host:", err)
	}
	s.ollama = api.NewClient(stringutil.OllamaHost(host.Schema+"://"+host.Domain+":"+host.Port), http.DefaultClient)
}

func (s *Service) GetHost() (resp types.JSResp) {
	o := storage.Ollama{}
	err := o.Read(s.ctx)
	if err != nil {
		log.Println("Error reading ollama host:", err)
	}

	if o.Schema == "" {
		o = storage.Ollama{
			Schema: "http",
			Domain: "localhost",
			Port:   "11434",
		}

		err = o.Create(s.ctx)
		if err != nil {
			log.Println("Error creating ollama host:", err)
		}
		log.Println("Default ollama host:", o.Schema+"://"+o.Domain+":"+o.Port)
	}
	resp.Success = true
	resp.Data = o.Schema + "://" + o.Domain + ":" + o.Port
	return
}

func (s *Service) SetHost(host string) (resp types.JSResp) {
	url := stringutil.OllamaHost(host)
	o := storage.Ollama{
		Schema: url.Scheme,
		Domain: url.Hostname(),
		Port:   url.Port(),
	}

	err := o.Update(s.ctx)
	if err != nil {
		log.Println("Error setting ollama host:", err)
	}

	resp.Success = true
	resp.Data = o.Schema + "://" + o.Domain + ":" + o.Port
	return
}

func (s *Service) Heartbeat() (resp types.JSResp) {
	err := s.ollama.Heartbeat(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	return
}

func (s *Service) List() (resp types.JSResp) {
	models, err := s.ollama.List(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	if len(models.Models) == 0 {
		resp.Success = false
		resp.Msg = "no models found"
		return
	}

	modelsByte, err := json.Marshal(models.Models)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
	resp.Data = string(modelsByte)
	return
}

func (s *Service) ListRunning() (resp types.JSResp) {
	models, err := s.ollama.ListRunning(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	if len(models.Models) == 0 {
		resp.Success = false
		resp.Msg = "no models found"
		return
	}

	modelsByte, err := json.Marshal(models.Models)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Data = string(modelsByte)
	return
}

func (s *Service) Delete(model string) (resp types.JSResp) {
	err := s.ollama.Delete(s.ctx, &api.DeleteRequest{Name: strings.ToLower(model)})
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	return
}

func (s *Service) Show(model string) (resp types.JSResp) {
	show, err := s.ollama.Show(s.ctx, &api.ShowRequest{Name: strings.ToLower(model)})
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	showByte, err := json.Marshal(show)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Data = string(showByte)
	return
}
