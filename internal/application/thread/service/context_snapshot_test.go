package service

import (
	"testing"
	"time"

	domainsession "dreamcreator/internal/domain/session"
)

func TestHasSessionContextSnapshot_IgnoreUpdatedAtOnly(t *testing.T) {
	t.Parallel()

	entry := domainsession.Entry{
		ContextUpdatedAt: time.Now(),
	}

	if hasSessionContextSnapshot(entry) {
		t.Fatal("expected updated_at-only snapshot to be treated as missing")
	}
}

func TestHasSessionContextSnapshot_WithTokens(t *testing.T) {
	t.Parallel()

	entry := domainsession.Entry{
		ContextTotalTokens: 1234,
	}

	if !hasSessionContextSnapshot(entry) {
		t.Fatal("expected snapshot with tokens to be treated as existing")
	}
}

