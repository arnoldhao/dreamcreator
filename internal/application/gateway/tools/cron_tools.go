package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	gatewaycron "dreamcreator/internal/application/gateway/cron"
	toolservice "dreamcreator/internal/application/tools/service"
)

func RegisterCronTool(ctx context.Context, toolSvc *toolservice.ToolService, executor *RegistryExecutor, scheduler *gatewaycron.Scheduler) {
	if toolSvc == nil || executor == nil {
		return
	}
	handler := runStubTool("cron unavailable")
	if scheduler != nil {
		handler = runCronTool(scheduler)
	}
	registerTool(ctx, toolSvc, executor, specCron(), handler)
}

func runCronTool(scheduler *gatewaycron.Scheduler) func(ctx context.Context, args string) (string, error) {
	return func(ctx context.Context, args string) (string, error) {
		if scheduler == nil {
			return "", errors.New("cron scheduler unavailable")
		}
		payload, err := parseToolArgs(args)
		if err != nil {
			return "", err
		}
		action := strings.ToLower(strings.TrimSpace(getStringArg(payload, "action")))
		if action == "" {
			return "", errors.New("action is required")
		}
		params := getMapArg(payload, "params")
		if params == nil {
			params = map[string]any{}
		}
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return "", err
		}

		switch action {
		case "status":
			return marshalResult(scheduler.Status()), nil
		case "list":
			input, err := gatewaycron.DecodeListInput(paramsJSON)
			if err != nil {
				return "", err
			}
			jobs, err := scheduler.List(ctx)
			if err != nil {
				return "", err
			}
			items := make([]gatewaycron.JobResponse, 0, len(jobs))
			for _, job := range jobs {
				if !input.IncludeDisabled && !job.Enabled {
					continue
				}
				items = append(items, gatewaycron.ToJobResponse(job))
			}
			return marshalResult(map[string]any{
				"items": items,
				"total": len(items),
			}), nil
		case "add":
			input, err := gatewaycron.DecodeCreateInput(paramsJSON)
			if err != nil {
				return "", err
			}
			if strings.TrimSpace(input.SessionKey) == "" {
				if runtimeSessionKey, _ := RuntimeContextFromContext(ctx); strings.TrimSpace(runtimeSessionKey) != "" {
					input.SessionKey = strings.TrimSpace(runtimeSessionKey)
				}
			}
			now := time.Now()
			job := gatewaycron.BuildCronJobFromCreate(input, now)
			stored, err := scheduler.Upsert(ctx, job)
			if err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"job": gatewaycron.ToJobResponse(stored)}), nil
		case "update":
			input, err := gatewaycron.DecodeUpdateInput(paramsJSON)
			if err != nil {
				return "", err
			}
			existing, ok := lookupCronToolJob(ctx, scheduler, strings.TrimSpace(input.ID))
			if !ok {
				return "", errors.New("job not found")
			}
			merged := gatewaycron.ApplyPatch(existing, input.Patch)
			merged.ID = strings.TrimSpace(input.ID)
			merged.JobID = strings.TrimSpace(input.ID)
			stored, err := scheduler.Upsert(ctx, merged)
			if err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"job": gatewaycron.ToJobResponse(stored)}), nil
		case "remove":
			input, err := gatewaycron.DecodeRemoveInput(paramsJSON)
			if err != nil {
				return "", err
			}
			if err := scheduler.Delete(ctx, strings.TrimSpace(input.ID)); err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"ok": true}), nil
		case "run":
			input, err := gatewaycron.DecodeRunInput(paramsJSON)
			if err != nil {
				return "", err
			}
			run, err := scheduler.RunJobWithMode(ctx, strings.TrimSpace(input.ID), strings.TrimSpace(input.Mode))
			if err != nil {
				return "", err
			}
			return marshalResult(map[string]any{"run": run}), nil
		case "runs":
			query, err := gatewaycron.DecodeRunsQuery(paramsJSON)
			if err != nil {
				return "", err
			}
			scope := strings.ToLower(strings.TrimSpace(query.Scope))
			if scope == "" {
				if strings.TrimSpace(query.ID) != "" {
					scope = "job"
				} else {
					scope = "all"
				}
			}
			jobID := ""
			if scope == "job" {
				jobID = strings.TrimSpace(query.ID)
			}
			runs, err := scheduler.ListRuns(ctx, gatewaycron.ListRunsQuery{
				JobID:            jobID,
				Statuses:         query.Statuses,
				DeliveryStatuses: query.DeliveryStatuses,
				Query:            strings.TrimSpace(query.Query),
				SortDir:          strings.TrimSpace(query.SortDir),
				Limit:            query.Limit,
				Offset:           query.Offset,
			})
			if err != nil {
				return "", err
			}
			return marshalResult(map[string]any{
				"items": runs.Items,
				"total": runs.Total,
			}), nil
		case "wake":
			input, err := gatewaycron.DecodeWakeInput(paramsJSON)
			if err != nil {
				return "", err
			}
			result, err := scheduler.Wake(ctx, input.Mode, input.Text, input.SessionKey)
			if err != nil {
				return "", err
			}
			return marshalResult(result), nil
		default:
			return "", errors.New("unsupported cron action: " + action)
		}
	}
}

func lookupCronToolJob(ctx context.Context, scheduler *gatewaycron.Scheduler, jobID string) (gatewaycron.CronJob, bool) {
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
