package extmgr

import (
	"encoding/json"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/extension"
)

// inboxVisibilityForConnectors gives a plugin only the boundary for a connector it contributes. Chat ids stay in
// manager.json and the initialize call; they are not copied into the extension registry's persisted settings.
func inboxVisibilityForConnectors(cfg *assistantwire.InboxVisibilityConfig, connectors []extension.ConnectorItem) json.RawMessage {
	for _, c := range connectors {
		if cfg.AppliesTo(c.ID) {
			raw, _ := json.Marshal(cfg)
			return raw
		}
	}
	return nil
}
