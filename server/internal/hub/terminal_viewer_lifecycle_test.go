package hub

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
)

func terminalViewerClient(h *Hub) *client {
	c := &client{hub: h, out: make(chan []byte, 32), subs: map[[2]string]bool{{"m1", "chat"}: true}}
	h.mu.Lock()
	if h.clients == nil {
		h.clients = map[*client]struct{}{}
	}
	h.clients[c] = struct{}{}
	h.mu.Unlock()
	return c
}

func terminalViewerReport(h *Hub) []assistantwire.SessionRef {
	h.reported.mu.Lock()
	defer h.reported.mu.Unlock()
	return append([]assistantwire.SessionRef(nil), h.reported.clients.Viewing...)
}

func waitTerminalViewerReport(t *testing.T, h *Hub, want []assistantwire.SessionRef) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if got := terminalViewerReport(h); reflect.DeepEqual(got, want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("fleet viewer report = %+v, want %+v", terminalViewerReport(h), want)
}

func TestTerminalWatchFailureClearsOnlyEndedViewerAndRefreshesReport(t *testing.T) {
	h := newTestHub(t)
	boardNode(t, h, "m1", func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method != "terminal.watch" {
			return map[string]any{}, nil
		}
		var p struct {
			SessionID string `json:"session_id"`
		}
		_ = json.Unmarshal(params, &p)
		if p.SessionID == "failed" {
			return nil, errors.New("watch failed")
		}
		return map[string]any{"control": false}, nil
	})
	c := terminalViewerClient(h)
	h.terminalSubscribe(context.Background(), c, "m1", "survivor", "", 0, nil, false)
	h.terminalSubscribe(context.Background(), c, "m1", "failed", "", 0, nil, false)

	deadline := time.After(3 * time.Second)
	for {
		select {
		case raw := <-c.out:
			var msg struct {
				Type  string `json:"type"`
				Error string `json:"error"`
			}
			_ = json.Unmarshal(raw, &msg)
			if msg.Type == "terminal" && msg.Error != "" {
				goto ended
			}
		case <-deadline:
			t.Fatal("watch failure did not end the terminal while the WebSocket client remained connected")
		}
	}

ended:
	want := []assistantwire.SessionRef{{MachineID: "m1", SessionID: "chat"}, {MachineID: "m1", SessionID: "survivor"}}
	waitTerminalViewerReport(t, h, want)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.termSubs[[3]string{"m1", "failed", "m1"}] {
		t.Fatal("failed terminal remained in the connected client's viewer subscriptions")
	}
	if !c.termSubs[[3]string{"m1", "survivor", "m1"}] {
		t.Fatal("ending one terminal removed the client's independent terminal subscription")
	}
	if !c.subs[[2]string{"m1", "chat"}] {
		t.Fatal("ending a terminal removed the independent chat subscription")
	}
}

func TestTerminalsEndedClearsMachineViewersAndKeepsChat(t *testing.T) {
	h := newTestHub(t)
	boardNode(t, h, "m1", func(context.Context, string, json.RawMessage) (any, error) {
		return map[string]any{"control": false}, nil
	})
	c := terminalViewerClient(h)
	h.terminalSubscribe(context.Background(), c, "m1", "one", "", 0, nil, false)
	h.terminalSubscribe(context.Background(), c, "m1", "two", "", 0, nil, false)
	h.terminalsEnded("m1", "machine disconnected")

	want := []assistantwire.SessionRef{{MachineID: "m1", SessionID: "chat"}}
	waitTerminalViewerReport(t, h, want)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.termSubs) != 0 {
		t.Fatalf("machine disconnect left terminal viewer entries: %+v", c.termSubs)
	}
	if !c.subs[[2]string{"m1", "chat"}] {
		t.Fatal("machine disconnect removed the independent chat subscription")
	}
}

func TestTerminalEndedPublishesDepartureWithConnectedViewers(t *testing.T) {
	h := newTestHub(t)
	boardNode(t, h, "m1", func(context.Context, string, json.RawMessage) (any, error) {
		return map[string]any{"control": false}, nil
	})
	clients := []*client{terminalViewerClient(h), terminalViewerClient(h)}
	for _, c := range clients {
		h.terminalSubscribe(context.Background(), c, "m1", "ended", "", 0, nil, false)
	}
	waitTerminalViewerReport(t, h, []assistantwire.SessionRef{
		{MachineID: "m1", SessionID: "chat"},
		{MachineID: "m1", SessionID: "ended"},
	})
	h.terminalEnded("m1", "ended", "stream ended", nil)
	waitTerminalViewerReport(t, h, []assistantwire.SessionRef{{MachineID: "m1", SessionID: "chat"}})
	for _, c := range clients {
		c.mu.Lock()
		remaining := len(c.termSubs)
		c.mu.Unlock()
		if remaining != 0 {
			t.Fatal("ended terminal left a viewer on a connected client")
		}
		h.mu.Lock()
		_, connected := h.clients[c]
		h.mu.Unlock()
		if !connected {
			t.Fatal("fixture disconnected the WebSocket instead of clearing its terminal")
		}
	}
}

func TestTerminalDropCleansViewerWhenRelayIsAlreadyGone(t *testing.T) {
	h := newTestHub(t)
	c := terminalViewerClient(h)
	c.termSubs = map[[3]string]bool{
		{"m1", "ended", "m1"}:          true,
		{"m1", "alias-ended", "alias"}: true,
	}
	// The relay has already been removed by terminalEnded; terminalDrop must still use the client's records.
	h.terminalDrop(c)
	want := []assistantwire.SessionRef{{MachineID: "m1", SessionID: "chat"}}
	waitTerminalViewerReport(t, h, want)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.termSubs) != 0 {
		t.Fatalf("terminalDrop left stale viewer entries without relays: %+v", c.termSubs)
	}
	if !c.subs[[2]string{"m1", "chat"}] {
		t.Fatal("terminalDrop removed the independent chat subscription")
	}
}
