package usage

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo  Repository
	now   func() time.Time
	newID func() string
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:  repo,
		now:   time.Now,
		newID: uuid.NewString,
	}
}

func (service *Service) Ingest(ctx context.Context, entry LedgerEntry) error {
	if service == nil || service.repo == nil {
		return errors.New("usage repository unavailable")
	}

	now := service.now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if entry.OccurredAt.IsZero() {
		entry.OccurredAt = entry.CreatedAt
	}

	entry.Category = normalizeCategory(entry.Category)
	entry.RequestSource = normalizeRequestSource(entry.RequestSource)
	entry.CostBasis = normalizeCostBasis(entry.CostBasis)
	entry.ProviderID = normalizeDimension(entry.ProviderID)
	entry.ModelName = normalizeDimension(entry.ModelName)
	entry.Channel = strings.TrimSpace(entry.Channel)

	if strings.TrimSpace(entry.RequestID) == "" {
		requestID := strings.TrimSpace(entry.ID)
		if requestID == "" {
			requestID = service.newID()
		}
		entry.RequestID = requestID
	}
	if strings.TrimSpace(entry.StepID) == "" {
		entry.StepID = "run"
	}
	if entry.Units <= 0 && (entry.InputTokens > 0 || entry.OutputTokens > 0) {
		entry.Units = entry.InputTokens + entry.OutputTokens
	}

	event := UsageEvent{
		ID:                service.newID(),
		RequestID:         strings.TrimSpace(entry.RequestID),
		StepID:            strings.TrimSpace(entry.StepID),
		ProviderID:        entry.ProviderID,
		ModelName:         entry.ModelName,
		Category:          entry.Category,
		Channel:           entry.Channel,
		RequestSource:     entry.RequestSource,
		UsageStatus:       "final",
		InputTokens:       maxInt(entry.InputTokens, 0),
		OutputTokens:      maxInt(entry.OutputTokens, 0),
		TotalTokens:       maxInt(entry.Units, 0),
		CachedInputTokens: maxInt(entry.CachedInputTokens, 0),
		ReasoningTokens:   maxInt(entry.ReasoningTokens, 0),
		RawUsageJSON:      strings.TrimSpace(entry.RawUsageJSON),
		OccurredAt:        entry.OccurredAt.UTC(),
		CreatedAt:         entry.CreatedAt.UTC(),
		UpdatedAt:         entry.CreatedAt.UTC(),
	}
	if event.TotalTokens <= 0 {
		event.TotalTokens = event.InputTokens + event.OutputTokens
	}

	storedEvent, err := service.repo.UpsertEvent(ctx, event)
	if err != nil {
		return err
	}
	entry.EventID = strings.TrimSpace(storedEvent.ID)
	entry.CreatedAt = entry.CreatedAt.UTC()
	entry.OccurredAt = entry.OccurredAt.UTC()

	if entry.Category == CategoryTokens {
		pricing, ok, err := service.repo.ResolvePricingVersion(ctx, entry.ProviderID, entry.ModelName, entry.OccurredAt)
		if err != nil {
			return err
		}
		if ok {
			entry.PricingVersionID = strings.TrimSpace(pricing.ID)
			breakdown := calculateCostBreakdown(entry, pricing)
			entry.InputCostMicros = breakdown.InputCostMicros
			entry.OutputCostMicros = breakdown.OutputCostMicros
			entry.CachedInputCostMicros = breakdown.CachedInputCostMicros
			entry.ReasoningCostMicros = breakdown.ReasoningCostMicros
			entry.RequestCostMicros = breakdown.RequestCostMicros
			entry.CostMicros = breakdown.TotalCostMicros
		}
	}

	if entry.CostMicros <= 0 {
		entry.CostMicros = entry.InputCostMicros + entry.OutputCostMicros + entry.CachedInputCostMicros + entry.ReasoningCostMicros + entry.RequestCostMicros
	}
	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = service.newID()
	}

	return service.repo.UpsertLedger(ctx, entry)
}

