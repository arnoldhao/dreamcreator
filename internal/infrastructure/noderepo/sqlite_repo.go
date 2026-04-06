package noderepo

import (
	"context"
	"database/sql"
	"dreamcreator/internal/infrastructure/persistence/sqlitedto"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/uptrace/bun"

	gatewaynodes "dreamcreator/internal/application/gateway/nodes"
)

type SQLiteNodeRegistryStore struct {
	db *bun.DB
}

type SQLiteNodeInvokeLogStore struct {
	db *bun.DB
}

type nodeRow = sqlitedto.NodeRow

type invokeRow = sqlitedto.NodeInvokeRow

func NewSQLiteNodeRegistryStore(db *bun.DB) *SQLiteNodeRegistryStore {
	return &SQLiteNodeRegistryStore{db: db}
}

func NewSQLiteNodeInvokeLogStore(db *bun.DB) *SQLiteNodeInvokeLogStore {
	return &SQLiteNodeInvokeLogStore{db: db}
}

func (store *SQLiteNodeRegistryStore) Save(ctx context.Context, descriptor gatewaynodes.NodeDescriptor) error {
	if store == nil || store.db == nil {
		return errors.New("node registry unavailable")
	}
	capabilities := sql.NullString{}
	if len(descriptor.Capabilities) > 0 {
		if data, err := json.Marshal(descriptor.Capabilities); err == nil {
			capabilities = sql.NullString{String: string(data), Valid: true}
		}
	}
	row := nodeRow{
		NodeID:       strings.TrimSpace(descriptor.NodeID),
		DisplayName:  nullString(descriptor.DisplayName),
		Platform:     nullString(descriptor.Platform),
		Version:      nullString(descriptor.Version),
		Capabilities: capabilities,
		Status:       nullString(descriptor.Status),
		UpdatedAt:    descriptor.UpdatedAt,
	}
	if row.UpdatedAt.IsZero() {
		row.UpdatedAt = time.Now()
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(node_id) DO UPDATE").
		Set("display_name = EXCLUDED.display_name").
		Set("platform = EXCLUDED.platform").
		Set("version = EXCLUDED.version").
		Set("capabilities_json = EXCLUDED.capabilities_json").
		Set("status = EXCLUDED.status").
		Set("updated_at = EXCLUDED.updated_at").
		Exec(ctx)
	return err
}

func (store *SQLiteNodeRegistryStore) List(ctx context.Context) ([]gatewaynodes.NodeDescriptor, error) {
	if store == nil || store.db == nil {
		return nil, errors.New("node registry unavailable")
	}
	rows := make([]nodeRow, 0)
	if err := store.db.NewSelect().Model(&rows).Order("updated_at DESC").Scan(ctx); err != nil {
		return nil, err
	}
	result := make([]gatewaynodes.NodeDescriptor, 0, len(rows))
	for _, row := range rows {
		result = append(result, rowToDescriptor(row))
	}
	return result, nil
}

func (store *SQLiteNodeRegistryStore) Get(ctx context.Context, nodeID string) (gatewaynodes.NodeDescriptor, error) {
	if store == nil || store.db == nil {
		return gatewaynodes.NodeDescriptor{}, errors.New("node registry unavailable")
	}
	row := new(nodeRow)
	if err := store.db.NewSelect().Model(row).Where("node_id = ?", nodeID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return gatewaynodes.NodeDescriptor{}, errors.New("node not found")
		}
		return gatewaynodes.NodeDescriptor{}, err
	}
	return rowToDescriptor(*row), nil
}

func (store *SQLiteNodeInvokeLogStore) Save(ctx context.Context, request gatewaynodes.NodeInvokeRequest, result gatewaynodes.NodeInvokeResult) error {
	if store == nil || store.db == nil {
		return errors.New("node invoke log store unavailable")
	}
	row := invokeRow{
		ID:         strings.TrimSpace(request.InvokeID),
		NodeID:     strings.TrimSpace(request.NodeID),
		Capability: strings.TrimSpace(request.Capability),
		Action:     strings.TrimSpace(request.Action),
		ArgsJSON:   strings.TrimSpace(request.Args),
		Status:     resolveInvokeStatus(result, nil),
		OutputJSON: strings.TrimSpace(result.Output),
		ErrorText:  strings.TrimSpace(result.Error),
		CreatedAt:  time.Now(),
	}
	_, err := store.db.NewInsert().Model(&row).
		On("CONFLICT(id) DO UPDATE").
		Set("status = EXCLUDED.status").
		Set("output_json = EXCLUDED.output_json").
		Set("error_text = EXCLUDED.error_text").
		Exec(ctx)
	return err
}

func rowToDescriptor(row nodeRow) gatewaynodes.NodeDescriptor {
	capabilities := []gatewaynodes.NodeCapability{}
	if row.Capabilities.Valid && strings.TrimSpace(row.Capabilities.String) != "" {
		_ = json.Unmarshal([]byte(row.Capabilities.String), &capabilities)
	}
	return gatewaynodes.NodeDescriptor{
		NodeID:       row.NodeID,
		DisplayName:  stringOrEmpty(row.DisplayName),
		Platform:     stringOrEmpty(row.Platform),
		Version:      stringOrEmpty(row.Version),
		Capabilities: capabilities,
		Status:       stringOrEmpty(row.Status),
		UpdatedAt:    row.UpdatedAt,
	}
}

func resolveInvokeStatus(result gatewaynodes.NodeInvokeResult, err error) string {
	if err != nil {
		return "failed"
	}
	if result.Ok {
		return "completed"
	}
	return "failed"
}

func nullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func stringOrEmpty(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
