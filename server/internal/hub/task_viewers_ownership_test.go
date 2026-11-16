package hub

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
)

func waitTaskViewerScope(t *testing.T, got <-chan assistantwire.Clients, want []assistantwire.SessionRef) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	var last assistantwire.Clients
	for {
		select {
		case last = <-got:
			if reflect.DeepEqual(last.Viewing, want) {
				return
			}
		case <-deadline:
			t.Fatalf("manager viewer scope = %+v, want %+v", last.Viewing, want)
		}
	}
}

func TestTaskViewersFollowMachineOwnershipChanges(t *testing.T) {
	for _, kind := range []string{"configured move", "kept move", "kept owner update"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			var h *Hub
			var a, b string
			var gotA, gotB chan assistantwire.Clients
			var move func(string) error
			if kind == "configured move" {
				hub, srv, _ := newBusHub(t)
				h = hub
				h.cfg.Machines = []MachineConfig{{ID: "box", Name: "Box", Kind: "local"}}
				a, b = assistantwire.AssistantID, "m-0b0b0b"
				gotA, gotB = make(chan assistantwire.Clients, 64), make(chan assistantwire.Clients, 64)
				connectClientsManager(t, h, srv, a, gotA)
				connectClientsManager(t, h, srv, b, gotB)
				move = func(to string) error {
					_, err := h.moveMachine(ctx, "box", to)
					return err
				}
			} else {
				h = newTestHub(t)
				h.store = newMeshStore()
				h.bus.idents = &keptIdents{ids: &memIdents{}, kept: map[string]json.RawMessage{}}
				if err := os.WriteFile(filepath.Join(h.cfg.dir, "hub.json"), []byte(`{"machines":[]}`), 0o600); err != nil {
					t.Fatal(err)
				}
				h.startDevices(ctx)
				srv := newHubServer(t, h)
				one := connectKeeper(t, h, srv, "m-0a0a0a", json.RawMessage(`{"id":"box","name":"Box","kind":"ssh","ssh_host":"box.example"}`))
				two := connectKeeper(t, h, srv, "m-0b0b0b")
				a, b, gotA, gotB = one.id, two.id, one.clients, two.clients
				h.devices.moving.Lock()
				err := h.setKept(ctx, "box", a)
				h.devices.moving.Unlock()
				if err != nil {
					t.Fatal(err)
				}
				move = func(to string) error {
					if kind == "kept move" {
						_, err := h.moveMachine(ctx, "box", to)
						return err
					}
					h.devices.moving.Lock()
					defer h.devices.moving.Unlock()
					return h.setKept(ctx, "box", to)
				}
			}

			oldA, oldB := h.busConnOf(a), h.busConnOf(b)
			c := &client{subs: map[[2]string]bool{{"box", "agent"}: true}}
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
			h.fleetClientsChanged()
			view := []assistantwire.SessionRef{{MachineID: "box", SessionID: "agent"}}
			waitTaskViewerScope(t, gotA, view)
			waitTaskViewerScope(t, gotB, nil)

			for _, to := range []string{b, a} {
				if err := move(to); err != nil {
					t.Fatalf("move to %s: %v", to, err)
				}
				if h.busConnOf(a) != oldA || h.busConnOf(b) != oldB || oldA == nil || oldB == nil {
					t.Fatal("the test reconnected a manager instead of refreshing its existing report")
				}
				if to == b {
					waitTaskViewerScope(t, gotA, nil)
					waitTaskViewerScope(t, gotB, view)
				} else {
					waitTaskViewerScope(t, gotA, view)
					waitTaskViewerScope(t, gotB, nil)
				}
				if got := h.viewedSessions(); !reflect.DeepEqual(got, view) {
					t.Fatalf("ownership changed the global viewer set: %+v", got)
				}
			}
		})
	}
}
