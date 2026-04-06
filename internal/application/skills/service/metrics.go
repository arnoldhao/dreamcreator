package service

import "sync/atomic"

type SkillsMetricsSnapshot struct {
	DiscoverTotal        int64 `json:"discoverTotal"`
	EligibleTotal        int64 `json:"eligibleTotal"`
	PromptTruncatedTotal int64 `json:"promptTruncatedTotal"`
	InstallAttempts      int64 `json:"installAttempts"`
	InstallSuccess       int64 `json:"installSuccess"`
	InstallFailed        int64 `json:"installFailed"`
}

type skillsMetrics struct {
	discoverTotal        atomic.Int64
	eligibleTotal        atomic.Int64
	promptTruncatedTotal atomic.Int64
	installAttempts      atomic.Int64
	installSuccess       atomic.Int64
	installFailed        atomic.Int64
}

func newSkillsMetrics() *skillsMetrics {
	return &skillsMetrics{}
}

func (metrics *skillsMetrics) snapshot() SkillsMetricsSnapshot {
	if metrics == nil {
		return SkillsMetricsSnapshot{}
	}
	return SkillsMetricsSnapshot{
		DiscoverTotal:        metrics.discoverTotal.Load(),
		EligibleTotal:        metrics.eligibleTotal.Load(),
		PromptTruncatedTotal: metrics.promptTruncatedTotal.Load(),
		InstallAttempts:      metrics.installAttempts.Load(),
		InstallSuccess:       metrics.installSuccess.Load(),
		InstallFailed:        metrics.installFailed.Load(),
	}
}

func (service *SkillsService) GetMetricsSnapshot() SkillsMetricsSnapshot {
	if service == nil {
		return SkillsMetricsSnapshot{}
	}
	return service.metrics.snapshot()
}

func (service *SkillsService) recordPromptDiscovery(discovered int, eligible int, truncated bool) {
	if service == nil || service.metrics == nil {
		return
	}
	if discovered > 0 {
		service.metrics.discoverTotal.Add(int64(discovered))
	}
	if eligible > 0 {
		service.metrics.eligibleTotal.Add(int64(eligible))
	}
	if truncated {
		service.metrics.promptTruncatedTotal.Add(1)
	}
}

func (service *SkillsService) recordInstallAttempt(success bool) {
	if service == nil || service.metrics == nil {
		return
	}
	service.metrics.installAttempts.Add(1)
	if success {
		service.metrics.installSuccess.Add(1)
		return
	}
	service.metrics.installFailed.Add(1)
}
