package persistence

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestEnsureSQLiteColumns_LegacyThreadsMigrationAddsLastInteractiveAt(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `
CREATE TABLE threads (
	id TEXT PRIMARY KEY,
	agent_id TEXT,
	assistant_id TEXT,
	title TEXT,
	title_is_default BOOLEAN NOT NULL DEFAULT 0,
	title_changed_by TEXT,
	status TEXT NOT NULL DEFAULT 'regular',
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	deleted_at TIMESTAMP,
	purge_after TIMESTAMP
);
CREATE TABLE thread_messages (
	id TEXT PRIMARY KEY,
	thread_id TEXT NOT NULL,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	parts_json TEXT NOT NULL DEFAULT '[]',
	created_at TIMESTAMP NOT NULL
);
`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	const ts = "2026-04-03T14:02:55Z"
	if _, err := db.ExecContext(ctx, `
INSERT INTO threads (id, agent_id, assistant_id, title, title_is_default, title_changed_by, status, created_at, updated_at)
VALUES (?, '', '', '', 0, '', 'regular', ?, ?)
`, "thread-1", ts, ts); err != nil {
		t.Fatalf("insert legacy thread: %v", err)
	}

	if err := ensureSQLiteColumns(ctx, db); err != nil {
		t.Fatalf("ensure columns: %v", err)
	}

	hasLastInteractive, err := sqliteTableHasColumn(ctx, db, "threads", "last_interactive_at")
	if err != nil {
		t.Fatalf("check threads.last_interactive_at: %v", err)
	}
	if !hasLastInteractive {
		t.Fatal("expected threads.last_interactive_at to exist")
	}

	hasMessageKind, err := sqliteTableHasColumn(ctx, db, "thread_messages", "kind")
	if err != nil {
		t.Fatalf("check thread_messages.kind: %v", err)
	}
	if !hasMessageKind {
		t.Fatal("expected thread_messages.kind to exist")
	}

	var backfilled int
	if err := db.QueryRowContext(ctx, `
SELECT CASE WHEN last_interactive_at = updated_at THEN 1 ELSE 0 END
FROM threads
WHERE id = ?
`, "thread-1").Scan(&backfilled); err != nil {
		t.Fatalf("query backfilled column: %v", err)
	}
	if backfilled != 1 {
		t.Fatal("expected last_interactive_at to be backfilled from updated_at")
	}
}

func TestEnsureSQLiteColumns_LegacyProvidersMigrationAddsCompatibility(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-providers.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `
CREATE TABLE providers (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	type TEXT NOT NULL,
	endpoint TEXT,
	enabled BOOLEAN NOT NULL DEFAULT 0,
	is_builtin BOOLEAN NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL
);
`); err != nil {
		t.Fatalf("create legacy provider schema: %v", err)
	}

	const ts = "2026-04-03T14:02:55Z"
	if _, err := db.ExecContext(ctx, `
INSERT INTO providers (id, name, type, endpoint, enabled, is_builtin, created_at, updated_at)
VALUES
	('deepseek', 'DeepSeek', 'openai', 'https://api.deepseek.com', 0, 1, ?, ?),
	('custom-anthropic', 'Custom Anthropic', 'anthropic', 'https://example.com', 0, 0, ?, ?),
	('custom-openai', 'Custom OpenAI', 'openai', 'https://example.com', 0, 0, ?, ?)
`, ts, ts, ts, ts, ts, ts); err != nil {
		t.Fatalf("insert legacy providers: %v", err)
	}

	if err := ensureSQLiteColumns(ctx, db); err != nil {
		t.Fatalf("ensure columns: %v", err)
	}

	hasCompatibility, err := sqliteTableHasColumn(ctx, db, "providers", "compatibility")
	if err != nil {
		t.Fatalf("check providers.compatibility: %v", err)
	}
	if !hasCompatibility {
		t.Fatal("expected providers.compatibility to exist")
	}

	rows, err := db.QueryContext(ctx, "SELECT id, compatibility FROM providers ORDER BY id ASC")
	if err != nil {
		t.Fatalf("query provider compatibilities: %v", err)
	}
	defer rows.Close()

	got := make(map[string]string)
	for rows.Next() {
		var id string
		var compatibility string
		if err := rows.Scan(&id, &compatibility); err != nil {
			t.Fatalf("scan provider compatibility: %v", err)
		}
		got[id] = compatibility
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate provider compatibilities: %v", err)
	}

	if got["deepseek"] != "deepseek" {
		t.Fatalf("expected deepseek compatibility=deepseek, got %q", got["deepseek"])
	}
	if got["custom-anthropic"] != "anthropic" {
		t.Fatalf("expected custom anthropic compatibility=anthropic, got %q", got["custom-anthropic"])
	}
	if got["custom-openai"] != "openai" {
		t.Fatalf("expected custom openai compatibility=openai, got %q", got["custom-openai"])
	}
}

func TestEnsureSQLiteColumns_LegacyGatewaySessionsMigrationAddsContextStateColumns(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "legacy-gateway-sessions.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, `
CREATE TABLE gateway_sessions (
	session_id TEXT PRIMARY KEY,
	session_key TEXT NOT NULL,
	agent_id TEXT,
	assistant_id TEXT,
	title TEXT,
	status TEXT,
	origin_json TEXT,
	context_prompt_tokens INTEGER,
	context_total_tokens INTEGER,
	context_window_tokens INTEGER,
	context_updated_at TIMESTAMP,
	context_fresh BOOLEAN,
	compaction_count INTEGER,
	memory_flush_compaction_count INTEGER,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("create legacy gateway_sessions schema: %v", err)
	}

	if err := ensureSQLiteColumns(ctx, db); err != nil {
		t.Fatalf("ensure columns: %v", err)
	}

	for _, column := range []string{
		"context_summary",
		"context_first_kept_message_id",
		"context_strategy_version",
		"context_compacted_at",
	} {
		hasColumn, err := sqliteTableHasColumn(ctx, db, "gateway_sessions", column)
		if err != nil {
			t.Fatalf("check gateway_sessions.%s: %v", column, err)
		}
		if !hasColumn {
			t.Fatalf("expected gateway_sessions.%s to exist", column)
		}
	}
}