func (service *Service) Status(ctx context.Context, request UsageStatusRequest) (UsageStatusResponse, error) {
	entries, window, err := service.loadEntries(ctx, request.Window, request.StartAt, request.EndAt, request.ProviderID, request.ModelName, request.Channel, request.Category, request.RequestSource, request.CostBasis)
	if err != nil {
		return UsageStatusResponse{}, err
	}
	groupBy := normalizeGroupBy(request.GroupBy)
	buckets, totals := aggregateEntries(entries, groupBy, request.TimezoneOffsetMinutes)
	return UsageStatusResponse{
		Window:  window,
		Totals:  totals,
		Buckets: buckets,
	}, nil
}

func (service *Service) Cost(ctx context.Context, request UsageCostRequest) (UsageCostResponse, error) {
	entries, window, err := service.loadEntries(ctx, request.Window, request.StartAt, request.EndAt, request.ProviderID, request.ModelName, request.Channel, request.Category, request.RequestSource, request.CostBasis)
	if err != nil {
		return UsageCostResponse{}, err
	}
	groupBy := normalizeGroupBy(request.GroupBy)
	lines, total := aggregateCost(entries, groupBy, request.TimezoneOffsetMinutes)
	return UsageCostResponse{
		Window:          window,
		TotalCostMicros: total,
		Lines:           lines,
	}, nil
}

func (service *Service) PricingList(ctx context.Context, request PricingListRequest) (PricingListResponse, error) {
	if service == nil || service.repo == nil {
		return PricingListResponse{}, errors.New("usage repository unavailable")
	}
	items, err := service.repo.ListPricingVersions(ctx, PricingVersionFilter{
		ProviderID: strings.TrimSpace(request.ProviderID),
		ModelName:  strings.TrimSpace(request.ModelName),
		Source:     strings.TrimSpace(request.Source),
		ActiveOnly: request.ActiveOnly,
	})
	if err != nil {
		return PricingListResponse{}, err
	}
	return PricingListResponse{Items: items}, nil
}

func (service *Service) PricingUpsert(ctx context.Context, request PricingUpsertRequest) (PricingVersion, error) {
	if service == nil || service.repo == nil {
		return PricingVersion{}, errors.New("usage repository unavailable")
	}
	providerID := strings.TrimSpace(request.ProviderID)
	modelName := strings.TrimSpace(request.ModelName)
	if providerID == "" || modelName == "" {
		return PricingVersion{}, errors.New("providerId and modelName are required")
	}
	effectiveFrom, err := time.Parse(time.RFC3339, strings.TrimSpace(request.EffectiveFrom))
	if err != nil {
		return PricingVersion{}, errors.New("effectiveFrom must be RFC3339")
	}
	var effectiveTo *time.Time
	if trimmed := strings.TrimSpace(request.EffectiveTo); trimmed != "" {
		parsed, parseErr := time.Parse(time.RFC3339, trimmed)
		if parseErr != nil {
			return PricingVersion{}, errors.New("effectiveTo must be RFC3339")
		}
		effectiveTo = &parsed
	}
	isActive := request.IsActive
	if strings.TrimSpace(request.ID) == "" && !request.IsActive {
		isActive = true
	}
	version := PricingVersion{
		ID:                    strings.TrimSpace(request.ID),
		ProviderID:            providerID,
		ModelName:             modelName,
		Currency:              normalizeCurrency(request.Currency),
		InputPerMillion:       clampFloat(request.InputPerMillion),
		OutputPerMillion:      clampFloat(request.OutputPerMillion),
		CachedInputPerMillion: clampFloat(request.CachedInputPerMillion),
		ReasoningPerMillion:   clampFloat(request.ReasoningPerMillion),
		AudioInputPerMillion:  clampFloat(request.AudioInputPerMillion),
		AudioOutputPerMillion: clampFloat(request.AudioOutputPerMillion),
		PerRequest:            clampFloat(request.PerRequest),
		Source:                normalizePricingSource(request.Source),
		EffectiveFrom:         effectiveFrom.UTC(),
		EffectiveTo:           effectiveTo,
		IsActive:              isActive,
		UpdatedBy:             strings.TrimSpace(request.UpdatedBy),
		UpdatedAt:             service.now().UTC(),
	}
	if version.ID == "" {
		version.ID = service.newID()
	}
	return service.repo.UpsertPricingVersion(ctx, version)
}

func (service *Service) PricingDelete(ctx context.Context, request PricingDeleteRequest) error {
	if service == nil || service.repo == nil {
		return errors.New("usage repository unavailable")
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return errors.New("id is required")
	}
	return service.repo.DeletePricingVersion(ctx, id)
}

