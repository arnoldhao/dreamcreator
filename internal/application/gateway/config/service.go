package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	telegrammenu "dreamcreator/internal/application/channels/telegram"
	settingsdto "dreamcreator/internal/application/settings/dto"
	settingsservice "dreamcreator/internal/application/settings/service"
	"go.uber.org/zap"
)

var ErrVersionConflict = errors.New("config version conflict")

type ReloadStep struct {
	Component string `json:"component"`
	Action    string `json:"action"`
	Reason    string `json:"reason,omitempty"`
}

type ReloadPlan struct {
	Mode  string       `json:"mode"`
	Steps []ReloadStep `json:"steps"`
}

type ConfigGetRequest struct {
	Path string `json:"path,omitempty"`
}

type ConfigGetResponse struct {
	Config  any `json:"config"`
	Version int `json:"version"`
}

type ConfigSetRequest struct {
	Path            string `json:"path"`
	Value           any    `json:"value"`
	ExpectedVersion int    `json:"expectedVersion,omitempty"`
}

type ConfigSetResponse struct {
	Version int `json:"version"`
}

type PatchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

type ConfigPatchRequest struct {
	Ops             []PatchOp `json:"ops"`
	DryRun          bool      `json:"dryRun"`
	ExpectedVersion int       `json:"expectedVersion,omitempty"`
}

type ConfigPatchResponse struct {
	Preview    any        `json:"preview"`
	Version    int        `json:"version"`
	ReloadPlan ReloadPlan `json:"reloadPlan"`
}

type ConfigApplyRequest struct {
	Config          any    `json:"config"`
	ExpectedVersion int    `json:"expectedVersion,omitempty"`
	Mode            string `json:"mode,omitempty"`
}

type ConfigApplyResponse struct {
	Version    int        `json:"version"`
	Applied    bool       `json:"applied"`
	ReloadPlan ReloadPlan `json:"reloadPlan"`
}

type ConfigSchemaRequest struct {
	Path string `json:"path,omitempty"`
}

type ConfigSchemaResponse struct {
	Schema   any               `json:"schema"`
	Examples any               `json:"examples,omitempty"`
	Help     map[string]string `json:"help,omitempty"`
}

type Service struct {
	settings        *settingsservice.SettingsService
	revisions       RevisionStore
	menuSyncer      MenuSyncer
	runtimeSyncer   RuntimeSyncer
	settingsApplier SettingsApplier
	now             func() time.Time
}

type RevisionStore interface {
	Record(ctx context.Context, revision ConfigRevision) error
	Get(ctx context.Context, id string) (ConfigRevision, error)
}

type MenuSyncer interface {
	SyncFromSettings(ctx context.Context, settings settingsdto.Settings) (telegrammenu.MenuSyncStatus, error)
}

type RuntimeSyncer interface {
	RefreshFromSettings(ctx context.Context, settings settingsdto.Settings) error
}

type SettingsApplier interface {
	ApplySettings(settings settingsdto.Settings)
}

type ConfigRevision struct {
	ID        string
	Version   int
	Config    any
	Plan      ReloadPlan
	CreatedAt time.Time
}

func NewService(settings *settingsservice.SettingsService, revisions RevisionStore, menuSyncer MenuSyncer) *Service {
	return &Service{
		settings:   settings,
		revisions:  revisions,
		menuSyncer: menuSyncer,
		now:        time.Now,
	}
}

func (service *Service) SetRuntimeSyncer(syncer RuntimeSyncer) {
	if service == nil {
		return
	}
	service.runtimeSyncer = syncer
}

func (service *Service) SetSettingsApplier(applier SettingsApplier) {
	if service == nil {
		return
	}
	service.settingsApplier = applier
}

func (service *Service) Get(ctx context.Context, request ConfigGetRequest) (ConfigGetResponse, error) {
	current, err := service.currentSettings(ctx)
	if err != nil {
		return ConfigGetResponse{}, err
	}
	configMap, err := toMap(current)
	if err != nil {
		return ConfigGetResponse{}, err
	}
	if strings.TrimSpace(request.Path) == "" {
		return ConfigGetResponse{Config: configMap, Version: current.Version}, nil
	}
	value, err := resolvePointer(configMap, request.Path)
	if err != nil {
		return ConfigGetResponse{}, err
	}
	return ConfigGetResponse{Config: value, Version: current.Version}, nil
}

