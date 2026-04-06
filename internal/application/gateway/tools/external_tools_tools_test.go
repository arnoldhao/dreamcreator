package tools

import (
	"context"
	"encoding/json"
	"testing"

	externaltoolsservice "dreamcreator/internal/application/externaltools/service"
	"dreamcreator/internal/domain/externaltools"
)

type externalToolsRepoStub struct {
	items map[string]externaltools.ExternalTool
}

func newExternalToolsRepoStub() *externalToolsRepoStub {
	return &externalToolsRepoStub{items: make(map[string]externaltools.ExternalTool)}
}

func (repo *externalToolsRepoStub) List(_ context.Context) ([]externaltools.ExternalTool, error) {
	result := make([]externaltools.ExternalTool, 0, len(repo.items))
	for _, item := range repo.items {
		result = append(result, item)
	}
	return result, nil
}

func (repo *externalToolsRepoStub) Get(_ context.Context, name string) (externaltools.ExternalTool, error) {
	item, ok := repo.items[name]
	if !ok {
		return externaltools.ExternalTool{}, externaltools.ErrToolNotFound
	}
	return item, nil
}

func (repo *externalToolsRepoStub) Save(_ context.Context, tool externaltools.ExternalTool) error {
	repo.items[string(tool.Name)] = tool
	return nil
}

func (repo *externalToolsRepoStub) Delete(_ context.Context, name string) error {
	delete(repo.items, name)
	return nil
}

func TestExternalToolsQueryStatusReturnsReadiness(t *testing.T) {
	t.Parallel()

	repo := newExternalToolsRepoStub()
	svc := externaltoolsservice.NewExternalToolsService(repo, nil, "")
	if err := svc.EnsureDefaults(context.Background()); err != nil {
		t.Fatalf("ensure defaults failed: %v", err)
	}

	handler := runExternalToolsQueryTool(svc)
	output, err := handler(context.Background(), `{"action":"status","name":"clawhub"}`)
	if err != nil {
		t.Fatalf("query status failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got: %#v", payload["ok"])
	}
	if payload["action"] != "status" {
		t.Fatalf("unexpected action: %#v", payload["action"])
	}
	if payload["tool"] != "clawhub" {
		t.Fatalf("unexpected tool: %#v", payload["tool"])
	}
	if payload["ready"] != false {
		t.Fatalf("expected ready=false, got: %#v", payload["ready"])
	}
	if payload["reason"] != "not_installed" {
		t.Fatalf("expected reason=not_installed, got: %#v", payload["reason"])
	}
}

func TestExternalToolsManageUnknownActionReturnsStructuredError(t *testing.T) {
	t.Parallel()

	repo := newExternalToolsRepoStub()
	svc := externaltoolsservice.NewExternalToolsService(repo, nil, "")
	if err := svc.EnsureDefaults(context.Background()); err != nil {
		t.Fatalf("ensure defaults failed: %v", err)
	}

	handler := runExternalToolsManageTool(svc)
	output, err := handler(context.Background(), `{"action":"noop","name":"clawhub"}`)
	if err != nil {
		t.Fatalf("manage call failed: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("unmarshal output failed: %v", err)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got: %#v", payload["ok"])
	}
	if payload["error"] != "unknown action" {
		t.Fatalf("unexpected error message: %#v", payload["error"])
	}
}
