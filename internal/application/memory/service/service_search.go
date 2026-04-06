package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

func (service *MemoryService) searchVector(
	ctx context.Context,
	assistantID string,
	threadID string,
	category string,
	scope string,
	identity memoryIdentityFilter,
	queryEmbedding []float32,
	limit int,
) ([]memoryVectorCandidate, error) {
	if len(queryEmbedding) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = maxCandidatePool
	}
	if limit > maxCandidatePool {
		limit = maxCandidatePool
	}
	if rows, err := service.searchVectorBySQLiteVec(ctx, assistantID, threadID, category, scope, identity, queryEmbedding, limit); err == nil && len(rows) > 0 {
		return rows, nil
	}
	return service.searchVectorByCosine(ctx, assistantID, threadID, category, scope, identity, queryEmbedding, limit)
}

func (service *MemoryService) searchVectorBySQLiteVec(
	ctx context.Context,
	assistantID string,
	threadID string,
	category string,
	scope string,
	identity memoryIdentityFilter,
	queryEmbedding []float32,
	limit int,
) ([]memoryVectorCandidate, error) {
	ok, err := service.hasVecTable(ctx)
	if err != nil || !ok {
		return nil, err
	}
	dim, _, err := service.currentVecDimension(ctx)
	if err != nil {
		return nil, err
	}
	if dim <= 0 || dim != len(queryEmbedding) {
		return nil, nil
	}
	where := []string{"c.assistant_id = ?"}
	args := []any{assistantID}
	join := ""
	needsCollectionJoin := category != "" || scope != "" || hasMemoryIdentityFilter(identity)
	if threadID != "" {
		where = append(where, "c.thread_id = ?")
		args = append(args, threadID)
	}
	if needsCollectionJoin {
		join = " JOIN memory_collections m ON m.id = c.chunk_id"
	}
	if category != "" {
		where = append(where, "m.category = ?")
		args = append(args, category)
	}
	if scope != "" {
		where = append(where, scopeFilterSQL("m.metadata_json"))
		args = append(args, defaultMemoryScope, scope)
	}
	appendIdentityWhere(&where, &args, "m.metadata_json", identity)
	queryVector := marshalEmbedding(queryEmbedding)
	if strings.TrimSpace(queryVector) == "" {
		return nil, nil
	}
	sqlQuery := "SELECT v.chunk_id, c.content, v.distance AS distance " +
		"FROM memory_chunks_vec v JOIN memory_chunks c ON c.chunk_id = v.chunk_id" + join +
		" WHERE " + strings.Join(where, " AND ") +
		" AND embedding MATCH vec_f32(?) AND k = ? " +
		" ORDER BY distance ASC LIMIT ?"
	args = append(args, queryVector, limit, limit)
	type row struct {
		ChunkID  string  `bun:"chunk_id"`
		Content  string  `bun:"content"`
		Distance float64 `bun:"distance"`
	}
	rows := make([]row, 0)
	if err := service.db.NewRaw(sqlQuery, args...).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]memoryVectorCandidate, 0, len(rows))
	for _, item := range rows {
		if !isFinite(item.Distance) {
			continue
		}
		result = append(result, memoryVectorCandidate{
			ID:          item.ChunkID,
			Content:     strings.TrimSpace(item.Content),
			VectorScore: clampFloat(1.0/(1.0+item.Distance), 0, 1),
		})
	}
	return result, nil
}

