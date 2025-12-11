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

// Deno镜像源配置
var DenoMirrors = map[string]Mirror{
	"github": {
		Name:        "github",
		DisplayName: "GitHub Official",
		Description: "Official Deno Release",
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

// Deno下载URL模板配置
var DenoDownloadTemplates = map[string]string{
	"github":  "https://github.com/denoland/deno/releases/download/{version}/{filename}",
	"ghproxy": "https://gh-proxy.com/github.com/denoland/deno/releases/download/{version}/{filename}",
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
	"deno": {
		"darwin":  "ghproxy",
		"windows": "ghproxy",
		"linux":   "ghproxy",
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

// GetDenoFileName 根据平台构建 Deno 发行文件名
func GetDenoFileName(osType, arch string) string {
	switch osType {
	case "windows":
		switch arch {
		case "amd64":
			return "deno-x86_64-pc-windows-msvc.zip"
		case "arm64":
			return "deno-aarch64-pc-windows-msvc.zip"
		default:
			return "deno-x86_64-pc-windows-msvc.zip"
		}
	case "darwin":
		switch arch {
		case "amd64":
			return "deno-x86_64-apple-darwin.zip"
		case "arm64":
			return "deno-aarch64-apple-darwin.zip"
		default:
			return "deno-x86_64-apple-darwin.zip"
		}
	case "linux":
		switch arch {
		case "amd64":
			return "deno-x86_64-unknown-linux-gnu.zip"
		case "arm64":
			return "deno-aarch64-unknown-linux-gnu.zip"
		default:
			return "deno-x86_64-unknown-linux-gnu.zip"
		}
	default:
		return "deno"
	}
}

// BuildDenoDownloadURL 构建 Deno 下载URL
func BuildDenoDownloadURL(mirror, version, osType, arch string) (string, error) {
	template, exists := DenoDownloadTemplates[mirror]
	if !exists {
		return "", fmt.Errorf("unsupported mirror: %s", mirror)
	}

	filename := GetDenoFileName(osType, arch)
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

// GetDenoAPIURL 返回 Deno Releases API 地址
func GetDenoAPIURL() (string, error) {
	url := "https://api.github.com/repos/denoland/deno/releases/latest"
	return url, nil
}

// Dependencies Embedded Versions
const (
	EMBEDDED_YTDLP_VERSION          = "2025.12.08"
	EMBEDDED_FFMPEG_VERSION_DARWIN  = "121793-g1eb2cbd865"
	EMBEDDED_FFMPEG_VERSION_WINDOWS = "7.1.2-4"
	EMBEDDED_DENO_VERSION           = "v2.5.6"

	// YTDLP_EJS_MIN_VERSION 定义支持 EJS 的 yt-dlp 最低版本
	YTDLP_EJS_MIN_VERSION = "2025.11.22"
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

func DenoEmbedVersion(osType string) (string, error) {
	return EMBEDDED_DENO_VERSION, nil
}
