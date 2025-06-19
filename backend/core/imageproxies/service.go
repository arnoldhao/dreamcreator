package imageproxies

import (
	"CanMe/backend/consts"
	"CanMe/backend/pkg/logger"
	"CanMe/backend/pkg/proxy"
	"CanMe/backend/storage"
	"CanMe/backend/types"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type Service struct {
	ctx context.Context
	// proxy
	proxyManager proxy.ProxyManager
	// storage
	storage *storage.BoltStorage
}

func NewService(proxyManager proxy.ProxyManager, storage *storage.BoltStorage) *Service {
	return &Service{
		proxyManager: proxyManager,
		storage:      storage,
	}
}

func (s *Service) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *Service) ProxyImage(imageUrl string) (*types.ImageInfo, error) {
	// 先尝试从缓存读取
	if cached, err := s.storage.GetImage(imageUrl); err == nil && cached != nil {
		return cached, nil
	}

	// 原始代理逻辑
	result, err := s.proxyImageWithoutCache(imageUrl)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	if err := s.storage.SaveImage(result); err != nil {
		// 缓存失败不影响主流程，只记录日志
		logger.GetLogger().Error("Fail to cache image", zap.Error(err))
	}

	return result, nil
}

// 清理图片缓存
func (s *Service) CleanImageCache(url string) error {
	return s.storage.DeleteImage(url)
}

func (s *Service) proxyImageWithoutCache(imageUrl string) (*types.ImageInfo, error) {
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("Fail to initial request: %w", err)
	}

	// Referer set to empty
	req.Header.Set("Referer", "")
	// 设置一个常见的 User-Agent
	req.Header.Set("User-Agent", consts.USER_AGENT)
	// 发送请求
	resp, err := s.proxyManager.GetHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("Fail to execute request: %w", err)
	}

	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		// 读取可能存在的错误信息体（可选）
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return nil, fmt.Errorf("获取图片失败，状态码: %d, 响应体: %s", resp.StatusCode, bodyString)
	}

	// 读取图片数据
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取图片数据失败: %w", err)
	}

	// 获取图片的 Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// 如果服务器未返回 Content-Type，可以尝试根据 URL 后缀猜测，或设置默认值
		if strings.HasSuffix(imageUrl, ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(imageUrl, ".gif") {
			contentType = "image/gif"
		} else if strings.HasSuffix(imageUrl, ".webp") {
			contentType = "image/webp"
		} else {
			contentType = "image/jpeg" // 默认设为 jpeg
		}
	}

	// 将图片数据进行 Base64 编码
	base64Data := base64.StdEncoding.EncodeToString(imageData)

	// 返回结果
	return &types.ImageInfo{
		URL:         imageUrl,
		Base64Data:  base64Data,
		ContentType: contentType,
	}, nil
}
