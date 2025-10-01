package downtasks

import (
	"dreamcreator/backend/consts"
	"dreamcreator/backend/pkg/browercookies"
	"dreamcreator/backend/pkg/dependencies"
	"dreamcreator/backend/pkg/dependencies/providers"
	"dreamcreator/backend/pkg/downinfo"
	"dreamcreator/backend/pkg/events"
	"dreamcreator/backend/pkg/logger"
	"dreamcreator/backend/pkg/proxy"
	"dreamcreator/backend/services/preferences"
	"dreamcreator/backend/storage"
	"dreamcreator/backend/types"

	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"encoding/json"
	"math"
	"net/url"
	"runtime"
	"sort"
	"unicode/utf8"

	"github.com/lrstanley/go-ytdlp"
	"go.uber.org/zap"
)

// helpers for unique appends
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// Service 处理视频下载和处理
type Service struct {
	ctx         context.Context
	taskManager *TaskManager
	// 事件总线
	eventBus      events.EventBus
	metadataCache sync.Map // 用于缓存视频元数据
	// proxy
	proxyManager proxy.ProxyManager
	// download
	downloadClient *downinfo.Client
	// pref
	pref *preferences.Service
	// ffmpeg exec path
	ffmpegExecPath string
	// bolt storage
	boltStorage *storage.BoltStorage

	// dependencies
	depManager dependencies.Manager

	// cookie manager
	cookieManager browercookies.CookieManager
}

func NewService(eventBus events.EventBus,
	proxyManager proxy.ProxyManager,
	downloadClient *downinfo.Client,
	pref *preferences.Service,
	boltStorage *storage.BoltStorage,
) *Service {

	// 创建依赖管理器
	depManager := dependencies.NewManager(eventBus, proxyManager, boltStorage)

	// 注册依赖提供者
	depManager.Register(providers.NewYTDLPProvider(eventBus))
	depManager.Register(providers.NewFFmpegProvider(eventBus))

	// 初始化默认依赖信息
	if err := depManager.InitializeDefaultDependencies(); err != nil {
		logger.Error("Failed to initialize default dependencies", zap.Error(err))
	}

	s := &Service{
		taskManager:    nil,
		eventBus:       eventBus,
		proxyManager:   proxyManager,
		downloadClient: downloadClient,
		pref:           pref,
		boltStorage:    boltStorage,
		depManager:     depManager,
		cookieManager:  browercookies.NewCookieManager(boltStorage, depManager),
	}

	return s
}

func (s *Service) SetContext(ctx context.Context) {
	s.ctx = ctx
	s.taskManager = NewTaskManager(ctx, s.boltStorage)
}

func (s *Service) ListTasks() []*types.DtTaskStatus {
	list := s.taskManager.ListTasks()
	// Return shallow copies for display safety; mutate only the copies here
	out := make([]*types.DtTaskStatus, 0, len(list))
	for _, t := range list {
		if t == nil {
			continue
		}
		c := *t // shallow copy
		// Sanitize thumbnail (trim quotes/whitespace)
		if c.Thumbnail != "" {
			th := strings.TrimSpace(c.Thumbnail)
			th = strings.Trim(th, "\"'")
			if strings.HasPrefix(th, "http:") {
				if parsed, err := url.Parse(th); err == nil {
					host := strings.ToLower(parsed.Host)
					if host == "i.ytimg.com" || strings.HasSuffix(host, ".ytimg.com") {
						parsed.Scheme = "https"
						th = parsed.String()
					}
				}
			}
			c.Thumbnail = th
		} else {
			if strings.Contains(strings.ToLower(c.Extractor), "youtube") && c.URL != "" {
				vid := ""
				if u, err := url.Parse(c.URL); err == nil {
					if v := u.Query().Get("v"); v != "" {
						vid = v
					}
					if vid == "" && strings.Contains(u.Host, "youtu.be") {
						p := strings.Trim(u.Path, "/")
						if p != "" {
							vid = p
						}
					}
				}
				if vid != "" {
					c.Thumbnail = "https://i.ytimg.com/vi/" + vid + "/hqdefault.jpg"
				}
			}
		}
		// Languages: derive strictly from files when available, without touching stored value
		if len(c.SubtitleProcess.Files) > 0 {
			seen := map[string]bool{}
			langs := make([]string, 0, len(c.SubtitleProcess.Files))
			for _, f := range c.SubtitleProcess.Files {
				base := filepath.Base(f)
				ext := filepath.Ext(base)
				noext := strings.TrimSuffix(base, ext)
				lang := strings.TrimPrefix(filepath.Ext(noext), ".")
				lang = strings.TrimSpace(lang)
				if lang == "" {
					continue
				}
				k := strings.ToLower(lang)
				if seen[k] {
					continue
				}
				seen[k] = true
				langs = append(langs, lang)
			}
			c.SubtitleProcess.Languages = langs
		} else {
			langs := c.SubtitleProcess.Languages
			if len(langs) > 0 {
				seen := map[string]bool{}
				outLangs := make([]string, 0, len(langs))
				for _, raw := range langs {
					v := strings.TrimSpace(raw)
					if v == "" {
						continue
					}
					k := strings.ToLower(strings.ReplaceAll(v, "_", "-"))
					if seen[k] {
						continue
					}
					seen[k] = true
					outLangs = append(outLangs, v)
				}
				c.SubtitleProcess.Languages = outLangs
			}
		}
		out = append(out, &c)
	}
	return out
}

func (s *Service) Path() string {
	return s.taskManager.Path()
}

// DeleteTask 删除指定ID的任务
func (s *Service) DeleteTask(id string) error {
	return s.taskManager.DeleteTask(id)
}

func (s *Service) GetTaskStatus(id string) (*types.DtTaskStatus, error) {
	status := s.taskManager.GetTask(id)
	if status == nil {
		return nil, fmt.Errorf("task not found")
	}
	return status, nil
}

// UpdateTask allows external callers (API layer) to persist changes to a task
func (s *Service) UpdateTask(task *types.DtTaskStatus) error {
	if task == nil {
		return fmt.Errorf("nil task")
	}
	s.taskManager.UpdateTask(task)
	return nil
}

func (s *Service) GetTaskStatusByURL(raw string) (bool, *types.DtTaskStatus, error) {
	target := normalizeURLForCompare(raw)
	list := s.taskManager.ListTasks() // use raw list to fetch pointer
	for _, task := range list {
		if task == nil {
			continue
		}
		if normalizeURLForCompare(task.URL) == target {
			return true, task, nil
		}
	}
	return false, nil, nil
}

// normalizeURLForCompare tries to normalize URLs for duplicate detection without network access.
func normalizeURLForCompare(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	u.User = nil
	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	// Normalize youtu.be short links to youtube watch URL
	if strings.EqualFold(u.Host, "youtu.be") {
		id := strings.Trim(u.Path, "/")
		if id != "" {
			u.Host = "www.youtube.com"
			u.Path = "/watch"
			q := url.Values{}
			q.Set("v", id)
			u.RawQuery = q.Encode()
			return u.String()
		}
	}
	// Remove common tracking params while keeping useful ones
	if u.RawQuery != "" {
		q := u.Query()
		for key := range q {
			lk := strings.ToLower(key)
			if strings.HasPrefix(lk, "utm_") || lk == "fbclid" || lk == "gclid" {
				q.Del(key)
			}
		}
		// sort keys deterministically
		if len(q) > 0 {
			// Re-encode preserves deterministic order in Go 1.20+
			u.RawQuery = q.Encode()
		} else {
			u.RawQuery = ""
		}
	}
	// remove trailing slash (except root)
	if len(u.Path) > 1 && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String()
}

func (s *Service) GetFormats() map[string][]*types.ConversionFormat {
	return s.taskManager.ListAvalibleConversionFormats()
}

