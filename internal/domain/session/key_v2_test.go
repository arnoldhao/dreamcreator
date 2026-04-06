package session

import "testing"

func TestSessionKeyBuildAndParse(t *testing.T) {
	key, err := BuildSessionKey(KeyParts{AgentID: "agent", Channel: "web", PrimaryID: "thread-1", ThreadRef: "thread-1"})
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	parts, err := ParseSessionKey(key)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if parts.Channel != "web" {
		t.Fatalf("unexpected channel: %s", parts.Channel)
	}
}

func TestNormalizeSessionKeyRejectsLegacyFormat(t *testing.T) {
	if _, _, err := NormalizeSessionKey("telegram::acct::thread-9"); err == nil {
		t.Fatalf("expected legacy format to be rejected")
	}
}
