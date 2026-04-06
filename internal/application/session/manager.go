package session

import (
	"context"
	"sync"
	"time"

	domainsession "dreamcreator/internal/domain/session"
)

type QueueTicket struct {
	id        string
	start     chan struct{}
	createdAt time.Time
	key       domainsession.Key
	payload   any
}

func (ticket *QueueTicket) Payload() any {
	if ticket == nil {
		return nil
	}
	return ticket.payload
}

func (ticket *QueueTicket) ID() string {
	if ticket == nil {
		return ""
	}
	return ticket.id
}

func (ticket *QueueTicket) CreatedAt() time.Time {
	if ticket == nil {
		return time.Time{}
	}
	return ticket.createdAt
}

func (ticket *QueueTicket) SetPayload(payload any) {
	if ticket == nil {
		return
	}
	ticket.payload = payload
}

type QueueLane string

const (
	QueueLanePriority QueueLane = "priority"
	QueueLaneDefault  QueueLane = "default"
	QueueLaneCollect  QueueLane = "collect"
)

type QueueSnapshot struct {
	TotalSessions  int                  `json:"totalSessions"`
	ActiveSessions int                  `json:"activeSessions"`
	Queues         []QueueStateSnapshot `json:"queues"`
}

type QueueStateSnapshot struct {
	Key         string              `json:"key"`
	Active      bool                `json:"active"`
	QueuedCount int                 `json:"queuedCount"`
	UpdatedAt   string              `json:"updatedAt"`
	Lanes       []QueueLaneSnapshot `json:"lanes,omitempty"`
}

type QueueLaneSnapshot struct {
	Lane        string `json:"lane"`
	QueuedCount int    `json:"queuedCount"`
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*queueState
}

type queueState struct {
	key       domainsession.Key
	active    *QueueTicket
	queues    map[QueueLane][]*QueueTicket
	updatedAt time.Time
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*queueState),
	}
}

func (manager *Manager) Enqueue(
	key domainsession.Key,
	mode domainsession.QueueMode,
	payload any,
	merge func(existing any, incoming any) (any, bool),
) (*QueueTicket, int, bool) {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	lookup := key.String()
	state := manager.sessions[lookup]
	if state == nil {
		state = &queueState{key: key, queues: make(map[QueueLane][]*QueueTicket)}
		manager.sessions[lookup] = state
	}

	ticket := &QueueTicket{
		id:        newQueueID(),
		start:     make(chan struct{}),
		createdAt: time.Now(),
		key:       key,
		payload:   payload,
	}

	if state.active == nil {
		state.active = ticket
		state.updatedAt = time.Now()
		close(ticket.start)
		return ticket, 0, false
	}

	lane := laneFromMode(mode)
	if mode == domainsession.QueueModeCollect && merge != nil {
		if target := state.lastQueuedInLane(lane); target != nil {
			if mergedPayload, ok := merge(target.payload, payload); ok {
				target.payload = mergedPayload
				state.updatedAt = time.Now()
				position := state.queuePosition(target)
				if position <= 0 {
					position = 1
				}
				return target, position, true
			}
		}
	}
	state.queues[lane] = append(state.queues[lane], ticket)
	state.updatedAt = time.Now()
	return ticket, state.queuedCount(), false
}

func (manager *Manager) Wait(ctx context.Context, ticket *QueueTicket) error {
	if ticket == nil {
		return nil
	}
	select {
	case <-ticket.start:
		return nil
	case <-ctx.Done():
		manager.Cancel(ticket)
		return ctx.Err()
	}
}

