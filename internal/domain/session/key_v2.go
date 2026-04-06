package session

import (
	"errors"
	"strings"
)

const sessionKeyDelimiter = "::"
const sessionKeyPrefix = "v2"

var ErrInvalidSessionKey = errors.New("invalid session key")

type KeyParts struct {
	AgentID   string
	Channel   string
	Scope     string
	PrimaryID string
	AccountID string
	ThreadRef string
}

func BuildSessionKey(parts KeyParts) (string, error) {
	primary := strings.TrimSpace(parts.PrimaryID)
	threadRef := strings.TrimSpace(parts.ThreadRef)
	if primary == "" && threadRef == "" {
		return "", ErrInvalidSessionKey
	}
	segments := []string{
		sessionKeyPrefix,
		normalizeKeyPart(parts.AgentID),
		normalizeKeyPart(parts.Channel),
		normalizeKeyPart(parts.Scope),
		normalizeKeyPart(primary),
		normalizeKeyPart(parts.AccountID),
		normalizeKeyPart(threadRef),
	}
	return strings.Join(segments, sessionKeyDelimiter), nil
}

func ParseSessionKey(raw string) (KeyParts, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return KeyParts{}, ErrInvalidSessionKey
	}
	if strings.HasPrefix(trimmed, sessionKeyPrefix+sessionKeyDelimiter) {
		parts := strings.Split(trimmed, sessionKeyDelimiter)
		if len(parts) < 7 {
			return KeyParts{}, ErrInvalidSessionKey
		}
		return KeyParts{
			AgentID:   denormalizeKeyPart(parts[1]),
			Channel:   denormalizeKeyPart(parts[2]),
			Scope:     denormalizeKeyPart(parts[3]),
			PrimaryID: denormalizeKeyPart(parts[4]),
			AccountID: denormalizeKeyPart(parts[5]),
			ThreadRef: denormalizeKeyPart(parts[6]),
		}, nil
	}
	return KeyParts{}, ErrInvalidSessionKey
}

func NormalizeSessionKey(raw string) (KeyParts, string, error) {
	parts, err := ParseSessionKey(raw)
	if err != nil {
		return KeyParts{}, "", err
	}
	newKey, err := BuildSessionKey(parts)
	if err != nil {
		return parts, "", err
	}
	return parts, newKey, nil
}

func denormalizeKeyPart(value string) string {
	if value == "-" {
		return ""
	}
	return value
}
