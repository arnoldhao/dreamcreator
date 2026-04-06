package methods

import (
	"context"
	"sort"
	"strings"
	"time"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewaycron "dreamcreator/internal/application/gateway/cron"
)

const (
	ScopeCronStatus    = "cron.status"
	ScopeCronList      = "cron.list"
	ScopeCronAdd       = "cron.add"
	ScopeCronUpdate    = "cron.update"
	ScopeCronRemove    = "cron.remove"
	ScopeCronRun       = "cron.run"
	ScopeCronRuns      = "cron.runs"
	ScopeCronRunDetail = "cron.runDetail"
	ScopeCronRunEvents = "cron.runEvents"
	ScopeCronWake      = "cron.wake"
)

func RegisterCron(router *controlplane.Router, scheduler *gatewaycron.Scheduler) {
	if router == nil || scheduler == nil {
		return
	}

	router.Register("cron.status", []string{ScopeCronStatus}, func(_ context.Context, _ *controlplane.SessionContext, _ []byte) (any, *controlplane.GatewayError) {
		return scheduler.Status(), nil
	})

	router.Register("cron.list", []string{ScopeCronList}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		payload, err := gatewaycron.DecodeListInput(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		jobs, err := scheduler.List(ctx)
		if err != nil {
			return nil, controlplane.NewGatewayError("internal_error", err.Error())
		}
		includeDisabled := payload.IncludeDisabled
		enabledFilter := strings.ToLower(strings.TrimSpace(payload.Enabled))
		query := strings.ToLower(strings.TrimSpace(payload.Query))
		filtered := make([]gatewaycron.CronJob, 0, len(jobs))
		for _, job := range jobs {
			if !includeDisabled && !job.Enabled {
				continue
			}
			if enabledFilter == "enabled" && !job.Enabled {
				continue
			}
			if enabledFilter == "disabled" && job.Enabled {
				continue
			}
			if query != "" {
				name := strings.ToLower(strings.TrimSpace(job.Name))
				id := strings.ToLower(strings.TrimSpace(job.JobID))
				if !strings.Contains(name, query) && !strings.Contains(id, query) {
					continue
				}
			}
			filtered = append(filtered, job)
		}

		sortBy := strings.ToLower(strings.TrimSpace(payload.SortBy))
		sortDir := strings.ToLower(strings.TrimSpace(payload.SortDir))
		if sortDir != "asc" {
			sortDir = "desc"
		}
		sort.Slice(filtered, func(i, j int) bool {
			left := filtered[i]
			right := filtered[j]
			cmp := 0
			switch sortBy {
			case "name":
				leftName := strings.ToLower(strings.TrimSpace(left.Name))
				rightName := strings.ToLower(strings.TrimSpace(right.Name))
				if leftName < rightName {
					cmp = -1
				} else if leftName > rightName {
					cmp = 1
				}
			case "nextrunatms":
				if left.State.NextRunAtMs < right.State.NextRunAtMs {
					cmp = -1
				} else if left.State.NextRunAtMs > right.State.NextRunAtMs {
					cmp = 1
				}
			default:
				if left.UpdatedAt.Before(right.UpdatedAt) {
					cmp = -1
				} else if left.UpdatedAt.After(right.UpdatedAt) {
					cmp = 1
				}
			}
			if cmp == 0 {
				leftID := strings.TrimSpace(left.JobID)
				rightID := strings.TrimSpace(right.JobID)
				if leftID < rightID {
					cmp = -1
				} else if leftID > rightID {
					cmp = 1
				}
			}
			if sortDir == "asc" {
				return cmp < 0
			}
			return cmp > 0
		})

		total := len(filtered)
		offset := payload.Offset
		if offset < 0 {
			offset = 0
		}
		if offset > total {
			offset = total
		}
		limit := payload.Limit
		if limit <= 0 {
			limit = total
		}
		if limit > 200 {
			limit = 200
		}
		end := offset + limit
		if end > total {
			end = total
		}
		items := filtered[offset:end]
		jobsResp := make([]gatewaycron.JobResponse, 0, len(items))
		for _, job := range items {
			jobsResp = append(jobsResp, gatewaycron.ToJobResponse(job))
		}
		nextOffset := 0
		hasMore := end < total
		if hasMore {
			nextOffset = end
		}
		return map[string]any{
			"items":      jobsResp,
			"total":      total,
			"offset":     offset,
			"limit":      limit,
			"hasMore":    hasMore,
			"nextOffset": nextOffset,
		}, nil
	})

	router.Register("cron.add", []string{ScopeCronAdd}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		input, err := gatewaycron.DecodeCreateInput(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		now := time.Now()
		job := gatewaycron.BuildCronJobFromCreate(input, now)
		stored, err := scheduler.Upsert(ctx, job)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return map[string]any{"job": gatewaycron.ToJobResponse(stored)}, nil
	})

	router.Register("cron.update", []string{ScopeCronUpdate}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		input, err := gatewaycron.DecodeUpdateInput(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		existing, ok := lookupCronJob(ctx, scheduler, strings.TrimSpace(input.ID))
		if !ok {
			return nil, controlplane.NewGatewayError("not_found", "job not found")
		}
		merged := gatewaycron.ApplyPatch(existing, input.Patch)
		merged.ID = strings.TrimSpace(input.ID)
		merged.JobID = strings.TrimSpace(input.ID)
		stored, err := scheduler.Upsert(ctx, merged)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return map[string]any{"job": gatewaycron.ToJobResponse(stored)}, nil
	})

	router.Register("cron.remove", []string{ScopeCronRemove}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		input, err := gatewaycron.DecodeRemoveInput(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		if err := scheduler.Delete(ctx, strings.TrimSpace(input.ID)); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				return nil, controlplane.NewGatewayError("not_found", err.Error())
			}
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return map[string]any{"ok": true}, nil
	})

	router.Register("cron.run", []string{ScopeCronRun}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		input, err := gatewaycron.DecodeRunInput(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		run, err := scheduler.RunJobWithMode(ctx, strings.TrimSpace(input.ID), strings.TrimSpace(input.Mode))
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				return nil, controlplane.NewGatewayError("not_found", err.Error())
			}
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return map[string]any{"run": run}, nil
	})

	router.Register("cron.runs", []string{ScopeCronRuns}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		payload, err := gatewaycron.DecodeRunsQuery(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		scope := strings.ToLower(strings.TrimSpace(payload.Scope))
		if scope == "" {
			if strings.TrimSpace(payload.ID) != "" {
				scope = "job"
			} else {
				scope = "all"
			}
		}
		jobID := ""
		if scope == "job" {
			jobID = strings.TrimSpace(payload.ID)
		}
		result, err := scheduler.ListRuns(ctx, gatewaycron.ListRunsQuery{
			JobID:            jobID,
			Statuses:         payload.Statuses,
			DeliveryStatuses: payload.DeliveryStatuses,
			Query:            strings.TrimSpace(payload.Query),
			SortDir:          strings.TrimSpace(payload.SortDir),
			Limit:            payload.Limit,
			Offset:           payload.Offset,
		})
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		hasMore := payload.Offset+len(result.Items) < result.Total
		nextOffset := 0
		if hasMore {
			nextOffset = payload.Offset + len(result.Items)
		}
		return map[string]any{
			"items":      result.Items,
			"total":      result.Total,
			"offset":     payload.Offset,
			"limit":      payload.Limit,
			"hasMore":    hasMore,
			"nextOffset": nextOffset,
		}, nil
	})

	router.Register("cron.runDetail", []string{ScopeCronRunDetail}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		input, err := gatewaycron.DecodeRunDetailInput(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		detail, err := scheduler.RunDetail(ctx, strings.TrimSpace(input.RunID), input.EventsLimit)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				return nil, controlplane.NewGatewayError("not_found", err.Error())
			}
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return detail, nil
	})

	router.Register("cron.runEvents", []string{ScopeCronRunEvents}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		input, err := gatewaycron.DecodeRunEventsInput(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		result, err := scheduler.ListRunEvents(ctx, gatewaycron.ListRunEventsQuery{
			RunID:   strings.TrimSpace(input.RunID),
			SortDir: strings.TrimSpace(input.SortDir),
			Limit:   input.Limit,
			Offset:  input.Offset,
		})
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				return nil, controlplane.NewGatewayError("not_found", err.Error())
			}
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		hasMore := input.Offset+len(result.Items) < result.Total
		nextOffset := 0
		if hasMore {
			nextOffset = input.Offset + len(result.Items)
		}
		return map[string]any{
			"items":      result.Items,
			"total":      result.Total,
			"offset":     input.Offset,
			"limit":      input.Limit,
			"hasMore":    hasMore,
			"nextOffset": nextOffset,
		}, nil
	})

	router.Register("cron.wake", []string{ScopeCronWake}, func(ctx context.Context, _ *controlplane.SessionContext, params []byte) (any, *controlplane.GatewayError) {
		payload, err := gatewaycron.DecodeWakeInput(params)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_params", err.Error())
		}
		result, err := scheduler.Wake(ctx, payload.Mode, payload.Text, payload.SessionKey)
		if err != nil {
			return nil, controlplane.NewGatewayError("invalid_request", err.Error())
		}
		return result, nil
	})
}

func lookupCronJob(ctx context.Context, scheduler *gatewaycron.Scheduler, jobID string) (gatewaycron.CronJob, bool) {
	if scheduler == nil {
		return gatewaycron.CronJob{}, false
	}
	jobs, err := scheduler.List(ctx)
	if err != nil {
		return gatewaycron.CronJob{}, false
	}
	needle := strings.TrimSpace(jobID)
	for _, job := range jobs {
		if strings.TrimSpace(job.JobID) == needle {
			return job, true
		}
	}
	return gatewaycron.CronJob{}, false
}