func (service *Service) PricingActivate(ctx context.Context, request PricingActivateRequest) error {
	if service == nil || service.repo == nil {
		return errors.New("usage repository unavailable")
	}
	id := strings.TrimSpace(request.ID)
	if id == "" {
		return errors.New("id is required")
	}
	return service.repo.ActivatePricingVersion(ctx, id)
}

func (service *Service) loadEntries(
	ctx context.Context,
	window string,
	startAt string,
	endAt string,
	providerID string,
	modelName string,
	channel string,
	category string,
	requestSource string,
	costBasis string,
) ([]LedgerEntry, string, error) {
	if service == nil || service.repo == nil {
		return nil, "", errors.New("usage repository unavailable")
	}
	requestSource = normalizeRequestSourceWithEmpty(requestSource)
	costBasis = normalizeCostBasisWithEmpty(costBasis)
	start, end, resolvedWindow := resolveTimeRange(service.now(), window, startAt, endAt)
	entries, err := service.repo.ListLedger(ctx, QueryFilter{
		StartAt:       start,
		EndAt:         end,
		ProviderID:    strings.TrimSpace(providerID),
		ModelName:     strings.TrimSpace(modelName),
		Channel:       strings.TrimSpace(channel),
		Category:      normalizeCategoryWithEmpty(category),
		RequestSource: requestSource,
		CostBasis:     costBasis,
	})
	if err != nil {
		return nil, resolvedWindow, err
	}
	return entries, resolvedWindow, nil
}

func resolveTimeRange(now time.Time, window string, startAt string, endAt string) (time.Time, time.Time, string) {
	start, end := time.Time{}, time.Time{}
	if startAt != "" {
		if parsed, err := time.Parse(time.RFC3339, startAt); err == nil {
			start = parsed
		}
	}
	if endAt != "" {
		if parsed, err := time.Parse(time.RFC3339, endAt); err == nil {
			end = parsed
		}
	}
	if !start.IsZero() || !end.IsZero() {
		if end.IsZero() {
			end = now
		}
		if start.IsZero() {
			start = end.Add(-24 * time.Hour)
		}
		return start, end, "custom"
	}
	trimmed := strings.ToLower(strings.TrimSpace(window))
	switch trimmed {
	case "1h":
		return now.Add(-time.Hour), now, "1h"
	case "24h":
		return now.Add(-24 * time.Hour), now, "24h"
	case "7d":
		return now.Add(-7 * 24 * time.Hour), now, "7d"
	case "30d":
		return now.Add(-30 * 24 * time.Hour), now, "30d"
	case "all":
		return time.Time{}, now, "all"
	default:
		return now.Add(-24 * time.Hour), now, "24h"
	}
}

func normalizeGroupBy(values []string) []string {
	if len(values) == 0 {
		return []string{"providerId"}
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, raw := range values {
		value := strings.ToLower(strings.TrimSpace(raw))
		switch value {
		case "provider", "providerid":
			value = "providerId"
		case "model", "modelname":
			value = "modelName"
		case "channel":
			value = "channel"
		case "category":
			value = "category"
		case "source", "requestsource", "request_source", "request-source":
			value = "requestSource"
		case "costbasis", "cost_basis", "cost-basis":
			value = "costBasis"
		case "day", "date", "bucketday", "bucket_day":
			value = "day"
		default:
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return []string{"providerId"}
	}
	return result
}

func normalizeRequestSource(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return RequestSourceUnknown
	case "dialogue", "dialog", "chat", "channel":
		return RequestSourceDialogue
	case "relay", "proxy", "forward":
		return RequestSourceRelay
	case "oneshot", "one-shot", "single", "single-shot", "title":
		return RequestSourceOneShot
	case "unknown":
		return RequestSourceUnknown
	default:
		return RequestSourceUnknown
	}
}

func normalizeRequestSourceWithEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return normalizeRequestSource(value)
}

func normalizeCategory(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return CategoryTokens
	case CategoryTokens:
		return CategoryTokens
	case CategoryContextToken:
		return CategoryContextToken
	case CategoryTTS:
		return CategoryTTS
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeCategoryWithEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return normalizeCategory(value)
}

func normalizeCostBasis(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", CostBasisEstimated:
		return CostBasisEstimated
	case CostBasisReconciled:
		return CostBasisReconciled
	default:
		return CostBasisEstimated
	}
}

func normalizeCostBasisWithEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return normalizeCostBasis(value)
}

func normalizeDimension(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

func normalizeCurrency(value string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(value))
	if trimmed == "" {
		return "USD"
	}
	return trimmed
}

func normalizePricingSource(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return "manual"
	}
	return trimmed
}

func clampFloat(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value < 0 {
		return 0
	}
	return value
}

type costBreakdown struct {
	InputCostMicros       int64
	OutputCostMicros      int64
	CachedInputCostMicros int64
	ReasoningCostMicros   int64
	RequestCostMicros     int64
	TotalCostMicros       int64
}

func calculateCostBreakdown(entry LedgerEntry, pricing PricingVersion) costBreakdown {
	inputCost := calculateTokenCostMicros(entry.InputTokens, pricing.InputPerMillion)
	outputCost := calculateTokenCostMicros(entry.OutputTokens, pricing.OutputPerMillion)
	cachedCost := calculateTokenCostMicros(entry.CachedInputTokens, pricing.CachedInputPerMillion)
	reasoningCost := calculateTokenCostMicros(entry.ReasoningTokens, pricing.ReasoningPerMillion)
	requestCost := int64(math.Round(pricing.PerRequest * 1_000_000))
	if requestCost < 0 {
		requestCost = 0
	}
	totalCost := inputCost + outputCost + cachedCost + reasoningCost + requestCost
	return costBreakdown{
		InputCostMicros:       inputCost,
		OutputCostMicros:      outputCost,
		CachedInputCostMicros: cachedCost,
		ReasoningCostMicros:   reasoningCost,
		RequestCostMicros:     requestCost,
		TotalCostMicros:       totalCost,
	}
}

func calculateTokenCostMicros(tokens int, usdPerMillion float64) int64 {
	if tokens <= 0 || usdPerMillion <= 0 {
		return 0
	}
	micros := float64(tokens) * usdPerMillion
	if math.IsNaN(micros) || math.IsInf(micros, 0) || micros <= 0 {
		return 0
	}
	return int64(math.Round(micros))
}

func aggregateEntries(entries []LedgerEntry, groupBy []string, timezoneOffsetMinutes int) ([]UsageBucket, UsageTotals) {
	type bucket struct {
		item UsageBucket
	}
	buckets := make(map[string]*bucket)
	totals := UsageTotals{}
	for _, entry := range entries {
		totals.Requests++
		totals.Units += entry.Units
		totals.InputTokens += entry.InputTokens
		totals.OutputTokens += entry.OutputTokens
		totals.CachedInputTokens += entry.CachedInputTokens
		totals.ReasoningTokens += entry.ReasoningTokens
		totals.CostMicros += entry.CostMicros
		keyParts := make([]string, 0, len(groupBy))
		item := UsageBucket{}
		for _, field := range groupBy {
			switch field {
			case "day":
				bucketStart, bucketEnd := usageDayBucket(entry.CreatedAt, timezoneOffsetMinutes)
				item.BucketStart = bucketStart
				item.BucketEnd = bucketEnd
				keyParts = append(keyParts, bucketStart)
			case "providerId":
				item.ProviderID = entry.ProviderID
				keyParts = append(keyParts, entry.ProviderID)
			case "modelName":
				item.ModelName = entry.ModelName
				keyParts = append(keyParts, entry.ModelName)
			case "channel":
				item.Channel = entry.Channel
				keyParts = append(keyParts, entry.Channel)
			case "category":
				item.Category = entry.Category
				keyParts = append(keyParts, entry.Category)
			case "requestSource":
				item.RequestSource = entry.RequestSource
				keyParts = append(keyParts, entry.RequestSource)
			case "costBasis":
				item.CostBasis = entry.CostBasis
				keyParts = append(keyParts, entry.CostBasis)
			}
		}
		key := strings.Join(keyParts, "::")
		if key == "" {
			key = "all"
		}
		existing := buckets[key]
		if existing == nil {
			item.Key = key
			buckets[key] = &bucket{item: item}
			existing = buckets[key]
		}
		existing.item.Requests++
		existing.item.Units += entry.Units
		existing.item.InputTokens += entry.InputTokens
		existing.item.OutputTokens += entry.OutputTokens
		existing.item.CachedInputTokens += entry.CachedInputTokens
		existing.item.ReasoningTokens += entry.ReasoningTokens
		existing.item.CostMicros += entry.CostMicros
	}
	result := make([]UsageBucket, 0, len(buckets))
	for _, item := range buckets {
		result = append(result, item.item)
	}
	sort.Slice(result, func(left, right int) bool {
		if groupByIncludes(groupBy, "day") && result[left].BucketStart != result[right].BucketStart {
			return result[left].BucketStart < result[right].BucketStart
		}
		if result[left].CostMicros != result[right].CostMicros {
			return result[left].CostMicros > result[right].CostMicros
		}
		if result[left].Requests != result[right].Requests {
			return result[left].Requests > result[right].Requests
		}
		return result[left].Key < result[right].Key
	})
	return result, totals
}

