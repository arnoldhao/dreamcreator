package ws

import (
	"context"
	"testing"
	"time"

	"dreamcreator/internal/application/events"
	"golang.org/x/net/websocket"
)

func TestWebSocketServerReplaysEventsByCursor(t *testing.T) {
	t.Parallel()

	bus := events.NewInMemoryBus()
	server := NewServer("127.0.0.1:0", bus)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Shutdown(context.Background())

	conn := mustDialWS(t, server.URL())
	_ = mustReadWS(t, conn) // system.hello
	if err := websocket.JSON.Send(conn, clientMessage{
		Action: "subscribe",
		Topics: []string{"chat.thread.updated"},
	}); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	if err := bus.Publish(context.Background(), events.Event{
		Topic: "chat.thread.updated",
		Type:  "thread-updated",
		Payload: map[string]any{
			"threadId": "t-1",
		},
	}); err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	first := mustReadWS(t, conn)
	if first.Seq == 0 {
		t.Fatalf("expected seq on first event")
	}
	afterSeq := first.Seq
	_ = conn.Close()

	if err := bus.Publish(context.Background(), events.Event{
		Topic: "chat.thread.updated",
		Type:  "thread-updated",
		Payload: map[string]any{
			"threadId": "t-2",
		},
	}); err != nil {
		t.Fatalf("publish 2: %v", err)
	}

	conn2 := mustDialWS(t, server.URL())
	_ = mustReadWS(t, conn2) // system.hello
	if err := websocket.JSON.Send(conn2, clientMessage{
		Action:  "subscribe",
		Topics:  []string{"chat.thread.updated"},
		Cursors: map[string]int64{"chat.thread.updated": afterSeq},
	}); err != nil {
		t.Fatalf("subscribe with cursor: %v", err)
	}

	replayed := mustReadWS(t, conn2)
	if !replayed.Replay {
		t.Fatalf("expected replayed event")
	}
	if replayed.Seq <= afterSeq {
		t.Fatalf("unexpected replay seq: got %d after %d", replayed.Seq, afterSeq)
	}
	if replayed.Topic != "chat.thread.updated" {
		t.Fatalf("unexpected replay topic: %s", replayed.Topic)
	}
}

func TestWebSocketServerReplayGapEmitsResyncRequired(t *testing.T) {
	t.Parallel()

	bus := events.NewInMemoryBus()
	server := NewServer("127.0.0.1:0", bus)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := server.Start(ctx); err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer server.Shutdown(context.Background())

	for i := 0; i < 600; i++ {
		if err := bus.Publish(context.Background(), events.Event{
			Topic: "chat.thread.updated",
			Type:  "thread-updated",
			Payload: map[string]any{
				"threadId": "t-gap",
				"index":    i,
			},
		}); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	conn := mustDialWS(t, server.URL())
	_ = mustReadWS(t, conn) // system.hello
	if err := websocket.JSON.Send(conn, clientMessage{
		Action:  "subscribe",
		Topics:  []string{"chat.thread.updated"},
		Cursors: map[string]int64{"chat.thread.updated": 1},
	}); err != nil {
		t.Fatalf("subscribe with stale cursor: %v", err)
	}

	// Replay gap emits a deterministic resync-required event first.
	event := mustReadWS(t, conn)
	if event.Type != "resync-required" {
		t.Fatalf("unexpected event type: %s", event.Type)
	}
	if event.Topic != "chat.thread.updated" {
		t.Fatalf("unexpected event topic: %s", event.Topic)
	}
}

func mustDialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, err := websocket.Dial(url, "", "http://localhost/")
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	return conn
}

func mustReadWS(t *testing.T, conn *websocket.Conn) outboundMessage {
	t.Helper()
	type result struct {
		msg outboundMessage
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var message outboundMessage
		err := websocket.JSON.Receive(conn, &message)
		ch <- result{msg: message, err: err}
	}()
	select {
	case received := <-ch:
		if received.err != nil {
			t.Fatalf("read ws message: %v", received.err)
		}
		return received.msg
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for ws message")
	}
	return outboundMessage{}
}
