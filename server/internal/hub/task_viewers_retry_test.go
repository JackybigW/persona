package hub

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/bus"
)

func TestTaskViewerReportsRetryAfterAllManagersFail(t *testing.T) {
	for _, forced := range []bool{false, true} {
		name := "ordinary retry"
		if forced {
			name = "forced ownership refresh"
		}
		t.Run(name, func(t *testing.T) {
			h := newTestHub(t)
			owners := []string{"manager-a", "manager-b"}
			h.devices.kept = map[string]string{"box": owners[0]}
			view := []assistantwire.SessionRef{{MachineID: "box", SessionID: "agent"}}
			addTestClient(h, &client{subs: map[[2]string]bool{{"box", "agent"}: true}})
			failing := true
			attempts := 0
			reports := map[string]assistantwire.Clients{}
			for _, owner := range owners {
				c := rosterConn(h, owner, writerFunc(func(raw []byte) (int, error) {
					var event struct {
						Event  string
						Params assistantwire.Clients
					}
					if err := json.Unmarshal(raw, &event); err != nil {
						return 0, err
					}
					if event.Event == assistantwire.EventClients {
						attempts++
						if failing {
							return 0, errors.New("manager connection unavailable")
						}
						reports[owner] = event.Params
					}
					return len(raw), nil
				}))
				c.presence = busPresence{ID: owner, Role: bus.RoleManager}
			}
			h.fleetClientsChanged()
			if attempts != len(owners) || h.reported.clientsSent || !reflect.DeepEqual(h.reported.clients.Viewing, view) {
				t.Fatalf("all failed sends were cached as delivered: attempts=%d sent=%v snapshot=%+v",
					attempts, h.reported.clientsSent, h.reported.clients)
			}
			failing = false
			if forced {
				h.devices.kept["box"] = owners[1]
				h.fleetClientsRefresh()
			} else {
				h.fleetClientsChanged()
			}
			current := owners[0]
			if forced {
				current = owners[1]
			}
			if attempts != 2*len(owners) || len(reports) != len(owners) || !h.reported.clientsSent {
				t.Fatalf("the unchanged global snapshot was not retried: attempts=%d reports=%+v", attempts, reports)
			}
			for _, owner := range owners {
				got := reports[owner].Viewing
				if owner == current && !reflect.DeepEqual(got, view) || owner != current && len(got) != 0 {
					t.Errorf("retry used stale ownership for %s: %+v", owner, got)
				}
			}
			if forced {
				// Once delivery succeeded, an ownership-only move still needs a forced report of the same global set.
				h.devices.kept["box"] = owners[0]
				h.fleetClientsRefresh()
				if attempts != 3*len(owners) || !reflect.DeepEqual(reports[owners[0]].Viewing, view) ||
					len(reports[owners[1]].Viewing) != 0 {
					t.Fatalf("forced refresh was suppressed by the last successful snapshot: attempts=%d reports=%+v", attempts, reports)
				}
			}
		})
	}
}
