package types

import (
	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/downinfo"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/proxy"
)

type Preferences struct {
	Behavior    PreferencesBehavior  `json:"behavior" yaml:"behavior"`
	General     PreferencesGeneral   `json:"general" yaml:"general"`
	Proxy       proxy.Config         `json:"proxy" yaml:"proxy"`
	Download    downinfo.Config      `json:"download" yaml:"download"`
	Logger      logger.Config        `json:"logger" yaml:"logger"`
	ListendInfo ListendInfo          `json:"listendInfo" yaml:"listend_info"`
	Telemetry   TelemetryPreferences `json:"telemetry" yaml:"telemetry"`
}

func NewPreferences() Preferences {
	return Preferences{
		Behavior: PreferencesBehavior{
			WindowWidth:  consts.DEFAULT_WINDOW_WIDTH,
			WindowHeight: consts.DEFAULT_WINDOW_HEIGHT,
		},
		General: PreferencesGeneral{
			Appearance:  "auto",
			Theme:       "blue",
			Language:    "auto",
			CheckUpdate: true,
		},
		Proxy: proxy.Config{
			Type: "system", // default use system proxy
		},
		Download: downinfo.Config{
			Dir: downinfo.GetDefaultDownloadDir(),
		},
		Logger:      *logger.DefaultConfig(),
		ListendInfo: DefaultListendInfo(),
		Telemetry: TelemetryPreferences{
			Enabled: true,
		},
	}
}

type PreferencesBehavior struct {
	Welcomed        bool `json:"welcomed" yaml:"welcomed"`
	WindowWidth     int  `json:"windowWidth" yaml:"window_width"`
	WindowHeight    int  `json:"windowHeight" yaml:"window_height"`
	WindowMaximised bool `json:"windowMaximised" yaml:"window_maximised"`
	WindowPosX      int  `json:"windowPosX" yaml:"window_pos_x"`
	WindowPosY      int  `json:"windowPosY" yaml:"window_pos_y"`
}

type PreferencesGeneral struct {
	// Appearance controls light/dark/auto color scheme
	Appearance string `json:"appearance" yaml:"appearance"`
	// Theme controls accent/highlight color (e.g., blue, purple)
	Theme string `json:"theme" yaml:"theme"`
	Language    string `json:"language" yaml:"language"`
	CheckUpdate bool   `json:"checkUpdate" yaml:"check_update"`
	SkipVersion string `json:"skipVersion" yaml:"skip_version,omitempty"`
}

type TelemetryPreferences struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	ClientID string `json:"clientId" yaml:"client_id"`
	AppID    string `json:"appId,omitempty" yaml:"-"`
	Endpoint string `json:"endpoint,omitempty" yaml:"-"`
	Version  string `json:"version,omitempty" yaml:"-"`
}

type ListendInfo struct {
	WS  ListendInfoDetails `json:"ws" yaml:"ws"`
	MCP ListendInfoDetails `json:"mcp" yaml:"mcp"`
}

type ListendInfoDetails struct {
	Protocol string `json:"protocol" yaml:"protocol"`
	IP       string `json:"ip" yaml:"ip"`
	Port     int    `json:"port" yaml:"port"`
	Path     string `json:"path" yaml:"path"`
}

func DefaultListendInfo() ListendInfo {
	return ListendInfo{
		WS: ListendInfoDetails{
			Protocol: "ws",
			IP:       "localhost",
			Port:     consts.WS_PORT,
			Path:     "ws",
		},
		MCP: ListendInfoDetails{
			Protocol: "http",
			IP:       "localhost",
			Port:     consts.MCP_SERVER_PORT,
			Path:     "sse",
		},
	}
}