func (service *MemoryService) searchVectorByCosine(
	ctx context.Context,
	assistantID string,
	threadID string,
	category string,
	scope string,
	identity memoryIdentityFilter,
	queryEmbedding []float32,
	limit int,
) ([]memoryVectorCandidate, error) {
	where := []string{"c.assistant_id = ?"}
	args := []any{assistantID}
	join := ""
	needsCollectionJoin := category != "" || scope != "" || hasMemoryIdentityFilter(identity)
	if threadID != "" {
		where = append(where, "c.thread_id = ?")
		args = append(args, threadID)
	}
	if needsCollectionJoin {
		join = " JOIN memory_collections m ON m.id = c.chunk_id"
	}
	if category != "" {
		where = append(where, "m.category = ?")
		args = append(args, category)
	}
	if scope != "" {
		where = append(where, scopeFilterSQL("m.metadata_json"))
		args = append(args, defaultMemoryScope, scope)
	}
	appendIdentityWhere(&where, &args, "m.metadata_json", identity)
	sqlQuery := "SELECT c.chunk_id, c.content, c.embedding_json FROM memory_chunks c" + join +
		" WHERE " + strings.Join(where, " AND ") +
		" ORDER BY c.created_at DESC LIMIT ?"
	args = append(args, limit)
	type row struct {
		ChunkID       string `bun:"chunk_id"`
		Content       string `bun:"content"`
		EmbeddingJSON string `bun:"embedding_json"`
	}
	rows := make([]row, 0)
	if err := service.db.NewRaw(sqlQuery, args...).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	result := make([]memoryVectorCandidate, 0, len(rows))
	for _, item := range rows {
		embedding := unmarshalEmbedding(item.EmbeddingJSON)
		if len(embedding) == 0 {
			continue
		}
		score := cosineSimilarity(queryEmbedding, embedding)
		if !isFinite(score) {
			continue
		}
		result = append(result, memoryVectorCandidate{
			ID:          item.ChunkID,
			Content:     strings.TrimSpace(item.Content),
			VectorScore: clampFloat((score+1)/2, 0, 1),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].VectorScore > result[j].VectorScore
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (service *MemoryService) searchText(
	ctx context.Context,
	assistantID string,
	threadID string,
	category string,
	scope string,
	identity memoryIdentityFilter,
	query string,
	limit int,
) ([]memoryTextCandidate, error) {
	normalizedQuery := strings.TrimSpace(query)
	if normalizedQuery == "" {
		return nil, nil
	}
	ftsQuery := buildFTSQuery(query)
	if ftsQuery == "" {
		return service.searchTextBySubstring(ctx, assistantID, threadID, category, scope, identity, normalizedQuery, limit)
	}
	if limit <= 0 {
		limit = maxCandidatePool
	}
	if limit > maxCandidatePool {
		limit = maxCandidatePool
	}

	where := []string{"memory_chunks_fts MATCH ?", "f.assistant_id = ?"}
	args := []any{ftsQuery, assistantID}
	if threadID != "" {
		where = append(where, "m.thread_id = ?")
		args = append(args, threadID)
	}
	if category != "" {
		where = append(where, "m.category = ?")
		args = append(args, category)
	}
	if scope != "" {
		where = append(where, scopeFilterSQL("m.metadata_json"))
		args = append(args, defaultMemoryScope, scope)
	}
	appendIdentityWhere(&where, &args, "m.metadata_json", identity)
	sqlQuery := "SELECT f.chunk_id, f.content, bm25(memory_chunks_fts) AS rank " +
		"FROM memory_chunks_fts f JOIN memory_collections m ON m.id = f.chunk_id " +
		"WHERE " + strings.Join(where, " AND ") +
		" ORDER BY rank ASC LIMIT ?"
	args = append(args, limit)

	type row struct {
		ChunkID string  `bun:"chunk_id"`
		Content string  `bun:"content"`
		Rank    float64 `bun:"rank"`
	}
	rows := make([]row, 0)
	if err := service.db.NewRaw(sqlQuery, args...).Scan(ctx, &rows); err != nil {
		return service.searchTextBySubstring(ctx, assistantID, threadID, category, scope, identity, normalizedQuery, limit)
	}
	result := make([]memoryTextCandidate, 0, len(rows))
	for _, item := range rows {
		score := bm25RankToScore(item.Rank)
		result = append(result, memoryTextCandidate{
			ID:        item.ChunkID,
			Content:   strings.TrimSpace(item.Content),
			TextScore: score,
		})
	}
	if len(result) == 0 {
		return service.searchTextBySubstring(ctx, assistantID, threadID, category, scope, identity, normalizedQuery, limit)
	}
	return result, nil
}

func (service *MemoryService) searchTextBySubstring(
	ctx context.Context,
	assistantID string,
	threadID string,
	category string,
	scope string,
	identity memoryIdentityFilter,
	query string,
	limit int,
) ([]memoryTextCandidate, error) {
	normalizedQuery := strings.TrimSpace(query)
	if normalizedQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = maxCandidatePool
	}
	if limit > maxCandidatePool {
		limit = maxCandidatePool
	}

	where := []string{"assistant_id = ?", "INSTR(LOWER(content), LOWER(?)) > 0"}
	args := []any{assistantID, normalizedQuery}
	if threadID != "" {
		where = append(where, "thread_id = ?")
		args = append(args, threadID)
	}
	if category != "" {
		where = append(where, "category = ?")
		args = append(args, category)
	}
	if scope != "" {
		where = append(where, scopeFilterSQL("metadata_json"))
		args = append(args, defaultMemoryScope, scope)
	}
	appendIdentityWhere(&where, &args, "metadata_json", identity)
	sqlQuery := "SELECT id, content, updated_at FROM memory_collections " +
		"WHERE " + strings.Join(where, " AND ") +
		" ORDER BY updated_at DESC LIMIT ?"
	args = append(args, limit)

	type row struct {
		ID        string    `bun:"id"`
		Content   string    `bun:"content"`
		UpdatedAt time.Time `bun:"updated_at"`
	}
	rows := make([]row, 0)
	if err := service.db.NewRaw(sqlQuery, args...).Scan(ctx, &rows); err != nil {
		return nil, err
	}

	result := make([]memoryTextCandidate, 0, len(rows))
	for _, item := range rows {
		score := substringMatchScore(item.Content, normalizedQuery)
		if score <= 0 {
			continue
		}
		result = append(result, memoryTextCandidate{
			ID:        item.ID,
			Content:   strings.TrimSpace(item.Content),
			TextScore: score,
		})
	}
	return result, nil
}

func mergeMemoryRankings(
	vectors []memoryVectorCandidate,
	texts []memoryTextCandidate,
	vectorWeight float64,
	textWeight float64,
) []memoryRanking {
	byID := make(map[string]memoryRanking)
	for _, item := range vectors {
		byID[item.ID] = memoryRanking{
			ID:          item.ID,
			Content:     item.Content,
			VectorScore: item.VectorScore,
		}
	}
	for _, item := range texts {
		rank := byID[item.ID]
		rank.ID = item.ID
		if strings.TrimSpace(rank.Content) == "" {
			rank.Content = item.Content
		}
		rank.TextScore = item.TextScore
		byID[item.ID] = rank
	}
	if len(byID) == 0 {
		return nil
	}
	wv := clampFloat(vectorWeight, 0, 1)
	wt := clampFloat(textWeight, 0, 1)
	hasVector := len(vectors) > 0
	hasText := len(texts) > 0
	if !hasVector {
		wv = 0
	}
	if !hasText {
		wt = 0
	}
	if wv == 0 && wt == 0 {
		if hasVector && hasText {
			wv, wt = 0.7, 0.3
		} else if hasVector {
			wv = 1
		} else {
			wt = 1
		}
	}
	sum := wv + wt
	if sum <= 0 {
		sum = 1
	}
	result := make([]memoryRanking, 0, len(byID))
	for _, rank := range byID {
		rank.Score = (wv*rank.VectorScore + wt*rank.TextScore) / sum
		result = append(result, rank)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Score == result[j].Score {
			return result[i].ID < result[j].ID
		}
		return result[i].Score > result[j].Score
	})
	return result
}

func (service *MemoryService) loadCollectionsByIDs(ctx context.Context, ids []string) (map[string]memoryCollectionRow, error) {
	if len(ids) == 0 {
		return map[string]memoryCollectionRow{}, nil
	}
	rows := make([]memoryCollectionRow, 0, len(ids))
	if err := service.db.NewSelect().Model(&rows).Where("id IN (?)", bun.In(ids)).Scan(ctx); err != nil {
		return nil, err
	}
	result := make(map[string]memoryCollectionRow, len(rows))
	for _, row := range rows {
		result[row.ID] = row
	}
	return result, nil
}

func (service *MemoryService) upsertFTS(ctx context.Context, chunk memoryChunkRow) error {
	_, _ = service.db.ExecContext(ctx, "DELETE FROM memory_chunks_fts WHERE chunk_id = ?", chunk.ChunkID)
	_, err := service.db.ExecContext(ctx,
		"INSERT INTO memory_chunks_fts(content, assistant_id, file_path, line_start, line_end, chunk_id) VALUES(?, ?, ?, ?, ?, ?)",
		chunk.Content,
		chunk.AssistantID,
		chunk.FilePath,
		chunk.LineStart,
		chunk.LineEnd,
		chunk.ChunkID,
	)
	return err
}

func (service *MemoryService) upsertVec(ctx context.Context, chunkID string, embedding []float32) error {
	chunkID = strings.TrimSpace(chunkID)
	if chunkID == "" {
		return nil
	}
	if len(embedding) == 0 {
		return service.deleteVec(ctx, chunkID)
	}
	enabled, err := service.ensureVecTable(ctx, len(embedding))
	if err != nil || !enabled {
		return err
	}
	vectorJSON := marshalEmbedding(embedding)
	if strings.TrimSpace(vectorJSON) == "" {
		return nil
	}
	_, err = service.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO memory_chunks_vec(chunk_id, embedding) VALUES(?, vec_f32(?))",
		chunkID,
		vectorJSON,
	)
	return err
}

