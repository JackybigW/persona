package assistant

import (
	"context"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/proto"
)

func TestReconnectInventoryOrdersConcurrentCardChanges(t *testing.T) {
	for _, withdrawing := range []bool{false, true} {
		name := "new card"
		if withdrawing {
			name = "withdrawn card"
		}
		t.Run(name, func(t *testing.T) {
			ft, wire := newHookedFleet(t)
			// This test controls the reconnect write itself. It needs no polling or expiry loops.
			ft.s.fleet.started.Do(func() {})
			ix := proto.Interaction{ID: "reconnect", Kind: proto.IxPermission, MachineID: "mac", SessionID: "s1", CreatedAt: time.Now()}
			if withdrawing {
				ft.hold(ix)
			}
			captured, release := make(chan struct{}), make(chan struct{})
			wire.hook = func(event string, params any) {
				if event == assistantwire.EventFleetInteractions {
					close(captured)
					<-release
				}
			}
			up, changed := make(chan struct{}), make(chan struct{})
			go func() { ft.s.FleetUp(context.Background()); close(up) }()
			<-captured
			started := make(chan struct{})
			go func() {
				close(started)
				if withdrawing {
					ft.s.finishInteraction(ix.ID)
				} else {
					ft.hold(ix)
					ft.s.tellCard(ix)
				}
				close(changed)
			}()
			<-started
			waitFor(t, "the pending set to change during reconnect", func() bool {
				return (len(ft.s.pendingInteractions()) == 0) == withdrawing
			})
			// Let an unsequenced change reach the transport while the snapshot is still in flight.
			select {
			case <-changed:
			case <-time.After(150 * time.Millisecond):
			}
			close(release)
			<-up
			<-changed
			want := assistantwire.EventFleetInteraction
			if withdrawing {
				want = assistantwire.EventFleetResolved
			}
			var order []string
			ft.f.mu.Lock()
			for _, tell := range ft.f.told {
				if tell.event == assistantwire.EventFleetInteractions || tell.event == want {
					order = append(order, tell.event)
				}
			}
			ft.f.mu.Unlock()
			if len(order) != 2 || order[0] != assistantwire.EventFleetInteractions || order[1] != want {
				t.Fatalf("the reconnect inventory overtook the card change: %v", order)
			}
		})
	}
}