func (service *Service) Set(ctx context.Context, request ConfigSetRequest) (ConfigSetResponse, error) {
	path := strings.TrimSpace(request.Path)
	if path == "" {
		return ConfigSetResponse{}, errors.New("config.set path is required")
	}
	ops := []PatchOp{{Op: "replace", Path: path, Value: request.Value}}
	preview, version, _, err := service.applyOps(ctx, ops, false, request.ExpectedVersion)
	if err != nil {
		return ConfigSetResponse{}, err
	}
	_ = preview
	return ConfigSetResponse{Version: version}, nil
}

func (service *Service) Patch(ctx context.Context, request ConfigPatchRequest) (ConfigPatchResponse, error) {
	if len(request.Ops) == 0 {
		return ConfigPatchResponse{}, errors.New("config.patch ops required")
	}
	preview, version, plan, err := service.applyOps(ctx, request.Ops, request.DryRun, request.ExpectedVersion)
	if err != nil {
		return ConfigPatchResponse{}, err
	}
	return ConfigPatchResponse{Preview: preview, Version: version, ReloadPlan: plan}, nil
}

func (service *Service) Apply(ctx context.Context, request ConfigApplyRequest) (ConfigApplyResponse, error) {
	if request.Config == nil {
		return ConfigApplyResponse{}, errors.New("config.apply config is required")
	}
	current, err := service.currentSettings(ctx)
	if err != nil {
		return ConfigApplyResponse{}, err
	}
	if request.ExpectedVersion > 0 && current.Version != request.ExpectedVersion {
		return ConfigApplyResponse{}, ErrVersionConflict
	}
	updateRequest, err := toUpdateRequest(request.Config)
	if err != nil {
		return ConfigApplyResponse{}, err
	}
	if service.settings == nil {
		plan := service.buildReloadPlan(nil, request.Mode)
		service.recordRevision(ctx, ConfigRevision{
			Version:   current.Version,
			Config:    request.Config,
			Plan:      plan,
			CreatedAt: service.now(),
		})
		return ConfigApplyResponse{Version: current.Version, Applied: false, ReloadPlan: plan}, nil
	}
	updated, err := service.settings.UpdateSettings(ctx, updateRequest)
	if err != nil {
		return ConfigApplyResponse{}, err
	}
	service.applySettings(updated)
	if service.menuSyncer != nil {
		updatedSnapshot := updated
		go func() {
			if _, err := service.menuSyncer.SyncFromSettings(context.Background(), updatedSnapshot); err != nil {
				zap.L().Warn("telegram menu sync failed", zap.Error(err))
			}
		}()
	}
	if service.runtimeSyncer != nil {
		updatedSnapshot := updated
		go func() {
			if err := service.runtimeSyncer.RefreshFromSettings(context.Background(), updatedSnapshot); err != nil {
				zap.L().Warn("telegram runtime refresh failed", zap.Error(err))
			}
		}()
	}
	plan := service.buildReloadPlan(nil, request.Mode)
	service.recordRevision(ctx, ConfigRevision{
		Version:   updated.Version,
		Config:    request.Config,
		Plan:      plan,
		CreatedAt: service.now(),
	})
	return ConfigApplyResponse{Version: updated.Version, Applied: true, ReloadPlan: plan}, nil
}

func (service *Service) Schema(_ context.Context, request ConfigSchemaRequest) (ConfigSchemaResponse, error) {
	schema := buildSchema(reflect.TypeOf(settingsdto.Settings{}))
	if strings.TrimSpace(request.Path) == "" {
		return ConfigSchemaResponse{Schema: schema}, nil
	}
	value, err := resolvePointer(schema, request.Path)
	if err != nil {
		return ConfigSchemaResponse{}, err
	}
	return ConfigSchemaResponse{Schema: value}, nil
}

func (service *Service) currentSettings(ctx context.Context) (settingsdto.Settings, error) {
	if service == nil || service.settings == nil {
		return settingsdto.Settings{}, errors.New("settings service unavailable")
	}
	return service.settings.GetSettings(ctx)
}

