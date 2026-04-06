package queue

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	gatewayevents "dreamcreator/internal/application/gateway/events"
	sessionmanager "dreamcreator/internal/application/session"
	domainsession "dreamcreator/internal/domain/session"
)

var ErrInvalidRequest = errors.New("invalid queue request")

type Policy struct {
	DefaultMode string
	DefaultCap  int
	ByChannel   map[string]ChannelPolicy
	GlobalCaps  GlobalCaps
}

type ChannelPolicy struct {
	Mode       string
	DebounceMs int
	Cap        int
	Drop       string
}

type GlobalCaps struct {
	Steer    int
	Followup int
	Collect  int
}

type LaneCaps struct {
	Main     int
	Subagent int
	Cron     int
}

const (
	LaneMain     = "main"
	LaneSubagent = "subagent"
	LaneCron     = "cron"
)

type ResolvedPolicy struct {
	Mode       domainsession.QueueMode
	DebounceMs int
	Cap        int
	Drop       string
}

type PolicyResolver struct {
	policy Policy
}

func NewPolicyResolver(policy Policy) *PolicyResolver {
	return &PolicyResolver{policy: policy}
}

func (resolver *PolicyResolver) Update(policy Policy) {
	if resolver == nil {
		return
	}
	resolver.policy = policy
}

func (resolver *PolicyResolver) Current() Policy {
	if resolver == nil {
		return Policy{}
	}
	return resolver.policy
}

func (resolver *PolicyResolver) Resolve(sessionKey string, mode string) ResolvedPolicy {
	mode = strings.TrimSpace(mode)
	resolvedMode := strings.TrimSpace(resolver.policy.DefaultMode)
	channelPolicy := ChannelPolicy{}
	if parts, _, err := domainsession.NormalizeSessionKey(sessionKey); err == nil {
		if policy, ok := resolver.policy.ByChannel[strings.TrimSpace(parts.Channel)]; ok {
			channelPolicy = policy
		}
	}
	if strings.TrimSpace(channelPolicy.Mode) != "" {
		resolvedMode = channelPolicy.Mode
	}
	if mode != "" {
		resolvedMode = mode
	}
	capValue := channelPolicy.Cap
	if capValue == 0 && resolver.policy.DefaultCap > 0 {
		capValue = resolver.policy.DefaultCap
	}
	return ResolvedPolicy{
		Mode:       domainsession.ParseQueueMode(resolvedMode),
		DebounceMs: channelPolicy.DebounceMs,
		Cap:        capValue,
		Drop:       strings.TrimSpace(channelPolicy.Drop),
	}
}

type EnqueueRequest struct {
	SessionKey string
	Mode       string
	Payload    any
}

type Ticket struct {
	TicketID   string `json:"ticketId"`
	SessionKey string `json:"sessionKey"`
	Lane       string `json:"lane"`
	Position   int    `json:"position"`
	Status     string `json:"status"`
}

