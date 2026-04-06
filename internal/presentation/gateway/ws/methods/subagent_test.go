package methods

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"

	"dreamcreator/internal/application/gateway/controlplane"
	gatewaysubagent "dreamcreator/internal/application/gateway/subagent"
	subagentservice "dreamcreator/internal/application/subagent/service"
)

type subagentRunStoreStub struct {
	mu      sync.RWMutex
	records map[string]subagentservice.RunRecord
}

func newSubagentRunStoreStub() *subagentRunStoreStub {
	return &subagentRunStoreStub{records: make(map[string]subagentservice.RunRecord)}
}

func (store *subagentRunStoreStub) Save(_ context.Context, record subagentservice.RunRecord) error {
	store.mu.Lock()
	store.records[record.RunID] = record
	store.mu.Unlock()
	return nil
}

func (store *subagentRunStoreStub) Get(_ context.Context, runID string) (subagentservice.RunRecord, error) {
	store.mu.RLock()
	record, ok := store.records[strings.TrimSpace(runID)]
	store.mu.RUnlock()
	if !ok {
		return subagentservice.RunRecord{}, subagentservice.ErrSubagentRunNotFound
	}
	return record, nil
}

func (store *subagentRunStoreStub) ListByParent(_ context.Context, parentSessionKey string) ([]subagentservice.RunRecord, error) {
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

func (store *subagentRunStoreStub) ListPendingAnnounce(_ context.Context) ([]subagentservice.RunRecord, error) {
	return []subagentservice.RunRecord{}, nil
}

func TestRegisterSubagentSpawnGetList(t *testing.T) {
	router := controlplane.NewRouter(nil)
	store := newSubagentRunStoreStub()
	gateway := gatewaysubagent.NewGatewayService(subagentservice.NewSpawner(), nil, store, nil, nil, nil, nil, nil, nil, nil)
	RegisterSubagent(router, gateway)

	session := &controlplane.SessionContext{ID: "test-session"}
	spawnResp := router.Handle(context.Background(), session, controlplane.RequestFrame{
		ID:     "1",
		Method: "subagent.spawn",
		Params: []byte(`{"parentSessionKey":"session-main","task":"collect logs","runTimeoutSeconds":30}`),
	})
	if !spawnResp.OK {
		t.Fatalf("spawn should succeed: %+v", spawnResp.Error)
	}
	record, ok := spawnResp.Payload.(subagentservice.RunRecord)
	if !ok {
		t.Fatalf("unexpected payload type: %T", spawnResp.Payload)
	}
	if record.RunID == "" {
		t.Fatalf("expected run id")
	}
	if record.Task != "collect logs" {
		t.Fatalf("unexpected task: %s", record.Task)
	}
	if record.RunTimeoutSeconds != 30 {
		t.Fatalf("unexpected timeout: %d", record.RunTimeoutSeconds)
	}

	getResp := router.Handle(context.Background(), session, controlplane.RequestFrame{
		ID:     "2",
		Method: "subagent.get",
		Params: []byte(`{"runId":"` + record.RunID + `"}`),
	})
	if !getResp.OK {
		t.Fatalf("get should succeed: %+v", getResp.Error)
	}
	gotRecord, ok := getResp.Payload.(subagentservice.RunRecord)
	if !ok {
		t.Fatalf("unexpected get payload type: %T", getResp.Payload)
	}
	if gotRecord.RunID != record.RunID {
		t.Fatalf("unexpected run id from get: %s", gotRecord.RunID)
	}

	listResp := router.Handle(context.Background(), session, controlplane.RequestFrame{
		ID:     "3",
		Method: "subagent.list",
		Params: []byte(`{"parentSessionKey":"session-main"}`),
	})
	if !listResp.OK {
		t.Fatalf("list should succeed: %+v", listResp.Error)
	}
	items, ok := listResp.Payload.([]subagentservice.RunRecord)
	if !ok {
		t.Fatalf("unexpected list payload type: %T", listResp.Payload)
	}
	if len(items) != 1 || items[0].RunID != record.RunID {
		t.Fatalf("unexpected list payload: %+v", items)
	}
}

func TestRegisterSubagentSpawnValidation(t *testing.T) {
	router := controlplane.NewRouter(nil)
	store := newSubagentRunStoreStub()
	gateway := gatewaysubagent.NewGatewayService(subagentservice.NewSpawner(), nil, store, nil, nil, nil, nil, nil, nil, nil)
	RegisterSubagent(router, gateway)

	session := &controlplane.SessionContext{ID: "test-session"}
	resp := router.Handle(context.Background(), session, controlplane.RequestFrame{
		ID:     "1",
		Method: "subagent.spawn",
		Params: []byte(`{"parentSessionKey":"session-main"}`),
	})
	if resp.OK {
		t.Fatalf("spawn without task should fail")
	}
	if resp.Error == nil || resp.Error.Code != "spawn_failed" {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	if resp.Error.Message != "task is required" {
		t.Fatalf("unexpected error message: %s", resp.Error.Message)
	}
}
