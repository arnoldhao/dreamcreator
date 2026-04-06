package session

import appsession "dreamcreator/internal/application/session"

type Service = appsession.Service
type Store = appsession.Store
type InMemoryStore = appsession.InMemoryStore
type Manager = appsession.Manager
type QueueTicket = appsession.QueueTicket
type QueueLane = appsession.QueueLane
type QueueSnapshot = appsession.QueueSnapshot
type QueueStateSnapshot = appsession.QueueStateSnapshot
type QueueLaneSnapshot = appsession.QueueLaneSnapshot
type CreateSessionRequest = appsession.CreateSessionRequest

const (
	QueueLanePriority = appsession.QueueLanePriority
	QueueLaneDefault  = appsession.QueueLaneDefault
	QueueLaneCollect  = appsession.QueueLaneCollect
)

var (
	NewService         = appsession.NewService
	NewInMemoryStore   = appsession.NewInMemoryStore
	NewManager         = appsession.NewManager
	ErrSessionNotFound = appsession.ErrSessionNotFound
)
