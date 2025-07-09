package browercookies

import (
	"CanMe/backend/types"
	"context"
)

// GetYTDLPPath 获取 yt-dlp 可执行文件的文件夹路径
func (c *cookieManager) YTDLPPath(ctx context.Context) (string, error) {
	return c.executablePath(ctx, types.DependencyYTDLP)
}

// YTDLPExecPath 获取 yt-dlp 可执行文件路径
func (c *cookieManager) YTDLPExecPath(ctx context.Context) (string, error) {
	return c.executableExecPath(ctx, types.DependencyYTDLP)
}

// executablePath 获取可执行文件的文件夹路径
func (c *cookieManager) executablePath(ctx context.Context, depType types.DependencyType) (string, error) {
	info, err := c.depManager.Get(ctx, depType)
	if err != nil {
		return "", err
	}

	return info.Path, nil
}

// executableExecPath 获取可执行文件路径
func (c *cookieManager) executableExecPath(ctx context.Context, depType types.DependencyType) (string, error) {
	info, err := c.depManager.Get(ctx, depType)
	if err != nil {
		return "", err
	}

	return info.ExecPath, nil
}
