package conformance_test

import (
	"slices"
	"testing"

	"github.com/nwverse/persona/sdk/go/wire"
	"github.com/nwverse/persona/server/internal/connector"
)

func TestInboxVisibilityReview12CanonicalCapabilityAndSDK(t *testing.T) {
	schemas := loadSchemas(t)
	definition := schemas.def(t, "ConnectorCapability")
	if !slices.Contains(asSlice(definition["enum"]), any(string(connector.CapInboxVisibility))) {
		t.Errorf("canonical ConnectorCapability enum %v omits %q", definition["enum"], connector.CapInboxVisibility)
	}
	if !slices.Contains(wire.ConnectorCapabilities, string(connector.CapInboxVisibility)) {
		t.Errorf("generated Go ConnectorCapabilities %v omits %q", wire.ConnectorCapabilities, connector.CapInboxVisibility)
	}
}