type Event struct {
	EventID    string    `json:"eventId"`
	Type       string    `json:"type"`
	SessionKey string    `json:"sessionKey"`
	TicketID   string    `json:"ticketId,omitempty"`
	Position   int       `json:"position,omitempty"`
	Lane       string    `json:"lane,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type Store interface {
	Save(ctx context.Context, ticket Ticket) error
}

type Manager struct {
	sessions *sessionmanager.Manager
	resolver *PolicyResolver
	store    Store
	events   *gatewayevents.Broker
	global   map[sessionmanager.QueueLane]chan struct{}
	lanes    map[string]chan struct{}
	mu       sync.Mutex
	tickets  map[string]*ticketState
	now      func() time.Time
	newID    func() string
}

type ticketState struct {
	sessionKey     string
	lane           sessionmanager.QueueLane
	position       int
	status         string
	createdAt      time.Time
	sessionTicket  *sessionmanager.QueueTicket
	globalAcquired bool
}

func NewManager(sessions *sessionmanager.Manager, resolver *PolicyResolver, store Store, events *gatewayevents.Broker) *Manager {
	if sessions == nil {
		sessions = sessionmanager.NewManager()
	}
	if resolver == nil {
		resolver = NewPolicyResolver(Policy{DefaultMode: string(domainsession.QueueModeFollowup)})
	}
	manager := &Manager{
		sessions: sessions,
		resolver: resolver,
		store:    store,
		events:   events,
		global:   make(map[sessionmanager.QueueLane]chan struct{}),
		lanes:    make(map[string]chan struct{}),
		tickets:  make(map[string]*ticketState),
		now:      time.Now,
		newID:    uuid.NewString,
	}
	manager.configureGlobalCaps(resolver.policy.GlobalCaps)
	return manager
}

func (manager *Manager) UpdateGlobalCaps(caps GlobalCaps) {
	if manager == nil {
		return
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.global = rebuildGlobalCaps(manager.global, caps)
}

func (manager *Manager) UpdatePolicy(policy Policy) {
	if manager == nil {
		return
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.resolver != nil {
		manager.resolver.Update(policy)
	}
	manager.global = rebuildGlobalCaps(manager.global, policy.GlobalCaps)
}

func (manager *Manager) UpdateLaneCaps(caps LaneCaps) {
	if manager == nil {
		return
	}
	manager.mu.Lock()
	manager.lanes = rebuildLaneCaps(manager.lanes, caps)
	type laneSnapshot struct {
		lane     string
		active   int
		capacity int
	}
	snapshots := make([]laneSnapshot, 0, len(manager.lanes))
	for lane, ch := range manager.lanes {
		snapshots = append(snapshots, laneSnapshot{
			lane:     lane,
			active:   len(ch),
			capacity: cap(ch),
		})
	}
	manager.mu.Unlock()
	for _, snapshot := range snapshots {
		manager.emitLaneDiagnostic(context.Background(), snapshot.lane, "lane_caps_updated", "", snapshot.active, snapshot.capacity)
	}
}

func (manager *Manager) Enqueue(ctx context.Context, request EnqueueRequest) (Ticket, Event, error) {
	if manager == nil {
		return Ticket{}, Event{}, ErrInvalidRequest
	}
	key, err := resolveQueueKey(request.SessionKey)
	if err != nil {
		return Ticket{}, Event{}, err
	}
	resolved := manager.resolver.Resolve(request.SessionKey, request.Mode)
	lane := queueTicketLane(resolved.Mode)
	now := manager.now()
	if resolved.DebounceMs > 0 {
		if existing := manager.sessions.LastQueuedTicket(key, lane); existing != nil {
			if now.Sub(existing.CreatedAt()) < time.Duration(resolved.DebounceMs)*time.Millisecond {
				existing.SetPayload(request.Payload)
				return manager.emitTicketEvent(ctx, request.SessionKey, existing.ID(), lane, "merged", 0, "")
			}
		}
	}
	if resolved.Cap > 0 {
		queuedCount := manager.sessions.QueuedCount(key, lane)
		if queuedCount >= resolved.Cap {
			dropPolicy := strings.ToLower(resolved.Drop)
			if dropPolicy == "old" {
				dropped := manager.sessions.DropOldestQueued(key, lane)
				if dropped != nil {
					manager.mu.Lock()
					delete(manager.tickets, dropped.ID())
					manager.mu.Unlock()
					_, _, _ = manager.emitTicketEvent(ctx, request.SessionKey, dropped.ID(), lane, "dropped", 0, "cap_old")
				}
			} else {
				return manager.emitTicketEvent(ctx, request.SessionKey, "", lane, "dropped", 0, "cap_new")
			}
		}
	}
	var merge func(existing any, incoming any) (any, bool)
	if resolved.Mode == domainsession.QueueModeCollect {
		merge = func(_ any, incoming any) (any, bool) {
			return incoming, true
		}
	}
	sessionTicket, position, merged := manager.sessions.Enqueue(key, resolved.Mode, request.Payload, merge)
	if merged && sessionTicket != nil {
		return manager.emitTicketEvent(ctx, request.SessionKey, sessionTicket.ID(), lane, "merged", position, "")
	}
	if sessionTicket == nil {
		return Ticket{}, Event{}, errors.New("queue ticket unavailable")
	}
	state := &ticketState{
		sessionKey:    request.SessionKey,
		lane:          lane,
		position:      position,
		status:        "queued",
		createdAt:     sessionTicket.CreatedAt(),
		sessionTicket: sessionTicket,
	}
	manager.mu.Lock()
	manager.tickets[sessionTicket.ID()] = state
	manager.mu.Unlock()
	return manager.emitTicketEvent(ctx, request.SessionKey, sessionTicket.ID(), lane, "queued", position, "")
}

func (manager *Manager) Wait(ctx context.Context, ticketID string) error {
	state := manager.lookup(ticketID)
	if state == nil || state.sessionTicket == nil {
		return ErrInvalidRequest
	}
	if err := manager.sessions.Wait(ctx, state.sessionTicket); err != nil {
		return err
	}
	if err := manager.acquireGlobal(ctx, state.lane); err != nil {
		manager.sessions.Cancel(state.sessionTicket)
		return err
	}
	manager.mu.Lock()
	state.globalAcquired = true
	state.status = "started"
	manager.mu.Unlock()
	_, _, _ = manager.emitTicketEvent(ctx, state.sessionKey, ticketID, state.lane, "started", state.position, "")
	return nil
}

func (manager *Manager) Done(ctx context.Context, ticketID string) {
	state := manager.lookup(ticketID)
	if state == nil {
		return
	}
	if state.globalAcquired {
		manager.releaseGlobal(state.lane)
	}
	manager.sessions.Done(state.sessionTicket)
	manager.mu.Lock()
	delete(manager.tickets, ticketID)
	manager.mu.Unlock()
	_, _, _ = manager.emitTicketEvent(ctx, state.sessionKey, ticketID, state.lane, "completed", state.position, "")
}

func (manager *Manager) Cancel(ctx context.Context, ticketID string) {
	state := manager.lookup(ticketID)
	if state == nil {
		return
	}
	if state.globalAcquired {
		manager.releaseGlobal(state.lane)
	}
	manager.sessions.Cancel(state.sessionTicket)
	manager.mu.Lock()
	delete(manager.tickets, ticketID)
	manager.mu.Unlock()
	_, _, _ = manager.emitTicketEvent(ctx, state.sessionKey, ticketID, state.lane, "cancelled", state.position, "")
}

func (manager *Manager) emitTicketEvent(ctx context.Context, sessionKey string, ticketID string, lane sessionmanager.QueueLane, eventType string, position int, reason string) (Ticket, Event, error) {
	now := manager.now()
	ticket := Ticket{
		TicketID:   ticketID,
		SessionKey: sessionKey,
		Lane:       string(lane),
		Position:   position,
		Status:     eventType,
	}
	if manager.store != nil {
		_ = manager.store.Save(ctx, Ticket{
			TicketID:   ticketID,
			SessionKey: sessionKey,
			Lane:       string(lane),
			Position:   position,
			Status:     eventType,
		})
	}
	event := Event{
		EventID:    manager.newID(),
		Type:       eventType,
		SessionKey: sessionKey,
		TicketID:   ticketID,
		Position:   position,
		Lane:       string(lane),
		Reason:     reason,
		Timestamp:  now,
	}
	if manager.events != nil {
		envelope := gatewayevents.Envelope{
			EventID:    event.EventID,
			Type:       "queue.updated",
			Topic:      "queue",
			SessionKey: sessionKey,
			Timestamp:  now,
		}
		_, _ = manager.events.Publish(ctx, envelope, event)
	}
	return ticket, event, nil
}

func (manager *Manager) lookup(ticketID string) *ticketState {
	if manager == nil || strings.TrimSpace(ticketID) == "" {
		return nil
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	return manager.tickets[ticketID]
}

func (manager *Manager) configureGlobalCaps(caps GlobalCaps) {
	manager.global = rebuildGlobalCaps(manager.global, caps)
}

func rebuildLaneCaps(current map[string]chan struct{}, caps LaneCaps) map[string]chan struct{} {
	next := make(map[string]chan struct{})
	apply := func(lane string, capValue int) {
		cleaned := strings.ToLower(strings.TrimSpace(lane))
		if cleaned == "" || capValue <= 0 {
			return
		}
		var used int
		if existing := current[cleaned]; existing != nil {
			used = len(existing)
		}
		if used > capValue {
			used = capValue
		}
		ch := make(chan struct{}, capValue)
		for i := 0; i < used; i++ {
			ch <- struct{}{}
		}
		next[cleaned] = ch
	}
	apply(LaneMain, caps.Main)
	apply(LaneSubagent, caps.Subagent)
	apply(LaneCron, caps.Cron)
	return next
}

func (manager *Manager) AcquireLane(ctx context.Context, lane string) error {
	if manager == nil {
		return nil
	}
	cleaned := strings.ToLower(strings.TrimSpace(lane))
	if cleaned == "" {
		cleaned = LaneMain
	}
	manager.mu.Lock()
	ch := manager.lanes[cleaned]
	capacity := 0
	if ch != nil {
		capacity = cap(ch)
	}
	manager.mu.Unlock()
	if ch == nil {
		return nil
	}
	select {
	case ch <- struct{}{}:
		manager.emitLaneDiagnostic(ctx, cleaned, "lane_acquired", "", len(ch), capacity)
		return nil
	case <-ctx.Done():
		manager.emitLaneDiagnostic(ctx, cleaned, "lane_acquire_failed", ctx.Err().Error(), len(ch), capacity)
		return ctx.Err()
	}
}

func (manager *Manager) ReleaseLane(lane string) {
	if manager == nil {
		return
	}
	cleaned := strings.ToLower(strings.TrimSpace(lane))
	if cleaned == "" {
		cleaned = LaneMain
	}
	manager.mu.Lock()
	ch := manager.lanes[cleaned]
	manager.mu.Unlock()
	if ch == nil {
		return
	}
	released := false
	select {
	case <-ch:
		released = true
	default:
	}
	if released {
		manager.emitLaneDiagnostic(context.Background(), cleaned, "lane_released", "", len(ch), cap(ch))
	}
}

func (manager *Manager) ResetAllLanes(ctx context.Context) {
	if manager == nil {
		return
	}
	manager.mu.Lock()
	if len(manager.lanes) == 0 {
		manager.mu.Unlock()
		manager.emitLaneDiagnostic(ctx, "", "lane_reset_all", "no_lanes", 0, 0)
		return
	}
	next := make(map[string]chan struct{}, len(manager.lanes))
	for lane, ch := range manager.lanes {
		capacity := cap(ch)
		if capacity <= 0 {
			continue
		}
		next[lane] = make(chan struct{}, capacity)
	}
	manager.lanes = next
	manager.mu.Unlock()
	for lane, ch := range next {
		manager.emitLaneDiagnostic(ctx, lane, "lane_reset_all", "reset", len(ch), cap(ch))
	}
}

func rebuildGlobalCaps(current map[sessionmanager.QueueLane]chan struct{}, caps GlobalCaps) map[sessionmanager.QueueLane]chan struct{} {
	next := make(map[sessionmanager.QueueLane]chan struct{})
	apply := func(lane sessionmanager.QueueLane, capValue int) {
		if capValue <= 0 {
			return
		}
		var used int
		if existing := current[lane]; existing != nil {
			used = len(existing)
		}
		if used > capValue {
			used = capValue
		}
		ch := make(chan struct{}, capValue)
		for i := 0; i < used; i++ {
			ch <- struct{}{}
		}
		next[lane] = ch
	}
	apply(sessionmanager.QueueLanePriority, caps.Steer)
	apply(sessionmanager.QueueLaneDefault, caps.Followup)
	apply(sessionmanager.QueueLaneCollect, caps.Collect)
	return next
}

func (manager *Manager) acquireGlobal(ctx context.Context, lane sessionmanager.QueueLane) error {
	ch := manager.global[lane]
	if ch == nil {
		return nil
	}
	select {
	case ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (manager *Manager) releaseGlobal(lane sessionmanager.QueueLane) {
	ch := manager.global[lane]
	if ch == nil {
		return
	}
	select {
	case <-ch:
	default:
	}
}

func resolveQueueKey(sessionKey string) (domainsession.Key, error) {
	trimmed := strings.TrimSpace(sessionKey)
	if trimmed == "" {
		return domainsession.Key{}, ErrInvalidRequest
	}
	parts, _, err := domainsession.NormalizeSessionKey(trimmed)
	if err == nil {
		thread := strings.TrimSpace(parts.ThreadRef)
		if thread == "" {
			thread = strings.TrimSpace(parts.PrimaryID)
		}
		channel := strings.TrimSpace(parts.Channel)
		if channel == "" {
			channel = "gateway"
		}
		return domainsession.NewKey(channel, strings.TrimSpace(parts.AccountID), thread)
	}
	return domainsession.NewKey("gateway", "", trimmed)
}

func queueTicketLane(mode domainsession.QueueMode) sessionmanager.QueueLane {
	switch mode {
	case domainsession.QueueModeSteer:
		return sessionmanager.QueueLanePriority
	case domainsession.QueueModeCollect:
		return sessionmanager.QueueLaneCollect
	default:
		return sessionmanager.QueueLaneDefault
	}
}

func (manager *Manager) emitLaneDiagnostic(ctx context.Context, lane string, eventType string, reason string, active int, capacity int) {
	if manager == nil || manager.events == nil {
		return
	}
	now := manager.now()
	envelope := gatewayevents.Envelope{
		EventID:   manager.newID(),
		Type:      "queue.lane_diagnostic",
		Topic:     "queue",
		Timestamp: now,
	}
	payload := map[string]any{
		"type":      strings.TrimSpace(eventType),
		"lane":      strings.TrimSpace(lane),
		"reason":    strings.TrimSpace(reason),
		"active":    active,
		"capacity":  capacity,
		"timestamp": now.Format(time.RFC3339Nano),
	}
	_, _ = manager.events.Publish(ctx, envelope, payload)
}
