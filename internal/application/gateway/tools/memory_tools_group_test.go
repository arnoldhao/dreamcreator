package tools

import (
	"context"
	"testing"
)

func TestMemoryQueryToolRequiresAction(t *testing.T) {
	t.Parallel()

	handler := runMemoryQueryTool(nil)
	_, err := handler(context.Background(), `{}`)
	if err == nil || err.Error() != "action is required" {
		t.Fatalf("expected action is required, got %v", err)
	}
}

func TestMemoryManageToolRequiresAction(t *testing.T) {
	t.Parallel()

	handler := runMemoryManageTool(nil)
	_, err := handler(context.Background(), `{}`)
	if err == nil || err.Error() != "action is required" {
		t.Fatalf("expected action is required, got %v", err)
	}
}

func TestMemoryQueryToolRoutesRecallAction(t *testing.T) {
	t.Parallel()

	handler := runMemoryQueryTool(nil)
	_, err := handler(context.Background(), `{"action":"recall","query":"demo"}`)
	if err == nil || err.Error() != "memory service unavailable" {
		t.Fatalf("expected memory service unavailable, got %v", err)
	}
}

func TestMemoryManageToolRoutesCreateAlias(t *testing.T) {
	t.Parallel()

	handler := runMemoryManageTool(nil)
	_, err := handler(context.Background(), `{"action":"create","text":"demo"}`)
	if err == nil || err.Error() != "memory service unavailable" {
		t.Fatalf("expected memory service unavailable, got %v", err)
	}
}

func TestMemoryManageToolRejectsUnknownAction(t *testing.T) {
	t.Parallel()

	handler := runMemoryManageTool(nil)
	_, err := handler(context.Background(), `{"action":"archive"}`)
	if err == nil || err.Error() != "unsupported memory_manage action" {
		t.Fatalf("expected unsupported memory_manage action, got %v", err)
	}
}
