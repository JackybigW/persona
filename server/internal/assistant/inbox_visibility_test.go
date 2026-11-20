package assistant

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
)

func visibilityTestConfig() *assistantwire.InboxVisibilityConfig {
	return &assistantwire.InboxVisibilityConfig{Sources: []string{"lark"}, Chats: []string{"synthetic-private"}}
}

func TestInboxVisibilityMatch(t *testing.T) {
	newItem := func(chat string, direct bool) *InboxItem {
		it := actionItem("lark:visibility", time.Now())
		it.Source = "lark"
		it.Lark = &InboxLark{FeedID: chat, IsGroup: !direct}
		return it
	}
	cfg := visibilityTestConfig()
	if inboxVisibilityMatch(newItem("synthetic-work-dm", true), cfg) {
		t.Fatal("an explicit private-chat list excluded an unlisted work DM")
	}
	if !inboxVisibilityMatch(newItem("synthetic-private", false), cfg) {
		t.Fatal("an explicitly listed group was not excluded")
	}
	cfg = &assistantwire.InboxVisibilityConfig{Sources: []string{"lark"}, DirectChats: true}
	if !inboxVisibilityMatch(newItem("synthetic-work-dm", true), cfg) || inboxVisibilityMatch(newItem("synthetic-group", false), cfg) {
		t.Fatal("the direct-chat switch did not distinguish direct and group chats")
	}
	if !inboxVisibilityMatch(&InboxItem{Source: "lark", Kind: KindDM}, cfg) {
		t.Fatal("unknown Lark chat type did not fail closed with the direct-chat switch on")
	}
	var zero *assistantwire.InboxVisibilityConfig
	if inboxVisibilityMatch(newItem("synthetic-private", true), zero) {
		t.Fatal("nil visibility config changed legacy behavior")
	}
}

func TestInboxVisibilityQuarantineReleasesQueuedAndHandling(t *testing.T) {
	for _, running := range []bool{false, true} {
		name := "queued"
		if running {
			name = "handling"
		}
		t.Run(name, func(t *testing.T) {
			h := agentHub(t)
			it := actionItem("lark:synthetic-private:1", time.Now())
			it.Source = "lark"
			it.Lark.FeedID = "synthetic-private"
			h.inbox.items[it.ID] = it
			now := time.Now()
			if got := h.inbox.escalate(now.Add(-time.Hour), now); len(got) != 1 {
				t.Fatal("test item did not enter AI handling")
			}
			if running {
				it.Agent.State, it.Agent.SessionID = AgentRunning, "synthetic-session"
				h.inbox.setHandling(it, HandlingLooking, "", now)
			}
			h.quarantineInboxVisibility(context.Background(), visibilityTestConfig())
			if !it.VisibilityExcluded {
				t.Fatal("quarantine marker was not persisted on the item")
			}
			checkPolicyReleased(t, it, h.inbox.userLang())
			if running && (it.Agent.MachineID != "" || it.Agent.SessionID != "" || it.Handling.MachineID != "" || it.Handling.SessionID != "") {
				t.Fatalf("quarantine left a live handling session attached: agent=%+v handling=%+v", it.Agent, it.Handling)
			}
			before := len(conversationStarts(h.f))
			h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
			if got := len(conversationStarts(h.f)); got != before {
				t.Fatalf("quarantined item started %d conversations", got-before)
			}
		})
	}
}

func TestInboxVisibilityQueuePassQuarantinesWithoutHandPass(t *testing.T) {
	h := agentHub(t)
	h.setConfig(func(c *assistantwire.InboxConfig) { c.Visibility = visibilityTestConfig() })
	h.inbox.cfgMu.Lock()
	h.inbox.cfg = assistantwire.InboxConfig{}
	h.inbox.cfgAt = time.Now()
	h.inbox.cfgMu.Unlock()
	it := actionItem("lark:synthetic-private:direct-queue", time.Now())
	it.Source = "lark"
	it.Lark.FeedID = "synthetic-private"
	it.Agent = &InboxAgent{State: AgentQueued}
	h.inbox.items[it.ID] = it
	before := len(conversationStarts(h.f))
	h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
	if !it.VisibilityExcluded {
		t.Fatal("queue pass did not persist the current visibility exclusion")
	}
	checkPolicyReleased(t, it, h.inbox.userLang())
	if got := len(conversationStarts(h.f)); got != before {
		t.Fatalf("newly excluded stored chat started %d handling conversations", got-before)
	}
}

