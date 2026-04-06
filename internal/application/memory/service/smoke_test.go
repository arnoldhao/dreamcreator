package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domainassistant "dreamcreator/internal/domain/assistant"
	"dreamcreator/internal/domain/providers"
	domainthread "dreamcreator/internal/domain/thread"
	"dreamcreator/internal/infrastructure/persistence"

	memorydto "dreamcreator/internal/application/memory/dto"
	settingsdto "dreamcreator/internal/application/settings/dto"
)

func TestMemoryServiceSmokeFlow(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/embeddings":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"embedding": []float64{0.11, 0.22, 0.33, 0.44}},
				},
			})
		case "/chat/completions":
			var req struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			userPrompt := ""
			for _, item := range req.Messages {
				if strings.TrimSpace(item.Role) == "user" {
					userPrompt = strings.TrimSpace(item.Content)
					break
				}
			}
			content := "[]"
			if strings.Contains(userPrompt, "extract up to") {
				content = `[{"content":"用户偏好简洁回复","category":"preference","confidence":0.9}]`
			} else if strings.Contains(userPrompt, "Summarize this session") {
				content = `{"summary":"会话总结：确定周五发布","memories":[{"content":"计划周五发布","category":"decision","confidence":0.85}]}`
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "chatcmpl-smoke",
				"choices": []map[string]any{
					{
						"finish_reason": "stop",
						"message": map[string]any{
							"role":    "assistant",
							"content": content,
						},
					},
				},
				"usage": map[string]any{
					"prompt_tokens":     10,
					"completion_tokens": 20,
					"total_tokens":      30,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer mockLLM.Close()

	db, err := persistence.OpenSQLite(ctx, persistence.SQLiteConfig{Path: filepath.Join(tmpDir, "memory-smoke.db")})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()

	memorySettings := settingsdto.MemorySettings{
		Enabled:           true,
		EmbeddingProvider: "mock-provider",
		EmbeddingModel:    "mock-embedding",
		LLMProvider:       "mock-provider",
		LLMModel:          "mock-chat",
		RecallTopK:        5,
		VectorWeight:      0.7,
		TextWeight:        0.3,
		RecencyWeight:     1,
		RecencyHalfLife:   14,
		MinScore:          0,
		AutoRecall:        true,
		AutoCapture:       true,
		SessionLifecycle:  true,
		CaptureMaxEntries: 3,
	}

	service := NewMemoryService(
		db.Bun,
		staticSettingsReader{settings: settingsdto.Settings{
			AgentModelProviderID: "mock-provider",
			AgentModelName:       "mock-chat",
			Memory:               memorySettings,
		}},
		staticAssistantReader{},
		nil,
		&staticMessageRepo{
			byThread: map[string][]domainthread.ThreadMessage{
				"thread-smoke": {
					{ID: "m1", ThreadID: "thread-smoke", Role: "user", Content: "这周五上线可以吗", CreatedAt: time.Now().Add(-2 * time.Minute)},
					{ID: "m2", ThreadID: "thread-smoke", Role: "assistant", Content: "可以，建议先灰度", CreatedAt: time.Now().Add(-1 * time.Minute)},
				},
			},
		},
		staticProviderRepo{
			provider: providers.Provider{
				ID:       "mock-provider",
				Name:     "mock",
				Type:     providers.ProviderTypeOpenAI,
				Endpoint: mockLLM.URL,
				Enabled:  true,
			},
		},
		staticModelRepo{},
		staticSecretRepo{
			secret: providers.ProviderSecret{
				ID:         "mock-provider",
				ProviderID: "mock-provider",
				APIKey:     "test-key",
			},
		},
	)

	const assistantID = "assistant-smoke"
	const threadID = "thread-smoke"

	stored, err := service.Store(ctx, memorydto.MemoryStoreRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Content:     "用户偏好中文且简洁",
		Category:    string(memorydto.MemoryCategoryPreference),
		Confidence:  0.9,
		Metadata: map[string]any{
			"source": "smoke",
		},
	})
	if err != nil {
		t.Fatalf("store memory: %v", err)
	}
	if strings.TrimSpace(stored.ID) == "" {
		t.Fatalf("store returned empty id")
	}

	codePref, err := service.Store(ctx, memorydto.MemoryStoreRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Content:     "用户喜欢写代码",
		Category:    string(memorydto.MemoryCategoryPreference),
		Confidence:  0.85,
	})
	if err != nil {
		t.Fatalf("store code preference memory: %v", err)
	}
	if strings.TrimSpace(codePref.ID) == "" {
		t.Fatalf("store code preference returned empty id")
	}

	retrieval, err := service.Recall(ctx, memorydto.MemoryRecallRequest{
		AssistantID: assistantID,
		Query:       "用户回复偏好",
		TopK:        3,
	})
	if err != nil {
		t.Fatalf("recall memory: %v", err)
	}
	if len(retrieval.Entries) == 0 {
		t.Fatalf("recall returned empty entries")
	}

	substringRetrieval, err := service.Recall(ctx, memorydto.MemoryRecallRequest{
		AssistantID: assistantID,
		Query:       "代码",
		TopK:        5,
	})
	if err != nil {
		t.Fatalf("recall substring memory: %v", err)
	}
	if len(substringRetrieval.Entries) == 0 {
		t.Fatalf("recall substring returned empty entries")
	}
	foundCodePreference := false
	for _, entry := range substringRetrieval.Entries {
		if entry.ID == codePref.ID {
			foundCodePreference = true
			break
		}
	}
	if !foundCodePreference {
		t.Fatalf("expected substring recall to include code preference memory")
	}

	updatedContent := "用户偏好中文、简洁且结构化"
	if _, err := service.Update(ctx, memorydto.MemoryUpdateRequest{
		AssistantID: assistantID,
		MemoryID:    stored.ID,
		Content:     &updatedContent,
	}); err != nil {
		t.Fatalf("update memory: %v", err)
	}

	oldScopeEntry, err := service.Store(ctx, memorydto.MemoryStoreRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Content:     "用户喜欢咖啡",
		Category:    string(memorydto.MemoryCategoryPreference),
		Scope:       "assistant",
		Confidence:  0.8,
	})
	if err != nil {
		t.Fatalf("store old scope memory: %v", err)
	}
	if _, err := service.db.ExecContext(
		ctx,
		"UPDATE memory_collections SET updated_at = ? WHERE id = ?",
		time.Now().UTC().Add(-30*24*time.Hour),
		oldScopeEntry.ID,
	); err != nil {
		t.Fatalf("backdate old scope memory: %v", err)
	}
	newScopeEntry, err := service.Store(ctx, memorydto.MemoryStoreRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Content:     "用户喜欢咖啡",
		Category:    string(memorydto.MemoryCategoryPreference),
		Scope:       "assistant",
		Confidence:  0.8,
	})
	if err != nil {
		t.Fatalf("store new scope memory: %v", err)
	}
	projectScopeEntry, err := service.Store(ctx, memorydto.MemoryStoreRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Content:     "项目Alpha将在周五发布",
		Category:    string(memorydto.MemoryCategoryDecision),
		Scope:       "project:alpha",
		Confidence:  0.85,
	})
	if err != nil {
		t.Fatalf("store project scope memory: %v", err)
	}

	recencyRetrieval, err := service.Recall(ctx, memorydto.MemoryRecallRequest{
		AssistantID: assistantID,
		Query:       "用户喜欢咖啡",
		TopK:        10,
		Scope:       "assistant",
	})
	if err != nil {
		t.Fatalf("recall recency/scope: %v", err)
	}
	newPos := -1
	oldPos := -1
	newSourceJSON := ""
	for idx, entry := range recencyRetrieval.Entries {
		if entry.ID == newScopeEntry.ID {
			newPos = idx
			newSourceJSON = entry.SourceJSON
		}
		if entry.ID == oldScopeEntry.ID {
			oldPos = idx
		}
	}
	if newPos < 0 {
		t.Fatalf("expected newest memory to be recalled, but it was missing")
	}
	if oldPos >= 0 && newPos > oldPos {
		t.Fatalf("expected newer memory ranked before old memory, got newPos=%d oldPos=%d", newPos, oldPos)
	}
	if !strings.Contains(newSourceJSON, "recencyScore") {
		t.Fatalf("expected recencyScore in source json, got %s", newSourceJSON)
	}

	assistantScopeEntries, err := service.List(ctx, memorydto.MemoryListRequest{
		AssistantID: assistantID,
		Scope:       "assistant",
		Limit:       50,
	})
	if err != nil {
		t.Fatalf("list assistant scope: %v", err)
	}
	for _, entry := range assistantScopeEntries {
		if entry.ID == projectScopeEntry.ID {
			t.Fatalf("project scope memory should not appear in assistant scope list")
		}
	}
	projectScopeEntries, err := service.List(ctx, memorydto.MemoryListRequest{
		AssistantID: assistantID,
		Scope:       "project:alpha",
		Limit:       50,
	})
	if err != nil {
		t.Fatalf("list project scope: %v", err)
	}
	projectFound := false
	for _, entry := range projectScopeEntries {
		if entry.ID == projectScopeEntry.ID {
			projectFound = true
			break
		}
	}
	if !projectFound {
		t.Fatalf("project scope memory should appear in project scope list")
	}
	projectStats, err := service.Stats(ctx, memorydto.MemoryStatsRequest{
		AssistantID: assistantID,
		Scope:       "project:alpha",
	})
	if err != nil {
		t.Fatalf("stats project scope: %v", err)
	}
	if projectStats.TotalCount == 0 {
		t.Fatalf("expected project scope stats to be non-zero")
	}
	deletedWrongScope, err := service.Forget(ctx, memorydto.MemoryForgetRequest{
		AssistantID: assistantID,
		MemoryID:    projectScopeEntry.ID,
		Scope:       "assistant",
	})
	if err != nil {
		t.Fatalf("forget with wrong scope: %v", err)
	}
	if deletedWrongScope {
		t.Fatalf("forget should not delete memory when scope mismatches")
	}
	deletedProjectScope, err := service.Forget(ctx, memorydto.MemoryForgetRequest{
		AssistantID: assistantID,
		MemoryID:    projectScopeEntry.ID,
		Scope:       "project:alpha",
	})
	if err != nil {
		t.Fatalf("forget with project scope: %v", err)
	}
	if !deletedProjectScope {
		t.Fatalf("forget should delete memory when scope matches")
	}

	telegramUserEntry, err := service.Store(ctx, memorydto.MemoryStoreRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Channel:     "telegram",
		AccountID:   "main-bot",
		UserID:      "u-1",
		Content:     "用户u-1偏好简体中文",
		Category:    string(memorydto.MemoryCategoryPreference),
		Scope:       "assistant",
		Confidence:  0.86,
	})
	if err != nil {
		t.Fatalf("store telegram user memory: %v", err)
	}
	telegramGroupEntry, err := service.Store(ctx, memorydto.MemoryStoreRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Channel:     "telegram",
		AccountID:   "main-bot",
		GroupID:     "g-100",
		Content:     "群组g-100约定每周五发布",
		Category:    string(memorydto.MemoryCategoryDecision),
		Scope:       "assistant",
		Confidence:  0.82,
	})
	if err != nil {
		t.Fatalf("store telegram group memory: %v", err)
	}
	discordUserEntry, err := service.Store(ctx, memorydto.MemoryStoreRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Channel:     "discord",
		AccountID:   "dc-bot",
		UserID:      "u-1",
		Content:     "discord用户u-1偏好英文",
		Category:    string(memorydto.MemoryCategoryPreference),
		Scope:       "assistant",
		Confidence:  0.8,
	})
	if err != nil {
		t.Fatalf("store discord user memory: %v", err)
	}

	telegramUserList, err := service.List(ctx, memorydto.MemoryListRequest{
		AssistantID: assistantID,
		Channel:     "telegram",
		AccountID:   "main-bot",
		UserID:      "u-1",
		Scope:       "assistant",
		Limit:       20,
	})
	if err != nil {
		t.Fatalf("list telegram user scope: %v", err)
	}
	telegramUserFound := false
	for _, entry := range telegramUserList {
		if entry.ID == telegramUserEntry.ID {
			telegramUserFound = true
		}
		if entry.ID == telegramGroupEntry.ID || entry.ID == discordUserEntry.ID {
			t.Fatalf("channel/principal filtering should exclude unrelated entries")
		}
	}
	if !telegramUserFound {
		t.Fatalf("expected telegram user entry in filtered list")
	}

	telegramGroupRecall, err := service.Recall(ctx, memorydto.MemoryRecallRequest{
		AssistantID: assistantID,
		Channel:     "telegram",
		AccountID:   "main-bot",
		GroupID:     "g-100",
		Query:       "每周五发布",
		TopK:        5,
		Scope:       "assistant",
	})
	if err != nil {
		t.Fatalf("recall telegram group memory: %v", err)
	}
	groupHit := false
	for _, entry := range telegramGroupRecall.Entries {
		if entry.ID == telegramGroupEntry.ID {
			groupHit = true
		}
		if entry.ID == discordUserEntry.ID {
			t.Fatalf("group recall should not include discord memory")
		}
	}
	if !groupHit {
		t.Fatalf("expected telegram group memory in recall results")
	}

	telegramGroupStats, err := service.Stats(ctx, memorydto.MemoryStatsRequest{
		AssistantID: assistantID,
		Channel:     "telegram",
		AccountID:   "main-bot",
		GroupID:     "g-100",
		Scope:       "assistant",
	})
	if err != nil {
		t.Fatalf("stats telegram group memory: %v", err)
	}
	if telegramGroupStats.TotalCount == 0 {
		t.Fatalf("expected telegram group stats to be non-zero")
	}

	buildOriginJSON := func(payload map[string]string) string {
		data, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			t.Fatalf("marshal origin json: %v", marshalErr)
		}
		return string(data)
	}
	now := time.Now().UTC()
	if _, err := service.db.ExecContext(
		ctx,
		"INSERT INTO gateway_sessions(session_id, session_key, assistant_id, status, origin_json, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)",
		"sess-user-u1",
		"telegram:thread:sess-user-u1",
		assistantID,
		"active",
		buildOriginJSON(map[string]string{
			"channel":       "telegram",
			"accountId":     "main-bot",
			"chatType":      "private",
			"peerId":        "u-1",
			"peerName":      "Alice",
			"peerUsername":  "alice",
			"peerAvatarUrl": "https://t.me/i/userpic/320/alice.jpg",
		}),
		now,
		now,
	); err != nil {
		t.Fatalf("insert gateway session user profile: %v", err)
	}
	if _, err := service.db.ExecContext(
		ctx,
		"INSERT INTO gateway_sessions(session_id, session_key, assistant_id, status, origin_json, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?)",
		"sess-group-g100",
		"telegram:thread:sess-group-g100",
		assistantID,
		"active",
		buildOriginJSON(map[string]string{
			"channel":       "telegram",
			"accountId":     "main-bot",
			"chatType":      "group",
			"peerId":        "g-100",
			"peerName":      "Dev Group",
			"peerUsername":  "devgroup",
			"peerAvatarUrl": "https://t.me/i/userpic/320/devgroup.jpg",
		}),
		now.Add(2*time.Second),
		now.Add(2*time.Second),
	); err != nil {
		t.Fatalf("insert gateway session group profile: %v", err)
	}

	userPrincipals, err := service.ListPrincipals(ctx, memorydto.MemoryPrincipalListRequest{
		AssistantID:   assistantID,
		Scope:         "assistant",
		Channel:       "telegram",
		AccountID:     "main-bot",
		PrincipalType: "user",
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("list user principals: %v", err)
	}
	var userPrincipal *memorydto.MemoryPrincipalItem
	for index := range userPrincipals {
		if userPrincipals[index].PrincipalID == "u-1" {
			userPrincipal = &userPrincipals[index]
			break
		}
	}
	if userPrincipal == nil {
		t.Fatalf("expected user principal u-1 in principal list")
	}
	if userPrincipal.Name != "Alice" {
		t.Fatalf("unexpected user principal name: %q", userPrincipal.Name)
	}
	if userPrincipal.AvatarURL == "" {
		t.Fatalf("expected user principal avatar url")
	}

	service.SetPrincipalProfileRefresher(staticPrincipalProfileRefresher{
		name:     "Alice Updated",
		username: "alice_updated",
		avatar:   "https://t.me/i/userpic/320/alice_updated.jpg",
	})
	refreshResult, err := service.RefreshPrincipal(ctx, memorydto.MemoryPrincipalRefreshRequest{
		AssistantID:   assistantID,
		PrincipalType: "user",
		PrincipalID:   "u-1",
	})
	if err != nil {
		t.Fatalf("refresh user principal profile: %v", err)
	}
	if refreshResult.UpdatedRows == 0 {
		t.Fatalf("expected refresh to update at least one session row")
	}
	refreshedUserPrincipals, err := service.ListPrincipals(ctx, memorydto.MemoryPrincipalListRequest{
		AssistantID:   assistantID,
		Scope:         "assistant",
		Channel:       "telegram",
		AccountID:     "main-bot",
		PrincipalType: "user",
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("list refreshed user principals: %v", err)
	}
	var refreshedUserPrincipal *memorydto.MemoryPrincipalItem
	for index := range refreshedUserPrincipals {
		if refreshedUserPrincipals[index].PrincipalID == "u-1" {
			refreshedUserPrincipal = &refreshedUserPrincipals[index]
			break
		}
	}
	if refreshedUserPrincipal == nil {
		t.Fatalf("expected refreshed user principal u-1 in principal list")
	}
	if refreshedUserPrincipal.Name != "Alice Updated" {
		t.Fatalf("unexpected refreshed user principal name: %q", refreshedUserPrincipal.Name)
	}
	if strings.TrimSpace(refreshedUserPrincipal.AvatarURL) == "" {
		t.Fatalf("expected refreshed user principal avatar")
	}
	if strings.TrimSpace(refreshedUserPrincipal.AvatarKey) == "" {
		t.Fatalf("expected refreshed user principal avatar key")
	}

	groupPrincipals, err := service.ListPrincipals(ctx, memorydto.MemoryPrincipalListRequest{
		AssistantID:   assistantID,
		Scope:         "assistant",
		Channel:       "telegram",
		AccountID:     "main-bot",
		PrincipalType: "group",
		Limit:         20,
	})
	if err != nil {
		t.Fatalf("list group principals: %v", err)
	}
	var groupPrincipal *memorydto.MemoryPrincipalItem
	for index := range groupPrincipals {
		if groupPrincipals[index].PrincipalID == "g-100" {
			groupPrincipal = &groupPrincipals[index]
			break
		}
	}
	if groupPrincipal == nil {
		t.Fatalf("expected group principal g-100 in principal list")
	}
	if groupPrincipal.Name != "Dev Group" {
		t.Fatalf("unexpected group principal name: %q", groupPrincipal.Name)
	}
	if groupPrincipal.AvatarURL == "" {
		t.Fatalf("expected group principal avatar url")
	}

	before, err := service.BuildRecallContext(ctx, memorydto.BeforeAgentStartRequest{
		AssistantID: assistantID,
		ThreadID:    threadID,
		Query:       "怎么回复这个用户",
		TopK:        3,
	})
	if err != nil {
		t.Fatalf("before_agent_start recall context: %v", err)
	}
	if !strings.Contains(before.InjectedContext, "<relevant-memories>") {
		t.Fatalf("before_agent_start did not inject memory context")
	}

	if err := service.HandleAgentEnd(ctx, memorydto.AgentEndRequest{
		Identity: memorydto.MemoryIdentity{
			AssistantID: assistantID,
			ThreadID:    threadID,
			Scope:       "assistant",
			Channel:     "aui",
			AccountID:   "local-app",
			UserID:      "user-42",
		},
		RunID: "run-smoke",
		Messages: []memorydto.MemoryMessage{
			{Role: "user", Content: "我喜欢简短回复，别太啰嗦"},
			{Role: "assistant", Content: "好的，我会尽量简洁"},
		},
	}); err != nil {
		t.Fatalf("agent_end lifecycle: %v", err)
	}
	agentEndRow := new(memoryCollectionRow)
	if err := service.db.NewSelect().Model(agentEndRow).
		Where("assistant_id = ?", assistantID).
		Where("thread_id = ?", threadID).
		Where("json_extract(metadata_json, '$.source') = ?", "agent_end").
		Where("json_extract(metadata_json, '$.runId') = ?", "run-smoke").
		Order("created_at DESC").
		Limit(1).
		Scan(ctx); err != nil {
		t.Fatalf("load agent_end memory row: %v", err)
	}
	agentEndMetadata := parseMetadataJSON(agentEndRow.MetadataJSON)
	if got := extractMemoryMetadataFieldFromMap(agentEndMetadata, "channel"); got != "aui" {
		t.Fatalf("agent_end metadata channel mismatch: got %q", got)
	}
	if got := extractMemoryMetadataFieldFromMap(agentEndMetadata, "accountId"); got != "local-app" {
		t.Fatalf("agent_end metadata accountId mismatch: got %q", got)
	}
	if got := extractMemoryMetadataFieldFromMap(agentEndMetadata, "userId"); got != "user-42" {
		t.Fatalf("agent_end metadata userId mismatch: got %q", got)
	}
	if got := extractMemoryMetadataFieldFromMap(agentEndMetadata, "groupId"); got != "" {
		t.Fatalf("agent_end metadata groupId should be empty, got %q", got)
	}
	if got := extractMemoryScopeFromMetadata(agentEndMetadata); got != "assistant" {
		t.Fatalf("agent_end metadata scope mismatch: got %q", got)
	}

	if err := service.HandleSessionLifecycle(ctx, memorydto.SessionLifecycleRequest{
		Identity: memorydto.MemoryIdentity{
			AssistantID: assistantID,
			ThreadID:    threadID,
			Scope:       "assistant",
			Channel:     "aui",
			AccountID:   "local-app",
			GroupID:     "group-7",
		},
		Event: memorydto.SessionLifecycleArchived,
	}); err != nil {
		t.Fatalf("session archived lifecycle: %v", err)
	}
	if err := service.HandleSessionLifecycle(ctx, memorydto.SessionLifecycleRequest{
		Identity: memorydto.MemoryIdentity{
			AssistantID: assistantID,
			ThreadID:    threadID,
			Scope:       "assistant",
			Channel:     "aui",
			AccountID:   "local-app",
			GroupID:     "group-7",
		},
		Event: memorydto.SessionLifecycleDeleted,
	}); err != nil {
		t.Fatalf("session deleted lifecycle: %v", err)
	}
	sessionRow := new(memoryCollectionRow)
	if err := service.db.NewSelect().Model(sessionRow).
		Where("assistant_id = ?", assistantID).
		Where("thread_id = ?", threadID).
		Where("json_extract(metadata_json, '$.source') = ?", "session_lifecycle").
		Order("created_at DESC").
		Limit(1).
		Scan(ctx); err != nil {
		t.Fatalf("load session_lifecycle memory row: %v", err)
	}
	sessionMetadata := parseMetadataJSON(sessionRow.MetadataJSON)
	if got := extractMemoryMetadataFieldFromMap(sessionMetadata, "channel"); got != "aui" {
		t.Fatalf("session metadata channel mismatch: got %q", got)
	}
	if got := extractMemoryMetadataFieldFromMap(sessionMetadata, "accountId"); got != "local-app" {
		t.Fatalf("session metadata accountId mismatch: got %q", got)
	}
	if got := extractMemoryMetadataFieldFromMap(sessionMetadata, "groupId"); got != "group-7" {
		t.Fatalf("session metadata groupId mismatch: got %q", got)
	}
	if got := extractMemoryMetadataFieldFromMap(sessionMetadata, "userId"); got != "" {
		t.Fatalf("session metadata userId should be empty, got %q", got)
	}
	if got := extractMemoryScopeFromMetadata(sessionMetadata); got != "assistant" {
		t.Fatalf("session metadata scope mismatch: got %q", got)
	}

	stats, err := service.Stats(ctx, memorydto.MemoryStatsRequest{AssistantID: assistantID})
	if err != nil {
		t.Fatalf("stats memory: %v", err)
	}
	if stats.TotalCount < 3 {
		t.Fatalf("expected >=3 memories after lifecycle writes, got %d", stats.TotalCount)
	}

	entries, err := service.List(ctx, memorydto.MemoryListRequest{AssistantID: assistantID, Limit: 20})
	if err != nil {
		t.Fatalf("list memory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("list returned empty")
	}

	ok, err := service.Forget(ctx, memorydto.MemoryForgetRequest{
		AssistantID: assistantID,
		MemoryID:    stored.ID,
	})
	if err != nil {
		t.Fatalf("forget memory: %v", err)
	}
	if !ok {
		t.Fatalf("forget should return true for existing memory")
	}
}

type staticSettingsReader struct {
	settings settingsdto.Settings
}

func (reader staticSettingsReader) GetSettings(_ context.Context) (settingsdto.Settings, error) {
	return reader.settings, nil
}

type staticAssistantReader struct{}

func (reader staticAssistantReader) Get(_ context.Context, id string) (domainassistant.Assistant, error) {
	if strings.TrimSpace(id) == "" {
		return domainassistant.Assistant{}, errors.New("assistant id required")
	}
	return domainassistant.Assistant{
		ID: id,
		Memory: domainassistant.AssistantMemory{
			Enabled: true,
		},
	}, nil
}

type staticProviderRepo struct {
	provider providers.Provider
}

func (repo staticProviderRepo) List(_ context.Context) ([]providers.Provider, error) {
	return []providers.Provider{repo.provider}, nil
}

func (repo staticProviderRepo) Get(_ context.Context, id string) (providers.Provider, error) {
	if strings.TrimSpace(id) == strings.TrimSpace(repo.provider.ID) {
		return repo.provider, nil
	}
	return providers.Provider{}, errors.New("provider not found")
}

func (repo staticProviderRepo) Save(_ context.Context, provider providers.Provider) error {
	repo.provider = provider
	return nil
}

func (repo staticProviderRepo) Delete(_ context.Context, _ string) error {
	return nil
}

type staticModelRepo struct{}

func (repo staticModelRepo) ListByProvider(_ context.Context, _ string) ([]providers.Model, error) {
	return nil, nil
}

func (repo staticModelRepo) Get(_ context.Context, _ string) (providers.Model, error) {
	return providers.Model{}, errors.New("model not found")
}

func (repo staticModelRepo) Save(_ context.Context, _ providers.Model) error {
	return nil
}

func (repo staticModelRepo) ReplaceByProvider(_ context.Context, _ string, _ []providers.Model) error {
	return nil
}

func (repo staticModelRepo) Delete(_ context.Context, _ string) error {
	return nil
}

type staticSecretRepo struct {
	secret providers.ProviderSecret
}

func (repo staticSecretRepo) GetByProviderID(_ context.Context, providerID string) (providers.ProviderSecret, error) {
	if strings.TrimSpace(providerID) == strings.TrimSpace(repo.secret.ProviderID) {
		return repo.secret, nil
	}
	return providers.ProviderSecret{}, errors.New("secret not found")
}

func (repo staticSecretRepo) Save(_ context.Context, secret providers.ProviderSecret) error {
	repo.secret = secret
	return nil
}

func (repo staticSecretRepo) DeleteByProviderID(_ context.Context, _ string) error {
	return nil
}

type staticMessageRepo struct {
	byThread map[string][]domainthread.ThreadMessage
}

func (repo *staticMessageRepo) ListByThread(_ context.Context, threadID string, limit int) ([]domainthread.ThreadMessage, error) {
	rows := append([]domainthread.ThreadMessage(nil), repo.byThread[strings.TrimSpace(threadID)]...)
	if limit > 0 && len(rows) > limit {
		return rows[:limit], nil
	}
	return rows, nil
}

func (repo *staticMessageRepo) Append(_ context.Context, message domainthread.ThreadMessage) error {
	threadID := strings.TrimSpace(message.ThreadID)
	repo.byThread[threadID] = append(repo.byThread[threadID], message)
	return nil
}

func (repo *staticMessageRepo) DeleteByThread(_ context.Context, threadID string) error {
	delete(repo.byThread, strings.TrimSpace(threadID))
	return nil
}

type staticPrincipalProfileRefresher struct {
	name     string
	username string
	avatar   string
	err      error
}

func (refresher staticPrincipalProfileRefresher) RefreshPrincipalProfile(
	_ context.Context,
	_ string,
	_ string,
	_ string,
	_ string,
) (string, string, string, error) {
	if refresher.err != nil {
		return "", "", "", refresher.err
	}
	return refresher.name, refresher.username, refresher.avatar, nil
}
