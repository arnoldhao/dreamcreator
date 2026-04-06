package persistence

import (
	"context"
	"database/sql"
	"os"
	"strings"
)

const (
	sqliteVecDisabledEnv = "DREAMCREATOR_SQLITE_VEC_DISABLED"
)

// initializeSQLiteVec performs a lightweight startup probe to verify the bundled
// sqlite-vec extension is available in the active sqlite driver.
func initializeSQLiteVec(ctx context.Context, db *sql.DB) {
	if db == nil || isSQLiteVecDisabled() {
		return
	}
	_ = probeSQLiteVec(ctx, db)
}

func probeSQLiteVec(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return sql.ErrConnDone
	}
	var version string
	if err := db.QueryRowContext(ctx, "SELECT vec_version()").Scan(&version); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS temp.__dreamcreator_vec_probe"); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "CREATE VIRTUAL TABLE temp.__dreamcreator_vec_probe USING vec0(embedding float[1])"); err != nil {
		return err
	}
	_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS temp.__dreamcreator_vec_probe")
	return nil
}

func isSQLiteVecDisabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(sqliteVecDisabledEnv)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}
