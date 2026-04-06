package events

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	appevents "dreamcreator/internal/application/events"
	librarydto "dreamcreator/internal/application/library/dto"
	"dreamcreator/internal/domain/library"
	domainsession "dreamcreator/internal/domain/session"
)

type LibraryEventBridge struct {
	bus        appevents.Bus
	operations library.OperationRepository
	events     *Broker
	mu         sync.RWMutex
	meta       map[string]libraryOperationMeta
}

type libraryOperationMeta struct {
	sessionKey string
	sessionID  string
	runID      string
}

func NewLibraryEventBridge(bus appevents.Bus, operations library.OperationRepository, events *Broker) *LibraryEventBridge {
	return &LibraryEventBridge{bus: bus, operations: operations, events: events, meta: make(map[string]libraryOperationMeta)}
}

func (bridge *LibraryEventBridge) Start() func() {
	if bridge == nil || bridge.bus == nil || bridge.operations == nil || bridge.events == nil {
		return func() {}
	}
	cancelOperation := bridge.bus.Subscribe("library.operation", bridge.handle)
	cancelFile := bridge.bus.Subscribe("library.file", bridge.handle)
	cancelHistory := bridge.bus.Subscribe("library.history", bridge.handle)
	cancelWorkspace := bridge.bus.Subscribe("library.workspace", bridge.handle)
	return func() {
		cancelOperation()
		cancelFile()
		cancelHistory()
		cancelWorkspace()
	}
}

func (bridge *LibraryEventBridge) handle(event appevents.Event) {
	meta := bridge.resolveMetaFromPayload(context.Background(), event.Payload)
	if meta.sessionKey == "" && meta.runID == "" {
		return
	}
	eventType := resolveLibraryEventType(event)
	if eventType == "" {
		return
	}
	envelope := Envelope{Type: eventType, Topic: "library", SessionKey: meta.sessionKey, SessionID: meta.sessionID, RunID: meta.runID, Timestamp: event.Timestamp}
	if envelope.Timestamp.IsZero() {
		envelope.Timestamp = time.Now()
	}
	payload := map[string]any{"topic": event.Topic, "type": event.Type, "payload": event.Payload}
	_, _ = bridge.events.Publish(context.Background(), envelope, payload)
}

func (bridge *LibraryEventBridge) resolveMetaFromPayload(ctx context.Context, payload any) libraryOperationMeta {
	if payload == nil {
		return libraryOperationMeta{}
	}
	switch value := payload.(type) {
	case librarydto.LibraryOperationDTO:
		return bridge.resolveMeta(ctx, value.ID, value.InputJSON)
	case *librarydto.LibraryOperationDTO:
		if value != nil {
			return bridge.resolveMeta(ctx, value.ID, value.InputJSON)
		}
	case librarydto.LibraryHistoryRecordDTO:
		if runID := strings.TrimSpace(value.Source.RunID); runID != "" {
			return libraryOperationMeta{runID: runID}
		}
		if operationID := strings.TrimSpace(value.Refs.OperationID); operationID != "" {
			return bridge.resolveMeta(ctx, operationID, "")
		}
	case *librarydto.LibraryHistoryRecordDTO:
		if value != nil {
			if runID := strings.TrimSpace(value.Source.RunID); runID != "" {
				return libraryOperationMeta{runID: runID}
			}
			if operationID := strings.TrimSpace(value.Refs.OperationID); operationID != "" {
				return bridge.resolveMeta(ctx, operationID, "")
			}
		}
	case librarydto.LibraryFileDTO:
		if operationID := strings.TrimSpace(value.LatestOperationID); operationID != "" {
			return bridge.resolveMeta(ctx, operationID, "")
		}
		if operationID := strings.TrimSpace(value.Origin.OperationID); operationID != "" {
			return bridge.resolveMeta(ctx, operationID, "")
		}
	case *librarydto.LibraryFileDTO:
		if value != nil {
			if operationID := strings.TrimSpace(value.LatestOperationID); operationID != "" {
				return bridge.resolveMeta(ctx, operationID, "")
			}
			if operationID := strings.TrimSpace(value.Origin.OperationID); operationID != "" {
				return bridge.resolveMeta(ctx, operationID, "")
			}
		}
	case librarydto.WorkspaceStateRecordDTO:
		if operationID := strings.TrimSpace(value.OperationID); operationID != "" {
			return bridge.resolveMeta(ctx, operationID, "")
		}
	case *librarydto.WorkspaceStateRecordDTO:
		if value != nil {
			if operationID := strings.TrimSpace(value.OperationID); operationID != "" {
				return bridge.resolveMeta(ctx, operationID, "")
			}
		}
	case map[string]any:
		if operationID := getString(value, "operationId", "operationID", "id"); operationID != "" {
			return bridge.resolveMeta(ctx, operationID, getString(value, "inputJson", "inputJSON"))
		}
	}
	return libraryOperationMeta{}
}

func (bridge *LibraryEventBridge) resolveMeta(ctx context.Context, operationID string, inputJSON string) libraryOperationMeta {
	trimmedID := strings.TrimSpace(operationID)
	if trimmedID == "" && strings.TrimSpace(inputJSON) == "" {
		return libraryOperationMeta{}
	}
	if trimmedID != "" {
		bridge.mu.RLock()
		cached, ok := bridge.meta[trimmedID]
		bridge.mu.RUnlock()
		if ok {
			return cached
		}
	}
	meta := extractOperationMeta(inputJSON)
	if trimmedID != "" && (meta.sessionKey == "" && meta.runID == "") {
		operation, err := bridge.operations.Get(ctx, trimmedID)
		if err == nil {
			meta = extractOperationMeta(operation.InputJSON)
		}
	}
	if meta.sessionKey != "" {
		if parts, _, err := domainsession.NormalizeSessionKey(meta.sessionKey); err == nil {
			meta.sessionID = strings.TrimSpace(parts.ThreadRef)
			if meta.sessionID == "" {
				meta.sessionID = strings.TrimSpace(parts.PrimaryID)
			}
		}
	}
	if trimmedID != "" {
		bridge.mu.Lock()
		bridge.meta[trimmedID] = meta
		bridge.mu.Unlock()
	}
	return meta
}

func extractOperationMeta(inputJSON string) libraryOperationMeta {
	if strings.TrimSpace(inputJSON) == "" {
		return libraryOperationMeta{}
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(inputJSON), &payload); err != nil {
		return libraryOperationMeta{}
	}
	return libraryOperationMeta{sessionKey: getString(payload, "sessionKey", "session_key"), runID: getString(payload, "runId", "runID")}
}

func resolveLibraryEventType(event appevents.Event) string {
	topic := strings.TrimSpace(event.Topic)
	suffix := "updated"
	if strings.EqualFold(strings.TrimSpace(event.Type), "delete") {
		suffix = "deleted"
	}
	switch topic {
	case "library.operation":
		return "library.operation." + suffix
	case "library.file":
		return "library.file." + suffix
	case "library.history":
		return "library.history." + suffix
	case "library.workspace":
		return "library.workspace." + suffix
	default:
		return ""
	}
}

func getString(values map[string]any, keys ...string) string {
	if values == nil {
		return ""
	}
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			if text, ok := raw.(string); ok {
				trimmed := strings.TrimSpace(text)
				if trimmed != "" {
					return trimmed
				}
			}
		}
	}
	return ""
}
