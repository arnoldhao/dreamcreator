package service

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	memorydto "dreamcreator/internal/application/memory/dto"
)

func (service *MemoryService) ImportDocs(ctx context.Context, request memorydto.ImportDocsRequest) error {
	if service == nil || service.db == nil {
		return errors.New("memory service unavailable")
	}
	assistantID := strings.TrimSpace(request.WorkspaceID)
	if assistantID == "" {
		return errors.New("workspaceId is required")
	}
	if !service.isAssistantMemoryEnabled(ctx, assistantID) {
		return errors.New("memory is disabled")
	}
	content := strings.TrimSpace(request.Content)
	if content == "" {
		return errors.New("content is required")
	}
	now := service.now().UTC()
	filePath := normalizeImportedDocPath(request.Name, now)
	chunks := splitDocIntoChunks(content, 900)
	if len(chunks) == 0 {
		return errors.New("content is empty")
	}
	if _, err := service.db.ExecContext(ctx,
		"INSERT INTO memory_files(assistant_id, file_path, content, updated_at) VALUES(?, ?, ?, ?) "+
			"ON CONFLICT(assistant_id, file_path) DO UPDATE SET content = excluded.content, updated_at = excluded.updated_at",
		assistantID, filePath, content, now,
	); err != nil {
		return err
	}

	staleIDs, err := service.listChunkIDsByFile(ctx, assistantID, filePath)
	if err != nil {
		return err
	}
	if len(staleIDs) > 0 {
		_, _ = service.db.NewDelete().Model((*memoryCollectionRow)(nil)).
			Where("assistant_id = ?", assistantID).
			Where("id IN (?)", bun.In(staleIDs)).
			Exec(ctx)
		_, _ = service.db.NewDelete().Model((*memoryChunkRow)(nil)).
			Where("assistant_id = ?", assistantID).
			Where("chunk_id IN (?)", bun.In(staleIDs)).
			Exec(ctx)
		for _, id := range staleIDs {
			_, _ = service.db.ExecContext(ctx, "DELETE FROM memory_chunks_fts WHERE chunk_id = ? AND assistant_id = ?", id, assistantID)
			_ = service.deleteVec(ctx, id)
		}
	}

	for idx, chunk := range chunks {
		memoryID := buildImportedChunkID(assistantID, filePath, idx)
		metadataJSON := compactJSON(map[string]any{
			"source":     "import_docs",
			"filePath":   filePath,
			"chunkIndex": idx + 1,
			"scope":      defaultMemoryScope,
		})
		collection := memoryCollectionRow{
			ID:           memoryID,
			AssistantID:  assistantID,
			ThreadID:     sql.NullString{},
			Category:     string(memorydto.MemoryCategoryOther),
			Content:      chunk.Content,
			MetadataJSON: metadataJSON,
			Confidence:   0.6,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if _, err := service.db.NewInsert().Model(&collection).
			On("CONFLICT(id) DO UPDATE").
			Set("assistant_id = EXCLUDED.assistant_id").
			Set("thread_id = EXCLUDED.thread_id").
			Set("category = EXCLUDED.category").
			Set("content = EXCLUDED.content").
			Set("metadata_json = EXCLUDED.metadata_json").
			Set("confidence = EXCLUDED.confidence").
			Set("updated_at = EXCLUDED.updated_at").
			Exec(ctx); err != nil {
			return err
		}

		embedding, _ := service.Embed(ctx, memorydto.EmbedRequest{
			AssistantID: assistantID,
			Input:       chunk.Content,
		})
		chunkRow := memoryChunkRow{
			ChunkID:       memoryID,
			AssistantID:   assistantID,
			ThreadID:      sql.NullString{},
			FilePath:      filePath,
			LineStart:     chunk.LineStart,
			LineEnd:       chunk.LineEnd,
			Content:       chunk.Content,
			EmbeddingJSON: marshalEmbedding(embedding),
			CreatedAt:     now,
		}
		if _, err := service.db.NewInsert().Model(&chunkRow).
			On("CONFLICT(chunk_id) DO UPDATE").
			Set("assistant_id = EXCLUDED.assistant_id").
			Set("thread_id = EXCLUDED.thread_id").
			Set("file_path = EXCLUDED.file_path").
			Set("line_start = EXCLUDED.line_start").
			Set("line_end = EXCLUDED.line_end").
			Set("content = EXCLUDED.content").
			Set("embedding_json = EXCLUDED.embedding_json").
			Exec(ctx); err != nil {
			return err
		}
		if err := service.upsertFTS(ctx, chunkRow); err != nil {
			return err
		}
		_ = service.upsertVec(ctx, chunkRow.ChunkID, embedding)
	}
	_, _ = service.RefreshAssistantSummary(ctx, assistantID)
	return nil
}

func (service *MemoryService) BuildIndex(ctx context.Context, request memorydto.BuildIndexRequest) error {
	if service == nil || service.db == nil {
		return errors.New("memory service unavailable")
	}
	assistantID := strings.TrimSpace(request.WorkspaceID)
	query := service.db.NewSelect().Model((*memoryChunkRow)(nil))
	rows := make([]memoryChunkRow, 0)
	if assistantID != "" {
		query = service.db.NewSelect().Model(&rows).Where("assistant_id = ?", assistantID)
	} else {
		query = service.db.NewSelect().Model(&rows)
	}
	if err := query.Scan(ctx); err != nil {
		return err
	}
	if assistantID != "" {
		_, _ = service.db.ExecContext(ctx,
			"DELETE FROM memory_chunks_fts WHERE assistant_id = ? AND chunk_id NOT IN (SELECT chunk_id FROM memory_chunks WHERE assistant_id = ?)",
			assistantID, assistantID,
		)
	} else {
		_, _ = service.db.ExecContext(ctx,
			"DELETE FROM memory_chunks_fts WHERE chunk_id NOT IN (SELECT chunk_id FROM memory_chunks)",
		)
		_, _ = service.db.ExecContext(ctx,
			"DELETE FROM memory_chunks_vec WHERE chunk_id NOT IN (SELECT chunk_id FROM memory_chunks)",
		)
	}
	for _, row := range rows {
		current := row
		if strings.TrimSpace(current.EmbeddingJSON) == "" {
			embedding, embedErr := service.Embed(ctx, memorydto.EmbedRequest{
				AssistantID: strings.TrimSpace(current.AssistantID),
				Input:       current.Content,
			})
			if embedErr == nil && len(embedding) > 0 {
				current.EmbeddingJSON = marshalEmbedding(embedding)
				_, _ = service.db.NewUpdate().Model((*memoryChunkRow)(nil)).
					Set("embedding_json = ?", current.EmbeddingJSON).
					Where("chunk_id = ?", current.ChunkID).
					Exec(ctx)
			}
		}
		if err := service.upsertFTS(ctx, current); err != nil {
			return err
		}
		_ = service.upsertVec(ctx, current.ChunkID, unmarshalEmbedding(current.EmbeddingJSON))
	}
	return nil
}

func (service *MemoryService) Store(ctx context.Context, request memorydto.MemoryStoreRequest) (memorydto.LTMEntry, error) {
	if service == nil || service.db == nil {
		return memorydto.LTMEntry{}, errors.New("memory service unavailable")
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
	content := strings.TrimSpace(request.Content)
	if content == "" {
		return memorydto.LTMEntry{}, errors.New("content is required")
	}
	assistantID, err := service.resolveAssistantID(ctx, requestIdentity.AssistantID, requestIdentity.ThreadID)
	if err != nil {
		return memorydto.LTMEntry{}, err
	}
	if !service.isAssistantMemoryEnabled(ctx, assistantID) {
		return memorydto.LTMEntry{}, errors.New("memory is disabled")
	}

	category := normalizeMemoryCategory(request.Category)
	if category == "" {
		category = string(memorydto.MemoryCategoryOther)
	}
	confidence := clampFloat(float64(request.Confidence), 0, 1)
	if confidence == 0 {
		confidence = 0.7
	}
	scope := normalizeMemoryScope(requestIdentity.Scope)
	identity := memoryIdentityFilterFromDTO(requestIdentity)
	metadata := ensureMemoryScopeMetadata(request.Metadata, scope)
	metadata = ensureMemoryIdentityMetadata(metadata, identity)
	metadataJSON := compactJSON(metadata)
	now := service.now().UTC()
	memoryID := strings.TrimSpace(service.newID())
	if memoryID == "" {
		memoryID = uuid.NewString()
	}
	threadID := strings.TrimSpace(requestIdentity.ThreadID)

	collection := memoryCollectionRow{
		ID:           memoryID,
		AssistantID:  assistantID,
		ThreadID:     nullString(threadID),
		Category:     category,
		Content:      content,
		MetadataJSON: metadataJSON,
		Confidence:   confidence,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if _, err := service.db.NewInsert().Model(&collection).Exec(ctx); err != nil {
		return memorydto.LTMEntry{}, err
	}

	embedding, _ := service.Embed(ctx, memorydto.EmbedRequest{
		AssistantID: assistantID,
		Input:       content,
	})
	chunk := memoryChunkRow{
		ChunkID:       memoryID,
		AssistantID:   assistantID,
		ThreadID:      nullString(threadID),
		FilePath:      filepath.Join("memory", "collections", memoryID+".md"),
		LineStart:     1,
		LineEnd:       1,
		Content:       content,
		EmbeddingJSON: marshalEmbedding(embedding),
		CreatedAt:     now,
	}
	if _, err := service.db.NewInsert().Model(&chunk).Exec(ctx); err != nil {
		return memorydto.LTMEntry{}, err
	}
	if err := service.upsertFTS(ctx, chunk); err != nil {
		return memorydto.LTMEntry{}, err
	}
	_ = service.upsertVec(ctx, chunk.ChunkID, embedding)

	_, _ = service.RefreshAssistantSummary(ctx, assistantID)
	return memorydto.LTMEntry{
		ID:          memoryID,
		AssistantID: assistantID,
		ThreadID:    threadID,
		Content:     content,
		Category:    category,
		Confidence:  float32(confidence),
		CreatedAt:   now.Format(time.RFC3339),
		UpdatedAt:   now.Format(time.RFC3339),
	}, nil
}

func (service *MemoryService) Forget(ctx context.Context, request memorydto.MemoryForgetRequest) (bool, error) {
	if service == nil || service.db == nil {
		return false, errors.New("memory service unavailable")
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
	memoryID := strings.TrimSpace(request.MemoryID)
	if memoryID == "" {
		return false, errors.New("memoryId is required")
	}
	assistantID, err := service.resolveAssistantID(ctx, requestIdentity.AssistantID, requestIdentity.ThreadID)
	if err != nil {
		return false, err
	}
	scope := normalizeMemoryScope(requestIdentity.Scope)
	identity := memoryIdentityFilterFromDTO(requestIdentity)
	query := service.db.NewDelete().Model((*memoryCollectionRow)(nil)).
		Where("id = ?", memoryID).
		Where("assistant_id = ?", assistantID)
	if scope != "" {
		query = query.Where(scopeFilterSQL("metadata_json"), defaultMemoryScope, scope)
	}
	identityWhere, identityArgs := buildIdentityWhere("metadata_json", identity)
	if identityWhere != "" {
		query = query.Where(identityWhere, identityArgs...)
	}
	result, err := query.Exec(ctx)
	if err != nil {
		return false, err
	}
	rows, _ := result.RowsAffected()
	if rows > 0 {
		_, _ = service.db.NewDelete().Model((*memoryChunkRow)(nil)).
			Where("chunk_id = ?", memoryID).
			Where("assistant_id = ?", assistantID).
			Exec(ctx)
		_, _ = service.db.ExecContext(ctx, "DELETE FROM memory_chunks_fts WHERE chunk_id = ? AND assistant_id = ?", memoryID, assistantID)
		_ = service.deleteVec(ctx, memoryID)
		_, _ = service.RefreshAssistantSummary(ctx, assistantID)
	}
	return rows > 0, nil
}

func (service *MemoryService) Update(ctx context.Context, request memorydto.MemoryUpdateRequest) (memorydto.LTMEntry, error) {
	if service == nil || service.db == nil {
		return memorydto.LTMEntry{}, errors.New("memory service unavailable")
	}
	requestIdentity := mergeMemoryIdentity(
		request.Identity,
		request.AssistantID,
		request.ThreadID,
		"",
		request.Channel,
		request.AccountID,
		request.UserID,
		request.GroupID,
	)
	memoryID := strings.TrimSpace(request.MemoryID)
	if memoryID == "" {
		return memorydto.LTMEntry{}, errors.New("memoryId is required")
	}
	assistantID, err := service.resolveAssistantID(ctx, requestIdentity.AssistantID, requestIdentity.ThreadID)
	if err != nil {
		return memorydto.LTMEntry{}, err
	}
	identity := memoryIdentityFilterFromDTO(requestIdentity)
	row := new(memoryCollectionRow)
	selectQuery := service.db.NewSelect().Model(row).
		Where("id = ?", memoryID).
		Where("assistant_id = ?", assistantID)
	identityWhere, identityArgs := buildIdentityWhere("metadata_json", identity)
	if identityWhere != "" {
		selectQuery = selectQuery.Where(identityWhere, identityArgs...)
	}
	if err := selectQuery.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memorydto.LTMEntry{}, errors.New("memory not found")
		}
		return memorydto.LTMEntry{}, err
	}

	updatedContent := row.Content
	if request.Content != nil {
		trimmed := strings.TrimSpace(*request.Content)
		if trimmed != "" {
			updatedContent = trimmed
		}
	}
	updatedCategory := row.Category
	if request.Category != nil {
		category := normalizeMemoryCategory(*request.Category)
		if category != "" {
			updatedCategory = category
		}
	}
	updatedConfidence := row.Confidence
	if request.Confidence != nil {
		updatedConfidence = clampFloat(float64(*request.Confidence), 0, 1)
	}
	existingMetadata := parseMetadataJSON(row.MetadataJSON)
	existingScope := extractMemoryScopeFromMetadata(existingMetadata)
	updatedScope := existingScope
	if request.Scope != nil {
		updatedScope = normalizeMemoryScope(*request.Scope)
	}
	updatedMetadataMap := cloneMetadataMap(existingMetadata)
	if len(request.Metadata) > 0 {
		updatedMetadataMap = mergeMetadataMaps(updatedMetadataMap, request.Metadata)
	}
	updatedMetadataMap = ensureMemoryScopeMetadata(updatedMetadataMap, updatedScope)
	updatedMetadataMap = ensureMemoryIdentityMetadata(updatedMetadataMap, identity)
	updatedMetadata := compactJSON(updatedMetadataMap)
	updatedAt := service.now().UTC()
	_, err = service.db.NewUpdate().Model((*memoryCollectionRow)(nil)).
		Set("content = ?", updatedContent).
		Set("category = ?", updatedCategory).
		Set("confidence = ?", updatedConfidence).
		Set("metadata_json = ?", updatedMetadata).
		Set("updated_at = ?", updatedAt).
		Where("id = ?", memoryID).
		Where("assistant_id = ?", assistantID).
		Exec(ctx)
	if err != nil {
		return memorydto.LTMEntry{}, err
	}

	embeddingJSON := ""
	if updatedContent != "" {
		embedding, _ := service.Embed(ctx, memorydto.EmbedRequest{
			AssistantID: assistantID,
			Input:       updatedContent,
		})
		embeddingJSON = marshalEmbedding(embedding)
	}
	_, _ = service.db.NewUpdate().Model((*memoryChunkRow)(nil)).
		Set("content = ?", updatedContent).
		Set("embedding_json = ?", embeddingJSON).
		Where("chunk_id = ?", memoryID).
		Where("assistant_id = ?", assistantID).
		Exec(ctx)

	chunk := memoryChunkRow{
		ChunkID:       memoryID,
		AssistantID:   assistantID,
		ThreadID:      row.ThreadID,
		FilePath:      filepath.Join("memory", "collections", memoryID+".md"),
		LineStart:     1,
		LineEnd:       1,
		Content:       updatedContent,
		EmbeddingJSON: embeddingJSON,
		CreatedAt:     row.CreatedAt,
	}
	if err := service.upsertFTS(ctx, chunk); err != nil {
		return memorydto.LTMEntry{}, err
	}
	_ = service.upsertVec(ctx, memoryID, unmarshalEmbedding(embeddingJSON))

	_, _ = service.RefreshAssistantSummary(ctx, assistantID)
	return memorydto.LTMEntry{
		ID:          memoryID,
		AssistantID: assistantID,
		ThreadID:    nullStringValue(row.ThreadID),
		Content:     updatedContent,
		Category:    updatedCategory,
		Confidence:  float32(updatedConfidence),
		CreatedAt:   row.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   updatedAt.Format(time.RFC3339),
	}, nil
}

func (service *MemoryService) listChunkIDsByFile(ctx context.Context, assistantID string, filePath string) ([]string, error) {
	type row struct {
		ChunkID string `bun:"chunk_id"`
	}
	rows := make([]row, 0)
	if err := service.db.NewRaw(
		"SELECT chunk_id FROM memory_chunks WHERE assistant_id = ? AND file_path = ?",
		assistantID,
		filePath,
	).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(rows))
	for _, item := range rows {
		if trimmed := strings.TrimSpace(item.ChunkID); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result, nil
}

func normalizeImportedDocPath(name string, now time.Time) string {
	base := strings.TrimSpace(name)
	if base == "" {
		base = "import-" + now.UTC().Format("20060102-150405") + ".md"
	} else {
		base = filepath.Base(base)
	}
	base = strings.TrimSpace(strings.ReplaceAll(base, " ", "_"))
	base = strings.ReplaceAll(base, "..", "")
	if base == "" {
		base = "import-" + now.UTC().Format("20060102-150405") + ".md"
	}
	if filepath.Ext(base) == "" {
		base += ".md"
	}
	return filepath.Join("memory", "docs", base)
}

func buildImportedChunkID(assistantID string, filePath string, chunkIndex int) string {
	seed := strings.ToLower(strings.TrimSpace(assistantID)) + "|" + strings.TrimSpace(filePath)
	hash := sha1.Sum([]byte(seed))
	return fmt.Sprintf("doc_%s_%03d", hex.EncodeToString(hash[:8]), chunkIndex+1)
}

func splitDocIntoChunks(content string, maxRunes int) []memoryDocChunk {
	if maxRunes <= 0 {
		maxRunes = 900
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	chunks := make([]memoryDocChunk, 0)
	currentLines := make([]string, 0)
	currentRunes := 0
	chunkStart := 1
	flush := func(lineEnd int) {
		if len(currentLines) == 0 {
			return
		}
		joined := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if joined != "" {
			chunks = append(chunks, memoryDocChunk{
				Content:   joined,
				LineStart: chunkStart,
				LineEnd:   lineEnd,
			})
		}
		currentLines = currentLines[:0]
		currentRunes = 0
	}
	for idx, line := range lines {
		lineNo := idx + 1
		additional := len([]rune(line))
		if len(currentLines) > 0 {
			additional++
		}
		if currentRunes+additional > maxRunes && len(currentLines) > 0 {
			flush(lineNo - 1)
			chunkStart = lineNo
		}
		if len(currentLines) == 0 {
			chunkStart = lineNo
		}
		currentLines = append(currentLines, line)
		currentRunes += additional
	}
	flush(len(lines))
	return chunks
}
