package languages

import (
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"context"
	"encoding/json"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Service struct {
	ctx context.Context
}

func New() *Service {
	return &Service{}
}

func (s *Service) RegisterService(ctx context.Context) {
	s.ctx = ctx
}

func (s *Service) GetLanguage() (resp types.JSResp) {
	langs := storage.NewLanguage()
	err := langs.List(s.ctx)
	// if error, return default language and create it
	if err != nil || langs == nil {
		langs = storage.DefaultLanguage()
		err = langs.Create(s.ctx)
		if err != nil {
			runtime.LogError(s.ctx, "create language failed: "+err.Error())
		}
	}

	langsByte, err := json.Marshal(langs)
	if err != nil {
		runtime.LogError(s.ctx, "marshal language failed: "+err.Error())
	}

	resp.Success = true
	resp.Data = string(langsByte)
	return
}

func (s *Service) AddLanguage(group, lang string) (resp types.JSResp) {
	langData, err := storage.NewLanguageData(group, lang)
	if err != nil {
		resp.Msg = "Create language failed: " + err.Error()
		return
	}

	exist, err := langData.Exist(s.ctx)
	if err != nil {
		resp.Msg = "Check language exist failed: " + err.Error()
		return
	}

	if exist {
		resp.Msg = "Language already exists"
		return
	}
	err = langData.Create(s.ctx)
	if err != nil {
		resp.Msg = "Create language failed: " + err.Error()
		return
	}

	langs := storage.NewLanguage()
	err = langs.List(s.ctx)
	if err != nil {
		resp.Msg = "Get language failed: " + err.Error()
		return
	}

	langsByte, err := json.Marshal(langs)
	if err != nil {
		runtime.LogError(s.ctx, "marshal language failed: "+err.Error())
	}

	resp.Success = true
	resp.Data = string(langsByte)
	return
}

func (s *Service) UpdateLanguage(group, lang string) (resp types.JSResp) {
	langData, err := storage.NewLanguageData(group, lang)
	if err != nil {
		resp.Msg = "Create language failed: " + err.Error()
		return
	}

	err = langData.Read(s.ctx)
	if err != nil {
		resp.Msg = "Language not found"
		return
	}

	err = langData.Update(s.ctx)
	if err != nil {
		resp.Msg = "Update language failed: " + err.Error()
		return
	}

	langs := storage.NewLanguage()
	err = langs.List(s.ctx)
	if err != nil {
		resp.Msg = "Get language failed: " + err.Error()
		return
	}

	langsByte, err := json.Marshal(langs)
	if err != nil {
		runtime.LogError(s.ctx, "marshal language failed: "+err.Error())
	}

	resp.Success = true
	resp.Data = string(langsByte)
	return
}
func (s *Service) DeleteLanguage(group, lang string) (resp types.JSResp) {
	langData, err := storage.NewLanguageData(group, lang)
	if err != nil {
		resp.Msg = "Create language failed: " + err.Error()
		return
	}

	err = langData.Read(s.ctx)
	if err != nil {
		resp.Msg = "Language not found"
		return
	}

	err = langData.Delete(s.ctx)
	if err != nil {
		resp.Msg = "Delete language failed: " + err.Error()
		return
	}

	resp.Success = true
	return
}
