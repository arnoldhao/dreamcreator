package session

import "strings"

type QueueMode string

const (
	QueueModeSteer    QueueMode = "steer"
	QueueModeFollowup QueueMode = "followup"
	QueueModeCollect  QueueMode = "collect"
)

type Key struct {
	Channel string
	Account string
	Thread  string
}

func NewKey(channel, account, thread string) (Key, error) {
	thread = strings.TrimSpace(thread)
	if thread == "" {
		return Key{}, ErrInvalidSession
	}
	return Key{
		Channel: strings.TrimSpace(channel),
		Account: strings.TrimSpace(account),
		Thread:  thread,
	}, nil
}

func (key Key) String() string {
	return strings.Join([]string{
		normalizeKeyPart(key.Channel),
		normalizeKeyPart(key.Account),
		normalizeKeyPart(key.Thread),
	}, "::")
}

func ParseQueueMode(raw string) QueueMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(QueueModeSteer):
		return QueueModeSteer
	case string(QueueModeCollect):
		return QueueModeCollect
	default:
		return QueueModeFollowup
	}
}

func normalizeKeyPart(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}
