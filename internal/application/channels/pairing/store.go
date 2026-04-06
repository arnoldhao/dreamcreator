package pairing

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	DefaultPendingTTL   = time.Hour
	DefaultMaxPending   = 3
	pairingCodeLength   = 8
	pairingCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

var (
	ErrInvalidChannel      = errors.New("invalid pairing channel")
	ErrInvalidRequestID    = errors.New("invalid pairing request id")
	ErrInvalidCode         = errors.New("invalid pairing code")
	ErrPairRequestNotFound = errors.New("pairing request not found")
)

type Request struct {
	ID         string            `json:"id"`
	Code       string            `json:"code"`
	CreatedAt  time.Time         `json:"createdAt"`
	LastSeenAt time.Time         `json:"lastSeenAt"`
	Meta       map[string]string `json:"meta,omitempty"`
}

type storeSnapshot struct {
	Version  int       `json:"version"`
	Requests []Request `json:"requests"`
}

type Store struct {
	channel    string
	ttl        time.Duration
	maxPending int
	now        func() time.Time
	mu         sync.Mutex
}

type UpsertResult struct {
	Code    string
	Created bool
}

func NewStore(channel string) (*Store, error) {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return nil, ErrInvalidChannel
	}
	return &Store{
		channel:    channel,
		ttl:        DefaultPendingTTL,
		maxPending: DefaultMaxPending,
		now:        time.Now,
	}, nil
}

func (store *Store) List(accountID string) ([]Request, error) {
	if store == nil {
		return nil, errors.New("pairing store unavailable")
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	path, err := store.resolvePath()
	if err != nil {
		return nil, err
	}
	snapshot, _, err := store.readSnapshot(path)
	if err != nil {
		return nil, err
	}
	changed := store.pruneSnapshot(&snapshot)
	filtered := filterByAccountID(snapshot.Requests, accountID)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
	})
	if changed {
		_ = store.writeSnapshot(path, snapshot)
	}
	return filtered, nil
}

func (store *Store) Upsert(params UpsertParams) (UpsertResult, error) {
	if store == nil {
		return UpsertResult{}, errors.New("pairing store unavailable")
	}
	id := strings.TrimSpace(params.ID)
	if id == "" {
		return UpsertResult{}, ErrInvalidRequestID
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	path, err := store.resolvePath()
	if err != nil {
		return UpsertResult{}, err
	}
	now := store.now()
	snapshot, _, err := store.readSnapshot(path)
	if err != nil {
		return UpsertResult{}, err
	}
	store.pruneSnapshot(&snapshot)

	existingCodes := make(map[string]struct{}, len(snapshot.Requests))
	for _, req := range snapshot.Requests {
		code := strings.ToUpper(strings.TrimSpace(req.Code))
		if code != "" {
			existingCodes[code] = struct{}{}
		}
	}

	normalizedMeta := normalizeMeta(params.Meta, params.AccountID)
	for i, req := range snapshot.Requests {
		if req.ID != id {
			continue
		}
		code := strings.ToUpper(strings.TrimSpace(req.Code))
		if code == "" {
			code = generateUniqueCode(existingCodes)
		}
		snapshot.Requests[i] = Request{
			ID:         id,
			Code:       code,
			CreatedAt:  resolveTime(req.CreatedAt, now),
			LastSeenAt: now,
			Meta:       mergeMeta(req.Meta, normalizedMeta),
		}
		snapshot.Requests = pruneExcess(snapshot.Requests, store.maxPending)
		if err := store.writeSnapshot(path, snapshot); err != nil {
			return UpsertResult{}, err
		}
		return UpsertResult{Code: code, Created: false}, nil
	}

	if store.maxPending > 0 && len(snapshot.Requests) >= store.maxPending {
		_ = store.writeSnapshot(path, snapshot)
		return UpsertResult{}, nil
	}
	code := generateUniqueCode(existingCodes)
	snapshot.Requests = append(snapshot.Requests, Request{
		ID:         id,
		Code:       code,
		CreatedAt:  now,
		LastSeenAt: now,
		Meta:       normalizedMeta,
	})
	snapshot.Requests = pruneExcess(snapshot.Requests, store.maxPending)
	if err := store.writeSnapshot(path, snapshot); err != nil {
		return UpsertResult{}, err
	}
	return UpsertResult{Code: code, Created: true}, nil
}

func (store *Store) Approve(code string, accountID string) (*Request, error) {
	return store.consume(code, accountID)
}

func (store *Store) Reject(code string, accountID string) (*Request, error) {
	return store.consume(code, accountID)
}

func (store *Store) consume(code string, accountID string) (*Request, error) {
	if store == nil {
		return nil, errors.New("pairing store unavailable")
	}
	normalizedCode := strings.ToUpper(strings.TrimSpace(code))
	if normalizedCode == "" {
		return nil, ErrInvalidCode
	}
	store.mu.Lock()
	defer store.mu.Unlock()

	path, err := store.resolvePath()
	if err != nil {
		return nil, err
	}
	snapshot, _, err := store.readSnapshot(path)
	if err != nil {
		return nil, err
	}
	store.pruneSnapshot(&snapshot)
	normalizedAccount := normalizeAccountID(accountID)

	for i, req := range snapshot.Requests {
		if strings.ToUpper(strings.TrimSpace(req.Code)) != normalizedCode {
			continue
		}
		if normalizedAccount != "" && !matchesAccountID(req, normalizedAccount) {
			continue
		}
		removed := req
		snapshot.Requests = append(snapshot.Requests[:i], snapshot.Requests[i+1:]...)
		if err := store.writeSnapshot(path, snapshot); err != nil {
			return nil, err
		}
		return &removed, nil
	}
	if err := store.writeSnapshot(path, snapshot); err != nil {
		return nil, err
	}
	return nil, ErrPairRequestNotFound
}

type UpsertParams struct {
	ID        string
	AccountID string
	Meta      map[string]string
}

func (store *Store) resolvePath() (string, error) {
	if store == nil {
		return "", errors.New("pairing store unavailable")
	}
	key, err := safeChannelKey(store.channel)
	if err != nil {
		return "", err
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	baseDir := filepath.Join(configDir, "dreamcreator", "credentials")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(baseDir, fmt.Sprintf("%s-pairing.json", key)), nil
}

func safeChannelKey(channel string) (string, error) {
	raw := strings.ToLower(strings.TrimSpace(channel))
	if raw == "" {
		return "", ErrInvalidChannel
	}
	safe := strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, raw)
	safe = strings.ReplaceAll(safe, "..", "_")
	if strings.TrimSpace(safe) == "" || safe == "_" {
		return "", ErrInvalidChannel
	}
	return safe, nil
}

func (store *Store) readSnapshot(path string) (storeSnapshot, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return storeSnapshot{Version: 1, Requests: nil}, false, nil
		}
		return storeSnapshot{}, false, err
	}
	if len(data) == 0 {
		return storeSnapshot{Version: 1, Requests: nil}, true, nil
	}
	var snapshot storeSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return storeSnapshot{}, true, err
	}
	if snapshot.Version == 0 {
		snapshot.Version = 1
	}
	return snapshot, true, nil
}