func (s *Service) newCommand(enbaledFFMpeg bool, cookiesFile string) (*ytdlp.Command, error) {
	// new
	dl := ytdlp.New()

	// proxy
	if httpProxy := s.proxyManager.GetProxyString(); httpProxy != "" {
		dl.SetEnvVar("HTTP_PROXY", httpProxy).
			SetEnvVar("HTTPS_PROXY", httpProxy)
	}

	// Ensure Python/yt-dlp emits UTF-8 on Windows and elsewhere
	// This avoids mojibake for non-ASCII filenames printed in logs
	dl.SetEnvVar("PYTHONUTF8", "1").
		SetEnvVar("PYTHONIOENCODING", "utf-8")

	// yt-dlp mustinstall
	ytExecPath, err := s.YTDLPExecPath()
	if err != nil {
		return nil, err
	}
	dl.SetExecutable(ytExecPath)

	// set temp dir: prefer fast system temp to avoid slow I/O (e.g., iCloud/AV scanning in Downloads)
	sysTmp := os.TempDir()
	tmpPath := filepath.Join(sysTmp, consts.AppDataDirName())
	var effectiveTmp string
	if err := os.MkdirAll(tmpPath, 0o755); err == nil {
		effectiveTmp = tmpPath
	} else {
		// fallback to app temp dir
		tempDir, derr := s.downDir("temp")
		if derr != nil {
			return nil, derr
		}
		effectiveTmp = tempDir
	}
	dl.SetEnvVar("TEMP", effectiveTmp)
	dl.SetEnvVar("TMP", effectiveTmp)
	// On POSIX systems, Python honors TMPDIR first. Set it to keep behavior consistent.
	if runtime.GOOS != "windows" {
		dl.SetEnvVar("TMPDIR", effectiveTmp)
	}

	// ffmpeg
	if enbaledFFMpeg {
		ffExecPath, err := s.FFMPEGExecPath()
		if err != nil {
			return nil, err
		}
		// set ffmpeg
		dl.FFmpegLocation(ffExecPath)
	}

	if cookiesFile != "" {
		// set cookies
		dl.Cookies(cookiesFile)
	}

	return dl, nil
}

// tempDir returns the preferred temporary directory for transient files.
func (s *Service) tempDir() (string, error) {
	sysTmp := os.TempDir()
	tmpPath := filepath.Join(sysTmp, consts.AppDataDirName())
	if err := os.MkdirAll(tmpPath, 0o755); err == nil {
		return tmpPath, nil
	}
	// fallback to app temp dir
	return s.downDir("temp")
}

