package hub

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// The hub keeps a manager's hello capabilities up to maxManagerCapabilities, so a capability past the cut is dropped
// without a word and the feature it gates silently never starts.
func TestCapabilityListBounds(t *testing.T) {
	var in []string
	for i := range maxManagerCapabilities + 1 {
		in = append(in, fmt.Sprintf("cap.%d", i))
	}
	in = append([]string{"", "  ", strings.Repeat("x", 65), "cap.0"}, in...)
	out := capabilityList(in)
	if len(out) != maxManagerCapabilities || out[0] != "cap.0" || out[len(out)-1] != fmt.Sprintf("cap.%d", maxManagerCapabilities-1) {
		t.Fatalf("kept %d: first %q, last %q", len(out), out[0], out[len(out)-1])
	}
}

// The real manager's hello must fit with room to spare: at the limit, the next capability a change adds would be
// cut (#13941: a project-review manager's 33rd capability was, once main's own list reached 32).
func TestManagerHelloCapabilitiesFit(t *testing.T) {
	h, server, _ := newBusHub(t)
	dialAssistant(t, h, server.URL, false)
	var n int
	eventually(t, 5*time.Second, "the manager's hello", func() bool {
		h.bus.mu.Lock()
		defer h.bus.mu.Unlock()
		for _, c := range h.bus.current {
			if c != nil && len(c.presence.Capabilities) > 0 {
				n = len(c.presence.Capabilities)
				return true
			}
		}
		return false
	})
	if n+8 > maxManagerCapabilities {
		t.Fatalf("the manager's hello has %d capabilities and the hub keeps %d: raise maxManagerCapabilities", n, maxManagerCapabilities)
	}
}
