package llms

import (
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type Service struct {
	ctx context.Context
}

func New() *Service {
	return &Service{}
}

func (s *Service) RegisterServices(ctx context.Context) {
	s.ctx = ctx
}

func (s *Service) ListLLMsAndModels() (resp types.JSResp) {
	llm := storage.LLM{}
	llms, err := llm.ListLLMsAndModels(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Data = llms
	return
}

func (s *Service) GetLLM(name string) (resp types.JSResp) {
	llm := storage.LLM{
		Name: name,
	}
	err := llm.Read(s.ctx, name)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Data = llm
	return
}

func (s *Service) AddLLM(requestData string) (resp types.JSResp) {
	// json unmarshal
	var llmData types.LLM
	err := json.Unmarshal([]byte(requestData), &llmData)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	// check if llm already exists
	llm := storage.LLM{}
	llms, err := llm.List(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}
	for _, l := range llms {
		if l.Name == llmData.Name {
			resp.Msg = fmt.Sprintf("llm %s already exists\n", llmData.Name)
			return
		}
	}

	// add llm
	llm = storage.LLM{
		Name:      llmData.Name,
		Region:    llmData.Region,
		BaseURL:   llmData.BaseURL,
		APIKey:    llmData.APIKey,
		Available: llmData.Available,
		Icon:      llmData.Icon,
		Show:      llmData.Show,
	}

	err = llm.Create(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	// read llm again
	err = llm.Read(s.ctx, llmData.Name)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	// update models
	if len(llmData.Models) > 0 {
		err := s.updateModels(llm.ID, llmData.Models)
		if err != nil {
			resp.Success = false
			resp.Msg = err.Error()
			return
		}
	}

	resp.Success = true
	resp.Data = llm
	return
}

func (s *Service) UpdateLLM(requestData string) (resp types.JSResp) {
	// json unmarshal
	var llmData types.LLM
	err := json.Unmarshal([]byte(requestData), &llmData)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	// check if llm name is empty
	if name := llmData.Name; name == "" {
		resp.Msg = "llm name is empty"
		return
	}

	// check if llm exists
	llm := storage.LLM{}
	llms, err := llm.List(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	exist := false
	for _, l := range llms {
		if l.Name == llmData.Name {
			exist = true
		}
	}

	if !exist {
		resp.Msg = fmt.Sprintf("llm %s not found\n", llmData.Name)
		return
	}

	// update llm
	llm = storage.LLM{
		Name:      llmData.Name,
		Region:    llmData.Region,
		BaseURL:   llmData.BaseURL,
		APIKey:    llmData.APIKey,
		Available: llmData.Available,
		Icon:      llmData.Icon,
		Show:      llmData.Show,
	}

	err = llm.Update(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	// read llm again
	err = llm.Read(s.ctx, llmData.Name)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	// update models
	if len(llmData.Models) > 0 {
		err := s.updateModels(llm.ID, llmData.Models)
		if err != nil {
			resp.Success = false
			resp.Msg = err.Error()
			return
		}
	}

	resp.Success = true
	resp.Data = llm
	return
}

func (s *Service) DeleteLLM(name string) (resp types.JSResp) {
	llm := storage.LLM{
		Name: name,
	}
	err := llm.Delete(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	return
}

func (s *Service) AddModel(llmName, modelName, modelDesc string) (resp types.JSResp) {
	llm := storage.LLM{}
	err := llm.Read(s.ctx, llmName)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	// check if model already exists
	model := storage.Model{}
	models, err := model.ListByLLM(s.ctx, llm.ID)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	if len(models) > 0 {
		for _, m := range models {
			if m.Name == modelName {
				resp.Msg = fmt.Sprintf("model %s already exists\n", modelName)
				return
			}
		}
	}

	// add model
	model = storage.Model{
		Name:        modelName,
		Available:   true,
		Description: modelDesc,
	}

	err = model.Create(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Data = model
	return
}

func (s *Service) UpdateModel(llmName, modelName, modelDesc string) (resp types.JSResp) {
	llm := storage.LLM{}
	err := llm.Read(s.ctx, llmName)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	model := storage.Model{}
	models, err := model.ListByLLM(s.ctx, llm.ID)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	exist := false
	for _, m := range models {
		if m.Name == modelName {
			exist = true
		}
	}

	if !exist {
		resp.Msg = fmt.Sprintf("model %s not found\n", modelName)
		return
	}

	// update model
	model = storage.Model{
		Name:        modelName,
		Available:   true,
		Description: modelDesc,
	}

	err = model.Update(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Data = model
	return
}

func (s *Service) DeleteModel(llmName, modelName string) (resp types.JSResp) {
	llm := storage.LLM{}
	err := llm.Read(s.ctx, llmName)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	model := storage.Model{}
	models, err := model.ListByLLM(s.ctx, llm.ID)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	exist := false
	for _, m := range models {
		if m.Name == modelName {
			exist = true
		}
	}

	if !exist {
		resp.Msg = fmt.Sprintf("model %s not found\n", modelName)
		return
	}

	model = storage.Model{
		Name: modelName,
	}

	err = model.Delete(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	return
}

func (s *Service) GetCurrentModel() (resp types.JSResp) {
	currentModel := storage.CurrentModel{}
	err := currentModel.Read(s.ctx)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	resp.Data = currentModel
	return

}

func (s *Service) UpdateCurrentModel(llmName, modelName string) (resp types.JSResp) {
	if llmName == "" || modelName == "" {
		resp.Success = false
		resp.Msg = "llmName or modelName is empty"
		return
	}

	currentModel := storage.CurrentModel{}
	if err := currentModel.Read(s.ctx); err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		err = currentModel.Create(s.ctx)
		if err != nil {
			resp.Success = false
			resp.Msg = err.Error()
			return
		}
	} else {
		currentModel.LLMName = llmName
		currentModel.ModelName = modelName
		err := currentModel.Update(s.ctx)
		if err != nil {
			resp.Success = false
			resp.Msg = err.Error()
			return
		}
	}

	resp.Success = true
	return
}

func (s *Service) RestoreAIs() (resp types.JSResp) {
	err := storage.DefaultLLMSAndModels(s.ctx)
	if err != nil {
		resp.Success = false
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
	return
}

func (s *Service) updateModels(llmID uint, newModels []*types.Model) error {
	model := storage.Model{}
	existingModels, err := model.ListByLLM(s.ctx, llmID)
	if err != nil {
		return err
	}

	existingModelMap := make(map[string]storage.Model)
	for _, m := range existingModels {
		existingModelMap[m.Name] = m
	}

	// steps
	// 1. update existed models
	// 2. create new models
	// 3. delete models that no longer exist
	for _, newModel := range newModels {
		if existingModel, ok := existingModelMap[newModel.Name]; ok {
			// update existed model
			existingModel.Available = newModel.Available
			existingModel.Description = newModel.Description
			if err := existingModel.Update(s.ctx); err != nil {
				return err
			}
			delete(existingModelMap, newModel.Name)
		} else {
			// create new model
			name := strings.ReplaceAll(newModel.Name, " ", "")
			if name == "" {
				continue
			}
			modelToCreate := storage.Model{
				LLMId:       llmID,
				Name:        name,
				Available:   newModel.Available,
				Description: newModel.Description,
			}
			if err := modelToCreate.Create(s.ctx); err != nil {
				return err
			}
		}
	}

	// delete models that no longer exist
	for _, modelToDelete := range existingModelMap {
		if err := modelToDelete.Delete(s.ctx); err != nil {
			return err
		}
	}

	return nil
}
