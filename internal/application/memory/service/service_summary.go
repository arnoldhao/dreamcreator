package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	assistantdto "dreamcreator/internal/application/assistant/dto"
	memorydto "dreamcreator/internal/application/memory/dto"
)

func (service *MemoryService) Stats(ctx context.Context, request memorydto.MemoryStatsRequest) (memorydto.MemoryStats, error) {
	if service == nil || service.db == nil {
		return memorydto.MemoryStats{}, errors.New("memory service unavailable")
	}
	requestIdentity := mergeMemoryIdentity(
		request.Identity,
		request.AssistantID,
		request.ThreadID,
		request.Scope,
		request.Channel,
		request.AccountID,
		request.UserID,
		request.GroupID,
	)
	assistantID := strings.TrimSpace(requestIdentity.AssistantID)
	threadFilter := strings.TrimSpace(requestIdentity.ThreadID)
	if assistantID == "" && threadFilter != "" {
		resolved, err := service.resolveAssistantID(ctx, "", threadFilter)
		if err == nil {
			assistantID = resolved
			threadFilter = ""
		}
	}
	scope := normalizeMemoryScope(requestIdentity.Scope)
	identity := memoryIdentityFilterFromDTO(requestIdentity)
	stats := memorydto.MemoryStats{
		CategoryCounts: make(map[string]int),
		HasFTS:         true,
	}

	where := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if assistantID != "" {
		where = append(where, "assistant_id = ?")
		args = append(args, assistantID)
	}
	if threadFilter != "" {
		where = append(where, "thread_id = ?")
		args = append(args, threadFilter)
	}
	if scope != "" {
		where = append(where, scopeFilterSQL("metadata_json"))
		args = append(args, defaultMemoryScope, scope)
	}
	appendIdentityWhere(&where, &args, "metadata_json", identity)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	countSQL := "SELECT COUNT(*) FROM memory_collections" + whereSQL
	if err := service.db.NewRaw(countSQL, args...).Scan(ctx, &stats.TotalCount); err != nil {
		return memorydto.MemoryStats{}, err
	}

	type categoryCount struct {
		Category string `bun:"category"`
		Count    int    `bun:"count"`
	}
	categoryRows := make([]categoryCount, 0)
	categorySQL := "SELECT category, COUNT(*) AS count FROM memory_collections" + whereSQL + " GROUP BY category"
	if err := service.db.NewRaw(categorySQL, args...).Scan(ctx, &categoryRows); err == nil {
		for _, row := range categoryRows {
			stats.CategoryCounts[normalizeMemoryCategory(row.Category)] = row.Count
		}
	}

	lastSQL := "SELECT MAX(updated_at) FROM memory_collections" + whereSQL
	var lastUpdatedRaw sql.NullString
	if err := service.db.NewRaw(lastSQL, args...).Scan(ctx, &lastUpdatedRaw); err == nil && lastUpdatedRaw.Valid {
		if parsed, ok := parseMemoryTimestamp(lastUpdatedRaw.String); ok {
			stats.LastUpdatedAt = parsed.UTC().Format(time.RFC3339)
			stats.LastMemoryAt = stats.LastUpdatedAt
		}
	}

	if assistantID == "" {
		distinctSQL := "SELECT COUNT(DISTINCT assistant_id) FROM memory_collections" + whereSQL
		_ = service.db.NewRaw(distinctSQL, args...).Scan(ctx, &stats.AssistantCount)
	} else {
		stats.AssistantCount = 1
	}

	settingsValue := service.loadMemorySettings(ctx)
	providerID, modelName := service.resolveEmbeddingModel(ctx, assistantID, settingsValue)
	if providerID != "" && modelName != "" {
		stats.ConfiguredModel = providerID + "/" + modelName
	}
	stats.HasEmbeddings = service.hasAnyEmbeddings(ctx, assistantID, threadFilter, scope, identity)
	return stats, nil
}