// createTempFile writes data to a uniquely-named temp file and returns its path.
func (s *Service) createTempFile(prefix string, data []byte) (string, error) {
	dir, err := s.tempDir()
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp(dir, prefix+"-*.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	return f.Name(), nil
}

// ParseURL 从 YouTube 获取视频内容信息
func (s *Service) ParseURL(url string, browser string) (*ytdlp.ExtractedInfo, error) {
	// 获取Cookies
	var cookiesFile string
	if browser != "" {
		netscapecookies, err := s.GetNetscapeCookiesByDomain(browser, url)
		if err == nil && netscapecookies != "" {
			cookiesFile, err = s.createTempFile("cookies", []byte(netscapecookies))
			if err != nil {
				return nil, err
			}
			defer os.Remove(cookiesFile)
		}
	}
	// 创建 yt-dlp 命令构建器
	dl, err := s.newCommand(false, cookiesFile)
	if err != nil {
		return nil, err
	}

	// 添加选项
	dl.SkipDownload().
		DumpSingleJSON().
		NoPlaylist() // 保持与下载流程一致，避免返回整份播放列表

	// 运行 yt-dlp 命令
	result, err := dl.Run(s.ctx, url)
	if err != nil {
		return nil, err
	}

	// 解析 JSON 数据为 ExtractedInfo 结构体
	var info ytdlp.ExtractedInfo
	if err := json.Unmarshal([]byte(result.Stdout), &info); err != nil {
		return nil, err
	}

	// 尝试将 playlist/multi-video 结果归一为首个可播放视频
	primary := s.selectPrimaryEntry(&info)
	if primary == nil {
		primary = &info
	}

	// 缓存元数据
	s.cacheMetadata(url, primary)

	return primary, nil
}

func (s *Service) selectPrimaryEntry(info *ytdlp.ExtractedInfo) *ytdlp.ExtractedInfo {
	if info == nil {
		return nil
	}

	// 若当前结果自身包含可用格式，则直接返回
	if len(info.Formats) > 0 || len(info.Entries) == 0 {
		return info
	}

	// 优先返回第一个具备 formats 的条目
	for _, entry := range info.Entries {
		if entry != nil && len(entry.Formats) > 0 {
			return entry
		}
	}

	// 回退：若所有条目缺少 formats，返回首个非空条目以供后续流程继续
	for _, entry := range info.Entries {
		if entry != nil {
			return entry
		}
	}

	return info
}

func (s *Service) getVideoMetadata(url, browser string) (*ytdlp.ExtractedInfo, error) {
	// 尝试从缓存获取元数据
	metadata, ok := s.getCachedMetadata(url)
	if ok {
		return metadata, nil
	}

	// 如果缓存中没有，则重新获取
	return s.ParseURL(url, browser)
}

type InfoChan chan *types.FillTaskInfo

// ProgressChan is a channel for receiving download progress updates
type ProgressChan chan *types.DtProgress

// Download 开始视频下载和处理流程
func (s *Service) Download(request *types.DtDownloadRequest) (*types.DtDownloadResponse, error) {
	// 创建新任务
	taskID := uuid.New().String()
	task := s.taskManager.CreateTask(taskID)

	task.Type = consts.TASK_TYPE_CUSTOM

	// 尝试从缓存获取元数据
	metadata, err := s.getVideoMetadata(request.URL, request.Browser)
	if err != nil {
		s.handleTaskError(task, err, nil)
		return nil, err
	}

	// request params
	task.DownloadSubs = request.DownloadSubs
	task.SubLangs = request.SubLangs
	task.SubFormat = request.SubFormat
	task.TranslateTo = request.TranslateTo
	task.SubtitleStyle = request.SubtitleStyle

	// recode info
	task.RecodeFormatNumber = request.RecodeFormatNumber
	if request.RecodeFormatNumber != 0 {
		recodeExt, err := s.taskManager.GetConversionFormatExtension(request.RecodeFormatNumber)
		if err != nil {
			// ignore
		} else {
			task.RecodeExtention = recodeExt
		}
	}

	// core metadata (defensive checks)
	if metadata.Extractor != nil {
		task.Extractor = *metadata.Extractor
	}
	if metadata.Title != nil {
		task.Title = *metadata.Title
	}
	if metadata.Thumbnail != nil {
		// Sanitize possible quoted/whitespace-wrapped values on Windows
		thumb := strings.TrimSpace(*metadata.Thumbnail)
		thumb = strings.Trim(thumb, "\"'")
		task.Thumbnail = thumb
	}
	task.URL = request.URL
	task.Stage = types.DtStageDownloading
	task.Percentage = 0
	task.FormatID = request.FormatID

	// 兼容Bilibili番剧
	if metadata.Uploader != nil {
		task.Uploader = *metadata.Uploader
	} else if metadata.Series != nil {
		task.Uploader = *metadata.Series
	} else if metadata.Extractor != nil {
		task.Uploader = *metadata.Extractor // default
	}
	if metadata.Duration != nil {
		task.Duration = *metadata.Duration
	}

	// 获取输出目录
	outputDir, err := s.downDir(task.Extractor)
	if err == nil {
		task.OutputDir = outputDir
		logger.Debug("download: set output dir", zap.String("taskId", task.ID), zap.String("outputDir", outputDir))
	}

	// 如果有格式信息，设置格式
	if formats := metadata.Formats; formats != nil {
		for _, format := range formats {
			if format.FormatID != nil && *format.FormatID == request.FormatID {
				if format.Extension != nil {
					task.Format = *format.Extension
				}
				// file size
				if format.FileSizeApprox != nil {
					task.FileSize = int64(*format.FileSizeApprox)
				} else if format.FileSize != nil {
					task.FileSize = int64(*format.FileSize)
				} else {
					task.FileSize = 0
				}

				// quality
				if format.Resolution != nil {
					task.Resolution = *format.Resolution
				} else if format.Height != nil && format.Width != nil {
					task.Resolution = fmt.Sprintf("%v x %v", *format.Width, *format.Height)
				} else {
					task.Resolution = "Unknown"
				}
				break
			}
		}
	}

	s.taskManager.UpdateTask(task)

	resp := &types.DtDownloadResponse{
		ID:     taskID,
		Status: types.DtStageDownloading,
	}

	// initial task info channel
	infoChan := make(InfoChan, 1)

	// 初始化进度通道
	progressChan := make(ProgressChan, 100)

	// 启动处理流程
	go s.processTask(task, &types.DownloadVideoRequest{
		Type:          task.Type,
		URL:           request.URL,
		Browser:       request.Browser,
		FormatID:      request.FormatID,
		DownloadSubs:  request.DownloadSubs,
		SubLangs:      request.SubLangs,
		SubFormat:     request.SubFormat,
		TranslateTo:   request.TranslateTo,
		SubtitleStyle: request.SubtitleStyle,
	}, infoChan, progressChan)

	// start info monitor
	go s.fillTaskInfo(infoChan)

	// 启动进度监控
	go s.monitorProgress(progressChan)

	return resp, nil
}

// QuickDownload 快速下载视频
func (s *Service) QuickDownload(request *types.DtQuickDownloadRequest) (*types.DtQuickDownloadResponse, error) {
	// 创建新任务
	taskID := uuid.New().String()
	task := s.taskManager.CreateTask(taskID)

	task.Type = request.Type
	task.URL = request.URL
	task.Browser = request.Browser

	task.Stage = types.DtStageDownloading
	task.Percentage = 0

	// recode info
	task.RecodeFormatNumber = request.RecodeFormatNumber
	if request.RecodeFormatNumber != 0 {
		recodeExt, err := s.taskManager.GetConversionFormatExtension(request.RecodeFormatNumber)
		if err != nil {
			// ignore
		} else {
			task.RecodeExtention = recodeExt
		}
	}

	// 不再在 Quick 模式返回前做元数据预取，依赖下载进度的首个回调填充任务信息，
	// 以加快接口返回速度并避免阻塞 UI。

	// define output dir, quick / mcp
	outputDir, err := s.downDir(request.Type)
	if err == nil {
		task.OutputDir = outputDir
	}

	s.taskManager.UpdateTask(task)

	resp := &types.DtQuickDownloadResponse{
		ID:     taskID,
		Status: types.DtStageDownloading,
	}

	// initial task info channel
	infoChan := make(InfoChan, 1)

	// 初始化进度通道
	progressChan := make(ProgressChan, 100)

	// 启动处理流程
	go s.processTask(task, &types.DownloadVideoRequest{
		Type:        request.Type,
		URL:         request.URL,
		Browser:     request.Browser,
		Video:       request.Video,
		BestCaption: request.BestCaption,
		// Trigger subtitle download in a separate step for quick mode when bestCaption is chosen
		DownloadSubs: request.BestCaption,
		SubFormat:    "best",
	}, infoChan, progressChan)

	// start info monitor
	go s.fillTaskInfo(infoChan)

	// 启动进度监控
	go s.monitorProgress(progressChan)

	return resp, nil
}

// 缓存元数据
type cachedMetadata struct {
	at   time.Time
	info *ytdlp.ExtractedInfo
}

const metadataTTL = 30 * time.Minute

func (s *Service) cacheMetadata(url string, metadata *ytdlp.ExtractedInfo) {
	s.metadataCache.Store(url, &cachedMetadata{at: time.Now(), info: metadata})
}

// 获取缓存的元数据（带 TTL）
func (s *Service) getCachedMetadata(url string) (*ytdlp.ExtractedInfo, bool) {
	value, ok := s.metadataCache.Load(url)
	if !ok {
		return nil, false
	}
	cm, ok := value.(*cachedMetadata)
	if !ok || cm == nil {
		return nil, false
	}
	if time.Since(cm.at) > metadataTTL {
		s.metadataCache.Delete(url)
		return nil, false
	}
	return cm.info, cm.info != nil
}

// processTask 处理任务的主流程
func (s *Service) processTask(task *types.DtTaskStatus, request *types.DownloadVideoRequest, infoChan InfoChan, progressChan ProgressChan) {
	// Ensure we always close infoChan to stop fillTaskInfo goroutine
	// after the video (and optional subtitle) processing completes.
	// Safe because sends to infoChan only occur during downloadVideo's
	// ProgressFunc while dl.Run is active.
	defer close(infoChan)
	defer close(progressChan)

	// 第一阶段：下载视频
	err := s.downloadVideo(task, request, infoChan, progressChan)
	if err != nil {
		s.handleTaskError(task, err, progressChan)
		return
	}
	s.taskManager.UpdateTask(task)

	// 第二阶段：翻译字幕（如果需要）
	if request.Type == consts.TASK_TYPE_CUSTOM {
		if request.DownloadSubs && request.TranslateTo != "" {
			task.Stage = types.DtStageTranslating
			s.taskManager.UpdateTask(task)

			// 发送阶段变更通知
			progressChan <- &types.DtProgress{
				ID:         task.ID,
				Type:       task.Type,
				Stage:      types.DtStageTranslating,
				Percentage: 0,
				StageInfo:  "Start translating subtitles",
			}

			subtitleFile, err := s.translateSubtitles(task, progressChan)
			if err != nil {
				s.handleTaskError(task, err, progressChan)
				return
			}
			task.TranslatedSubs = append(task.TranslatedSubs, subtitleFile)
			// add to all files
			task.AllFiles = append(task.AllFiles, subtitleFile)
			s.taskManager.UpdateTask(task)
		} else {
			task.TranslatedSubs = []string{}
		}

		// 第三阶段：嵌入字幕（如果需要）
		if request.DownloadSubs && request.TranslateTo != "" {
			task.Stage = types.DtStageEmbedding
			s.taskManager.UpdateTask(task)

			// 发送阶段变更通知
			progressChan <- &types.DtProgress{
				ID:         task.ID,
				Type:       task.Type,
				Stage:      types.DtStageEmbedding,
				Percentage: 0,
				StageInfo:  "Start embedding subtitles",
			}

			embeddedVideo, err := s.embedSubtitles(task, progressChan)
			if err != nil {
				s.handleTaskError(task, err, progressChan)
				return
			}
			task.EmbeddedVideoFiles = append(task.EmbeddedVideoFiles, embeddedVideo)
			// add to all files
			task.AllFiles = append(task.AllFiles, embeddedVideo)
			s.taskManager.UpdateTask(task)
		} else {
			task.EmbeddedVideoFiles = []string{}
		}
	}
	// 完成所有处理
	task.Stage = types.DtStageCompleted
	s.taskManager.UpdateTask(task)

	// 发送完成通知
	progressChan <- &types.DtProgress{
		ID:            task.ID,
		Type:          task.Type,
		Stage:         types.DtStageCompleted,
		Percentage:    100,
		StageInfo:     "Processing completed",
		EstimatedTime: "completed",
	}
}

// handleTaskError 处理任务错误
func (s *Service) handleTaskError(task *types.DtTaskStatus, err error, progressChan ProgressChan) {
	task.Stage = types.DtStageFailed
	task.Error = err.Error()
	s.taskManager.UpdateTask(task)

	progressChan <- &types.DtProgress{
		ID:         task.ID,
		Type:       task.Type,
		Stage:      types.DtStageFailed,
		Error:      err.Error(),
		Percentage: 0,
		StageInfo:  "Processing failed",
	}
}

// downloadVideo 实现视频下载阶段
func (s *Service) downloadVideo(task *types.DtTaskStatus, request *types.DownloadVideoRequest, infoChan InfoChan, progressChan ProgressChan) error {
	// 发送阶段开始通知：仅 video，在字幕分步下载时再单独发布 subtitle:start
	if s.eventBus != nil {
		s.eventBus.Publish(s.ctx, &events.BaseEvent{
			ID:        uuid.New().String(),
			Type:      consts.TopicDowntasksStage,
			Source:    "downtasks",
			Timestamp: time.Now(),
			Data:      &types.DTStageEvent{ID: task.ID, Kind: "video", Action: "start"},
		})
	}
	// persist process start
	task.DownloadProcess.Video = "working"
	s.taskManager.UpdateTask(task)
	progressChan <- &types.DtProgress{
		ID:         task.ID,
		Type:       task.Type,
		Stage:      types.DtStageDownloading,
		Percentage: 0,
		StageInfo:  "Start downloading video",
	}

	// 获取Cookies
	var cookiesFile string
	if request.Browser != "" {
		netscapecookies, err := s.GetNetscapeCookiesByDomain(request.Browser, request.URL)
		if err == nil && netscapecookies != "" {
			var cerr error
			cookiesFile, cerr = s.createTempFile("cookies", []byte(netscapecookies))
			if cerr != nil {
				return cerr
			}
			defer os.Remove(cookiesFile)
		}
	}

	dl, err := s.newCommand(true, cookiesFile)
	if err != nil {
		s.handleTaskError(task, err, progressChan)
		return err
	}

	if task.Type == "custom" {
		metadata, err := s.getVideoMetadata(request.URL, request.Browser)
		if err != nil {
			s.handleTaskError(task, err, progressChan)
			return err
		}

		// 检查请求的 format_id 是否存在 VCodec 且不存在 ACodes的情况，这种需要增加 bestaudio
		var videoExt string
		if request.FormatID != "" {
			needAudio := false
			for _, format := range metadata.Formats {
				if format.FormatID != nil && *format.FormatID == request.FormatID {
					if format.VCodec != nil && *format.VCodec != "none" {
						if format.ACodec == nil || *format.ACodec == "none" {
							needAudio = true
						}
					}

					if format.Extension != nil {
						videoExt = *format.Extension
					}
					break
				}
			}

			if needAudio {
				if videoExt == "mp4" {
					// MP4 视频，使用 M4A 音频
					dl.Format(request.FormatID + "+bestaudio[ext=m4a]")
					dl.MergeOutputFormat("mp4")
				} else if videoExt == "webm" {
					// WebM 视频，使用 WebM 音频
					dl.Format(request.FormatID + "+bestaudio[ext=webm]")
					dl.MergeOutputFormat("webm")
				} else {
					// 其他情况，让 yt-dlp 自行决定
					dl.Format(request.FormatID + "+bestaudio")
					// 保持原来的设置
					dl.MergeOutputFormat("mp4/webm")
				}
			} else {
				dl.Format(request.FormatID)
			}
		} else {
			dl.UnsetFormat()
		}

		// 字幕分步下载，避免影响视频进度回调

	} else { // if type == quick || mcp
		// format
		if request.Video != "" {
			switch request.Video {
			case "best":
				dl.UnsetFormat()
			default:
				dl.Format(request.Video)
			}
		}
	}

	// 设置工作目录和输出文件
	// Quick 模式下允许覆盖已有文件（强制重新下载）；其他模式保持不覆盖行为
	dl.SetWorkDir(task.OutputDir).
		NoPlaylist()
	if request.Type == consts.TASK_TYPE_QUICK {
		// Quick 模式强制覆盖，确保已存在视频也会重新下载
		dl.ForceOverwrites()
	} else {
		dl.NoOverwrites()
	}
	dl.Output("%(title)s_%(height)sp_%(fps)dfps.%(ext)s").
		NoRestrictFilenames().
		NoWindowsFilenames()
	// print-to-file 已移除：依赖快照差异与兜底扫描
	logger.Debug("download: command prepared",
		zap.String("taskId", task.ID),
		zap.String("workDir", task.OutputDir),
		zap.String("stage", string(task.Stage)),
		zap.String("type", task.Type),
	)

	// Recode
	if task.RecodeExtention != "" {
		dl.RecodeVideo(task.RecodeExtention)
	}

	var once sync.Once
	// speed smoother for stable bandwidth reporting
	ss := newSpeedSmoother(2*time.Second, 2.5) // τ=2s, 峰值抑制系数=2.5

	// 设置进度回调（更高频率，避免小文件/网络快时错过间隔）
	dl.ProgressFunc(250*time.Millisecond, func(update ytdlp.ProgressUpdate) {
		once.Do(func() {
			infoChan <- &types.FillTaskInfo{
				ID:   task.ID,
				Info: update.Info,
			}
		})

		// 平滑瞬时速度（时间常数型 EMA + 峰值抑制）
		bps, ok := ss.Update(time.Now(), update.DownloadedBytes)
		speedStr := ""
		if ok && bps > 0 {
			speedStr = formatBandwidth(bps)
		}

		// ETA display: show blank when unknown, and "completed" only at 100%
		eta := update.ETA()
		etaStr := ""
		if update.Percent() >= 100 {
			etaStr = "completed"
		} else if eta > 0 {
			etaStr = formatDuration(eta)
		}

		progress := &types.DtProgress{
			ID:            task.ID,
			Type:          task.Type,
			Stage:         types.DtStageDownloading,
			Percentage:    update.Percent(),
			Speed:         speedStr,
			Downloaded:    fmt.Sprintf("%.2f MB", float64(update.DownloadedBytes)/1024/1024),
			TotalSize:     fmt.Sprintf("%.2f MB", float64(update.TotalBytes)/1024/1024),
			EstimatedTime: etaStr,
		}

		select {
		case progressChan <- progress:
		case <-s.ctx.Done():
			progress.Stage = types.DtStageCancelled
			progress.Error = "download cancelled"
			progressChan <- progress
			return
		default:
			// Channel is full, skip this update
		}
	})

	// 记录目录快照与开始时间（用于输出文件增量检测）
	videoStartedAt := time.Now()
	beforeSnap := s.dirSnapshot(task.OutputDir)

	// 执行下载
	result, err := dl.Run(s.ctx, request.URL)
	if err != nil {
		s.handleTaskError(task, err, progressChan)
		return fmt.Errorf("Download video failed: %w", err)
	}
	// Log completion with sanitized args (avoid leaking URL queries)
	sanitizedArgs := sanitizeArgs(result.Args)
	logger.Debug("download: yt-dlp finished",
		zap.String("taskId", task.ID),
		zap.Int("stdoutLen", len(result.Stdout)),
		zap.Int("stderrLen", len(result.Stderr)),
		zap.String("exec", result.Executable),
		zap.Strings("args", sanitizedArgs),
	)
	// Truncated stdout/stderr at debug level to reduce volume and avoid sensitive leakage
	if result.Stdout != "" {
		logger.Debug("yt-dlp stdout (truncated)", zap.String("taskId", task.ID), zap.String("stdout", truncateString(result.Stdout, 4000)))
	}
	if result.Stderr != "" {
		logger.Debug("yt-dlp stderr (truncated)", zap.String("taskId", task.ID), zap.String("stderr", truncateString(result.Stderr, 4000)))
	}

	// 解析下载结果（stdout 回放）
	s.parseYtdlpOutput(task, result)

	// 使用目录快照差异，补录本次新增文件（避免编码/输出差异），并基于扩展名做“分类但不丢弃”
	if added := s.diffNewFiles(beforeSnap, task.OutputDir, videoStartedAt); len(added) > 0 {
		// Strictly filter by title_ prefix to avoid cross-task interference
		// when multiple tasks share the same output directory.
		var prefix string
		if t := strings.TrimSpace(task.Title); t != "" {
			prefix = t + "_"
		}
		for _, p := range added {
			if prefix != "" {
				base := filepath.Base(p)
				if !strings.HasPrefix(base, prefix) {
					continue
				}
			}
			cls := classifyByExt(p)
			switch cls {
			case "subtitle":
				if !contains(task.SubtitleFiles, p) {
					task.SubtitleFiles = append(task.SubtitleFiles, p)
				}
			case "video":
				if !contains(task.VideoFiles, p) {
					task.VideoFiles = append(task.VideoFiles, p)
				}
			}
			if !contains(task.AllDownloadedFiles, p) {
				task.AllDownloadedFiles = append(task.AllDownloadedFiles, p)
			}
			if !contains(task.AllFiles, p) {
				task.AllFiles = append(task.AllFiles, p)
			}
		}
		logger.Debug("video snapshot diff",
			zap.String("taskId", task.ID),
			zap.Int("addedCount", len(added)),
			zap.Int("videoCount", len(task.VideoFiles)),
			zap.Int("subtitleCount", len(task.SubtitleFiles)),
		)
	}
	// 兜底：如果未检测到视频文件，按标题前缀扫描目录增补
	if len(task.VideoFiles) == 0 {
		newly := s.scanSubtitleFiles(task.OutputDir, task.Title, videoStartedAt)
		picked := []string{}
		for _, p := range newly {
			if classifyByExt(p) == "video" {
				picked = append(picked, p)
				if !contains(task.VideoFiles, p) {
					task.VideoFiles = append(task.VideoFiles, p)
				}
				if !contains(task.AllDownloadedFiles, p) {
					task.AllDownloadedFiles = append(task.AllDownloadedFiles, p)
				}
				if !contains(task.AllFiles, p) {
					task.AllFiles = append(task.AllFiles, p)
				}
			}
		}
		if len(picked) > 0 {
			logger.Debug("video scan fallback",
				zap.String("taskId", task.ID),
				zap.Int("pickedVideoCount", len(picked)),
			)
		}
		// 如果还是未命中（例如目标文件在本次任务前已存在且未更新），再进行一次“无时间限制”的前缀扫描，
		// 以便把已存在的重复文件也纳入输出列表，满足“output info 填上重复文件”的需求。
		if len(task.VideoFiles) == 0 && strings.TrimSpace(task.Title) != "" {
			older := s.scanSubtitleFiles(task.OutputDir, task.Title, time.Time{}) // 零值时间 => 不做时间过滤
			picked2 := []string{}
			for _, p := range older {
				if classifyByExt(p) != "video" {
					continue
				}
				picked2 = append(picked2, p)
				if !contains(task.VideoFiles, p) {
					task.VideoFiles = append(task.VideoFiles, p)
				}
				if !contains(task.AllDownloadedFiles, p) {
					task.AllDownloadedFiles = append(task.AllDownloadedFiles, p)
				}
				if !contains(task.AllFiles, p) {
					task.AllFiles = append(task.AllFiles, p)
				}
			}
			if len(picked2) > 0 {
				logger.Debug("video scan legacy/prefetched",
					zap.String("taskId", task.ID),
					zap.Int("pickedVideoCount", len(picked2)),
				)
			}
		}
	}
	// 下载与合并阶段完成后的阶段事件（基于输出解析结果的总结事件）
	// 注意：不要依赖文件探测是否命中才发送“完成”事件，否则前端可能永远停留在“合并中”。
	// 只要 yt-dlp 正常结束，就补发 video/merge/finalize 的 complete 事件，确保前端能走到终态。
	if s.eventBus != nil {
		// video complete
		s.eventBus.Publish(s.ctx, &events.BaseEvent{ID: uuid.New().String(), Type: consts.TopicDowntasksStage, Source: "downtasks", Timestamp: time.Now(), Data: &types.DTStageEvent{ID: task.ID, Kind: "video", Action: "complete"}})
		// 合并/收尾 complete：无条件补发，避免前端卡在“合并”阶段
		s.eventBus.Publish(s.ctx, &events.BaseEvent{ID: uuid.New().String(), Type: consts.TopicDowntasksStage, Source: "downtasks", Timestamp: time.Now(), Data: &types.DTStageEvent{ID: task.ID, Kind: "merge", Action: "complete"}})
		s.eventBus.Publish(s.ctx, &events.BaseEvent{ID: uuid.New().String(), Type: consts.TopicDowntasksStage, Source: "downtasks", Timestamp: time.Now(), Data: &types.DTStageEvent{ID: task.ID, Kind: "finalize", Action: "complete"}})
	}
	// 同步持久化过程状态
	task.DownloadProcess.Video = "done"
	if task.DownloadProcess.Merge == "" || task.DownloadProcess.Merge == "working" {
		task.DownloadProcess.Merge = "done"
	}
	if task.DownloadProcess.Finalize == "" || task.DownloadProcess.Finalize == "working" {
		task.DownloadProcess.Finalize = "done"
	}
	s.taskManager.UpdateTask(task)

	// 分步下载字幕，避免影响视频进度输出
	if request.DownloadSubs {
		if err := s.downloadSubtitlesOnly(task, request); err != nil {
			logger.Error("download subtitles failed", zap.Error(err))
		}
	}

	return nil
}

// 单独下载字幕，避免与视频下载进度互相影响
func (s *Service) downloadSubtitlesOnly(task *types.DtTaskStatus, request *types.DownloadVideoRequest) error {
	// 获取Cookies
	var cookiesFile string
	if request.Browser != "" {
		netscapecookies, err := s.GetNetscapeCookiesByDomain(request.Browser, request.URL)
		if err == nil && netscapecookies != "" {
			cookiesFile, err = s.createTempFile("cookies", []byte(netscapecookies))
			if err != nil {
				return err
			}
			defer os.Remove(cookiesFile)
		}
	}

	dl, err := s.newCommand(true, cookiesFile)
	if err != nil {
		return err
	}

	// 仅下载字幕
	dl.SkipDownload()
	dl.WriteSubs()
	if len(request.SubLangs) > 0 {
		dl.SubLangs(strings.Join(request.SubLangs, ","))
	} else {
		dl.SubLangs("all")
	}
	// Use effective subtitle format (default to "best" when not provided)
	effectiveSubFormat := request.SubFormat
	if strings.TrimSpace(effectiveSubFormat) == "" {
		effectiveSubFormat = "best"
	}
	dl.SubFormat(effectiveSubFormat)

	// 工作目录保持一致，输出模板保持一致（便于生成相同基名的字幕文件）
	dl.SetWorkDir(task.OutputDir).NoPlaylist()
	if request.Type == consts.TASK_TYPE_QUICK {
		dl.ForceOverwrites()
	} else {
		dl.NoOverwrites()
	}
	dl.Output("%(title)s_%(height)sp_%(fps)dfps.%(ext)s").
		NoRestrictFilenames().
		NoWindowsFilenames()
	// print-to-file removed; rely on snapshot diff + stdout parsing

	task.SubtitleProcess.Status = "working"
	// 在开始阶段不写入“best”占位，避免前端看到不准确的格式；
	// 如果调用方明确指定了具体格式（非空且非best），则可先行写入。
	if strings.TrimSpace(request.SubFormat) != "" && !strings.EqualFold(request.SubFormat, "best") {
		task.SubtitleProcess.Format = request.SubFormat
	}
	task.SubtitleProcess.OutputDir = task.OutputDir
	s.taskManager.UpdateTask(task)
	if s.eventBus != nil {
		s.eventBus.Publish(s.ctx, &events.BaseEvent{ID: uuid.New().String(), Type: consts.TopicDowntasksStage, Source: "downtasks", Timestamp: time.Now(), Data: &types.DTStageEvent{ID: task.ID, Kind: "subtitle", Action: "start"}})
	}

	// 运行
	startedAt := time.Now()
	beforeSnap := s.dirSnapshot(task.OutputDir)
	result, err := dl.Run(s.ctx, request.URL)
	if err != nil {
		return err
	}
	// Log completion with sanitized args (avoid leaking URL queries)
	sanitizedArgs := sanitizeArgs(result.Args)
	logger.Debug("subtitles-only: yt-dlp finished",
		zap.String("taskId", task.ID),
		zap.Int("stdoutLen", len(result.Stdout)),
		zap.Int("stderrLen", len(result.Stderr)),
		zap.String("exec", result.Executable),
		zap.Strings("args", sanitizedArgs),
	)
	// Truncated stdout/stderr at debug level to reduce volume and avoid sensitive leakage
	if result.Stdout != "" {
		logger.Debug("yt-dlp stdout (subs, truncated)", zap.String("taskId", task.ID), zap.String("stdout", truncateString(result.Stdout, 4000)))
	}
	if result.Stderr != "" {
		logger.Debug("yt-dlp stderr (subs, truncated)", zap.String("taskId", task.ID), zap.String("stderr", truncateString(result.Stderr, 4000)))
	}
	// 解析输出（以获取字幕文件名）
	s.parseYtdlpOutput(task, result)
	// 报告文件已移除：依赖快照差异与兜底扫描
	// 目录快照差异获取本次新增文件（分类但不丢弃)
	beforeSubtitleCount := len(task.SubtitleFiles)
	added := s.diffNewFiles(beforeSnap, task.OutputDir, startedAt)
	if len(added) > 0 {
		// Strictly filter by title_ prefix to reduce cross-task interference
		var prefix string
		if t := strings.TrimSpace(task.Title); t != "" {
			prefix = t + "_"
		}
		picked := []string{}
		for _, p := range added {
			if prefix != "" {
				base := filepath.Base(p)
				if !strings.HasPrefix(base, prefix) {
					continue
				}
			}
			cls := classifyByExt(p)
			if cls == "subtitle" {
				picked = append(picked, p)
			}
			if cls == "subtitle" {
				if !contains(task.SubtitleFiles, p) {
					task.SubtitleFiles = append(task.SubtitleFiles, p)
				}
			} else if cls == "video" {
				if !contains(task.VideoFiles, p) {
					task.VideoFiles = append(task.VideoFiles, p)
				}
			}
			if !contains(task.AllDownloadedFiles, p) {
				task.AllDownloadedFiles = append(task.AllDownloadedFiles, p)
			}
			if !contains(task.AllFiles, p) {
				task.AllFiles = append(task.AllFiles, p)
			}
		}
		logger.Debug("subs snapshot diff",
			zap.String("taskId", task.ID),
			zap.Int("addedCount", len(added)),
			zap.Int("pickedSubtitleCount", len(picked)),
		)
	}
	// 兜底：按标题前缀扫描目录（若快照差异未新增字幕）
	if len(task.SubtitleFiles) == beforeSubtitleCount {
		newly := s.scanSubtitleFiles(task.OutputDir, task.Title, startedAt)
		if len(newly) > 0 {
			picked := []string{}
			for _, p := range newly {
				cls := classifyByExt(p)
				if cls == "subtitle" {
					picked = append(picked, p)
				}
				if cls == "subtitle" {
					if !contains(task.SubtitleFiles, p) {
						task.SubtitleFiles = append(task.SubtitleFiles, p)
					}
				} else if cls == "video" {
					if !contains(task.VideoFiles, p) {
						task.VideoFiles = append(task.VideoFiles, p)
					}
				}
				if !contains(task.AllDownloadedFiles, p) {
					task.AllDownloadedFiles = append(task.AllDownloadedFiles, p)
				}
				if !contains(task.AllFiles, p) {
					task.AllFiles = append(task.AllFiles, p)
				}
			}
			logger.Debug("subs scan fallback",
				zap.String("taskId", task.ID),
				zap.Int("scannedCount", len(newly)),
				zap.Int("pickedSubtitleCount", len(picked)),
			)
		}
	}
	// 同步 SubtitleProcess 信息（严格以下载到的文件为准），使用受锁更新避免竞态
	s.taskManager.UpdateTaskWith(task.ID, func(tt *types.DtTaskStatus) {
		tt.SubtitleProcess.Files = append([]string{}, tt.SubtitleFiles...)
		if len(tt.SubtitleProcess.Files) > 0 {
			langs := make([]string, 0, len(tt.SubtitleProcess.Files))
			seen := map[string]bool{}
			for _, f := range tt.SubtitleProcess.Files {
				base := filepath.Base(f)
				ext := filepath.Ext(base)
				noext := strings.TrimSuffix(base, ext)
				lang := strings.TrimPrefix(filepath.Ext(noext), ".")
				lang = strings.TrimSpace(lang)
				if lang == "" {
					continue
				}
				lck := strings.ToLower(lang)
				if seen[lck] {
					continue
				}
				seen[lck] = true
				langs = append(langs, lang)
			}
			tt.SubtitleProcess.Languages = langs
			// Derive subtitle format string from scanned files
			tt.SubtitleProcess.Format = deriveSubtitleFormats(tt.SubtitleProcess.Files)
		} else {
			tt.SubtitleProcess.Languages = nil
		}
		tt.SubtitleProcess.Status = "done"
	})
	if s.eventBus != nil {
		s.eventBus.Publish(s.ctx, &events.BaseEvent{ID: uuid.New().String(), Type: consts.TopicDowntasksStage, Source: "downtasks", Timestamp: time.Now(), Data: &types.DTStageEvent{ID: task.ID, Kind: "subtitle", Action: "complete"}})
	}
	return nil
}

func (s *Service) fillTaskInfo(infoChan InfoChan) {
	for taskInfo := range infoChan {
		if taskInfo == nil {
			continue
		}

		var updated *types.DtTaskStatus
		updated = s.taskManager.UpdateTaskWith(taskInfo.ID, func(task *types.DtTaskStatus) {
			if taskInfo.Info.Extractor != nil {
				task.Extractor = *taskInfo.Info.Extractor
			}
			if taskInfo.Info.Title != nil {
				task.Title = *taskInfo.Info.Title
			}
			// thumbnail with fallback (esp. on Windows/YouTube where field may be empty)
			var thumb string
			if taskInfo.Info.Thumbnail != nil {
				thumb = strings.TrimSpace(*taskInfo.Info.Thumbnail)
				thumb = strings.Trim(thumb, "\"'")
			}
			if thumb == "" {
				id := strings.TrimSpace(taskInfo.Info.ID)
				if id != "" && strings.Contains(strings.ToLower(task.Extractor), "youtube") {
					thumb = "https://i.ytimg.com/vi/" + id + "/hqdefault.jpg"
				}
			}
			if strings.HasPrefix(thumb, "http:") {
				thumb = "https:" + strings.TrimPrefix(thumb, "http:")
			}
			task.Thumbnail = thumb

			// 兼容Bilibili番剧
			if taskInfo.Info.Uploader != nil {
				task.Uploader = *taskInfo.Info.Uploader
			} else if taskInfo.Info.Series != nil {
				task.Uploader = *taskInfo.Info.Series
			} else if taskInfo.Info.Extractor != nil {
				task.Uploader = *taskInfo.Info.Extractor
			}
			if taskInfo.Info.Duration != nil {
				task.Duration = *taskInfo.Info.Duration
			}

			if task.Format == "" {
				task.Format = taskInfo.Info.Extension
				if taskInfo.Info.FileSizeApprox != nil {
					task.FileSize = int64(*taskInfo.Info.FileSizeApprox)
				} else if taskInfo.Info.FileSize != nil {
					task.FileSize = int64(*taskInfo.Info.FileSize)
				} else {
					task.FileSize = 0
				}
				if taskInfo.Info.Resolution != nil {
					task.Resolution = *taskInfo.Info.Resolution
				} else if taskInfo.Info.Height != nil && taskInfo.Info.Width != nil {
					task.Resolution = fmt.Sprintf("%v x %v", *taskInfo.Info.Width, *taskInfo.Info.Height)
				} else {
					task.Resolution = "Unknown"
				}
			}
		})

		if updated != nil {
			logger.Debug("fillTaskInfo: core metadata",
				zap.String("taskId", updated.ID),
				zap.String("extractor", updated.Extractor),
				zap.String("title", updated.Title),
				zap.String("thumb", updated.Thumbnail),
			)
			// 仅当首次填充格式等信息时给前端一个刷新信号
			event := &events.BaseEvent{
				ID:        uuid.New().String(),
				Type:      consts.TopicDowntasksInstalling,
				Source:    "downtasks",
				Timestamp: time.Now(),
				Data:      &types.DTSignal{ID: updated.ID, Type: updated.Type, Stage: updated.Stage, Refresh: true},
				Metadata:  map[string]interface{}{"task": updated},
			}
			s.eventBus.Publish(s.ctx, event)
		}
	}
}

// parseYtdlpOutput 解析yt-dlp的输出结果，提取最终保存的文件信息
func (s *Service) parseYtdlpOutput(task *types.DtTaskStatus, result *ytdlp.Result) {
	// 保留 yt-dlp 原始输出，不做全局编码回退（避免把 UTF-8 误判为 GBK 导致乱码）
	// 依赖 PYTHONUTF8/PYTHONIOENCODING=utf-8 与 --print 输出，尽量保持 UTF‑8
	stdout := result.Stdout
	wasUTF8 := utf8.ValidString(stdout)
	// Debug log: summarize stdout status
	logger.Debug("yt-dlp stdout summary",
		zap.String("taskId", task.ID),
		zap.String("os", runtime.GOOS),
		zap.Int("len", len(stdout)),
		zap.Bool("wasUTF8", wasUTF8),
	)
	// Log a few sample lines (truncated) for diagnostics
	{
		sample := stdout
		if len(sample) > 2000 {
			sample = sample[:2000]
		}
		logger.Debug("yt-dlp stdout sample", zap.String("taskId", task.ID), zap.String("sample", sample))
	}
	// 按行解析输出
	lines := strings.Split(stdout, "\n")
	// 阶段起始信号（尽可能从输出中识别；注意这里是在进程结束后回放日志，非实时）
	var mergeStarted, finalizeStarted, subtitleStarted bool
	for _, line := range lines {
		// 检测下载目标文件
		if strings.Contains(line, "[download] Destination:") {
			filename := strings.TrimPrefix(line, "[download] Destination: ")
			filename = strings.TrimSpace(filename)
			_ = normalizePath(task.OutputDir, filename)
			// 不在此写入 task 的文件列表，统一由目录快照差异处理
		}

		// 检测合并文件
		if strings.Contains(line, "[Merger] Merging formats into") {
			if !mergeStarted && s.eventBus != nil {
				s.eventBus.Publish(s.ctx, &events.BaseEvent{ID: uuid.New().String(), Type: consts.TopicDowntasksStage, Source: "downtasks", Timestamp: time.Now(), Data: &types.DTStageEvent{ID: task.ID, Kind: "merge", Action: "start"}})
				mergeStarted = true
				task.DownloadProcess.Merge = "working"
				s.taskManager.UpdateTask(task)
			}
			filenameWithQuotes := strings.TrimPrefix(line, "[Merger] Merging formats into ")
			filename := strings.Trim(strings.TrimPrefix(filenameWithQuotes, "\""), "\"")
			_ = normalizePath(task.OutputDir, filename)
			// 不在此写入 task 的文件列表，统一由目录快照差异处理
		}

		// 检测删除的临时文件
		if strings.Contains(line, "Deleting original file") {
			if !finalizeStarted && s.eventBus != nil {
				s.eventBus.Publish(s.ctx, &events.BaseEvent{ID: uuid.New().String(), Type: consts.TopicDowntasksStage, Source: "downtasks", Timestamp: time.Now(), Data: &types.DTStageEvent{ID: task.ID, Kind: "finalize", Action: "start"}})
				finalizeStarted = true
				task.DownloadProcess.Finalize = "working"
				s.taskManager.UpdateTask(task)
			}
			filenameWithQuotes := strings.TrimPrefix(line, "Deleting original file ")
			filename := strings.TrimSuffix(filenameWithQuotes, " (pass -k to keep)")
			_ = normalizePath(task.OutputDir, filename)
		}

		// 检测字幕写入/下载提示
		lower := strings.ToLower(line)
		if (strings.Contains(lower, "[subtitle]") || strings.Contains(lower, "writing video subtitles") || strings.Contains(lower, "writing subtitles")) && !subtitleStarted {
			if s.eventBus != nil {
				s.eventBus.Publish(s.ctx, &events.BaseEvent{ID: uuid.New().String(), Type: consts.TopicDowntasksStage, Source: "downtasks", Timestamp: time.Now(), Data: &types.DTStageEvent{ID: task.ID, Kind: "subtitle", Action: "start"}})
			}
			// 尝试提取写入的字幕文件名（仅记录，不修改任务文件列表；文件最终以目录快照差异为准）
			if idx := strings.Index(lower, "writing video subtitles to"); idx >= 0 {
				// 形如: Writing video subtitles to: "path/to/file.srt"
				// 简单从冒号后截取
				parts := strings.SplitN(line[idx:], ":", 2)
				if len(parts) == 2 {
					name := strings.TrimSpace(parts[1])
					name = strings.Trim(name, "\"")
					_ = normalizePath(task.OutputDir, name)
					// 不在此处改动 SubtitleFiles/AllFiles/Languages/Format
				}
			}
			subtitleStarted = true
		}

		// 检测字幕相关行（仅用于内部状态机，不记录日志）
		if strings.Contains(lower, "downloading subtitles") || strings.Contains(lower, "downloading subtitle") ||
			strings.Contains(lower, "writing subtitles to") || strings.Contains(lower, "writing subtitle to") ||
			strings.Contains(lower, "extracting subtitles") || strings.Contains(lower, "converting subtitle") {
			// no-op: preserve for possible future heuristics
		}
	}

	// 此函数不再更新 VideoFiles/AllDownloadedFiles，避免与目录快照差异重复；仅在检测到字幕写入时增加 SubtitleFiles 以及 AllFiles。
	// 诊断性日志已移除（避免噪音）；如需，改为 debug 级别并在关键路径添加。
	// 如果已获取到字幕文件，但未由请求参数精确指定，则从现有文件扩展名聚合生成格式串（如 "vtt" 或 "vtt/srt/xml"）
	if task.SubtitleProcess.Format == "" {
		// Prefer SubtitleProcess.Files when available; else fall back to SubtitleFiles
		src := task.SubtitleProcess.Files
		if len(src) == 0 {
			src = task.SubtitleFiles
		}
		if len(src) > 0 {
			task.SubtitleProcess.Format = deriveSubtitleFormats(src)
		}
	}
	// 更新任务
	s.taskManager.UpdateTask(task)
	// Final snapshot removed: avoid verbose logs

	// 创建事件
	event := &events.BaseEvent{
		ID:        uuid.New().String(),
		Type:      consts.TopicDowntasksInstalling,
		Source:    "downtasks",
		Timestamp: time.Now(),
		Data: &types.DTSignal{
			ID:      task.ID,
			Type:    task.Type,
			Stage:   task.Stage,
			Refresh: true,
		},
		Metadata: map[string]interface{}{
			"task": task,
		},
	}

	s.eventBus.Publish(s.ctx, event)
}

// translateSubtitles 实现字幕翻译阶段
func (s *Service) translateSubtitles(task *types.DtTaskStatus, progressChan ProgressChan) (string, error) {
	// TODO
	_ = progressChan
	// 返回翻译后的字幕文件路径
	translatedSubFile := fmt.Sprintf("downloads/%s.translated.srt", task.ID)

	// 在实际实现中，这里应该调用翻译API并保存翻译后的字幕

	return translatedSubFile, nil
}

// detectFileType 根据文件名判断文件类型
func (s *Service) detectFileType(filename string) string {
	// 检查是否是字幕文件
	subtitleExts := []string{".vtt", ".srt", ".ass", ".ssa"}
	for _, ext := range subtitleExts {
		if strings.HasSuffix(filename, ext) {
			return "subtitle"
		}
	}

	// 默认为视频文件
	return "video"
}

// classifyByExt returns one of: "video", "subtitle", "other" based on common extensions.
// 不用于过滤，只用于分类，所有文件仍会加入 AllDownloadedFiles/AllFiles。
func classifyByExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".vtt", ".srt", ".ass", ".ssa":
		return "subtitle"
	case ".mp4", ".webm", ".mkv", ".mov", ".m4v", ".ts", ".avi", ".flv", ".ogg", ".ogv":
		return "video"
	default:
		return "other"
	}
}

