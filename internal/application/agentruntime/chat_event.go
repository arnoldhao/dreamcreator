package agentruntime

import (
	"encoding/json"
	"strings"

	"dreamcreator/internal/application/chatevent"
)

const chatEventType = "data-agent-event"

func EncodeChatEvent(event Event) (chatevent.Event, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return chatevent.Event{}, err
	}
	transient := true
	return chatevent.Event{
		Type:      chatEventType,
		Data:      data,
		Transient: &transient,
	}, nil
}

func DecodeChatEvent(event chatevent.Event) (Event, bool) {
	if strings.TrimSpace(event.Type) != chatEventType || len(event.Data) == 0 {
		return Event{}, false
	}
	var decoded Event
	if err := json.Unmarshal(event.Data, &decoded); err != nil {
		return Event{}, false
	}
	decoded.Type = EventType(strings.TrimSpace(string(decoded.Type)))
	if decoded.Type == "" {
		return Event{}, false
	}
	return decoded, true
}