func (service *MemoryService) GetSummary(ctx context.Context, request memorydto.MemorySummaryRequest) (memorydto.MemorySummary, error) {
	assistantID := strings.TrimSpace(request.AssistantID)
	// Summary is a dashboard-style aggregate and should include all scopes
	// (assistant/user/group) for the selected assistant.
	stats, err := service.Stats(ctx, memorydto.MemoryStatsRequest{
		AssistantID: assistantID,
		Scope:       allMemoryScopeToken,
	})
	if err != nil {
		return memorydto.MemorySummary{}, err
	}
	scopeCounts, err := service.querySummaryLabelCounts(ctx, assistantID, "scope", defaultMemoryScope)
	if err != nil {
		return memorydto.MemorySummary{}, err
	}
	channelCounts, err := service.querySummaryLabelCounts(ctx, assistantID, "channel", "")
	if err != nil {
		return memorydto.MemorySummary{}, err
	}
	accountCounts, err := service.querySummaryLabelCounts(ctx, assistantID, "accountId", "")
	if err != nil {
		return memorydto.MemorySummary{}, err
	}
	threadCount, err := service.querySummaryDistinctThreadCount(ctx, assistantID)
	if err != nil {
		return memorydto.MemorySummary{}, err
	}
	userCount, err := service.querySummaryDistinctMetadataCount(ctx, assistantID, "userId")
	if err != nil {
		return memorydto.MemorySummary{}, err
	}
	groupCount, err := service.querySummaryDistinctMetadataCount(ctx, assistantID, "groupId")
	if err != nil {
		return memorydto.MemorySummary{}, err
	}
	storage, err := service.collectSummaryStorageStats(ctx, assistantID)
	if err != nil {
		return memorydto.MemorySummary{}, err
	}
	summary := ""
	if assistantID != "" {
		if refreshed, refreshErr := service.RefreshAssistantSummary(ctx, assistantID); refreshErr == nil {
			summary = strings.TrimSpace(refreshed)
		}
		// If assistant has memories but text summary is empty (or refresh failed),
		// fallback to a deterministic stats-based summary to keep UI informative.
		if summary == "" && stats.TotalCount > 0 {
			summary = service.buildAssistantSummaryFallback(stats)
		}
	} else {
		summary = service.buildGlobalSummary(stats)
	}
	return memorydto.MemorySummary{
		AssistantID:     assistantID,
		Summary:         strings.TrimSpace(summary),
		TotalMemories:   stats.TotalCount,
		AssistantCount:  stats.AssistantCount,
		ThreadCount:     threadCount,
		UserCount:       userCount,
		GroupCount:      groupCount,
		ChannelCount:    len(channelCounts),
		AccountCount:    len(accountCounts),
		CategoryCounts:  ensureCountMap(stats.CategoryCounts),
		ScopeCounts:     ensureCountMap(scopeCounts),
		ChannelCounts:   ensureCountMap(channelCounts),
		AccountCounts:   ensureCountMap(accountCounts),
		PrincipalCounts: map[string]int{"user": userCount, "group": groupCount},
		Storage:         storage,
		LastUpdatedAt:   stats.LastUpdatedAt,
	}, nil
}

func (service *MemoryService) querySummaryLabelCounts(
	ctx context.Context,
	assistantID string,
	fieldKey string,
	defaultValue string,
) (map[string]int, error) {
	if service == nil || service.db == nil {
		return map[string]int{}, errors.New("memory service unavailable")
	}
	result := map[string]int{}
	whereSQL, args := summaryAssistantWhereSQL(strings.TrimSpace(assistantID))
	fieldExpr := metadataFieldSQL("metadata_json", fieldKey)
	labelExpr := fieldExpr
	if strings.TrimSpace(defaultValue) != "" {
		escaped := strings.ReplaceAll(strings.TrimSpace(defaultValue), "'", "''")
		labelExpr = "COALESCE(NULLIF(" + fieldExpr + ", ''), '" + escaped + "')"
	} else {
		whereSQL = appendConditionToWhereSQL(whereSQL, fieldExpr+" <> ''")
	}
	querySQL := "SELECT " + labelExpr + " AS label, COUNT(*) AS count FROM memory_collections" + whereSQL + " GROUP BY label"
	type labelCountRow struct {
		Label string `bun:"label"`
		Count int    `bun:"count"`
	}
	rows := make([]labelCountRow, 0)
	if err := service.db.NewRaw(querySQL, args...).Scan(ctx, &rows); err != nil {
		return result, err
	}
	for _, row := range rows {
		label := strings.TrimSpace(row.Label)
		if label == "" {
			continue
		}
		result[label] = row.Count
	}
	return result, nil
}

