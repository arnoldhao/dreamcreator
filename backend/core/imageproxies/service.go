package imageproxies

import (
	"context"
	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/proxy"
	"dreamcreator/backend/storage"
	"dreamcreator/backend/types"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
		logger.Error("image proxy: fetch failed", zap.String("url", imageUrl), zap.Error(err))
		return nil, err
	}

	// 存入缓存
	if err := s.storage.SaveImage(result); err != nil {
		// 缓存失败不影响主流程，只记录日志
		logger.Error("Fail to cache image", zap.Error(err))
	}

	return result, nil
}

// 清理图片缓存
func (s *Service) CleanImageCache(url string) error {
	return s.storage.DeleteImage(url)
}

func (s *Service) proxyImageWithoutCache(imageUrl string) (*types.ImageInfo, error) {
	// Build request
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("Fail to initial request: %w", err)
	}

	// Parse URL to set better anti-hotlink headers
	u, _ := url.Parse(imageUrl)
	defaultRef := ""
	if u != nil && u.Scheme != "" && u.Host != "" {
		defaultRef = u.Scheme + "://" + u.Host + "/"
	}
	ref := defaultRef
	// Domain-specific referer mapping
	host := strings.ToLower(u.Host)
	if strings.HasSuffix(host, ".hdslb.com") || strings.Contains(host, "bilibili") {
		ref = "https://www.bilibili.com/"
	}
	// Apply headers
	if ref != "" {
		req.Header.Set("Referer", ref)
		req.Header.Set("Origin", strings.TrimRight(ref, "/"))
	}
	// Some providers (e.g., YouTube thumbnail CDN) prefer youtube referer
	if strings.Contains(host, "ytimg.com") || strings.Contains(host, "youtube.com") {
		req.Header.Set("Referer", "https://www.youtube.com/")
		req.Header.Set("Origin", "https://www.youtube.com")
	}
	// Common UA/Accept
	req.Header.Set("User-Agent", consts.USER_AGENT)
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	// 发送前不再打印详细代理调试日志

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
		logger.Warn("image proxy: non-200",
			zap.String("url", imageUrl),
			zap.Int("status", resp.StatusCode),
			zap.String("contentType", resp.Header.Get("Content-Type")),
			zap.Int("errBodyLen", len(bodyString)),
		)
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