func (service *Service) applyOps(ctx context.Context, ops []PatchOp, dryRun bool, expectedVersion int) (map[string]any, int, ReloadPlan, error) {
	current, err := service.currentSettings(ctx)
	if err != nil {
		return nil, 0, ReloadPlan{}, err
	}
	if expectedVersion > 0 && current.Version != expectedVersion {
		return nil, current.Version, ReloadPlan{}, ErrVersionConflict
	}
	configMap, err := toMap(current)
	if err != nil {
		return nil, 0, ReloadPlan{}, err
	}
	preview, changedPaths, err := applyPatchOps(configMap, ops)
	if err != nil {
		return nil, 0, ReloadPlan{}, err
	}
	plan := service.buildReloadPlan(changedPaths, "")
	if dryRun || service.settings == nil {
		return preview, current.Version, plan, nil
	}
	updateRequest, err := toUpdateRequest(preview)
	if err != nil {
		return nil, 0, ReloadPlan{}, err
	}
	updated, err := service.settings.UpdateSettings(ctx, updateRequest)
	if err != nil {
		return nil, 0, ReloadPlan{}, err
	}
	service.applySettings(updated)
	if service.menuSyncer != nil && shouldSyncTelegramMenu(changedPaths) {
		updatedSnapshot := updated
		go func() {
			if _, err := service.menuSyncer.SyncFromSettings(context.Background(), updatedSnapshot); err != nil {
				zap.L().Warn("telegram menu sync failed", zap.Error(err))
			}
		}()
	}
	if service.runtimeSyncer != nil && shouldSyncTelegramMenu(changedPaths) {
		updatedSnapshot := updated
		go func() {
			if err := service.runtimeSyncer.RefreshFromSettings(context.Background(), updatedSnapshot); err != nil {
				zap.L().Warn("telegram runtime refresh failed", zap.Error(err))
			}
		}()
	}
	service.recordRevision(ctx, ConfigRevision{
		Version:   updated.Version,
		Config:    preview,
		Plan:      plan,
		CreatedAt: service.now(),
	})
	return preview, updated.Version, plan, nil
}

func (service *Service) applySettings(updated settingsdto.Settings) {
	if service == nil || service.settingsApplier == nil {
		return
	}
	updatedSnapshot := updated
	go service.settingsApplier.ApplySettings(updatedSnapshot)
}

func (service *Service) buildReloadPlan(changed []string, mode string) ReloadPlan {
	selectedMode := strings.TrimSpace(mode)
	if selectedMode == "" {
		selectedMode = "hot"
	}
	stepReason := ""
	if len(changed) > 0 {
		stepReason = fmt.Sprintf("changed: %s", strings.Join(changed, ", "))
	}
	return ReloadPlan{
		Mode: selectedMode,
		Steps: []ReloadStep{
			{
				Component: "settings",
				Action:    "reload",
				Reason:    stepReason,
			},
		},
	}
}

func (service *Service) recordRevision(ctx context.Context, revision ConfigRevision) {
	if service == nil || service.revisions == nil {
		return
	}
	rev := revision
	if rev.CreatedAt.IsZero() {
		rev.CreatedAt = service.now()
	}
	if rev.ID == "" {
		rev.ID = rev.CreatedAt.Format(time.RFC3339Nano)
	}
	_ = service.revisions.Record(ctx, rev)
}

func toMap(value any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoded := make(map[string]any)
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func toUpdateRequest(value any) (settingsdto.UpdateSettingsRequest, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return settingsdto.UpdateSettingsRequest{}, err
	}
	var request settingsdto.UpdateSettingsRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return settingsdto.UpdateSettingsRequest{}, err
	}
	return request, nil
}

func buildSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct:
		props := make(map[string]any)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name := strings.Split(tag, ",")[0]
			if name == "" {
				name = field.Name
			}
			props[name] = buildSchema(field.Type)
		}
		return map[string]any{
			"type":       "object",
			"properties": props,
		}
	case reflect.Map:
		valueSchema := buildSchema(t.Elem())
		return map[string]any{
			"type":                 "object",
			"additionalProperties": valueSchema,
		}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": buildSchema(t.Elem()),
		}
	case reflect.Interface:
		return map[string]any{}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	default:
		return map[string]any{"type": "string"}
	}
}