func (service *MemoryService) querySummaryDistinctThreadCount(ctx context.Context, assistantID string) (int, error) {
	if service == nil || service.db == nil {
		return 0, errors.New("memory service unavailable")
	}
	whereSQL, args := summaryAssistantWhereSQL(strings.TrimSpace(assistantID))
	whereSQL = appendConditionToWhereSQL(whereSQL, "thread_id IS NOT NULL AND TRIM(thread_id) <> ''")
	querySQL := "SELECT COUNT(DISTINCT thread_id) FROM memory_collections" + whereSQL
	var count int
	if err := service.db.NewRaw(querySQL, args...).Scan(ctx, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (service *MemoryService) querySummaryDistinctMetadataCount(ctx context.Context, assistantID string, fieldKey string) (int, error) {
	if service == nil || service.db == nil {
		return 0, errors.New("memory service unavailable")
	}
	whereSQL, args := summaryAssistantWhereSQL(strings.TrimSpace(assistantID))
	fieldExpr := metadataFieldSQL("metadata_json", fieldKey)
	whereSQL = appendConditionToWhereSQL(whereSQL, fieldExpr+" <> ''")
	querySQL := "SELECT COUNT(DISTINCT " + fieldExpr + ") FROM memory_collections" + whereSQL
	var count int
	if err := service.db.NewRaw(querySQL, args...).Scan(ctx, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (service *MemoryService) collectSummaryStorageStats(ctx context.Context, assistantID string) (memorydto.MemorySummaryStorage, error) {
	if service == nil || service.db == nil {
		return memorydto.MemorySummaryStorage{}, errors.New("memory service unavailable")
	}
	result := memorydto.MemorySummaryStorage{}
	whereSQL, args := summaryAssistantWhereSQL(strings.TrimSpace(assistantID))
	collectionsSQL := "SELECT COALESCE(SUM(" +
		"LENGTH(COALESCE(content, '')) + " +
		"LENGTH(COALESCE(metadata_json, '')) + " +
		"LENGTH(COALESCE(category, ''))), 0) FROM memory_collections" + whereSQL
	if err := service.db.NewRaw(collectionsSQL, args...).Scan(ctx, &result.CollectionsBytes); err != nil {
		return memorydto.MemorySummaryStorage{}, err
	}
	chunkWhereSQL, chunkArgs := summaryAssistantWhereSQL(strings.TrimSpace(assistantID))
	chunksSQL := "SELECT COALESCE(SUM(" +
		"LENGTH(COALESCE(content, '')) + " +
		"LENGTH(COALESCE(embedding_json, '')) + " +
		"LENGTH(COALESCE(file_path, ''))), 0) FROM memory_chunks" + chunkWhereSQL
	if err := service.db.NewRaw(chunksSQL, chunkArgs...).Scan(ctx, &result.ChunksBytes); err != nil {
		return memorydto.MemorySummaryStorage{}, err
	}
	avatarBaseDir, avatarBaseErr := memoryAvatarBaseDir()
	if avatarBaseErr == nil {
		result.AvatarCacheBytes = directoryFileBytes(avatarBaseDir)
	}
	result.TotalBytes = result.CollectionsBytes + result.ChunksBytes + result.AvatarCacheBytes
	return result, nil
}

func summaryAssistantWhereSQL(assistantID string) (string, []any) {
	normalizedAssistantID := strings.TrimSpace(assistantID)
	if normalizedAssistantID == "" {
		return "", []any{}
	}
	return " WHERE assistant_id = ?", []any{normalizedAssistantID}
}

func appendConditionToWhereSQL(whereSQL string, condition string) string {
	normalizedCondition := strings.TrimSpace(condition)
	if normalizedCondition == "" {
		return whereSQL
	}
	normalizedWhere := strings.TrimSpace(whereSQL)
	if normalizedWhere == "" {
		return " WHERE " + normalizedCondition
	}
	return " " + normalizedWhere + " AND " + normalizedCondition
}

func ensureCountMap(values map[string]int) map[string]int {
	if len(values) == 0 {
		return map[string]int{}
	}
	result := make(map[string]int, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

func (service *MemoryService) buildGlobalSummary(stats memorydto.MemoryStats) string {
	lines := []string{
		"# Memory Overview",
		"",
		fmt.Sprintf("Total memories: %d", stats.TotalCount),
		fmt.Sprintf("Assistants: %d", stats.AssistantCount),
	}
	if stats.LastUpdatedAt != "" {
		lines = append(lines, "Last updated: "+stats.LastUpdatedAt)
	}
	if len(stats.CategoryCounts) > 0 {
		lines = append(lines, "", "By category:")
		keys := make([]string, 0, len(stats.CategoryCounts))
		for key := range stats.CategoryCounts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- %s: %d", key, stats.CategoryCounts[key]))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (service *MemoryService) buildAssistantSummaryFallback(stats memorydto.MemoryStats) string {
	lines := []string{
		"# Assistant Memory Summary",
		"",
		fmt.Sprintf("Total memories: %d", stats.TotalCount),
	}
	if stats.LastUpdatedAt != "" {
		lines = append(lines, "Last updated: "+stats.LastUpdatedAt)
	}
	if len(stats.CategoryCounts) > 0 {
		lines = append(lines, "", "By category:")
		keys := make([]string, 0, len(stats.CategoryCounts))
		for key := range stats.CategoryCounts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("- %s: %d", key, stats.CategoryCounts[key]))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func (service *MemoryService) hasAnyEmbeddings(
	ctx context.Context,
	assistantID string,
	threadID string,
	scope string,
	identity memoryIdentityFilter,
) bool {
	where := []string{"c.embedding_json IS NOT NULL", "c.embedding_json != ''"}
	args := []any{}
	if strings.TrimSpace(assistantID) != "" {
		where = append(where, "c.assistant_id = ?")
		args = append(args, strings.TrimSpace(assistantID))
	}
	if strings.TrimSpace(threadID) != "" {
		where = append(where, "c.thread_id = ?")
		args = append(args, strings.TrimSpace(threadID))
	}
	if scope != "" {
		where = append(where, scopeFilterSQL("m.metadata_json"))
		args = append(args, defaultMemoryScope, scope)
	}
	appendIdentityWhere(&where, &args, "m.metadata_json", identity)
	sqlQuery := "SELECT COUNT(*) FROM memory_chunks c JOIN memory_collections m ON m.id = c.chunk_id WHERE " + strings.Join(where, " AND ")
	count := 0
	if err := service.db.NewRaw(sqlQuery, args...).Scan(ctx, &count); err != nil {
		return false
	}
	return count > 0
}

func (service *MemoryService) GetAssistantMemorySummary(ctx context.Context, assistantID string) (assistantdto.AssistantMemorySummary, error) {
	summary, err := service.GetSummary(ctx, memorydto.MemorySummaryRequest{AssistantID: assistantID})
	if err != nil {
		return assistantdto.AssistantMemorySummary{}, err
	}
	return assistantdto.AssistantMemorySummary{Summary: summary.Summary}, nil
}
