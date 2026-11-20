package extmgr

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/extension"
)

func TestInboxConfigViewCarriesVisibilityFromManagerConfig(t *testing.T) {
	visibility := &assistantwire.InboxVisibilityConfig{Sources: []string{"lark"}, Chats: []string{"synthetic-private"}}
	h := &Manager{cfg: &Config{Inbox: InboxConfig{Visibility: visibility}}}
	got := h.inboxConfigView(context.Background()).Visibility
	if got != visibility {
		t.Fatalf("inbox visibility = %#v, want the manager's configured boundary %#v", got, visibility)
	}
	if len(got.Chats) != 1 || got.Chats[0] != "synthetic-private" || got.DirectChats {
		t.Fatalf("inbox visibility changed in transit: %#v", got)
	}
}

func TestInboxVisibilityInitializeTargetsOnlySelectedConnector(t *testing.T) {
	visibility := &assistantwire.InboxVisibilityConfig{Sources: []string{"lark"}, Chats: []string{"synthetic-private"}}
	raw := inboxVisibilityForConnectors(visibility, []extension.ConnectorItem{{ID: "other"}, {ID: "lark"}})
	var got assistantwire.InboxVisibilityConfig
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("selected connector did not receive visibility: %v", err)
	}
	if len(got.Chats) != 1 || got.Chats[0] != "synthetic-private" {
		t.Fatalf("initialize visibility = %#v", got)
	}
	if raw := inboxVisibilityForConnectors(visibility, []extension.ConnectorItem{{ID: "other"}}); raw != nil {
		t.Fatalf("unselected extension received private chat ids: %s", raw)
	}
	if raw := inboxVisibilityForConnectors(nil, []extension.ConnectorItem{{ID: "lark"}}); raw != nil {
		t.Fatalf("nil visibility changed legacy initialize params: %s", raw)
	}
}
