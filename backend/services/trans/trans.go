package trans

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/llms"
	"CanMe/backend/pkg/llms/ollama"
	"CanMe/backend/pkg/llms/openailike"
	"CanMe/backend/pkg/subs/others"
	innerinterfaces "CanMe/backend/services/innerInterfaces"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	stringutil "CanMe/backend/utils/stringUtil"
	timeutil "CanMe/backend/utils/timeUtil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/asticode/go-astisub"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

type WorkQueue struct {
	mutex sync.RWMutex

	ctx        context.Context
	wsService  innerinterfaces.WebSocketServiceInterface
	pending    chan *TranslationWorkQueue
	processing map[string]*TranslationWorkQueue
	completed  chan *TranslationWorkQueue
}

type TranslationWorkQueue struct {
	ctx                 context.Context
	client              llms.Interface
	SubtitleInfo        SubtitleInfo              `json:"subtitle_info"`
	OriginalCaptions    *astisub.Subtitles        `json:"original_captions"`
	TranslationStatus   storage.TranslationStatus `json:"translation_status"`
	TranslationProgress float64                   `json:"translation_progress"`
	ActionDescription   string                    `json:"action_description"`
}

type SubtitleInfo struct {
	Key      string `json:"key"`
	FileName string `json:"file_name"`
	Language string `json:"language"`
	Stream   bool   `json:"stream"`
	Models   string `json:"models"`
	Brief    string `json:"brief"`
	Captions string `json:"captions"`
}

func New() *WorkQueue {
	return &WorkQueue{
		pending:    make(chan *TranslationWorkQueue, consts.TRANSLATION_WORK_QUEUE_MAX_SIZE),
		processing: make(map[string]*TranslationWorkQueue, consts.TRANSLATION_WORK_QUEUE_MAX_SIZE),
		completed:  make(chan *TranslationWorkQueue, consts.TRANSLATION_WORK_QUEUE_MAX_SIZE),
	}
}

// Process run when service start
func (wq *WorkQueue) Process(ctx context.Context, wsService innerinterfaces.WebSocketServiceInterface) {
	wq.ctx = ctx
	wq.wsService = wsService
	go func() {
		for {
			select {
			case <-wq.ctx.Done():
				return
			case info := <-wq.pending:
				// process processing map
				wq.setProcessingInfo(info.SubtitleInfo.Key, info)

				// handle translation
				go func() {
					err := wq.handleTranslation(info.SubtitleInfo.Key)
					if err != nil {
						runtime.LogError(wq.ctx, "translation worker translation error: "+err.Error())
					}
				}()

			case info := <-wq.completed:
				go func() {
					err := wq.handleCompleted(info)
					if err != nil {
						// handle error
						runtime.LogError(wq.ctx, "translation worker completed error: "+err.Error())
					}
				}()
			}
		}
	}()
}

// AddTranslation add translation to work queue by backend from websocket etc.
func (wq *WorkQueue) AddTranslation(id, originalSubId, lang string) (err error) {
	// request params check
	if id == "" {
		return fmt.Errorf("id is empty")
	}

	if originalSubId == "" {
		return fmt.Errorf("original subtitle id is empty")
	}

	if lang == "" {
		return fmt.Errorf("language is empty")
	}

	// get client
	client, cLLM, cModel, err := wq.currentModelClient()
	if err != nil {
		return err
	}

	// get original subtitle
	originalSub := storage.Subtitles{}
	err = originalSub.Read(wq.ctx, originalSubId)
	if err != nil {
		return fmt.Errorf("read original subtitle failed: " + err.Error())
	}

	// unmarshal captions
	var captions *astisub.Subtitles
	if err = json.Unmarshal([]byte(originalSub.Captions), &captions); err != nil {
		return fmt.Errorf("unmarshal captions failed: " + err.Error())
	}

	// initialize tranlation subtitle
	transSub := &TranslationWorkQueue{
		ctx:    wq.ctx,
		client: client,
		SubtitleInfo: SubtitleInfo{
			Key:      id,
			FileName: originalSub.FileName + "_" + lang,
			Language: lang,
			Stream:   true,
			Models:   cLLM + ":" + cModel,
			Brief:    "",
			Captions: "",
		},
		OriginalCaptions:    captions,
		TranslationStatus:   storage.StatusRunning,
		TranslationProgress: 0,
		ActionDescription:   "",
	}

	// push to pending queue
	wq.pending <- transSub

	return
}

// CancelTranslation cancel translation by frontend
func (wq *WorkQueue) CancelTranslation(key string) (resp types.JSResp) {
	if info, ok := wq.getProcessingInfo(key); ok {
		// update translation info
		info.TranslationStatus = storage.StatusCanceled
		info.ActionDescription = string(storage.StatusCanceled) + "by user"

		// push to completed queue
		wq.completed <- info

		// remove processing info
		wq.removeProcessingInfo(key)
	} else {
		resp.Msg = "translation not found"
		return
	}

	resp.Success = true
	return
}