func TestInboxVisibilityQuarantineBlocksExistingSuggestion(t *testing.T) {
	h, lark, it := handledHub(t)
	it.Source = "lark"
	it.Lark.FeedID = "synthetic-private"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic reply before quarantine"}`); err != nil {
		t.Fatal(err)
	}
	h.setConfig(func(c *assistantwire.InboxConfig) { c.Visibility = visibilityTestConfig() })
	h.quarantineInboxVisibility(context.Background(), visibilityTestConfig())
	if currentSuggestion(it) == nil {
		t.Fatal("quarantine deleted the suggestion instead of retaining its audit")
	}
	if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
		t.Fatal("quarantined suggestion was confirmed")
	}
	if len(lark.got) != 0 {
		t.Fatalf("quarantined suggestion called the sender: %+v", lark.got)
	}
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic replacement"}`); err == nil {
		t.Fatal("quarantined item accepted a new suggestion")
	}
}

func TestInboxVisibilityFreshSuggestAndConfirm(t *testing.T) {
	h, lark, it := handledHub(t)
	it.Source = "lark"
	it.Lark.FeedID = "synthetic-private"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic reply before visibility changes"}`); err != nil {
		t.Fatal(err)
	}
	h.setConfig(func(c *assistantwire.InboxConfig) { c.Visibility = visibilityTestConfig() })
	h.inbox.cfgMu.Lock()
	h.inbox.cfg = assistantwire.InboxConfig{}
	h.inbox.cfgAt = time.Now()
	h.inbox.cfgMu.Unlock()
	g := currentSuggestion(it)
	if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
		t.Fatal("stale disabled cache confirmed a newly excluded reply")
	}
	if currentSuggestion(it) != g {
		t.Fatal("fresh visibility denial consumed the suggestion")
	}
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic replacement through stale cache"}`); err == nil {
		t.Fatal("stale disabled cache published a reply for a newly excluded chat")
	}
	if len(lark.got) != 0 {
		t.Fatalf("fresh visibility denial called the sender: %+v", lark.got)
	}
}

func TestInboxVisibilityRecheckAfterReservationSave(t *testing.T) {
	h, lark, it := handledHub(t)
	it.Source = "lark"
	it.Lark.FeedID = "synthetic-private"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic reply before visibility changes"}`); err != nil {
		t.Fatal(err)
	}
	st := &recoveryBeforeWriteStore{inboxMemStore: h.st, before: func() {
		h.setConfig(func(c *assistantwire.InboxConfig) { c.Visibility = visibilityTestConfig() })
	}}
	h.store = func() Store { return st }
	if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
		t.Fatal("reply was confirmed after its chat became excluded during persistence")
	}
	if len(lark.got) != 0 {
		t.Fatalf("pre-send visibility recheck called the sender: %+v", lark.got)
	}
}

func TestInboxVisibilityFinalSendUsesFreshConfig(t *testing.T) {
	h, lark, it := handledHub(t)
	it.Source = "lark"
	it.Lark.FeedID = "synthetic-private"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic reply before the final visibility gate"}`); err != nil {
		t.Fatal(err)
	}
	h.inbox.cfgMu.Lock()
	h.inbox.cfg = assistantwire.InboxConfig{}
	h.inbox.cfgAt = time.Now()
	h.inbox.cfgMu.Unlock()
	h.in.mu.Lock()
	h.in.configQueue = []assistantwire.InboxConfig{
		{},                                   // confirmation before the edit-veto preflight
		{},                                   // confirmation after the veto
		{},                                   // reservation recheck
		{},                                   // per-item send-mutex recheck before the veto
		{Visibility: visibilityTestConfig()}, // final authorization immediately before the outward send
	}
	h.in.mu.Unlock()
	if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
		t.Fatal("reply bypassed the final fresh visibility gate")
	}
	if len(lark.got) != 0 {
		t.Fatalf("final fresh visibility denial called the sender: %+v", lark.got)
	}
}

func TestInboxVisibilityQuarantineKeepsSentAuditWithoutResending(t *testing.T) {
	h, lark, it := handledHub(t)
	it.Source = "lark"
	it.Lark.FeedID = "synthetic-private"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic already-sent reply"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err != nil {
		t.Fatal(err)
	}
	if len(lark.got) != 1 {
		t.Fatalf("test setup sent %d replies, want one", len(lark.got))
	}
	a := suggestionAttempt(it, 1)
	if a == nil || a.Status != suggestionSent {
		t.Fatalf("test setup has no sent attempt: %+v", a)
	}
	text, proposal := a.Text, a.Proposal
	h.quarantineInboxVisibility(context.Background(), visibilityTestConfig())
	h.quarantineInboxVisibility(context.Background(), visibilityTestConfig())
	if !it.VisibilityExcluded || len(it.SuggestionAttempts) != 1 || a.Status != suggestionSent || a.Text != text || a.Proposal != proposal {
		t.Fatalf("sent audit changed during quarantine: %+v", it.SuggestionAttempts)
	}
	if len(lark.got) != 1 {
		t.Fatalf("quarantine re-sent the historical attempt: %+v", lark.got)
	}
}

