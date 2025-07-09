package browercookies

import (
	"CanMe/backend/consts"
	"os"
)

// GetCookieFilePath 获取指定浏览器的 cookies 文件路径
func GetCookieFilePath(browserType consts.BrowserType) string {
	homeDir, _ := os.UserHomeDir()
	return getCookieFilePath(browserType, homeDir)
}