func (wq *WorkQueue) handleTranslation(id string) (err error) {
	// get processing info
	info, ok := wq.getProcessingInfo(id)
	if !ok {
		return fmt.Errorf("translation: %v not found", id)
	}

	// original captions check
	var oCapLen int
	var items []*astisub.Item
	if i := info.OriginalCaptions.Items; i == nil || len(i) < 1 {
		return fmt.Errorf("original captions not found")
	} else {
		oCapLen = len(i)
		items = i
	}

	// for loop translation
	var briefTimes int
	for i, item := range items {
		info, ok := wq.getProcessingInfo(id)
		// map check
		if !ok {
			// if !ok, tranlate work is not processcing status, log and return
			runtime.LogInfof(wq.ctx, "translation: %v not found", id)
			return
		}

		// wss emit: 1.captions timestamp emit
		progress := float64(i+1) / float64(oCapLen) * 100
		timestamp := fmt.Sprintf("%d\n%s --> %s\n", i+1, timeutil.FormatDurationSRT(item.StartAt), timeutil.FormatDurationSRT(item.EndAt))
		wq.wsService.SendToClient(id, types.TranslateResponse{
			ID:       id,
			Content:  timestamp,
			Status:   string(storage.StatusRunning),
			Progress: progress,
		}.WSResponseMessage())

		// wss emit: 2.captions text
		var captionText string
		transCompleted := false
		translatedTextChan, errChan := info.client.ChatCompletionStream(info.ctx, consts.TranslatePrompt(info.SubtitleInfo.Language), item.String())
		for !transCompleted {
			select {
			case err, ok := <-errChan:
				// if channel is closed, break to stop this for loop
				if !ok {
					transCompleted = true
					break
				}

				if err != nil {
					// update translation info
					info.TranslationStatus = storage.StatusError
					info.ActionDescription = string(storage.StatusError) + ": " + err.Error()
					info.TranslationProgress = progress

					// 2.push to complated channel
					wq.completed <- info

					// 3.remove current id porgresscing
					wq.removeProcessingInfo(id)

					// 4.return error(break all loops and return)
					return err
				}

			case text, ok := <-translatedTextChan:
				// if channel is closed, break
				if !ok {
					transCompleted = true
					break
				}

				wq.wsService.SendToClient(id, types.TranslateResponse{
					ID:       id,
					Content:  text,
					Status:   string(storage.StatusRunning),
					Progress: progress,
				}.WSResponseMessage())

				// append text to captionText
				captionText += text
			}
		}

		// wss emit: 3.captions spaces
		wq.wsService.SendToClient(id, types.TranslateResponse{
			ID:       id,
			Content:  "\n\n",
			Status:   string(storage.StatusRunning),
			Progress: progress,
		}.WSResponseMessage())

		// generate brief
		if briefTimes < 3 {
			info.SubtitleInfo.Brief += captionText
			briefTimes++
		}

		// generate captions
		info.SubtitleInfo.Captions += timestamp + captionText + "\n\n"
		info.TranslationProgress = progress

		// save map
		_, err = wq.updateProcessingInfo(id, info)
		if err != nil {
			runtime.LogError(info.ctx, "update processing info failed: "+err.Error())
			return
		}

		// wss emit: 4.translation progress
		wq.wsService.SendToClient(id, types.TranslateResponse{
			ID:       id,
			Status:   string(storage.StatusRunning),
			Progress: progress,
		}.WSResponseMessage())

		// if last item, push to completed queue
		if i == oCapLen-1 {
			// update translation info
			info.TranslationStatus = storage.StatusCompleted
			info.ActionDescription = string(storage.StatusCompleted)
			info.TranslationProgress = 100

			// push to complated channel
			wq.completed <- info

			// remove processing info
			wq.removeProcessingInfo(id)

			// return
			return nil
		}
	}

	return
}

