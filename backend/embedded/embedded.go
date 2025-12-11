package embedded

import (
	"embed"
	"fmt"
	"runtime"

	"dreamcreator/backend/consts"
	"dreamcreator/backend/types"
)

//go:embed binaries/*
var binaries embed.FS

// 提供全局访问函数
func GetEmbeddedBinaries() embed.FS {
	return binaries
}

func GetEmbeddedBinaryVersion(dpType types.DependencyType) (string, error) {
	osType := runtime.GOOS
	switch dpType {
	case types.DependencyYTDLP:
		return consts.YtdlpEmbedVersion(osType)
	case types.DependencyFFmpeg:
		return consts.FfmpegEmbedVersion(osType)
	case types.DependencyDeno:
		return consts.DenoEmbedVersion(osType)

	default:
		return "", fmt.Errorf("unsupported dependency type: %s", dpType)
	}
}
