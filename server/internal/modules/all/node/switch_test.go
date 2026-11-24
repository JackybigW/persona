package node

import (
	"testing"

	"github.com/nwverse/persona/server/internal/moduleswitch/switchtest"
)

// switchPending are the node root's calls of provider seams that do not follow the run-time switch yet (#14405). It
// only shrinks: an entry no longer called fails.
var switchPending = []string{
	"commandport.AddCommand",
	"iphoneport.Provide",
	"screenport.Provide",
}

// Every provider seam a node-side module of this build fills follows the run-time module switch as the node reads it
// from its hub's features (moduleswitch.FromFeatures): present with the module on, absent with it switched off.
func TestEverySeamFollowsTheSwitch(t *testing.T) {
	switchtest.Check(t, ".", want, nil, switchPending)
}