func (wq *WorkQueue) handleCompleted(info *TranslationWorkQueue) (err error) {
	// convert srt to astisub
	subs := &others.Others{}
	captions, err := subs.Format(info.ctx, info.SubtitleInfo.FileName+".srt", info.SubtitleInfo.Captions)
	if err != nil {
		runtime.LogError(info.ctx, "convert srt to astisub failed: "+err.Error())
		return
	}

	captionsByte, err := json.Marshal(captions)
	if err != nil {
		runtime.LogError(info.ctx, "marshal json failed: "+err.Error())
		return
	}

	sub := &storage.Subtitles{}
	// save to db
	err = sub.Read(info.ctx, info.SubtitleInfo.Key)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // if not found, create
			persistSub := storage.Subtitles{
				Key:                 info.SubtitleInfo.Key,
				FileName:            info.SubtitleInfo.FileName,
				Language:            info.SubtitleInfo.Language,
				Stream:              info.SubtitleInfo.Stream,
				Models:              info.SubtitleInfo.Models,
				Brief:               info.SubtitleInfo.Brief,
				Captions:            string(captionsByte),
				TranslationStatus:   info.TranslationStatus,
				TranslationProgress: info.TranslationProgress,
				ActionDescription:   info.ActionDescription,
			}
			err = persistSub.Create(info.ctx)
			if err != nil {
				runtime.LogError(info.ctx, "create translation subtitle failed: "+err.Error())
			}
		} else {
			runtime.LogError(info.ctx, "update translation subtitle failed: "+err.Error())
		}
	} else { // if found, update
		// update items
		sub.Brief = info.SubtitleInfo.Brief
		sub.Captions = string(captionsByte)
		sub.TranslationStatus = info.TranslationStatus
		sub.TranslationProgress = info.TranslationProgress
		sub.ActionDescription = info.ActionDescription

		// update to db
		err = sub.Update(info.ctx)
		if err != nil {
			runtime.LogError(info.ctx, "update translation subtitle failed: "+err.Error())
		}
	}
	// wss final status emit
	var emitString string
	switch info.TranslationStatus {
	case storage.StatusCanceled: // emit cancel
		emitString = types.TranslateResponse{
			ID:       info.SubtitleInfo.Key,
			Status:   string(storage.StatusCanceled),
			Progress: info.TranslationProgress,
			Message:  "Task Canceled",
		}.WSResponseMessage()
	case storage.StatusCompleted: // emit complate
		emitString = types.TranslateResponse{
			ID:       info.SubtitleInfo.Key,
			Status:   string(storage.StatusCompleted),
			Progress: info.TranslationProgress,
			Message:  "Task Completed",
		}.WSResponseMessage()
	case storage.StatusError: // emit error
		emitString = types.TranslateResponse{
			ID:       info.SubtitleInfo.Key,
			Status:   string(storage.StatusError),
			Progress: info.TranslationProgress,
			Error:    true,
			Message:  info.ActionDescription,
		}.WSResponseMessage()
	default:
		runtime.LogError(info.ctx, "translation worker handle completed unknown status")
		return
	}

	// wss emit
	wq.wsService.SendToClient(info.SubtitleInfo.Key, emitString)

	// remove from processing list
	wq.wsService.RemoveTranslation(info.SubtitleInfo.Key)

	return
}

func (wq *WorkQueue) currentModelClient() (client llms.Interface, cLLM, cModel string, err error) {
	currentModel := storage.CurrentModel{}
	err = currentModel.Read(wq.ctx)
	if err != nil {
		return nil, "", "", fmt.Errorf("read current model failed: " + err.Error())
	}
	cLLM = currentModel.LLMName
	cModel = currentModel.ModelName

	if cLLM != "" && cModel != "" {
		llm := storage.LLM{}
		err = llm.Read(wq.ctx, cLLM)
		if err != nil {
			return nil, "", "", fmt.Errorf("read llm failed: " + err.Error())
		}
		baseURL := llm.BaseURL
		token := llm.APIKey

		if baseURL != "" && token != "" {
			if cLLM == "ollama" {
				client = ollama.New(cModel, stringutil.OllamaHost(baseURL))
			} else {
				client = openailike.New(token, baseURL, cModel)
			}
		} else {
			return nil, "", "", fmt.Errorf("llm baseurl or token is empty")
		}
	} else {
		return nil, "", "", fmt.Errorf("current model is not set")
	}

	return
}

// getProcessingInfo
func (wq *WorkQueue) getProcessingInfo(key string) (*TranslationWorkQueue, bool) {
	wq.mutex.RLock()
	defer wq.mutex.RUnlock()
	info, ok := wq.processing[key]
	return info, ok
}

// setProcessingInfo
func (wq *WorkQueue) setProcessingInfo(key string, info *TranslationWorkQueue) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()
	wq.processing[key] = info
}

func (wq *WorkQueue) updateProcessingInfo(key string, info *TranslationWorkQueue) (success bool, err error) {
	if _, ok := wq.processing[key]; !ok {
		return false, fmt.Errorf("translation: %v not found", key)
	}
	wq.mutex.Lock()
	defer wq.mutex.Unlock()
	wq.processing[key] = info
	return true, nil
}

// removeProcessingInfo
func (wq *WorkQueue) removeProcessingInfo(key string) {
	wq.mutex.Lock()
	defer wq.mutex.Unlock()
	delete(wq.processing, key)
}
