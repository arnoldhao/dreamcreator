package observability

import "testing"

func TestParseLogLineConsoleFormat(t *testing.T) {
	line := "2026-03-13T09:24:05.397+0800\tINFO\tupdate/service.go:110\tupdate: checking for updates\t{\"currentVersion\":\"dev\"}"

	record := parseLogLine(line)

	if record.TS != "2026-03-13T09:24:05.397+0800" {
		t.Fatalf("unexpected ts: %q", record.TS)
	}
	if record.Level != "info" {
		t.Fatalf("unexpected level: %q", record.Level)
	}
	if record.Component != "update/service.go:110" {
		t.Fatalf("unexpected component: %q", record.Component)
	}
	if record.Message != "update: checking for updates" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if got := record.Fields["currentVersion"]; got != "dev" {
		t.Fatalf("unexpected fields currentVersion: %#v", got)
	}
}

func TestParseLogLineJSONFormat(t *testing.T) {
	line := "{\"level\":\"info\",\"ts\":\"2026-03-13T09:24:05.397+0800\",\"caller\":\"app/bootstrap.go:201\",\"msg\":\"application started\",\"language\":\"zh-CN\"}"

	record := parseLogLine(line)

	if record.TS != "2026-03-13T09:24:05.397+0800" {
		t.Fatalf("unexpected ts: %q", record.TS)
	}
	if record.Level != "info" {
		t.Fatalf("unexpected level: %q", record.Level)
	}
	if record.Component != "app/bootstrap.go:201" {
		t.Fatalf("unexpected component: %q", record.Component)
	}
	if record.Message != "application started" {
		t.Fatalf("unexpected message: %q", record.Message)
	}
	if got := record.Fields["language"]; got != "zh-CN" {
		t.Fatalf("unexpected fields language: %#v", got)
	}
}
