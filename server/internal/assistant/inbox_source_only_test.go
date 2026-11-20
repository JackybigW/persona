package assistant

import (
	"context"
	"strings"
	"testing"

	"github.com/nwverse/persona/server/internal/assistantwire"
)

func TestSourceOnlyInboxAgentSkipsKnowledgeContext(t *testing.T) {
	ctx := context.Background()
	h, e := newInboxHubKB(t)
	if err := h.inboxReady(ctx); err != nil {
		t.Fatal(err)
	}
	waitNeedsYouIdle(t, h.Service)
	goldenKB(t, e)
	h.setConfig(func(c *assistantwire.InboxConfig) {
		c.Agent.ReadProfile = "source_only"
		c.Agent.ConnectorTools = []string{"lark_messages"}
	})
	it := *goldenItems(t)["mention"]
	it.Agent = &InboxAgent{State: AgentQueued}
	h.inbox.items[it.ID] = &it
	h.startInboxAgents(ctx, InboxAgentConfig{Enabled: true, ReadProfile: "source_only", ConnectorTools: []string{"lark_messages"}})
	if it.Agent.SessionID == "" {
		t.Fatalf("the source-only agent did not start: %+v", it.Agent)
	}
	text := h.f.lastSend(t)["text"].(string)
	if strings.Contains(text, "## 知识库里的背景") {
		t.Fatalf("source_only injected knowledge-base context: %q", text)
	}
	if !strings.Contains(text, it.Context) || it.Context == "" {
		t.Fatalf("source_only lost the current item context: %q", text)
	}
}