func (manager *Manager) Done(ticket *QueueTicket) {
	if ticket == nil {
		return
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()

	lookup := ticket.key.String()
	state := manager.sessions[lookup]
	if state == nil {
		return
	}
	if state.active != nil && state.active.id == ticket.id {
		state.active = nil
		if next := state.nextQueuedTicket(); next != nil {
			state.active = next
			close(next.start)
		}
		state.updatedAt = time.Now()
		if state.active == nil && state.queuedCount() == 0 {
			delete(manager.sessions, lookup)
		}
		return
	}
	for lane, queue := range state.queues {
		for i, queued := range queue {
			if queued.id == ticket.id {
				state.queues[lane] = append(queue[:i], queue[i+1:]...)
				state.updatedAt = time.Now()
				break
			}
		}
	}
	if state.active == nil && state.queuedCount() == 0 {
		delete(manager.sessions, lookup)
	}
}

func (manager *Manager) Cancel(ticket *QueueTicket) {
	manager.Done(ticket)
}

func (manager *Manager) QueuedCount(key domainsession.Key, lane QueueLane) int {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	state := manager.sessions[key.String()]
	if state == nil {
		return 0
	}
	return len(state.queues[lane])
}

func (manager *Manager) LastQueuedTicket(key domainsession.Key, lane QueueLane) *QueueTicket {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	state := manager.sessions[key.String()]
	if state == nil {
		return nil
	}
	queue := state.queues[lane]
	if len(queue) == 0 {
		return nil
	}
	return queue[len(queue)-1]
}

func (manager *Manager) DropOldestQueued(key domainsession.Key, lane QueueLane) *QueueTicket {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	state := manager.sessions[key.String()]
	if state == nil {
		return nil
	}
	queue := state.queues[lane]
	if len(queue) == 0 {
		return nil
	}
	oldest := queue[0]
	state.queues[lane] = queue[1:]
	state.updatedAt = time.Now()
	if state.active == nil && state.queuedCount() == 0 {
		delete(manager.sessions, key.String())
	}
	return oldest
}

func (manager *Manager) Snapshot() QueueSnapshot {
	manager.mu.Lock()
	defer manager.mu.Unlock()

	snapshot := QueueSnapshot{
		TotalSessions: len(manager.sessions),
	}
	queues := make([]QueueStateSnapshot, 0, len(manager.sessions))
	for _, state := range manager.sessions {
		if state == nil {
			continue
		}
		active := state.active != nil
		if active {
			snapshot.ActiveSessions++
		}
		laneSnapshots := make([]QueueLaneSnapshot, 0, len(state.queues))
		for lane, queue := range state.queues {
			if len(queue) == 0 {
				continue
			}
			laneSnapshots = append(laneSnapshots, QueueLaneSnapshot{
				Lane:        string(lane),
				QueuedCount: len(queue),
			})
		}
		queues = append(queues, QueueStateSnapshot{
			Key:         state.key.String(),
			Active:      active,
			QueuedCount: state.queuedCount(),
			UpdatedAt:   state.updatedAt.Format(time.RFC3339),
			Lanes:       laneSnapshots,
		})
	}
	snapshot.Queues = queues
	return snapshot
}

func laneFromMode(mode domainsession.QueueMode) QueueLane {
	switch mode {
	case domainsession.QueueModeSteer:
		return QueueLanePriority
	case domainsession.QueueModeCollect:
		return QueueLaneCollect
	default:
		return QueueLaneDefault
	}
}

func (state *queueState) queuedCount() int {
	if state == nil {
		return 0
	}
	count := 0
	for _, queue := range state.queues {
		count += len(queue)
	}
	return count
}

func (state *queueState) nextQueuedTicket() *QueueTicket {
	if state == nil {
		return nil
	}
	for _, lane := range []QueueLane{QueueLanePriority, QueueLaneDefault, QueueLaneCollect} {
		queue := state.queues[lane]
		if len(queue) == 0 {
			continue
		}
		next := queue[0]
		state.queues[lane] = queue[1:]
		return next
	}
	return nil
}

func (state *queueState) lastQueuedInLane(lane QueueLane) *QueueTicket {
	if state == nil {
		return nil
	}
	queue := state.queues[lane]
	if len(queue) == 0 {
		return nil
	}
	return queue[len(queue)-1]
}

func (state *queueState) queuePosition(ticket *QueueTicket) int {
	if state == nil || ticket == nil {
		return 0
	}
	position := 0
	for _, lane := range []QueueLane{QueueLanePriority, QueueLaneDefault, QueueLaneCollect} {
		queue := state.queues[lane]
		for _, queued := range queue {
			position++
			if queued == ticket {
				return position
			}
		}
	}
	return position
}

func newQueueID() string {
	return time.Now().UTC().Format("20060102T150405.000000000")
}
