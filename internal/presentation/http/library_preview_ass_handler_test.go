package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLibraryPreviewASSHandlerLifecycle(t *testing.T) {
	handler := NewLibraryPreviewASSHandler()

	createReq := httptest.NewRequest(http.MethodPost, "/api/library/preview-ass", bytes.NewBufferString(`{"content":"[Script Info]\nTitle: Test"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()

	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusOK {
		t.Fatalf("expected create status %d, got %d", http.StatusOK, createResp.Code)
	}

	var created previewASSUpsertResponse
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected preview ass id")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/library/preview-ass/"+created.ID, nil)
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected get status %d, got %d", http.StatusOK, getResp.Code)
	}
	if body := getResp.Body.String(); body != "[Script Info]\nTitle: Test" {
		t.Fatalf("unexpected body: %q", body)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/library/preview-ass/"+created.ID, nil)
	deleteResp := httptest.NewRecorder()
	handler.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusNoContent {
		t.Fatalf("expected delete status %d, got %d", http.StatusNoContent, deleteResp.Code)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/api/library/preview-ass/"+created.ID, nil)
	missingResp := httptest.NewRecorder()
	handler.ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("expected missing status %d, got %d", http.StatusNotFound, missingResp.Code)
	}
}

func TestLibraryPreviewASSHandlerPrunesExpiredEntries(t *testing.T) {
	handler := NewLibraryPreviewASSHandler()
	now := time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC)
	handler.now = func() time.Time { return now }
	handler.ttl = time.Minute

	id := handler.put("[Script Info]")

	now = now.Add(2 * time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/api/library/preview-ass/"+id, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected pruned status %d, got %d", http.StatusNotFound, resp.Code)
	}
}