// normalizePath converts possibly-relative filenames from yt-dlp output into absolute paths
// based on the given output directory, and cleans the result.
func normalizePath(outDir, p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	if !filepath.IsAbs(p) && outDir != "" {
		p = filepath.Join(outDir, p)
	}
	return filepath.Clean(p)
}

// truncateString limits the size of a string for logging to avoid flooding logs.
func truncateString(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	const suffix = "…(truncated)"
	if max <= len(suffix) {
		return s[:max]
	}
	return s[:max-len(suffix)] + suffix
}

// sanitizeArgs removes sensitive query strings from http(s) URLs in args for logging.
func sanitizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = sanitizeArg(a)
	}
	return out
}

func sanitizeArg(a string) string {
	s := strings.TrimSpace(a)
	if s == "" {
		return a
	}
	if u, err := url.Parse(s); err == nil {
		if u.Scheme == "http" || u.Scheme == "https" {
			u.User = nil
			u.RawQuery = ""
			return u.String()
		}
	}
	return a
}

// deriveSubtitleFormats aggregates and formats distinct subtitle extensions from files.
// Returns a display string like "vtt" or "vtt/srt/xml" in a stable, preferred order.
func deriveSubtitleFormats(files []string) string {
	if len(files) == 0 {
		return ""
	}
	seen := map[string]bool{}
	for _, f := range files {
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(f), "."))
		if ext == "" {
			continue
		}
		seen[ext] = true
	}
	if len(seen) == 0 {
		return ""
	}
	// Preferred order for readability
	pref := []string{"vtt", "srt", "ass", "ssa", "ttml", "xml", "sbv", "lrc", "sub", "idx", "scc", "dfxp", "txt", "json"}
	// Build final list: known in preferred order, then unknown in alphabetical order
	final := make([]string, 0, len(seen))
	for _, p := range pref {
		if seen[p] {
			final = append(final, p)
		}
	}
	var unknown []string
	for ext := range seen {
		known := false
		for _, p := range pref {
			if ext == p {
				known = true
				break
			}
		}
		if !known {
			unknown = append(unknown, ext)
		}
	}
	sort.Strings(unknown)
	final = append(final, unknown...)
	return strings.Join(final, "/")
}