func (store *Store) writeSnapshot(path string, snapshot storeSnapshot) error {
	if snapshot.Version == 0 {
		snapshot.Version = 1
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "pairing-*.json")
	if err != nil {
		return err
	}
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snapshot); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (store *Store) pruneSnapshot(snapshot *storeSnapshot) bool {
	if snapshot == nil {
		return false
	}
	now := store.now()
	changed := false
	kept := snapshot.Requests[:0]
	for _, req := range snapshot.Requests {
		if isExpired(req, now, store.ttl) {
			changed = true
			continue
		}
		kept = append(kept, req)
	}
	snapshot.Requests = pruneExcess(kept, store.maxPending)
	if len(snapshot.Requests) != len(kept) {
		changed = true
	}
	return changed
}

func isExpired(req Request, now time.Time, ttl time.Duration) bool {
	if ttl <= 0 {
		return false
	}
	if req.CreatedAt.IsZero() {
		return true
	}
	return now.Sub(req.CreatedAt) > ttl
}

func pruneExcess(requests []Request, maxPending int) []Request {
	if maxPending <= 0 || len(requests) <= maxPending {
		return requests
	}
	sort.Slice(requests, func(i, j int) bool {
		return resolveLastSeen(requests[i]).Before(resolveLastSeen(requests[j]))
	})
	if len(requests) > maxPending {
		return append([]Request(nil), requests[len(requests)-maxPending:]...)
	}
	return requests
}

func resolveLastSeen(req Request) time.Time {
	if !req.LastSeenAt.IsZero() {
		return req.LastSeenAt
	}
	return req.CreatedAt
}

func generateUniqueCode(existing map[string]struct{}) string {
	for attempt := 0; attempt < 500; attempt++ {
		code := randomCode()
		if _, ok := existing[code]; !ok {
			return code
		}
	}
	return randomCode()
}

func randomCode() string {
	var builder strings.Builder
	builder.Grow(pairingCodeLength)
	max := big.NewInt(int64(len(pairingCodeAlphabet)))
	for i := 0; i < pairingCodeLength; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return fmt.Sprintf("%d", time.Now().UnixNano())[:pairingCodeLength]
		}
		builder.WriteByte(pairingCodeAlphabet[n.Int64()])
	}
	return builder.String()
}

func normalizeAccountID(accountID string) string {
	return strings.ToLower(strings.TrimSpace(accountID))
}

func matchesAccountID(req Request, accountID string) bool {
	if accountID == "" {
		return true
	}
	if req.Meta == nil {
		return false
	}
	return normalizeAccountID(req.Meta["accountId"]) == accountID
}

func filterByAccountID(requests []Request, accountID string) []Request {
	normalized := normalizeAccountID(accountID)
	if normalized == "" {
		return append([]Request(nil), requests...)
	}
	filtered := make([]Request, 0, len(requests))
	for _, req := range requests {
		if matchesAccountID(req, normalized) {
			filtered = append(filtered, req)
		}
	}
	return filtered
}

func normalizeMeta(meta map[string]string, accountID string) map[string]string {
	if len(meta) == 0 && strings.TrimSpace(accountID) == "" {
		return nil
	}
	result := make(map[string]string)
	for key, value := range meta {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		result[key] = trimmed
	}
	if accountID != "" {
		result["accountId"] = strings.TrimSpace(accountID)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func mergeMeta(existing, next map[string]string) map[string]string {
	if len(next) == 0 {
		if len(existing) == 0 {
			return nil
		}
		out := make(map[string]string, len(existing))
		for key, value := range existing {
			out[key] = value
		}
		return out
	}
	out := make(map[string]string, len(next))
	for key, value := range next {
		out[key] = value
	}
	return out
}

func resolveTime(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback
	}
	return value
}