func aggregateCost(entries []LedgerEntry, groupBy []string, timezoneOffsetMinutes int) ([]UsageCostLine, int64) {
	type bucket struct {
		item UsageCostLine
	}
	buckets := make(map[string]*bucket)
	var total int64
	for _, entry := range entries {
		total += entry.CostMicros
		keyParts := make([]string, 0, len(groupBy))
		item := UsageCostLine{}
		for _, field := range groupBy {
			switch field {
			case "day":
				bucketStart, bucketEnd := usageDayBucket(entry.CreatedAt, timezoneOffsetMinutes)
				item.BucketStart = bucketStart
				item.BucketEnd = bucketEnd
				keyParts = append(keyParts, bucketStart)
			case "providerId":
				item.ProviderID = entry.ProviderID
				keyParts = append(keyParts, entry.ProviderID)
			case "modelName":
				item.ModelName = entry.ModelName
				keyParts = append(keyParts, entry.ModelName)
			case "channel":
				item.Channel = entry.Channel
				keyParts = append(keyParts, entry.Channel)
			case "category":
				item.Category = entry.Category
				keyParts = append(keyParts, entry.Category)
			case "requestSource":
				item.RequestSource = entry.RequestSource
				keyParts = append(keyParts, entry.RequestSource)
			case "costBasis":
				item.CostBasis = entry.CostBasis
				keyParts = append(keyParts, entry.CostBasis)
			}
		}
		key := strings.Join(keyParts, "::")
		if key == "" {
			key = "all"
		}
		existing := buckets[key]
		if existing == nil {
			buckets[key] = &bucket{item: item}
			existing = buckets[key]
		}
		existing.item.Requests++
		existing.item.CostMicros += entry.CostMicros
	}
	result := make([]UsageCostLine, 0, len(buckets))
	for _, item := range buckets {
		result = append(result, item.item)
	}
	sort.Slice(result, func(left, right int) bool {
		if groupByIncludes(groupBy, "day") && result[left].BucketStart != result[right].BucketStart {
			return result[left].BucketStart < result[right].BucketStart
		}
		if result[left].CostMicros != result[right].CostMicros {
			return result[left].CostMicros > result[right].CostMicros
		}
		if result[left].Requests != result[right].Requests {
			return result[left].Requests > result[right].Requests
		}
		return buildCostLineKey(result[left]) < buildCostLineKey(result[right])
	})
	return result, total
}

func buildCostLineKey(line UsageCostLine) string {
	return strings.Join([]string{
		line.BucketStart,
		line.ProviderID,
		line.ModelName,
		line.Channel,
		line.Category,
		line.RequestSource,
		line.CostBasis,
	}, "::")
}

func groupByIncludes(groupBy []string, field string) bool {
	for _, item := range groupBy {
		if item == field {
			return true
		}
	}
	return false
}

func usageDayBucket(value time.Time, timezoneOffsetMinutes int) (string, string) {
	if value.IsZero() {
		return "unknown", ""
	}
	if timezoneOffsetMinutes < -14*60 || timezoneOffsetMinutes > 14*60 {
		timezoneOffsetMinutes = 0
	}
	location := time.FixedZone("usage-bucket", -timezoneOffsetMinutes*60)
	local := value.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	return start.Format("2006-01-02"), start.AddDate(0, 0, 1).Format("2006-01-02")
}

func maxInt(value int, lowerBound int) int {
	if value < lowerBound {
		return lowerBound
	}
	return value
}