// embedSubtitles 实现字幕嵌入阶段
func (s *Service) embedSubtitles(task *types.DtTaskStatus, progressChan ProgressChan) (string, error) {
	// TODO
	_ = progressChan
	// 返回最终视频文件路径
	embeddedVideo := fmt.Sprintf("downloads/%s.embedded.mp4", task.ID)

	// 在实际实现中，这里应该调用FFmpeg等工具嵌入字幕

	return embeddedVideo, nil
}

// report helpers removed: switched to snapshot diffs and stdout parsing only
// scanSubtitleFiles scans dir for files whose base name starts with "<title>_"
// and whose modification time is newer than startedAt. Returns absolute paths.
// 不限制扩展名：全部返回，由上层据扩展名进行分类（video/subtitle/other）。
func (s *Service) scanSubtitleFiles(dir, title string, startedAt time.Time) []string {
	if dir == "" || title == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	prefix := title + "_"
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// must start with our output template prefix
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		info, err2 := e.Info()
		if err2 != nil {
			continue
		}
		// only accept files created/modified during/after this run (with small clock skew tolerance)
		if info.ModTime().Before(startedAt.Add(-5 * time.Second)) {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

// directory snapshot & diff helpers
type fileMeta struct {
	Size    int64
	ModTime time.Time
}

func (s *Service) dirSnapshot(dir string) map[string]fileMeta {
	m := map[string]fileMeta{}
	if dir == "" {
		return m
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return m
	}
	for _, e := range entries {
		if info, err := e.Info(); err == nil {
			m[e.Name()] = fileMeta{Size: info.Size(), ModTime: info.ModTime()}
		}
	}
	return m
}

// diffNewFiles lists absolute paths of files present now but not in the snapshot before,
// and with modtime not earlier than startedAt - small skew.
func (s *Service) diffNewFiles(before map[string]fileMeta, dir string, startedAt time.Time) []string {
	out := []string{}
	if dir == "" {
		return out
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	skew := startedAt.Add(-5 * time.Second)
	for _, e := range entries {
		name := e.Name()
		if _, ok := before[name]; ok {
			continue
		}
		info, err2 := e.Info()
		if err2 != nil {
			continue
		}
		if info.ModTime().Before(skew) {
			continue
		}
		// exclude common non-final artifacts
		low := strings.ToLower(name)
		if strings.HasSuffix(low, ".part") || strings.HasSuffix(low, ".ytdl") || strings.HasSuffix(low, ".temp") {
			continue
		}
		if strings.HasSuffix(low, ".info.json") || strings.HasSuffix(low, ".description") {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

// monitorProgress 监控进度并发送到前端
func (s *Service) monitorProgress(progressChan ProgressChan) {
	for progress := range progressChan {
		if progress == nil {
			continue
		}

		// 根据不同阶段和状态打印不同的日志
		switch progress.Stage {
		case types.DtStageCompleted:
			// normalize ETA display when completed
			progress.EstimatedTime = "completed"
			logger.Info("Task completed",
				zap.String("id", progress.ID),
			)
		case types.DtStageFailed:
			logger.Error("Task failed",
				zap.String("id", progress.ID),
				zap.String("error", progress.Error),
			)
		}

		s.taskManager.UpdateTaskWith(progress.ID, func(task *types.DtTaskStatus) {
			task.UpdateFromProgress(progress)
		})

		// eventbus
		// 创建事件
		event := &events.BaseEvent{
			ID:        uuid.New().String(),
			Type:      consts.TopicDowntasksProgress,
			Source:    "downtasks",
			Timestamp: time.Now(),
			Data:      progress,
			Metadata: map[string]interface{}{
				"progress": progress,
			},
		}

		s.eventBus.Publish(s.ctx, event)
	}
}

func (s *Service) downDir(source string) (string, error) {
	dreamcreatorDir := s.downloadClient.GetDownloadDirWithDreamcreator()
	if dreamcreatorDir == "" {
		return "", fmt.Errorf("dreamcreator dir is empty")
	}

	// Sanitize subdir name to avoid accidental traversal or separators
	safe := strings.TrimSpace(source)
	safe = filepath.Clean(safe)
	safe = filepath.Base(safe)
	if safe == "." || safe == ".." || safe == "" {
		safe = "general"
	}

	dir := filepath.Join(dreamcreatorDir, safe)
	// check if source dir is exsited
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// if source dir is not exsited, create it
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create source directory: %w", err)
		}
	}

	return dir, nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second) // 四舍五入到最接近的秒

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// speedSmoother 提供时间常数 EMA 的速度平滑与尖峰抑制
type speedSmoother struct {
	tauSec      float64 // 时间常数（秒）
	spikeFactor float64 // 尖峰允许倍数（相对基线）
	lastTime    time.Time
	lastBytes   int
	smoothBps   float64
	start       time.Time
	totalBytes  int64
}

func newSpeedSmoother(tau time.Duration, spikeFactor float64) *speedSmoother {
	if tau <= 0 {
		tau = 2 * time.Second
	}
	if spikeFactor <= 1 {
		spikeFactor = 2.5
	}
	return &speedSmoother{tauSec: tau.Seconds(), spikeFactor: spikeFactor}
}

// Update 以累计字节数更新平滑速度；返回平滑后的 B/s
func (s *speedSmoother) Update(now time.Time, downloadedBytes int) (float64, bool) {
	if s.lastTime.IsZero() {
		s.lastTime = now
		s.start = now
		s.lastBytes = downloadedBytes
		s.smoothBps = 0
		s.totalBytes = 0
		return 0, false
	}
	dt := now.Sub(s.lastTime).Seconds()
	if dt <= 0 {
		return s.smoothBps, s.smoothBps > 0
	}
	delta := downloadedBytes - s.lastBytes
	if delta < 0 {
		// 非单调回退（切片/阶段切换），视为 0 增量
		delta = 0
	}
	s.totalBytes += int64(delta)

	inst := float64(delta) / dt // B/s

	// 时间常数型 EMA：alpha 随 dt 调整
	alpha := 1 - math.Exp(-dt/s.tauSec)
	if alpha < 0.01 {
		alpha = 0.01
	}
	if alpha > 1.0 {
		alpha = 1.0
	}

	// 计算平均速率作为基线
	avg := s.AvgBps(now)
	base := s.smoothBps
	if avg > base {
		base = avg
	}
	if base <= 0 {
		base = inst
	}

	// 尖峰抑制：限制瞬时值不超过基线的 N 倍
	maxInst := s.spikeFactor * base
	if inst > maxInst {
		inst = maxInst
	}

	if s.smoothBps <= 0 {
		s.smoothBps = inst
	} else {
		s.smoothBps = s.smoothBps + alpha*(inst-s.smoothBps)
	}

	s.lastTime = now
	s.lastBytes = downloadedBytes
	return s.smoothBps, true
}

// AvgBps 返回从 start 起到 now 的平均速率（B/s）
func (s *speedSmoother) AvgBps(now time.Time) float64 {
	if s.start.IsZero() {
		return 0
	}
	dt := now.Sub(s.start).Seconds()
	if dt <= 0 {
		return 0
	}
	return float64(s.totalBytes) / dt
}

// formatBandwidth 按量级自适应带宽显示（B/s、KB/s、MB/s、GB/s，1024 进位）
func formatBandwidth(bps float64) string {
	if bps <= 0 {
		return ""
	}
	const (
		KB = 1024.0
		MB = 1024.0 * KB
		GB = 1024.0 * MB
	)
	switch {
	case bps >= GB:
		return fmt.Sprintf("%.2f GB/s", bps/GB)
	case bps >= MB:
		return fmt.Sprintf("%.2f MB/s", bps/MB)
	case bps >= KB:
		return fmt.Sprintf("%.2f KB/s", bps/KB)
	default:
		return fmt.Sprintf("%.0f B/s", bps)
	}
}

// Close 关闭服务，清理资源
func (s *Service) Close() error {
	// 关闭任务管理器，确保持久化存储正确关闭
	if s.taskManager != nil {
		return s.taskManager.Close()
	}
	return nil
}