func TestInboxVisibilityHandedSuggestionUsesFreshConfig(t *testing.T) {
	h, _, it := handoffHub(t)
	handOff(t, h, it)
	it.Source = "lark"
	it.Lark.FeedID = "synthetic-private"
	h.setConfig(func(c *assistantwire.InboxConfig) { c.Visibility = visibilityTestConfig() })
	h.inbox.cfgMu.Lock()
	h.inbox.cfg = assistantwire.InboxConfig{}
	h.inbox.cfgAt = time.Now()
	h.inbox.cfgMu.Unlock()
	if _, err := mainSuggest(h, "m1", `{"item":"lark:f1:100","kind":"agent","text":"Inspect the synthetic private request"}`); err == nil {
		t.Fatal("handed-off suggestion used the stale disabled visibility cache")
	}
	if currentSuggestion(it) != nil {
		t.Fatalf("newly excluded handed-off suggestion was published: %+v", currentSuggestion(it))
	}
}

func TestInboxVisibilityHandedAgentUsesFreshConfig(t *testing.T) {
	h := agentHub(t)
	h.setConfig(func(c *assistantwire.InboxConfig) {
		c.Agent.Enabled = true
		c.Visibility = visibilityTestConfig()
	})
	h.inbox.cfgMu.Lock()
	h.inbox.cfg = assistantwire.InboxConfig{}
	h.inbox.cfgAt = time.Now()
	h.inbox.cfgMu.Unlock()
	it := disabledTwinItem(h, "visibility-handed-agent", HandlingDone, AgentSkipped, "parent-session")
	it.Source = "lark"
	it.Lark = &InboxLark{FeedID: "synthetic-private", IsGroup: true}
	g := &InboxSuggestion{Version: 1, Kind: SuggestAgent, Text: "Inspect the synthetic private request", At: time.Now(), MachineID: "server"}
	if _, err := h.confirmHandedAgent(context.Background(), it, g, g.Text, false, SessionRef{MachineID: "server", SessionID: "handoff-session"}, "Synthetic"); err == nil {
		t.Fatal("handed-off Agent bypassed fresh visibility")
	}
	h.f.mu.Lock()
	starts := len(h.f.starts)
	h.f.mu.Unlock()
	if starts != 0 {
		t.Fatalf("newly excluded handed-off Agent started %d sessions", starts)
	}
}

func TestInboxVisibilityToolVetoBeforeConnectorCall(t *testing.T) {
	h, _, it := handledHub(t)
	ext := &policyExtensions{read: map[string]bool{"lark_messages": true, "lark_search": true}}
	h.SetExtensions(ext)
	h.setConfig(func(c *assistantwire.InboxConfig) {
		c.Agent.ReadProfile = "source_only"
		c.Agent.ConnectorTools = []string{"lark_messages", "lark_search"}
		c.Visibility = visibilityTestConfig()
	})
	h.originSet(t.Context(), "server", "s1", OriginInbox, it.ID, "")
	for _, call := range []struct {
		name string
		args string
	}{
		{name: "lark_messages", args: `{"chat_id":"synthetic-private"}`},
		{name: "lark_messages", args: `{"thread":"synthetic-unverifiable-thread"}`},
		{name: "lark_search", args: `{"query":"synthetic"}`},
	} {
		res := h.mcpCallWithOrigin(t.Context(), "server", "s1", call.name, json.RawMessage(call.args), true, OriginInbox)
		if !res.IsError {
			t.Errorf("visibility allowed %s %s: %+v", call.name, call.args, res)
		}
	}
	if len(ext.calls) != 0 {
		t.Fatalf("visibility vetoes reached the connector: %v", ext.calls)
	}
	res := h.mcpCallWithOrigin(t.Context(), "server", "s1", "lark_messages", json.RawMessage(`{"chat_id":"synthetic-work-dm"}`), true, OriginInbox)
	if res.IsError || len(ext.calls) != 1 {
		t.Fatalf("unlisted work DM did not retain its read path: %+v, calls=%v", res, ext.calls)
	}
}
