package all

import (
	"testing"

	"github.com/nwverse/persona/server/internal/moduleswitch/switchtest"
)

// switchPending are the composition root's calls of provider seams that do not follow the run-time switch yet
// (#14405). It only shrinks: an entry no longer called fails.
var switchPending = []string{
	"builtinext.AddProgram",
	"builtinext.ProvideVoice",
	"builtinext.ProvideVolc",
	"botport.Provide",
	"cliport.Provide",
	"larkhost.Provide",
	"mailport.Provide",
	"recordinghost.Provide",
}

// Every provider seam a module of this build fills follows the run-time module switch (docs/architecture/
// core-and-modules.md, "The switch"): present with the module on, absent with it switched off, exactly as in a build
// without it. A new seam that ignores the switch fails here. Run with each persona_without_<id> tag as well.
func TestEverySeamFollowsTheSwitch(t *testing.T) {
	switchtest.Check(t, ".", wantModules, []string{"kernel.Provide"}, switchPending)
}
