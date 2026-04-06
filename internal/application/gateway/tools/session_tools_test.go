package tools

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"testing"

	gatewaysubagent "dreamcreator/internal/application/gateway/subagent"
	subagentservice "dreamcreator/internal/application/subagent/service"
	"dreamcreator/internal/infrastructure/llm"
)

type runStoreStub struct {
	mu      sync.RWMutex
	records map[string]subagentservice.RunRecord
}

func newRunStoreStub() *runStoreStub {
	return &runStoreStub{records: make(map[string]subagentservice.RunRecord)}
}

func (store *runStoreStub) Save(_ context.Context, record subagentservice.RunRecord) error {
	store.mu.Lock()
	store.records[record.RunID] = record
	store.mu.Unlock()
	return nil
}

func (store *runStoreStub) Get(_ context.Context, runID string) (subagentservice.RunRecord, error) {
	store.mu.RLock()
	record, ok := store.records[strings.TrimSpace(runID)]
	store.mu.RUnlock()
	if !ok {
		return subagentservice.RunRecord{}, subagentservice.ErrSubagentRunNotFound
	}
	return record, nil
}

func (store *runStoreStub) ListByParent(_ context.Context, parentSessionKey string) ([]subagentservice.RunRecord, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	items := make([]subagentservice.RunRecord, 0)
	for _, record := range store.records {
		if strings.TrimSpace(record.ParentSessionKey) == strings.TrimSpace(parentSessionKey) {
			items = append(items, record)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (store *runStoreStub) ListPendingAnnounce(_ context.Context) ([]subagentservice.RunRecord, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()
	items := make([]subagentservice.RunRecord, 0)
	for _, record := range store.records {
		if record.AnnounceSentAt == nil && record.Status != subagentservice.RunStatusRunning {
			items = append(items, record)
		}
	}
	return items, nil
}

func TestSessionsSpawnToolAccepted(t *testing.T) {
	store := newRunStoreStub()
	gateway := gatewaysubagent.NewGatewayService(subagentservice.NewSpawner(), nil, store, nil, nil, nil, nil, nil, nil, nil)
	handler := runSessionsSpawnTool(gateway)

	ctx := WithRuntimeContext(context.Background(), "session-main", "run-parent")
	ctx = llm.WithRuntimeParams(ctx, llm.RuntimeParams{
		ProviderID:    "provider-a",
		ModelName:     "model-a",
		ThinkingLevel: "high",
	})
	output, err := handler(ctx, `{"task":"整理结果","runTimeoutSeconds":12}`)
	if err != nil {
		t.Fatalf("sessions_spawn error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if payload["status"] != "accepted" {
		t.Fatalf("unexpected status: %v", payload["status"])
	}
	runID, _ := payload["runId"].(string)
	if strings.TrimSpace(runID) == "" {
		t.Fatalf("expected run id in output")
	}
	record, err := store.Get(context.Background(), runID)
	if err != nil {
		t.Fatalf("record not found: %v", err)
	}
	if record.ParentSessionKey != "session-main" {
		t.Fatalf("unexpected parent session key: %s", record.ParentSessionKey)
	}
	if record.ParentRunID != "run-parent" {
		t.Fatalf("unexpected parent run id: %s", record.ParentRunID)
	}
	if record.Task != "整理结果" {
		t.Fatalf("unexpected task: %s", record.Task)
	}
	if record.RunTimeoutSeconds != 12 {
		t.Fatalf("unexpected timeout: %d", record.RunTimeoutSeconds)
	}
	if record.CallerModel != "provider-a/model-a" {
		t.Fatalf("unexpected caller model: %s", record.CallerModel)
	}
	if record.CallerThinking != "high" {
		t.Fatalf("unexpected caller thinking: %s", record.CallerThinking)
	}
	if !strings.HasPrefix(record.ChildSessionKey, "agent:") {
		t.Fatalf("unexpected child session key: %s", record.ChildSessionKey)
	}
}

func TestSessionsSpawnToolRequiresTask(t *testing.T) {
	store := newRunStoreStub()
	gateway := gatewaysubagent.NewGatewayService(subagentservice.NewSpawner(), nil, store, nil, nil, nil, nil, nil, nil, nil)
	handler := runSessionsSpawnTool(gateway)
	_, err := handler(WithRuntimeContext(context.Background(), "session-main", "run-parent"), `{}`)
	if err == nil || err.Error() != "task is required" {
		t.Fatalf("expected task is required error, got %v", err)
	}
}

func TestSubagentsToolRequiresTargetForInfo(t *testing.T) {
	store := newRunStoreStub()
	gateway := gatewaysubagent.NewGatewayService(subagentservice.NewSpawner(), nil, store, nil, nil, nil, nil, nil, nil, nil)
	handler := runSubagentsTool(gateway)
	_, err := handler(WithRuntimeContext(context.Background(), "session-main", "run-parent"), `{"action":"info"}`)
	if err == nil || err.Error() != "target is required" {
		t.Fatalf("expected target is required, got %v", err)
	}
}

func TestSubagentsToolProfilesActionRemoved(t *testing.T) {
	store := newRunStoreStub()
	gateway := gatewaysubagent.NewGatewayService(subagentservice.NewSpawner(), nil, store, nil, nil, nil, nil, nil, nil, nil)
	handler := runSubagentsTool(gateway)
	_, err := handler(WithRuntimeContext(context.Background(), "session-main", "run-parent"), `{"action":"profiles"}`)
	if err == nil || err.Error() != "action profiles is not supported" {
		t.Fatalf("expected profiles action removed error, got %v", err)
	}
}

func TestResolveSubagentTargetSupportsIndexAndScope(t *testing.T) {
	store := newRunStoreStub()
	gateway := gatewaysubagent.NewGatewayService(subagentservice.NewSpawner(), nil, store, nil, nil, nil, nil, nil, nil, nil)

	first, err := gateway.Spawn(context.Background(), subagentservice.SpawnRequest{
		ParentSessionKey: "session-a",
		Task:             "a1",
	})
	if err != nil {
		t.Fatalf("spawn first failed: %v", err)
	}
	second, err := gateway.Spawn(context.Background(), subagentservice.SpawnRequest{
		ParentSessionKey: "session-a",
		Task:             "a2",
	})
	if err != nil {
		t.Fatalf("spawn second failed: %v", err)
	}
	other, err := gateway.Spawn(context.Background(), subagentservice.SpawnRequest{
		ParentSessionKey: "session-b",
		Task:             "b1",
	})
	if err != nil {
		t.Fatalf("spawn other failed: %v", err)
	}

	targetID, err := resolveSubagentTarget(context.Background(), gateway, "session-a", "#1")
	if err != nil {
		t.Fatalf("resolve #1 failed: %v", err)
	}
	if targetID != second.RunID {
		t.Fatalf("expected latest run id %s, got %s", second.RunID, targetID)
	}

	_, err = resolveSubagentTarget(context.Background(), gateway, "session-a", other.RunID)
	if err == nil || err.Error() != "subagent not found" {
		t.Fatalf("expected scoped not found, got %v", err)
	}

	targetID, err = resolveSubagentTarget(context.Background(), gateway, "session-a", first.RunID)
	if err != nil {
		t.Fatalf("resolve run id failed: %v", err)
	}
	if targetID != first.RunID {
		t.Fatalf("unexpected resolved run id: %s", targetID)
	}
}