func (service *MemoryService) deleteVec(ctx context.Context, chunkID string) error {
	ok, err := service.hasVecTable(ctx)
	if err != nil || !ok {
		return err
	}
	_, err = service.db.ExecContext(ctx, "DELETE FROM memory_chunks_vec WHERE chunk_id = ?", strings.TrimSpace(chunkID))
	return err
}

func (service *MemoryService) ensureVecTable(ctx context.Context, dim int) (bool, error) {
	if dim <= 0 {
		return false, nil
	}
	ok, err := service.isSQLiteVecEnabled(ctx)
	if err != nil || !ok {
		return false, err
	}
	currentDim, exists, err := service.currentVecDimension(ctx)
	if err != nil {
		return false, err
	}
	if exists {
		return currentDim == dim, nil
	}
	createSQL := fmt.Sprintf(
		"CREATE VIRTUAL TABLE IF NOT EXISTS memory_chunks_vec USING vec0(chunk_id TEXT PRIMARY KEY, embedding float[%d] distance_metric=cosine)",
		dim,
	)
	if _, err := service.db.ExecContext(ctx, createSQL); err != nil {
		return false, err
	}
	return true, nil
}

func (service *MemoryService) isSQLiteVecEnabled(ctx context.Context) (bool, error) {
	version := ""
	if err := service.db.NewRaw("SELECT vec_version()").Scan(ctx, &version); err != nil {
		return false, nil
	}
	return strings.TrimSpace(version) != "", nil
}

func (service *MemoryService) hasVecTable(ctx context.Context) (bool, error) {
	value := ""
	if err := service.db.NewRaw(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'memory_chunks_vec' LIMIT 1",
	).Scan(ctx, &value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(value) != "", nil
}

func (service *MemoryService) currentVecDimension(ctx context.Context) (int, bool, error) {
	sqlText := ""
	if err := service.db.NewRaw(
		"SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'memory_chunks_vec' LIMIT 1",
	).Scan(ctx, &sqlText); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	match := vecDimPattern.FindStringSubmatch(sqlText)
	if len(match) != 2 {
		return 0, true, nil
	}
	dim, convErr := parseInt(match[1])
	if convErr != nil {
		return 0, true, nil
	}
	return dim, true, nil
}
