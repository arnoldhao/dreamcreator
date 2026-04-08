package update

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"dreamcreator/internal/application/events"
	"dreamcreator/internal/application/softwareupdate"
	"dreamcreator/internal/domain/update"
	"go.uber.org/zap"
)

type Downloader interface {
	Download(ctx context.Context, url string, progress func(int)) (string, error)
}

type Installer interface {
	Install(ctx context.Context, artifactPath string) error
	RestartToApply(ctx context.Context) error
}

type downloadURLSelector interface {
	SelectDownloadURLs(ctx context.Context, urls []string) []string
}

type Notifier interface {
	SetUpdateAvailable(available bool)
	NotifyUpdateState(info update.Info)
}

type Service struct {
	mu             sync.Mutex
	state          update.Info
	catalog        *softwareupdate.Service
	downloader     Downloader
	installer      Installer
	bus            events.Bus
	notifier       Notifier
	now            func() time.Time
	scheduleTicker *time.Ticker
	cancelSchedule context.CancelFunc
	downloadURLs   []string
}

type ServiceParams struct {
	Catalog    *softwareupdate.Service
	Downloader Downloader
	Installer  Installer
	Bus        events.Bus
	Notifier   Notifier
}

func NewService(params ServiceParams) *Service {
	return &Service{
		catalog:    params.Catalog,
		downloader: params.Downloader,
		installer:  params.Installer,
		bus:        params.Bus,
		notifier:   params.Notifier,
		now:        time.Now,
		state: update.Info{
			Kind:   update.KindApp,
			Status: update.StatusIdle,
		},
	}
}

func (service *Service) State() update.Info {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.state
}

// PublishCurrentState pushes current state to subscribers/notifier.
func (service *Service) PublishCurrentState() {
	service.publishState()
}

// SetCurrentVersion seeds the current version so it can be surfaced before any check.
func (service *Service) SetCurrentVersion(version string) {
	service.mu.Lock()
	service.state.CurrentVersion = update.NormalizeVersion(version)
	if service.state.CurrentVersion != "" &&
		service.state.LatestVersion != "" &&
		update.CompareVersion(service.state.CurrentVersion, service.state.LatestVersion) >= 0 {
		service.state.Status = update.StatusIdle
		service.state.Progress = 0
		service.state.DownloadURL = ""
		service.state.Message = ""
		service.downloadURLs = nil
	}
	service.mu.Unlock()
}

func (service *Service) CheckForUpdate(ctx context.Context, currentVersion string) (update.Info, error) {
	service.mu.Lock()
	if currentVersion != "" {
		service.state.CurrentVersion = update.NormalizeVersion(currentVersion)
	}
	service.setStatusLocked(update.StatusChecking, 0, "")
	state := service.state
	service.mu.Unlock()
	service.publishSnapshot(state)
	zap.L().Info("update: checking for updates",
		zap.String("currentVersion", state.CurrentVersion),
	)

	if service.catalog == nil {
		return service.publishError(fmt.Errorf("software update catalog not configured"))
	}

	_, _ = service.catalog.RefreshCatalog(ctx, softwareupdate.Request{AppVersion: state.CurrentVersion})
	release, err := service.catalog.ResolveAppRelease(ctx, softwareupdate.AppRequest{
		CurrentVersion: state.CurrentVersion,
	})

	if err != nil {
		service.mu.Lock()
		service.setStatusLocked(update.StatusError, service.state.Progress, err.Error())
		state := service.state
		service.mu.Unlock()
		go service.publishSnapshot(state)
		return state, err
	}

	downloadURLs := service.selectDownloadURLs(ctx, release.Asset.DownloadURLs())

	service.mu.Lock()
	defer service.mu.Unlock()
	latest := update.NormalizeVersion(release.Version)
	current := update.NormalizeVersion(service.state.CurrentVersion)
	service.state.LatestVersion = latest
	service.state.Changelog = release.Notes
	service.state.DownloadURL = ""
	if len(downloadURLs) > 0 {
		service.state.DownloadURL = downloadURLs[0]
	}
	service.state.CheckedAt = service.now()
	service.downloadURLs = downloadURLs
	zap.L().Info("update: check result",
		zap.String("currentVersion", current),
		zap.String("latestVersion", latest),
		zap.Bool("hasDownload", service.state.DownloadURL != ""),
	)

	if current != "" && latest != "" && update.CompareVersion(current, latest) >= 0 {
		service.setStatusLocked(update.StatusNoUpdate, 0, "")
		state := service.state
		service.downloadURLs = nil
		service.notifyAvailability(false)
		go service.publishSnapshot(state)
		return state, nil
	}

	if service.state.DownloadURL == "" {
		service.setStatusLocked(update.StatusError, service.state.Progress, "no downloadable asset for update")
		state := service.state
		go service.publishSnapshot(state)
		return state, fmt.Errorf("no downloadable asset for update")
	}

	service.setStatusLocked(update.StatusAvailable, 0, "")
	state = service.state
	service.notifyAvailability(true)
	go service.publishSnapshot(state)
	return state, nil
}