func shouldSyncTelegramMenu(changed []string) bool {
	if len(changed) == 0 {
		return false
	}
	for _, path := range changed {
		normalized := strings.TrimSpace(path)
		if normalized == "" {
			continue
		}
		if normalized == "channels" || strings.HasPrefix(normalized, "channels/telegram") {
			return true
		}
	}
	return false
}

func applyPatchOps(input map[string]any, ops []PatchOp) (map[string]any, []string, error) {
	preview, err := cloneMap(input)
	if err != nil {
		return nil, nil, err
	}
	changed := make([]string, 0, len(ops))
	for _, op := range ops {
		if err := applyPatchOp(preview, op); err != nil {
			return nil, nil, err
		}
		changed = append(changed, strings.TrimPrefix(op.Path, "/"))
	}
	return preview, changed, nil
}

func applyPatchOp(target map[string]any, op PatchOp) error {
	action := strings.ToLower(strings.TrimSpace(op.Op))
	if action == "" {
		return errors.New("patch op missing action")
	}
	path := strings.TrimSpace(op.Path)
	if path == "" || path == "/" {
		return errors.New("patch op path is required")
	}
	segments, err := parsePointer(path)
	if err != nil {
		return err
	}
	parent, key, err := resolveContainer(target, segments, action == "add" || action == "replace")
	if err != nil {
		return err
	}
	switch container := parent.(type) {
	case map[string]any:
		switch action {
		case "add", "replace":
			container[key] = op.Value
		case "remove":
			delete(container, key)
		default:
			return fmt.Errorf("unsupported patch op: %s", action)
		}
	case []any:
		return errors.New("array patch operations are not supported")
	default:
		return errors.New("invalid patch target")
	}
	return nil
}

func resolvePointer(root any, path string) (any, error) {
	segments, err := parsePointer(path)
	if err != nil {
		return nil, err
	}
	current := root
	for _, segment := range segments {
		switch node := current.(type) {
		case map[string]any:
			next, ok := node[segment]
			if !ok {
				return nil, fmt.Errorf("path not found: %s", path)
			}
			current = next
		case []any:
			idx, err := strconv.Atoi(segment)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, fmt.Errorf("invalid array index: %s", segment)
			}
			current = node[idx]
		default:
			return nil, fmt.Errorf("invalid path segment: %s", segment)
		}
	}
	return current, nil
}

func resolveContainer(root any, segments []string, create bool) (any, string, error) {
	if len(segments) == 0 {
		return nil, "", errors.New("empty path")
	}
	current := root
	for i := 0; i < len(segments)-1; i++ {
		segment := segments[i]
		switch node := current.(type) {
		case map[string]any:
			next, ok := node[segment]
			if !ok {
				if !create {
					return nil, "", fmt.Errorf("path not found: %s", strings.Join(segments[:i+1], "/"))
				}
				child := make(map[string]any)
				node[segment] = child
				current = child
				continue
			}
			current = next
		case []any:
			idx, err := strconv.Atoi(segment)
			if err != nil || idx < 0 {
				return nil, "", fmt.Errorf("invalid array index: %s", segment)
			}
			if idx >= len(node) {
				if !create {
					return nil, "", fmt.Errorf("array index out of range: %s", segment)
				}
				for len(node) <= idx {
					node = append(node, make(map[string]any))
				}
			}
			current = node[idx]
		default:
			return nil, "", fmt.Errorf("invalid path segment: %s", segment)
		}
	}
	return current, segments[len(segments)-1], nil
}

func parsePointer(path string) ([]string, error) {
	if path == "" || path == "/" {
		return []string{}, nil
	}
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid pointer: %s", path)
	}
	raw := strings.Split(path[1:], "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.ReplaceAll(part, "~1", "/")
		part = strings.ReplaceAll(part, "~0", "~")
		parts = append(parts, part)
	}
	return parts, nil
}

func cloneMap(value map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}
