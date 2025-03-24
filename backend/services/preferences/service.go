package preferences

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/downinfo"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/pkg/proxy"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

type Service struct {
	pref          *storage.PreferencesStorage
	clientVersion string

	ctx            context.Context
	proxyClient    *proxy.Client
	downloadClient *downinfo.Client
}

var preferences *Service
var oncePreferences sync.Once

func New() *Service {
	if preferences == nil {
		oncePreferences.Do(func() {
			preferences = &Service{
				pref:          storage.NewPreferences(),
				clientVersion: consts.APP_VERSION,
				proxyClient:   nil,
			}

			// 初始化日志系统
			pref := preferences.pref.GetPreferences()
			if err := logger.InitLogger(&pref.Logger); err != nil {
				fmt.Printf("初始化日志系统失败: %v\n", err)
			}

			// 初始化其他全局配置
			preferences.updateDownloadConfig()
		})
	}
	return preferences
}

func (p *Service) SetPackageClients(proxyClient *proxy.Client, downloadClient *downinfo.Client) {
	// proxy
	p.proxyClient = proxyClient
	p.OnProxyChanged(func(config *proxy.Config) {
		p.proxyClient.SetConfig(config)
	})

	// 初始化代理配置
	p.CheckAndUpdateProxy()

	// download info
	p.downloadClient = downloadClient
	p.OnDownloadInfoChanged(func(config *downinfo.Config) {
		p.downloadClient.SetConfig(config)
	})

	// 初始化下载配置
	p.CheckAndUpdateDownloadInfo()

}

func (p *Service) SetContext(ctx context.Context) {
	p.ctx = ctx
}

func (p *Service) GetPreferences() (resp types.JSResp) {
	resp.Data = p.pref.GetPreferences()
	resp.Success = true
	return
}

func (p *Service) SetPreferences(pf types.Preferences) (resp types.JSResp) {
	err := p.pref.SetPreferences(&pf)
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	p.UpdateEnv()
	p.UpdateGlobalConfig()
	resp.Success = true
	return
}

func (p *Service) UpdatePreferences(value map[string]any) (resp types.JSResp) {
	err := p.pref.UpdatePreferences(value)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
	return
}

func (p *Service) RestorePreferences() (resp types.JSResp) {
	defaultPref := p.pref.RestoreDefault()
	resp.Data = map[string]any{
		"pref": defaultPref,
	}
	resp.Success = true
	return
}

func (p *Service) GetLanguage() string {
	pref := p.pref.GetPreferences()
	return pref.General.Language
}

func (p *Service) SetAppVersion(ver string) {
	if !strings.HasPrefix(ver, "v") {
		p.clientVersion = "v" + ver
	} else {
		p.clientVersion = ver
	}
}

func (p *Service) GetAppVersion() (resp types.JSResp) {
	resp.Success = true
	resp.Data = map[string]any{
		"version": p.clientVersion,
	}
	return
}

func (p *Service) SaveWindowSize(width, height int, maximised bool) {
	if maximised {
		// do not update window size if maximised state
		p.UpdatePreferences(map[string]any{
			"behavior.windowMaximised": true,
		})
	} else if width >= consts.MIN_WINDOW_WIDTH && height >= consts.MIN_WINDOW_HEIGHT {
		p.UpdatePreferences(map[string]any{
			"behavior.windowWidth":     width,
			"behavior.windowHeight":    height,
			"behavior.windowMaximised": false,
		})
	}
}

func (p *Service) GetWindowSize() (width, height int, maximised bool) {
	data := p.pref.GetPreferences()
	width, height, maximised = data.Behavior.WindowWidth, data.Behavior.WindowHeight, data.Behavior.WindowMaximised
	if width <= 0 {
		width = consts.DEFAULT_WINDOW_WIDTH
	}
	if height <= 0 {
		height = consts.DEFAULT_WINDOW_HEIGHT
	}
	return
}

func (p *Service) GetWindowPosition(ctx context.Context) (x, y int) {
	data := p.pref.GetPreferences()
	x, y = data.Behavior.WindowPosX, data.Behavior.WindowPosY
	width, height := data.Behavior.WindowWidth, data.Behavior.WindowHeight
	var screenWidth, screenHeight int
	if screens, err := runtime.ScreenGetAll(ctx); err == nil {
		for _, screen := range screens {
			if screen.IsCurrent {
				screenWidth, screenHeight = screen.Size.Width, screen.Size.Height
				break
			}
		}
	}
	if screenWidth <= 0 || screenHeight <= 0 {
		screenWidth, screenHeight = consts.DEFAULT_WINDOW_WIDTH, consts.DEFAULT_WINDOW_HEIGHT
	}
	if x <= 0 || x+width > screenWidth || y <= 0 || y+height > screenHeight {
		// out of screen, reset to center
		x, y = (screenWidth-width)/2, (screenHeight-height)/2
	}
	return
}

func (p *Service) SaveWindowPosition(x, y int) {
	if x > 0 || y > 0 {
		p.UpdatePreferences(map[string]any{
			"behavior.windowPosX": x,
			"behavior.windowPosY": y,
		})
	}
}

type latestRelease struct {
	Name    string `json:"name"`
	TagName string `json:"tag_name"`
	Url     string `json:"url"`
	HtmlUrl string `json:"html_url"`
}

func (p *Service) CheckForUpdate() (resp types.JSResp) {
	// request latest version
	res, err := http.Get(consts.CHECK_UPDATE_URL)
	if err != nil || res.StatusCode != http.StatusOK {
		resp.Msg = "network error"
		return
	}

	var respObj latestRelease
	err = json.NewDecoder(res.Body).Decode(&respObj)
	if err != nil {
		resp.Msg = "invalid content"
		return
	}

	// compare with current version
	resp.Success = true
	resp.Data = map[string]any{
		"version":  p.clientVersion,
		"latest":   respObj.TagName,
		"page_url": respObj.HtmlUrl,
	}
	return
}

// UpdateEnv Update System Environment
func (p *Service) UpdateEnv() {
	if p.GetLanguage() == "zh" {
		os.Setenv("LANG", "zh_CN.UTF-8")
	} else {
		os.Unsetenv("LANG")
	}
}

func (p *Service) updateDownloadConfig() {
	pref := p.pref.GetPreferences()
	downloadDir := pref.Download.Dir
	if len(downloadDir) > 0 {
		if p.downloadClient != nil {
			p.downloadClient.SetDir(downloadDir)
		}
	}
}

// UpdateGlobalConfig 更新全局配置（不包括日志配置）
func (p *Service) UpdateGlobalConfig() {
	p.updateDownloadConfig()
}

// SetLoggerConfig 更新日志配置
func (p *Service) SetLoggerConfig(config logger.Config) (resp types.JSResp) {
	// 检查配置是否真的变化了
	pref := p.pref.GetPreferences()
	if reflect.DeepEqual(pref.Logger, config) {
		resp.Success = true
		return
	}

	// 保存新配置
	pref.Logger = config
	err := p.pref.SetPreferences(&pref)
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	// 重新初始化日志系统
	if err := logger.InitLogger(&config); err != nil {
		logger.Error("Failed to update logger config", zap.Error(err))
		resp.Msg = err.Error()
		return
	}

	resp.Success = true
	return
}