func (service *Service) DownloadUpdate(ctx context.Context) (update.Info, error) {
	service.mu.Lock()
	downloadURLs := service.resolveDownloadURLsLocked()
	service.setStatusLocked(update.StatusDownloading, 0, "")
	state := service.state
	service.mu.Unlock()
	service.publishSnapshot(state)

	if len(downloadURLs) == 0 {
		return service.publishError(fmt.Errorf("missing download url"))
	}
	if service.downloader == nil {
		return service.publishError(fmt.Errorf("downloader not configured"))
	}

	var path string
	var err error
	for _, downloadURL := range downloadURLs {
		path, err = service.downloader.Download(ctx, downloadURL, func(progress int) {
			service.mu.Lock()
			service.state.Progress = progress
			state := service.state
			service.mu.Unlock()
			service.publishSnapshot(state)
		})
		if err == nil {
			break
		}
		zap.L().Warn("update: download source failed", zap.String("url", downloadURL), zap.Error(err))
	}
	if err != nil {
		return service.publishError(err)
	}

	service.mu.Lock()
	service.state.Progress = 100
	service.state.Message = path
	service.setStatusLocked(update.StatusInstalling, 100, "")
	installingState := service.state
	service.mu.Unlock()
	service.publishSnapshot(installingState)

	if service.installer != nil {
		if err := service.installer.Install(ctx, path); err != nil {
			return service.publishError(err)
		}
	}

	service.mu.Lock()
	service.setStatusLocked(update.StatusReadyToRestart, 100, "")
	finalState := service.state
	service.mu.Unlock()
	service.notifyAvailability(true)
	service.publishSnapshot(finalState)

	return finalState, nil
}

func (service *Service) RestartToApply(ctx context.Context) (update.Info, error) {
	if service.installer == nil {
		return service.publishError(fmt.Errorf("installer not configured"))
	}
	if err := service.installer.RestartToApply(ctx); err != nil {
		return service.publishError(err)
	}
	service.mu.Lock()
	service.setStatusLocked(update.StatusIdle, 0, "")
	service.downloadURLs = nil
	state := service.state
	service.mu.Unlock()
	service.notifyAvailability(false)
	service.publishSnapshot(state)
	return state, nil
}

// ScheduleAutoCheck starts a ticker-based auto check; call StopAutoCheck on shutdown.
func (service *Service) ScheduleAutoCheck(ctx context.Context, initialDelay time.Duration, interval time.Duration, currentVersion string) {
	service.StopAutoCheck()
	if initialDelay <= 0 {
		initialDelay = 3 * time.Minute
	}
	if interval <= 0 {
		interval = time.Hour
	}

	runCtx, cancel := context.WithCancel(ctx)
	service.cancelSchedule = cancel

	go func() {
		select {
		case <-time.After(initialDelay):
			service.safeCheck(runCtx, currentVersion)
		case <-runCtx.Done():
			return
		}

		ticker := time.NewTicker(interval)
		service.mu.Lock()
		service.scheduleTicker = ticker
		service.mu.Unlock()

		for {
			select {
			case <-ticker.C:
				service.safeCheck(runCtx, currentVersion)
			case <-runCtx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (service *Service) StopAutoCheck() {
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.cancelSchedule != nil {
		service.cancelSchedule()
	}
	if service.scheduleTicker != nil {
		service.scheduleTicker.Stop()
	}
	service.scheduleTicker = nil
	service.cancelSchedule = nil
}

func (service *Service) safeCheck(ctx context.Context, currentVersion string) {
	_, _ = service.CheckForUpdate(ctx, currentVersion) // errors already published
}

func (service *Service) publishError(err error) (update.Info, error) {
	service.mu.Lock()
	service.setStatusLocked(update.StatusError, service.state.Progress, err.Error())
	state := service.state
	service.mu.Unlock()
	service.publishSnapshot(state)
	return state, err
}

func (service *Service) setStatusLocked(status update.Status, progress int, message string) {
	service.state.Status = status
	if progress >= 0 && progress <= 100 {
		service.state.Progress = progress
	}
	service.state.Message = message
}

func (service *Service) publishState() {
	service.mu.Lock()
	state := service.state
	service.mu.Unlock()
	service.publishSnapshot(state)
}

func (service *Service) publishSnapshot(state update.Info) {
	if service.bus != nil {
		_ = service.bus.Publish(context.Background(), events.Event{
			Topic:   "update.status",
			Type:    "status",
			Payload: state,
		})
	}
	if service.notifier != nil {
		service.notifier.NotifyUpdateState(state)
	}
}

func (service *Service) notifyAvailability(available bool) {
	if service.notifier == nil {
		return
	}
	service.notifier.SetUpdateAvailable(available)
}

func (service *Service) resolveDownloadURLsLocked() []string {
	if len(service.downloadURLs) > 0 {
		return slices.Clone(service.downloadURLs)
	}
	if strings.TrimSpace(service.state.DownloadURL) == "" {
		return nil
	}
	return []string{service.state.DownloadURL}
}

func (service *Service) selectDownloadURLs(ctx context.Context, urls []string) []string {
	if selector, ok := service.installer.(downloadURLSelector); ok && selector != nil {
		if selected := selector.SelectDownloadURLs(ctx, urls); len(selected) > 0 {
			return selected
		}
	}
	return slices.Clone(urls)
}
