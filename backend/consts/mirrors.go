package consts

import (
	"fmt"
	"strings"
)

// Mirror 镜像源信息
type Mirror struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Region      string `json:"region"`
	Speed       string `json:"speed"`
}

// FFmpeg镜像源配置
var FFmpegMirrors = map[string]Mirror{
	"evermeet": {
		Name:        "evermeet",
		DisplayName: "Evermeet",
		Description: "macOS Official Build",
		Region:      "Global",
		Speed:       "midium",
	},
	"ghproxy": {
		Name:        "ghproxy",
		DisplayName: "GitHub Proxy",
		Description: "Github Proxy Mirror",
		Region:      "Global",
		Speed:       "fast",
	},
}

// YTDLP镜像源配置
var YTDLPMirrors = map[string]Mirror{
	"github": {
		Name:        "github",
		DisplayName: "GitHub Official",
		Description: "Official yt-dlp Release",
		Region:      "Global",
		Speed:       "medium",
	},
	"ghproxy": {
		Name:        "ghproxy",
		DisplayName: "GitHub Proxy",
		Description: "GitHub Proxy Mirror",
		Region:      "China",
		Speed:       "fast",
	},
}

// FFmpeg下载URL配置
var FFmpegDownloadURLs = map[string]map[string]map[string]string{
	"windows": {
		"amd64": {
			"ghproxy": "https://gh-proxy.com/github.com/jellyfin/jellyfin-ffmpeg/releases/download/{version}/{filename}.zip",
		},
		"arm64": {
			"ghproxy": "https://gh-proxy.com/github.com/jellyfin/jellyfin-ffmpeg/releases/download/{version}/{filename}.zip",
		},
	},
	"darwin": {
		"amd64": {
			"evermeet": "https://evermeet.cx/ffmpeg/{filename}-{version}.zip",
		},
		"arm64": {
			"evermeet": "https://evermeet.cx/ffmpeg/{filename}-{version}.zip",
		},
	},
}

var FFmpegAPIURLs = map[string]map[string]map[string]string{
	"windows": {
		"amd64": {
			"ghproxy": "https://api.github.com/repos/jellyfin/jellyfin-ffmpeg/releases/latest",
		},
		"arm64": {
			"ghproxy": "https://api.github.com/repos/jellyfin/jellyfin-ffmpeg/releases/latest",
		},
	},
	"darwin": {
		"amd64": {
			"evermeet": "https://evermeet.cx/ffmpeg/info/ffmpeg/snapshot",
		},
		"arm64": {
			"evermeet": "https://evermeet.cx/ffmpeg/info/ffmpeg/snapshot",
		},
	},
}

// YTDLP下载URL模板配置
var YTDLPDownloadTemplates = map[string]string{
	"github":  "https://github.com/yt-dlp/yt-dlp/releases/download/{version}/{filename}",
	"ghproxy": "https://gh-proxy.com/github.com/yt-dlp/yt-dlp/releases/download/{version}/{filename}",
}

// 平台默认镜像源
var DefaultMirrors = map[string]map[string]string{
	"ffmpeg": {
		"darwin":  "evermeet",
		"windows": "ghproxy",
	},
	"yt-dlp": {
		"darwin":  "ghproxy",
		"windows": "ghproxy",
	},
}

// 获取依赖的推荐镜像源
func GetRecommendedMirror(depType, osType string) string {
	if depMirrors, exists := DefaultMirrors[depType]; exists {
		if mirror, exists := depMirrors[osType]; exists {
			return mirror
		}
	}
	return "ghproxy" // 默认回退
}

// 获取YTDLP文件名
func GetYTDLPFileName(osType string) string {
	switch osType {
	case "windows":
		return "yt-dlp.exe"
	case "darwin":
		return "yt-dlp_macos"
	case "linux":
		return "yt-dlp"
	default:
		return "yt-dlp"
	}
}

// 构建YTDLP下载URL
func BuildYTDLPDownloadURL(mirror, version, osType string) (string, error) {
	template, exists := YTDLPDownloadTemplates[mirror]
	if !exists {
		return "", fmt.Errorf("unsupported mirror: %s", mirror)
	}

	filename := GetYTDLPFileName(osType)
	url := strings.ReplaceAll(template, "{version}", version)
	url = strings.ReplaceAll(url, "{filename}", filename)

	return url, nil
}

func GetFFMPEGFileName(osType, version, arch string) string {
	var archSuffix string
	if arch == "arm64" {
		archSuffix = "winarm64"
	} else {
		archSuffix = "win64"
	}

	// 去掉v前缀
	if strings.HasPrefix(version, "v") {
		version = version[1:]
	}

	switch osType {
	case "windows":
		return fmt.Sprintf("jellyfin-ffmpeg_%s_portable_%s-clang-gpl", version, archSuffix)
	case "darwin":
		return "ffmpeg"
	default:
		return "ffmpeg"
	}
}

func BuildFFMPEGDownloadURL(mirror, version, osType, arch string) (string, error) {
	template, exists := FFmpegDownloadURLs[osType][arch][mirror]
	if !exists {
		return "", fmt.Errorf("unsupported mirror: %s", mirror)
	}

	filename := GetFFMPEGFileName(osType, version, arch)
	url := strings.ReplaceAll(template, "{version}", version)
	url = strings.ReplaceAll(url, "{filename}", filename)

	return url, nil
}

func GetFFMPEGAPIURL(mirror, osType, arch string) (string, error) {
	url, exists := FFmpegAPIURLs[osType][arch][mirror]
	if !exists {
		return "", fmt.Errorf("unsupported mirror: %s", mirror)
	}

	return url, nil
}

func GetYTDLPAPIURL() (string, error) {
	url := "https://api.github.com/repos/yt-dlp/yt-dlp/releases/latest"

	return url, nil
}

// Dependencies Embedded Versions
const (
	EMBEDDED_YTDLP_VERSION          = "2025.10.22"
	EMBEDDED_FFMPEG_VERSION_DARWIN  = "121725-g7b18eafabd"
	EMBEDDED_FFMPEG_VERSION_WINDOWS = "7.1.2-4"
)

func YtdlpEmbedVersion(osType string) (string, error) {
	return EMBEDDED_YTDLP_VERSION, nil
}

func FfmpegEmbedVersion(osType string) (string, error) {
	switch osType {
	case "windows":
		return EMBEDDED_FFMPEG_VERSION_WINDOWS, nil
	case "darwin":
		return EMBEDDED_FFMPEG_VERSION_DARWIN, nil
	default:
		return "", fmt.Errorf("unsupported os type: %s", osType)
	}
}
