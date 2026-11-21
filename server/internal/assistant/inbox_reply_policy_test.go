package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/rpc"
)

func replyPolicyTestConfig() *assistantwire.InboxReplyPolicyConfig {
	return &assistantwire.InboxReplyPolicyConfig{
		Enabled: true, Revision: "test-v1", Sources: []string{"lark"}, UnknownMode: "owner_only",
		Routes: map[string]string{
			"personal_transaction": "owner_only", "security_compliance": "owner_only",
			"approval_permission": "owner_only", "decision_delegation": "owner_only",
			"technical_qa": "bot", "research": "bot", "alert_triage": "bot", "ppe_deploy": "bot", "mr_precheck": "bot",
			"no_reply": "no_reply", "ordinary_coordination": "owner_draft", "logistical_followup": "owner_draft",
			"acknowledgement": "owner_draft", "other_owner_reply": "owner_draft", "unknown": "owner_only",
		},
	}
}

func TestInboxReplyPolicyRoutes(t *testing.T) {
	cfg := replyPolicyTestConfig()
	tests := []struct{ category, want string }{
		{"personal_transaction", "owner_only"}, {"security_compliance", "owner_only"},
		{"approval_permission", "owner_only"}, {"decision_delegation", "owner_only"},
		{"technical_qa", "bot"}, {"research", "bot"}, {"alert_triage", "bot"}, {"ppe_deploy", "bot"}, {"mr_precheck", "bot"},
		{"ordinary_coordination", "owner_draft"}, {"logistical_followup", "owner_draft"}, {"acknowledgement", "owner_draft"},
		{"no_reply", "no_reply"}, {"unknown", "owner_only"}, {"", "owner_only"}, {"not-a-category", "owner_only"},
	}
	for _, tc := range tests {
		if got := replyRoute(tc.category, cfg); got != tc.want {
			t.Errorf("replyRoute(%q) = %q, want %q", tc.category, got, tc.want)
		}
	}
	if got := replyRoute("ordinary_coordination", nil); got != "" {
		t.Fatalf("nil policy route = %q, want legacy behavior", got)
	}
	cfg.Enabled = false
	if got := replyRoute("ordinary_coordination", cfg); got != "" {
		t.Fatalf("disabled policy route = %q, want legacy behavior", got)
	}
	cfg = replyPolicyTestConfig()
	cfg.Routes["approval_permission"] = "owner_draft"
	cfg.Routes["technical_qa"] = "owner_draft"
	cfg.Routes["unknown"] = "owner_draft"
	cfg.Routes["no_reply"] = "owner_draft"
	cfg.UnknownMode = "owner_draft"
	for _, tc := range []struct{ category, want string }{
		{"approval_permission", "owner_only"}, {"technical_qa", "bot"},
		{"unknown", "owner_only"}, {"no_reply", "no_reply"}, {"ordinary_coordination", "owner_draft"},
	} {
		if got := replyRoute(tc.category, cfg); got != tc.want {
			t.Errorf("unsafe override %s = %q, want %q", tc.category, got, tc.want)
		}
	}
	cfg = replyPolicyTestConfig()
	cfg.Revision = ""
	for _, category := range []string{"approval_permission", "technical_qa", "ordinary_coordination", "unknown"} {
		if got := replyRoute(category, cfg); got != "owner_only" {
			t.Errorf("empty revision %s = %q, want owner_only", category, got)
		}
	}
	for _, tc := range []struct{ name, key, value string }{
		{"invalid category route", "research", "send"},
		{"unknown category key", "misspelled_category", "owner_draft"},
		{"invalid unknown mode", "", "send"},
	} {
		cfg = replyPolicyTestConfig()
		if tc.key == "" {
			cfg.UnknownMode = tc.value
		} else {
			cfg.Routes[tc.key] = tc.value
		}
		for _, category := range []string{"technical_qa", "ordinary_coordination", "unknown"} {
			if got := replyRoute(category, cfg); got != "owner_only" {
				t.Errorf("%s: %s = %q, want owner_only", tc.name, category, got)
			}
		}
	}
}

func TestInboxReplyPolicyQueueGate(t *testing.T) {
	for _, route := range []string{"owner_only", "bot", "no_reply"} {
		t.Run(route, func(t *testing.T) {
			h := agentHub(t)
			h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
			it := actionItem("lark:queued:"+route, time.Now())
			it.Source = "lark"
			it.ReplyCategory, it.ReplyMode = map[string]string{"owner_only": "approval_permission", "bot": "technical_qa", "no_reply": "no_reply"}[route], route
			it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
			it.Agent = &InboxAgent{State: AgentQueued}
			h.inbox.items[it.ID] = it
			before := len(conversationStarts(h.f))
			h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
			checkPolicyReleased(t, it, h.inbox.userLang())
			if got := len(conversationStarts(h.f)); got != before {
				t.Fatalf("policy route %s started %d handling conversations", route, got-before)
			}
		})
	}
	t.Run("unclassified waits for a later classification pass", func(t *testing.T) {
		h := agentHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		it := actionItem("lark:queued:unclassified", time.Now())
		it.Source = "lark"
		it.Agent = &InboxAgent{State: AgentQueued}
		h.inbox.items[it.ID] = it
		before := len(conversationStarts(h.f))
		h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
		if got := len(conversationStarts(h.f)); got != before {
			t.Fatalf("unclassified item started %d handling conversations", got-before)
		}
		if it.Agent.State != AgentQueued || it.Handling != nil && it.Handling.State == HandlingDone {
			t.Fatalf("unclassified item was released instead of waiting: agent=%+v handling=%+v", it.Agent, it.Handling)
		}
	})
}

func TestInboxReplyPolicyOverridesTwinFirst(t *testing.T) {
	h := agentHub(t)
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
	it := actionItem("lark:queued:twin-first", time.Now())
	it.Source = "lark"
	it.Lark = &InboxLark{FeedID: "synthetic-chat", IsGroup: true}
	it.ReplyCategory, it.ReplyMode = "approval_permission", "owner_only"
	it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
	it.Agent = &InboxAgent{State: AgentQueued}
	h.inbox.items[it.ID] = it
	setPolicyFor(t, h, InboxPolicy{Source: "lark", Tag: "chat", Value: "synthetic-chat", Twin: TwinFirst})
	before := len(conversationStarts(h.f))
	h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
	checkPolicyReleased(t, it, h.inbox.userLang())
	if got := len(conversationStarts(h.f)); got != before {
		t.Fatalf("TwinFirst overrode owner_only and started %d conversations", got-before)
	}
}

func TestInboxReplyPolicyClassifiesTwinFirstFYIBeforeQueue(t *testing.T) {
	h := agentHub(t)
	h.in.model = fakeModel(t, `{"category":"no_reply"}`)
	h.setConfig(func(c *assistantwire.InboxConfig) {
		c.Agent.Enabled = true
		c.ReplyPolicy = replyPolicyTestConfig()
	})
	it := actionItem("lark:queued:twin-first-fyi", time.Now())
	it.Source, it.Tier, it.TriagedAt = "lark", TierFYI, it.Last
	it.Lark = &InboxLark{FeedID: "synthetic-fyi-chat", IsGroup: true}
	h.inbox.items[it.ID] = it
	setPolicyFor(t, h, InboxPolicy{Source: "lark", Tag: "chat", Value: "synthetic-fyi-chat", Twin: TwinFirst})
	h.classifyInboxReplyPolicies(context.Background(), h.inboxConfig(context.Background()).ReplyPolicy)
	if it.ReplyCategory != "no_reply" || it.ReplyMode != "no_reply" || !replyPolicyClassificationCurrent(it, replyPolicyTestConfig()) {
		t.Fatalf("TwinFirst FYI remained unclassifiable: %+v", it)
	}
	h.inbox.escalate(time.Now().Add(-time.Hour), time.Now())
	before := len(conversationStarts(h.f))
	h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
	checkPolicyReleased(t, it, h.inbox.userLang())
	if got := len(conversationStarts(h.f)); got != before {
		t.Fatalf("classified TwinFirst FYI started %d conversations", got-before)
	}
}

func TestInboxReplyPolicyFreshAdmissionAndHandback(t *testing.T) {
	t.Run("queue admission observes a newly enabled policy", func(t *testing.T) {
		h := agentHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		h.inbox.cfgMu.Lock()
		h.inbox.cfg = assistantwire.InboxConfig{}
		h.inbox.cfgAt = time.Now()
		h.inbox.cfgMu.Unlock()
		it := actionItem("lark:fresh-queue", time.Now())
		it.Source = "lark"
		it.Agent = &InboxAgent{State: AgentQueued}
		h.inbox.items[it.ID] = it
		before := len(conversationStarts(h.f))
		h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
		if got := len(conversationStarts(h.f)); got != before {
			t.Fatalf("stale disabled cache admitted %d conversations", got-before)
		}
		if it.Agent.State != AgentQueued {
			t.Fatalf("unclassified newly enabled item left the queue: %+v", it.Agent)
		}
	})
	t.Run("suggestion observes a newly enabled policy", func(t *testing.T) {
		h, _, it := handledHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		h.inbox.cfgMu.Lock()
		h.inbox.cfg = assistantwire.InboxConfig{}
		h.inbox.cfgAt = time.Now()
		h.inbox.cfgMu.Unlock()
		if _, err := suggestTool(t, h, `{"kind":"reply","text":"Freshly forbidden synthetic draft"}`); err == nil {
			t.Fatal("stale disabled cache published an unclassified reply")
		}
		if currentSuggestion(it) != nil {
			t.Fatalf("freshly forbidden suggestion was stored: %+v", currentSuggestion(it))
		}
	})
	t.Run("policy changes while a suggestion waits for agent association", func(t *testing.T) {
		h := agentHub(t)
		oldWait := inboxDraftWait
		inboxDraftWait = time.Second
		t.Cleanup(func() { inboxDraftWait = oldWait })
		h.inbox.cfgMu.Lock()
		h.inbox.cfg = assistantwire.InboxConfig{}
		h.inbox.cfgAt = time.Now()
		h.inbox.cfgMu.Unlock()
		h.in.mu.Lock()
		h.in.configQueue = []assistantwire.InboxConfig{{}, {ReplyPolicy: replyPolicyTestConfig()}}
		h.in.mu.Unlock()
		it := actionItem("lark:association-policy-race", time.Now())
		it.Source = "lark"
		it.Agent = &InboxAgent{State: AgentRunning, Task: "synthetic-task"}
		h.inbox.items[it.ID] = it
		go func() {
			time.Sleep(250 * time.Millisecond)
			h.inbox.mu.Lock()
			it.Agent.SessionID = "session-policy-association"
			h.inbox.mu.Unlock()
		}()
		_, _, err := h.suggest(context.Background(), "session-policy-association", suggestIn{Kind: SuggestReply, Text: "Synthetic draft across a policy change"})
		if !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("suggestion used the pre-wait policy, error = %v", err)
		}
		if currentSuggestion(it) != nil {
			t.Fatalf("policy-enabled suggestion was published after the association wait: %+v", currentSuggestion(it))
		}
	})
	t.Run("association cannot change during the post-wait fresh read", func(t *testing.T) {
		h := agentHub(t)
		oldWait := inboxDraftWait
		inboxDraftWait = time.Second
		t.Cleanup(func() { inboxDraftWait = oldWait })
		h.inbox.cfgMu.Lock()
		h.inbox.cfg = assistantwire.InboxConfig{}
		h.inbox.cfgAt = time.Now()
		h.inbox.cfgMu.Unlock()
		h.in.mu.Lock()
		h.in.configQueue = []assistantwire.InboxConfig{{}, {}}
		reads := 0
		h.in.configHook = func() {
			reads++
			if reads == 2 {
				h.inbox.mu.Lock()
				for _, item := range h.inbox.items {
					if item.ID == "lark:association-binding-race" {
						item.Agent.SessionID = "session-B"
					}
				}
				h.inbox.mu.Unlock()
			}
		}
		h.in.mu.Unlock()
		it := actionItem("lark:association-binding-race", time.Now())
		it.Source = "lark"
		it.Agent = &InboxAgent{State: AgentRunning, Task: "synthetic-task"}
		h.inbox.items[it.ID] = it
		go func() {
			time.Sleep(250 * time.Millisecond)
			h.inbox.mu.Lock()
			it.Agent.SessionID = "session-A"
			h.inbox.mu.Unlock()
		}()
		_, _, err := h.suggest(context.Background(), "session-A", suggestIn{Kind: SuggestReply, Text: "Delayed synthetic suggestion"})
		if !errors.Is(err, errNotProcessing) {
			t.Fatalf("suggestion lost its session binding during the fresh read, error = %v", err)
		}
		if currentSuggestion(it) != nil {
			t.Fatalf("session A published against session B's item: %+v", currentSuggestion(it))
		}
	})
	t.Run("handback confirmation is not exempt", func(t *testing.T) {
		h := agentHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		it := disabledTwinItem(h, "policy-handback", HandlingDone, AgentSkipped, "parent-session")
		it.Source = "lark"
		it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
		g := &InboxSuggestion{Version: 1, Kind: SuggestReply, Text: "Synthetic handback", Handback: true, At: time.Now(), Conversation: &SessionRef{MachineID: "server", SessionID: "handoff-session"}}
		snapshotReplyPolicy(g, it, replyPolicyTestConfig())
		it.Suggestion = g
		it.ReplyCategory, it.ReplyMode = "no_reply", "no_reply"
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("revoked handback confirmation error = %v", err)
		}
		if currentSuggestion(it) != g {
			t.Fatal("revoked handback card was consumed")
		}
		if err := replyPolicySendGate(it, g, replyPolicyTestConfig()); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("revoked handback final send gate = %v", err)
		}
	})
	t.Run("stale disabled cache does not skip an edited handback veto", func(t *testing.T) {
		h := agentHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		h.inbox.cfgMu.Lock()
		h.inbox.cfg = assistantwire.InboxConfig{}
		h.inbox.cfgAt = time.Now()
		h.inbox.cfgMu.Unlock()
		it := disabledTwinItem(h, "policy-handback-edit", HandlingDone, AgentSkipped, "parent-session")
		it.Source = "lark"
		it.Context = "Synthetic original handback conversation"
		it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
		g := &InboxSuggestion{Version: 1, Kind: SuggestReply, Text: "Synthetic handback", Handback: true, At: time.Now(), Conversation: &SessionRef{MachineID: "server", SessionID: "handoff-session"}}
		snapshotReplyPolicy(g, it, replyPolicyTestConfig())
		it.Suggestion = g
		h.in.model = fakeModel(t, `{"category":"approval_permission"}`)
		edited := "I approve the synthetic restricted handback"
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm, Text: &edited}, true); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("stale-cache edited handback error = %v", err)
		}
		if currentSuggestion(it) != g {
			t.Fatal("restricted edited handback card was consumed")
		}
	})
	t.Run("policy changes during an edited handback veto", func(t *testing.T) {
		h := agentHub(t)
		oldCfg := assistantwire.InboxConfig{ReplyPolicy: replyPolicyTestConfig()}
		newCfg := assistantwire.InboxConfig{ReplyPolicy: replyPolicyTestConfig()}
		newCfg.ReplyPolicy.Revision = "test-v2"
		h.inbox.cfgMu.Lock()
		h.inbox.cfg = oldCfg
		h.inbox.cfgAt = time.Now()
		h.inbox.cfgMu.Unlock()
		h.in.mu.Lock()
		h.in.configQueue = []assistantwire.InboxConfig{oldCfg, newCfg}
		h.in.mu.Unlock()
		it := disabledTwinItem(h, "policy-handback-revision", HandlingDone, AgentSkipped, "parent-session")
		it.Source = "lark"
		it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
		g := &InboxSuggestion{Version: 1, Kind: SuggestReply, Text: "Synthetic handback", Handback: true, At: time.Now(), Conversation: &SessionRef{MachineID: "server", SessionID: "handoff-session"}}
		snapshotReplyPolicy(g, it, replyPolicyTestConfig())
		it.Suggestion = g
		h.in.model = fakeModel(t, `{"category":"ordinary_coordination"}`)
		edited := "An ordinary synthetic handback edit"
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm, Text: &edited}, true); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("policy revision changed during handback veto, error = %v", err)
		}
		if currentSuggestion(it) != g {
			t.Fatal("handback card was consumed across a policy revision")
		}
	})
}

func TestInboxReplySuggestionGate(t *testing.T) {
	for _, tc := range []struct {
		category, mode string
		allowed        bool
	}{
		{"personal_transaction", "owner_only", false}, {"security_compliance", "owner_only", false},
		{"approval_permission", "owner_only", false}, {"decision_delegation", "owner_only", false},
		{"technical_qa", "bot", false}, {"research", "bot", false}, {"alert_triage", "bot", false},
		{"ppe_deploy", "bot", false}, {"mr_precheck", "bot", false}, {"unknown", "owner_only", false},
		{"ordinary_coordination", "owner_draft", true},
	} {
		t.Run(tc.category, func(t *testing.T) {
			h, _, it := handledHub(t)
			h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
			it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = tc.category, tc.mode, replyPolicyContentVersion(it), "test-v1"
			_, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic coordination reply"}`)
			if tc.allowed && err != nil {
				t.Fatalf("owner draft refused: %v", err)
			}
			if !tc.allowed && err == nil {
				t.Fatal("non-owner-draft reply was accepted")
			}
			if !tc.allowed && currentSuggestion(it) != nil {
				t.Fatalf("refused suggestion was stored: %+v", currentSuggestion(it))
			}
			if tc.allowed {
				g := currentSuggestion(it)
				if g == nil || g.ReplyCategory != tc.category || g.ReplyMode != "owner_draft" || g.ReplyClassifiedOf != replyPolicyContentVersion(it) || g.ReplyPolicyRev != "test-v1" {
					t.Fatalf("server did not snapshot trusted policy fields: %+v", g)
				}
			}
		})
	}
}

func TestInboxReplyPolicyRefreshesUnattemptedCardSnapshot(t *testing.T) {
	h, _, it := handledHub(t)
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
	it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Same synthetic coordination reply"}`); err != nil {
		t.Fatal(err)
	}
	it.ReplyPolicyRev = "test-v2"
	h.setConfig(func(c *assistantwire.InboxConfig) {
		c.ReplyPolicy = replyPolicyTestConfig()
		c.ReplyPolicy.Revision = "test-v2"
	})
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Same synthetic coordination reply"}`); err != nil {
		t.Fatal(err)
	}
	g := currentSuggestion(it)
	if g == nil || g.Version != 2 || g.ReplyPolicyRev != "test-v2" || it.SuggestSeq != 2 {
		t.Fatalf("identical text kept a stale unattempted card: suggestion=%+v seq=%d", g, it.SuggestSeq)
	}
}

func TestInboxReplyPolicyAgentConfirmationGate(t *testing.T) {
	for _, tc := range []struct {
		name       string
		legacy     bool
		restricted bool
	}{
		{name: "legacy Agent suggestion", legacy: true},
		{name: "category became restricted", restricted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := agentHub(t)
			h.setConfig(func(c *assistantwire.InboxConfig) {
				c.Agent.Enabled = true
				c.ReplyPolicy = replyPolicyTestConfig()
			})
			it := disabledTwinItem(h, "policy-agent", HandlingDone, AgentSkipped, "parent-session")
			it.Source = "lark"
			it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
			g := &InboxSuggestion{Version: 1, Kind: SuggestAgent, Text: "Inspect the synthetic request", At: time.Now(), MachineID: "server"}
			if !tc.legacy {
				snapshotReplyPolicy(g, it, replyPolicyTestConfig())
			}
			it.Suggestion = g
			if tc.restricted {
				it.ReplyCategory, it.ReplyMode = "approval_permission", "owner_only"
			}
			if err := replyPolicyActionGate(it, g, h.inboxConfig(context.Background()).ReplyPolicy); !errors.Is(err, errReplyPolicyDenied) {
				t.Fatalf("test setup did not restrict the Agent suggestion: %v, source=%q policy=%+v", err, it.Source, h.inboxConfig(context.Background()).ReplyPolicy)
			}
			if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); !errors.Is(err, errReplyPolicyDenied) {
				t.Fatalf("restricted Agent confirmation error = %v", err)
			}
			if got := currentSuggestion(it); got != g {
				t.Fatalf("denied Agent confirmation consumed its draft: %+v", got)
			}
			h.f.mu.Lock()
			starts := len(h.f.starts)
			h.f.mu.Unlock()
			if starts != 0 {
				t.Fatalf("restricted Agent confirmation started %d Agents", starts)
			}
		})
	}
	t.Run("handed-off Agent rechecks fresh policy", func(t *testing.T) {
		h := agentHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) {
			c.Agent.Enabled = true
			c.ReplyPolicy = replyPolicyTestConfig()
		})
		it := disabledTwinItem(h, "policy-handed-agent", HandlingDone, AgentSkipped, "parent-session")
		it.Source = "lark"
		it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
		g := &InboxSuggestion{Version: 1, Kind: SuggestAgent, Text: "Inspect the synthetic request", At: time.Now(), MachineID: "server"}
		snapshotReplyPolicy(g, it, replyPolicyTestConfig())
		it.ReplyCategory, it.ReplyMode = "approval_permission", "owner_only"
		_, err := h.confirmHandedAgent(context.Background(), it, g, g.Text, false, SessionRef{MachineID: "server", SessionID: "handoff-session"}, "Synthetic")
		if !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("handed-off restricted Agent error = %v", err)
		}
		h.f.mu.Lock()
		starts := len(h.f.starts)
		h.f.mu.Unlock()
		if starts != 0 {
			t.Fatalf("handed-off restricted Agent started %d Agents", starts)
		}
	})
}

func TestInboxReplySuggestionSendGate(t *testing.T) {
	newDraft := func(t *testing.T) (*inboxHub, *extensionLark, *InboxItem) {
		h, lark, it := handledHub(t)
		h.in.model = fakeModel(t, `{"category":"ordinary_coordination"}`)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
		if _, err := suggestTool(t, h, `{"kind":"reply","text":"The original synthetic reply"}`); err != nil {
			t.Fatal(err)
		}
		return h, lark, it
	}
	t.Run("ordinary draft stores proposal actual and category", func(t *testing.T) {
		h, lark, it := newDraft(t)
		edited := "The owner-edited synthetic reply"
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm, Text: &edited}, true); err != nil {
			t.Fatal(err)
		}
		if len(lark.got) != 1 || lark.got[0].Text != edited {
			t.Fatalf("confirmed sender calls = %+v", lark.got)
		}
		a := suggestionAttempt(it, 1)
		if a == nil || a.Status != suggestionSent || a.Text != edited || a.Proposal == nil || a.Proposal.Text != "The original synthetic reply" ||
			a.Proposal.ReplyCategory != "ordinary_coordination" || a.Proposal.ReplyMode != "owner_draft" || a.Proposal.ReplyPolicyRev != "test-v1" {
			t.Fatalf("attempt snapshot = %+v", a)
		}
	})
	for _, tc := range []struct {
		name   string
		mutate func(*InboxItem)
	}{
		{"legacy draft has no category", func(it *InboxItem) {
			it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "", "", "", ""
			it.Suggestion.ReplyCategory, it.Suggestion.ReplyMode, it.Suggestion.ReplyClassifiedOf, it.Suggestion.ReplyPolicyRev = "", "", "", ""
		}},
		{"policy revision changed", func(it *InboxItem) { it.ReplyPolicyRev = "test-v2" }},
		{"content changed", func(it *InboxItem) { it.Last = "newer-content" }},
		{"route is no longer owner draft", func(it *InboxItem) { it.ReplyCategory, it.ReplyMode = "technical_qa", "bot" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, lark, it := newDraft(t)
			tc.mutate(it)
			if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
				t.Fatal("stale or forbidden reply was confirmed")
			}
			if len(lark.got) != 0 {
				t.Fatalf("refused confirmation called sender: %+v", lark.got)
			}
		})
	}
	t.Run("policy changes while the reservation is saved", func(t *testing.T) {
		h, lark, it := newDraft(t)
		st := &recoveryBeforeWriteStore{inboxMemStore: h.st, before: func() {
			h.inbox.mu.Lock()
			defer h.inbox.mu.Unlock()
			it.ReplyCategory, it.ReplyMode = "approval_permission", "owner_only"
		}}
		h.store = func() Store { return st }
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
			t.Fatal("reply was confirmed after its policy became forbidden during persistence")
		}
		if len(lark.got) != 0 {
			t.Fatalf("pre-send recheck missed the policy change and called sender: %+v", lark.got)
		}
	})
	t.Run("policy revision changes while the reservation is saved", func(t *testing.T) {
		h, lark, it := newDraft(t)
		st := &recoveryBeforeWriteStore{inboxMemStore: h.st, before: func() {
			// Deliberately keep the manager's five-second cache alive. Only the fresh outbound read may observe this.
			h.in.mu.Lock()
			cfg := replyPolicyTestConfig()
			cfg.Revision = "test-v2"
			h.in.cfg.ReplyPolicy = cfg
			h.in.mu.Unlock()
		}}
		h.store = func() Store { return st }
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
			t.Fatal("reply was confirmed against a superseded policy revision during persistence")
		}
		if len(lark.got) != 0 {
			t.Fatalf("pre-send recheck missed the revision change and called sender: %+v", lark.got)
		}
	})
	t.Run("fresh configuration failure is known not sent", func(t *testing.T) {
		h, lark, it := newDraft(t)
		st := &recoveryBeforeWriteStore{inboxMemStore: h.st, before: func() {
			h.in.mu.Lock()
			h.in.configErr = errors.New("synthetic configuration outage")
			h.in.mu.Unlock()
		}}
		h.store = func() Store { return st }
		_, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true)
		var rpcErr *rpc.Error
		if !errors.As(err, &rpcErr) || rpcErr.Code != assistantwire.CodeInboxSendFailed {
			t.Fatalf("fresh configuration failure error = %v", err)
		}
		if len(lark.got) != 0 {
			t.Fatalf("configuration outage called sender: %+v", lark.got)
		}
		a := suggestionAttempt(it, 1)
		if a == nil || a.Status != suggestionFailed || !a.Retryable || a.Code != assistantwire.CodeInboxSendFailed {
			t.Fatalf("known-none configuration failure was not retryable: %+v", a)
		}
	})
	t.Run("empty or null configuration is not authorization", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			raw  json.RawMessage
		}{
			{name: "empty", raw: json.RawMessage{}},
			{name: "null", raw: json.RawMessage("null")},
		} {
			t.Run(tc.name, func(t *testing.T) {
				h, lark, it := newDraft(t)
				st := &recoveryBeforeWriteStore{inboxMemStore: h.st, before: func() {
					h.in.mu.Lock()
					h.in.configRaw = tc.raw
					h.in.mu.Unlock()
				}}
				h.store = func() Store { return st }
				_, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true)
				var rpcErr *rpc.Error
				if !errors.As(err, &rpcErr) || rpcErr.Code != assistantwire.CodeInboxSendFailed {
					t.Fatalf("absent configuration error = %v", err)
				}
				if len(lark.got) != 0 {
					t.Fatalf("absent configuration called sender: %+v", lark.got)
				}
			})
		}
	})
	t.Run("inert disabled configuration changes preserve legacy sends", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			before assistantwire.InboxConfig
			after  assistantwire.InboxConfig
		}{
			{name: "nil to disabled", after: assistantwire.InboxConfig{ReplyPolicy: &assistantwire.InboxReplyPolicyConfig{Enabled: false}}},
			{name: "disabled revision", before: assistantwire.InboxConfig{ReplyPolicy: &assistantwire.InboxReplyPolicyConfig{Revision: "unused-v1"}},
				after: assistantwire.InboxConfig{ReplyPolicy: &assistantwire.InboxReplyPolicyConfig{Revision: "unused-v2"}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				h, lark, it := handledHub(t)
				if _, err := suggestTool(t, h, `{"kind":"reply","text":"Legacy synthetic reply"}`); err != nil {
					t.Fatal(err)
				}
				st := &recoveryBeforeWriteStore{inboxMemStore: h.st, before: func() {
					h.in.mu.Lock()
					h.in.configQueue = []assistantwire.InboxConfig{tc.before, tc.after}
					h.in.mu.Unlock()
				}}
				h.store = func() Store { return st }
				if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err != nil {
					t.Fatalf("inert disabled configuration rejected a legacy send: %v", err)
				}
				if len(lark.got) != 1 {
					t.Fatalf("legacy sender calls = %+v", lark.got)
				}
			})
		}
	})
	t.Run("policy changes between the two final fresh reads", func(t *testing.T) {
		h, lark, it := newDraft(t)
		st := &recoveryBeforeWriteStore{inboxMemStore: h.st, before: func() {
			oldCfg := assistantwire.InboxConfig{ReplyPolicy: replyPolicyTestConfig()}
			newCfg := assistantwire.InboxConfig{ReplyPolicy: replyPolicyTestConfig()}
			newCfg.ReplyPolicy.Revision = "test-v2"
			h.in.mu.Lock()
			h.in.configQueue = []assistantwire.InboxConfig{oldCfg, newCfg}
			h.in.mu.Unlock()
		}}
		h.store = func() Store { return st }
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
			t.Fatal("reply used the pre-veto policy after the final fresh read changed")
		}
		if len(lark.got) != 0 {
			t.Fatalf("second fresh-read race called sender: %+v", lark.got)
		}
	})
	t.Run("a stale edit veto does not reject a newly legacy send", func(t *testing.T) {
		h, lark, it := newDraft(t)
		h.in.mu.Lock()
		h.in.cfg.ReplyPolicy = nil
		h.in.mu.Unlock()
		h.in.model = fakeModel(t, `{"category":"approval_permission"}`)
		edited := "I approve the synthetic legacy request"
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm, Text: &edited}, true); err != nil {
			t.Fatalf("stale cached veto rejected a newly disabled policy: %v", err)
		}
		if len(lark.got) != 1 {
			t.Fatalf("newly legacy sender calls = %+v", lark.got)
		}
	})
	t.Run("policy snapshots do not change uncertain send identity", func(t *testing.T) {
		h, lark, it := newDraft(t)
		g := currentSuggestion(it)
		old := copySuggestion(g)
		old.ReplyCategory, old.ReplyMode, old.ReplyPolicyRev, old.ReplyReviewed = "acknowledgement", "owner_draft", "test-v0", true
		proof := &InboxSuggestionAttempt{Version: 0, Status: suggestionUnknown, Text: g.Text, ClientID: namedClientID("same synthetic send"),
			Proposal: old, Target: suggestionReplyTarget(it)}
		keepSuggestionAttempt(it, proof)
		if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err == nil {
			t.Fatal("an uncertain equivalent reply was allowed to send again")
		}
		if len(lark.got) != 0 {
			t.Fatalf("policy snapshot change defeated send-once protection: %+v", lark.got)
		}
		a := suggestionAttempt(it, 1)
		if a == nil || a.ClientID != proof.ClientID || a.Status != suggestionUnknown {
			t.Fatalf("uncertain receipt was not inherited: %+v", a)
		}
	})
	t.Run("owner edit is classified with its original conversation", func(t *testing.T) {
		h, lark, it := newDraft(t)
		it.Context = "Synthetic scheduling context with a conditional request for access: ORIGINAL-AUTHORITY-MARKER"
		model, calls := captureModel(t, `{"category":"approval_permission"}`)
		h.in.model = model
		edited := "Yes, go ahead: EDIT-ASSENT-MARKER"
		_, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm, Text: &edited}, true)
		if !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("contextual owner edit error = %v", err)
		}
		call := onlyCall(t, calls())
		if !strings.Contains(call.User, "ORIGINAL-AUTHORITY-MARKER") || !strings.Contains(call.User, "EDIT-ASSENT-MARKER") {
			t.Fatalf("edit veto lost its context or replacement text: %q", call.User)
		}
		if len(lark.got) != 0 {
			t.Fatalf("contextual approval edit called sender: %+v", lark.got)
		}
	})
	t.Run("owner edit introduces approval", func(t *testing.T) {
		h, lark, it := newDraft(t)
		h.in.model = fakeModel(t, `{"category":"approval_permission"}`)
		edited := "I approve access and authorize the synthetic request"
		_, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm, Text: &edited}, true)
		if err == nil || !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("edited approval confirmation error = %v", err)
		}
		if len(lark.got) != 0 {
			t.Fatalf("edited approval called sender: %+v", lark.got)
		}
	})
}

func TestInboxReplyPolicyOnDemandDraftGate(t *testing.T) {
	t.Run("nil policy keeps the draft transient", func(t *testing.T) {
		h, _, it := handledHub(t)
		h.in.model = fakeModel(t, "Legacy transient on-demand draft")
		if _, err := h.draftReply(t.Context(), it.ID, ""); err != nil {
			t.Fatalf("legacy on-demand draft failed: %v", err)
		}
		if currentSuggestion(it) != nil || mirroredLegacyReply(it) != nil {
			t.Fatalf("nil-policy on-demand draft changed legacy state: %+v", currentSuggestion(it))
		}
	})
	t.Run("owner-only draft is refused before the model", func(t *testing.T) {
		h, lark, it := handledHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		it.ReplyCategory, it.ReplyMode = "approval_permission", "owner_only"
		it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
		model, calls := captureModel(t, "Synthetic owner-voice approval")
		h.in.model = model
		code, _ := inboxCall(t, h.routes(), "POST", "/api/v1/inbox/"+it.ID+"/draft", `{}`)
		if code != http.StatusForbidden {
			t.Fatalf("owner-only on-demand draft status = %d, want %d", code, http.StatusForbidden)
		}
		if got := len(calls()); got != 0 {
			t.Fatalf("owner-only on-demand draft made %d model calls", got)
		}
		if currentSuggestion(it) != nil || mirroredLegacyReply(it) != nil {
			t.Fatalf("owner-only on-demand draft left a sendable suggestion: %+v", currentSuggestion(it))
		}
		if len(lark.got) != 0 {
			t.Fatalf("owner-only on-demand draft called sender: %+v", lark.got)
		}
	})
	t.Run("policy revision changes while the model runs", func(t *testing.T) {
		h, lark, it := handledHub(t)
		it.ReplyCategory, it.ReplyMode = "ordinary_coordination", "owner_draft"
		it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
		model, calls := captureModel(t, "Synthetic draft from the revoked policy revision")
		h.in.model = model
		next := replyPolicyTestConfig()
		next.Revision = "test-v2"
		h.in.mu.Lock()
		h.in.configQueue = []assistantwire.InboxConfig{{ReplyPolicy: replyPolicyTestConfig()}, {ReplyPolicy: next}}
		h.in.mu.Unlock()
		if _, err := h.draftReply(t.Context(), it.ID, ""); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("draft from the old policy revision error = %v", err)
		}
		if got := len(calls()); got != 1 {
			t.Fatalf("revision-race draft made %d model calls, want 1", got)
		}
		if currentSuggestion(it) != nil || mirroredLegacyReply(it) != nil {
			t.Fatalf("revoked policy revision published a sendable draft: %+v", currentSuggestion(it))
		}
		if len(lark.got) != 0 {
			t.Fatalf("revoked policy revision called sender: %+v", lark.got)
		}
	})
	t.Run("another device publishes while the model runs", func(t *testing.T) {
		h, lark, it := handledHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		it.ReplyCategory, it.ReplyMode = "ordinary_coordination", "owner_draft"
		it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
		var changed atomic.Bool
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if changed.CompareAndSwap(false, true) {
				h.inbox.mu.Lock()
				now := time.Now().UTC()
				g := &InboxSuggestion{Version: 1, Kind: SuggestReply, Text: "Newer draft from another device", At: now, SourceOf: suggestionReplyTarget(it)}
				snapshotReplyPolicy(g, it, replyPolicyTestConfig())
				it.SuggestSeq, it.Suggestion = 1, g
				h.inbox.setHandling(it, HandlingDone, "", now)
				h.inbox.mu.Unlock()
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]any{"type": "text", "text": "Stale on-demand draft"}}})
		}))
		t.Cleanup(srv.Close)
		h.in.model = modelAt(t, srv.URL)
		if _, err := h.draftReply(t.Context(), it.ID, ""); !errors.Is(err, errSuggestionStale) {
			t.Fatalf("on-demand draft replaced a newer device draft: %v", err)
		}
		if !changed.Load() {
			t.Fatal("test model did not publish the other device's draft")
		}
		if g := currentSuggestion(it); g == nil || g.Text != "Newer draft from another device" || g.Version != 1 {
			t.Fatalf("on-demand draft overwrote the newer device state: %+v", g)
		}
		if len(lark.got) != 0 {
			t.Fatalf("concurrent on-demand draft called sender: %+v", lark.got)
		}
	})
	for _, tc := range []struct {
		name     string
		fromCard bool
	}{
		{name: "direct reply"},
		{name: "workbench card reply", fromCard: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, lark, it := handledHub(t)
			h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
			it.ReplyCategory, it.ReplyMode = "ordinary_coordination", "owner_draft"
			it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
			h.in.model = fakeModel(t, "Synthetic on-demand coordination draft")
			text, err := h.draftReply(t.Context(), it.ID, "")
			if err != nil {
				t.Fatalf("ordinary on-demand draft failed: %v", err)
			}
			g := mirroredLegacyReply(it)
			if g == nil || g.Text != text || !replyPolicySnapshotCurrent(it, replyPolicyTestConfig()) {
				t.Fatalf("on-demand draft has no current policy-bound mirror: agent=%+v suggestion=%+v", it.Agent, currentSuggestion(it))
			}
			h.in.model = fakeModel(t, `{"category":"approval_permission"}`)
			edited := "I approve the synthetic on-demand permission request"
			if tc.fromCard {
				_, err = h.replyInboxFrom(t.Context(), it.ID, edited, "synthetic-workbench-send", true)
			} else {
				_, err = h.replyInboxChecked(t.Context(), it.ID, edited, "", false, nil, nil)
			}
			if !errors.Is(err, errReplyPolicyDenied) {
				t.Fatalf("edited on-demand draft error = %v", err)
			}
			if len(lark.got) != 0 {
				t.Fatalf("edited on-demand draft reached sender through the direct-owner path: %+v", lark.got)
			}
		})
	}
	t.Run("unmodified on-demand draft sends through its mirror", func(t *testing.T) {
		h, lark, it := handledHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		it.ReplyCategory, it.ReplyMode = "ordinary_coordination", "owner_draft"
		it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
		h.in.model = fakeModel(t, "Synthetic confirmed on-demand draft")
		text, err := h.draftReply(t.Context(), it.ID, "")
		if err != nil {
			t.Fatalf("ordinary on-demand draft failed: %v", err)
		}
		if mirroredLegacyReply(it) == nil {
			t.Fatal("on-demand draft did not use the old-client mirror")
		}
		if _, err := h.replyInboxChecked(t.Context(), it.ID, text, "", false, nil, nil); err != nil {
			t.Fatalf("confirmed on-demand draft failed: %v", err)
		}
		if len(lark.got) != 1 || lark.got[0].Text != text {
			t.Fatalf("confirmed on-demand sender calls = %+v", lark.got)
		}
	})
}

func TestInboxReplyPolicyMergedContextInvalidatesDraft(t *testing.T) {
	h, lark, it := handledHub(t)
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
	// A full survivor used to clip the entire appended member away, leaving both the text and its authorization hash unchanged.
	it.Context = strings.Repeat("s", mergeCtxMax)
	it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic scheduling assent"}`); err != nil {
		t.Fatal(err)
	}
	oldLast := it.Last
	member := actionItem("lark:restricted-member", it.UpdatedAt.Add(-time.Second))
	member.Source = "lark"
	member.Context = "Synthetic request to approve access and authorize a restricted permission change"
	h.inbox.mu.Lock()
	h.inbox.items[member.ID] = member
	survivor, _ := h.inbox.merge([]*InboxItem{it, member}, time.Now())
	h.inbox.mu.Unlock()
	if survivor != it || it.Last != oldLast {
		t.Fatalf("test merge unexpectedly changed the survivor or Last: %+v", it)
	}
	if _, err := h.actOnSuggestion(context.Background(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); !errors.Is(err, errReplyPolicyDenied) {
		t.Fatalf("clipped restricted member retained its old authorization: %v", err)
	}
	if len(lark.got) != 0 {
		t.Fatalf("merged restricted context called sender: %+v", lark.got)
	}
	model, calls := captureModel(t, `{"category":"ordinary_coordination"}`)
	h.in.model = model
	cfg := replyPolicyTestConfig()
	for _, revision := range []string{"test-v1", "test-v2"} {
		cfg.Revision = revision
		h.classifyInboxReplyPolicies(t.Context(), cfg)
		if it.ReplyCategory != "unknown" || it.ReplyMode != "owner_only" {
			t.Fatalf("clipped merge was reauthorized under %s: category=%q mode=%q", revision, it.ReplyCategory, it.ReplyMode)
		}
	}
	if got := len(calls()); got != 0 {
		t.Fatalf("clipped merge made %d classification calls instead of failing closed", got)
	}
}

func TestInboxReplyPolicyCrossSourceMergeKeepsMemberScope(t *testing.T) {
	newMerged := func(t *testing.T) (*inboxHub, *extensionLark, *InboxItem) {
		h, lark, it := handledHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) {
			c.ReplyPolicy = replyPolicyTestConfig()
			c.ReplyPolicy.Sources = []string{"mail"}
		})
		// The survivor itself is outside the configured source list, so this suggestion has no policy snapshot.
		if _, err := suggestTool(t, h, `{"kind":"reply","text":"Synthetic cross-source owner draft"}`); err != nil {
			t.Fatal(err)
		}
		member := actionItem("mail:restricted-member", it.UpdatedAt.Add(-time.Second))
		member.Source = "mail"
		member.Context = "Synthetic request to approve access and authorize a restricted permission change"
		h.inbox.mu.Lock()
		h.inbox.items[member.ID] = member
		survivor, _ := h.inbox.merge([]*InboxItem{it, member}, time.Now())
		h.inbox.mu.Unlock()
		if survivor != it || survivor.Source != "lark" || len(survivor.MergedFrom) != 1 || survivor.MergedFrom[0].Source != "mail" {
			t.Fatalf("test did not build a cross-source merge: %+v", survivor)
		}
		return h, lark, it
	}
	t.Run("existing draft cannot be confirmed", func(t *testing.T) {
		h, lark, it := newMerged(t)
		if _, err := h.actOnSuggestion(t.Context(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("cross-source merged draft confirmation error = %v", err)
		}
		if len(lark.got) != 0 {
			t.Fatalf("restricted member escaped its source policy through a merge: %+v", lark.got)
		}
	})
	t.Run("new draft cannot be published without classifying the member", func(t *testing.T) {
		h, lark, _ := newMerged(t)
		if _, err := suggestTool(t, h, `{"kind":"reply","text":"Replacement cross-source owner draft"}`); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("cross-source replacement suggestion error = %v", err)
		}
		if len(lark.got) != 0 {
			t.Fatalf("cross-source replacement suggestion called sender: %+v", lark.got)
		}
	})
	t.Run("undo retains the source boundary for retained member text", func(t *testing.T) {
		h, lark, it := newMerged(t)
		h.inbox.mu.Lock()
		h.inbox.unmerge(it, time.Now())
		h.inbox.mu.Unlock()
		if !strings.Contains(it.Context, "restricted permission change") {
			t.Fatal("test undo unexpectedly removed the member's retained context")
		}
		if _, err := suggestTool(t, h, `{"kind":"reply","text":"Post-undo cross-source owner draft"}`); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("post-undo retained member text escaped its source policy: %v", err)
		}
		if len(lark.got) != 0 {
			t.Fatalf("post-undo cross-source suggestion called sender: %+v", lark.got)
		}
	})
}

func TestInboxReplyPolicyClassificationRejectsContextChangeDuringModelCall(t *testing.T) {
	h := agentHub(t)
	it := actionItem("lark:classification-context-race", time.Now())
	it.Source = "lark"
	it.Agent = &InboxAgent{State: AgentQueued}
	h.inbox.items[it.ID] = it
	var changed atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if changed.CompareAndSwap(false, true) {
			h.inbox.mu.Lock()
			it.Context += "\n\nSynthetic restricted context arrived during classification"
			h.inbox.mu.Unlock()
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]any{"type": "text", "text": `{"category":"ordinary_coordination"}`}}})
	}))
	t.Cleanup(srv.Close)
	h.in.model = modelAt(t, srv.URL)
	cfg := replyPolicyTestConfig()
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = cfg })
	h.classifyInboxReplyPolicies(context.Background(), cfg)
	if !changed.Load() {
		t.Fatal("test model did not change the context")
	}
	if it.ReplyClassifiedOf != "" || it.ReplyPolicyRev != "" || it.ReplyCategory != "" || it.ReplyMode != "" {
		t.Fatalf("classification from the old context was committed after a merge: %+v", it)
	}
	if replyPolicyClassificationCurrent(it, cfg) {
		t.Fatal("changed context was treated as currently classified")
	}
}

func TestInboxReplyPolicyLegacyMirroredDraftGate(t *testing.T) {
	newDraft := func(t *testing.T, enabled bool) (*inboxHub, *extensionLark, *InboxItem) {
		h, lark, it := handledHub(t)
		if enabled {
			h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
			it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
		}
		if _, err := suggestTool(t, h, `{"kind":"reply","text":"Mirrored synthetic AI draft"}`); err != nil {
			t.Fatal(err)
		}
		if mirroredLegacyReply(it) == nil {
			t.Fatalf("test setup did not expose the old-client mirror: agent=%+v suggestion=%+v", it.Agent, currentSuggestion(it))
		}
		return h, lark, it
	}
	t.Run("pre-policy mirrored draft cannot send after enablement", func(t *testing.T) {
		h, lark, it := newDraft(t, false)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		if _, err := h.replyInboxChecked(context.Background(), it.ID, currentSuggestion(it).Text, "", false, nil, nil); err == nil {
			t.Fatal("old client sent a pre-policy AI draft without a policy snapshot")
		}
		if len(lark.got) != 0 {
			t.Fatalf("pre-policy old-client draft called sender: %+v", lark.got)
		}
	})
	for _, tc := range []struct {
		name   string
		mutate func(*inboxHub, *InboxItem)
	}{
		{name: "route revoked", mutate: func(_ *inboxHub, it *InboxItem) {
			it.ReplyCategory, it.ReplyMode = "approval_permission", "owner_only"
		}},
		{name: "revision revoked", mutate: func(h *inboxHub, _ *InboxItem) {
			h.setConfig(func(c *assistantwire.InboxConfig) {
				c.ReplyPolicy = replyPolicyTestConfig()
				c.ReplyPolicy.Revision = "test-v2"
			})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, lark, it := newDraft(t, true)
			tc.mutate(h, it)
			if _, err := h.replyInboxChecked(context.Background(), it.ID, currentSuggestion(it).Text, "", false, nil, nil); err == nil {
				t.Fatal("old client sent after its policy authorization was revoked")
			}
			if len(lark.got) != 0 {
				t.Fatalf("revoked old-client draft called sender: %+v", lark.got)
			}
		})
	}
	t.Run("edited approval is vetoed", func(t *testing.T) {
		h, lark, it := newDraft(t, true)
		h.in.model = fakeModel(t, `{"category":"approval_permission"}`)
		if _, err := h.replyInboxChecked(context.Background(), it.ID, "I approve the synthetic restricted request", "", false, nil, nil); !errors.Is(err, errReplyPolicyDenied) {
			t.Fatalf("old-client edited approval error = %v", err)
		}
		if len(lark.got) != 0 {
			t.Fatalf("old-client edited approval called sender: %+v", lark.got)
		}
	})
	t.Run("authorized owner draft sends", func(t *testing.T) {
		h, lark, it := newDraft(t, true)
		if _, err := h.replyInboxChecked(context.Background(), it.ID, currentSuggestion(it).Text, "", false, nil, nil); err != nil {
			t.Fatalf("authorized old-client owner draft failed: %v", err)
		}
		if len(lark.got) != 1 {
			t.Fatalf("authorized old-client sender calls = %+v", lark.got)
		}
	})
	t.Run("direct owner-authored reply stays legacy", func(t *testing.T) {
		h, lark, it := handledHub(t)
		h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
		if mirroredLegacyReply(it) != nil {
			t.Fatal("test setup unexpectedly has an AI mirror")
		}
		if _, err := h.replyInboxChecked(context.Background(), it.ID, "Direct owner-authored synthetic reply", "", false, nil, nil); err != nil {
			t.Fatalf("direct owner-authored reply was gated: %v", err)
		}
		if len(lark.got) != 1 {
			t.Fatalf("direct owner-authored sender calls = %+v", lark.got)
		}
	})
	for _, tc := range []struct {
		name   string
		policy *assistantwire.InboxReplyPolicyConfig
	}{
		{name: "nil policy"},
		{name: "disabled policy", policy: &assistantwire.InboxReplyPolicyConfig{Enabled: false, Revision: "unused-v1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, lark, it := newDraft(t, false)
			h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = tc.policy })
			if _, err := h.replyInboxChecked(context.Background(), it.ID, currentSuggestion(it).Text, "", false, nil, nil); err != nil {
				t.Fatalf("legacy mirrored draft failed with %s: %v", tc.name, err)
			}
			if len(lark.got) != 1 {
				t.Fatalf("legacy mirrored sender calls = %+v", lark.got)
			}
		})
	}
}

func TestInboxReplyPolicyLegacyDraftRejectedWhileSendWaits(t *testing.T) {
	h, lark, it := handledHub(t)
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
	it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Mirrored draft rejected on another device"}`); err != nil {
		t.Fatal(err)
	}
	text := currentSuggestion(it).Text
	entered := make(chan struct{})
	once := &inboxReply{beforeLock: func() { close(entered) }}
	once.mu.Lock()
	unlocked := false
	defer func() {
		if !unlocked {
			once.mu.Unlock()
		}
	}()
	h.inbox.replies.Store(it.ID, once)
	result := make(chan error, 1)
	go func() {
		_, err := h.replyInboxChecked(t.Context(), it.ID, text, "", false, nil, nil)
		result <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("old-client send did not reach the per-item send lock")
	}
	// Rejection removes the mirror while the old-client request is queued behind the send mutex.
	if _, err := h.actOnSuggestion(t.Context(), it.ID, suggestionReq{Version: 1, Action: suggestReject}, true); err != nil {
		t.Fatalf("rejecting the mirrored draft failed: %v", err)
	}
	once.mu.Unlock()
	unlocked = true
	select {
	case err := <-result:
		if err == nil {
			t.Fatal("queued old-client request sent a draft rejected while it waited")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("queued old-client send did not finish")
	}
	if len(lark.got) != 0 {
		t.Fatalf("rejected old-client draft reached sender after the mirror disappeared: %+v", lark.got)
	}
}

func TestInboxReplyPolicyDeniedEditSurvivesConcurrentDisable(t *testing.T) {
	h, lark, it := handledHub(t)
	h.in.model = fakeModel(t, `{"category":"approval_permission"}`)
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
	it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "ordinary_coordination", "owner_draft", replyPolicyContentVersion(it), "test-v1"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Original synthetic reply"}`); err != nil {
		t.Fatal(err)
	}
	// Prime the display cache, then make the veto's fresh read enabled and the second fresh read disabled. Calling the
	// complete action proves that the post-veto read cannot discard an established denial and reach the real send path.
	h.inboxConfig(t.Context())
	h.in.mu.Lock()
	h.in.configQueue = []assistantwire.InboxConfig{{ReplyPolicy: replyPolicyTestConfig()}, {}}
	h.in.mu.Unlock()
	edited := "I approve the synthetic permission request"
	_, err := h.actOnSuggestion(t.Context(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm, Text: &edited}, true)
	if !errors.Is(err, errReplyPolicyDenied) {
		t.Fatalf("concurrent disable cleared an established edit veto: %v", err)
	}
	if len(lark.got) != 0 {
		t.Fatalf("rejected owner edit reached the sender through confirmation: %+v", lark.got)
	}
}

func TestInboxReplyPolicyConfigFailureDoesNotBlockOtherSuggestions(t *testing.T) {
	for _, kind := range []string{SuggestConclusion, SuggestTodo} {
		t.Run(kind, func(t *testing.T) {
			h, _, it := handledHub(t)
			it.Suggestion = &InboxSuggestion{Version: 1, Kind: kind, Text: "Synthetic non-reply action", At: time.Now()}
			h.in.mu.Lock()
			h.in.configErr = errors.New("synthetic reply-policy configuration outage")
			h.in.mu.Unlock()
			if _, err := h.actOnSuggestion(t.Context(), it.ID, suggestionReq{Version: 1, Action: suggestConfirm}, true); err != nil {
				t.Fatalf("reply-policy configuration outage blocked %s confirmation: %v", kind, err)
			}
		})
	}
}

func TestInboxReplyPolicyClassificationFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name          string
		requireReview bool
		emptyContext  bool
	}{
		{name: "independent review failure", requireReview: true},
		{name: "empty input", emptyContext: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := agentHub(t)
			cfg := replyPolicyTestConfig()
			cfg.RequireIndependentReview = tc.requireReview
			h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = cfg })
			it := actionItem("lark:classification", time.Now())
			it.Source = "lark"
			it.Agent = &InboxAgent{State: AgentQueued}
			if tc.emptyContext {
				it.Context = ""
			}
			h.inbox.items[it.ID] = it
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			h.classifyInboxReplyPolicies(ctx, cfg)
			if it.ReplyCategory != "unknown" || it.ReplyMode != "owner_only" || it.ReplyClassifiedOf != replyPolicyContentVersion(it) || it.ReplyPolicyRev != "test-v1" {
				t.Fatalf("classification failure did not fail closed: %+v", it)
			}
			before := len(conversationStarts(h.f))
			h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
			checkPolicyReleased(t, it, h.inbox.userLang())
			if got := len(conversationStarts(h.f)); got != before {
				t.Fatalf("failed classification started %d handling conversations", got-before)
			}
		})
	}
}

func TestInboxReplyPolicyReviewRequirementChangeReclassifies(t *testing.T) {
	h := agentHub(t)
	h.in.model = fakeModel(t, `{"category":"ordinary_coordination"}`)
	cfg := replyPolicyTestConfig()
	cfg.RequireIndependentReview = true
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = cfg })
	it := actionItem("lark:review-required", time.Now())
	it.Source = "lark"
	it.ReplyCategory, it.ReplyMode = "ordinary_coordination", "owner_draft"
	it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
	h.inbox.items[it.ID] = it
	h.classifyInboxReplyPolicies(context.Background(), cfg)
	if !it.ReplyReviewed || !replyPolicyClassificationCurrent(it, cfg) {
		t.Fatalf("enabling independent review left an unreviewed owner draft current: %+v", it)
	}
}

func TestInboxReplyPolicyIndependentReviewDisagreementFailsClosed(t *testing.T) {
	h := agentHub(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		category := "ordinary_coordination"
		if calls.Add(1) == 2 {
			category = "approval_permission"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]any{"type": "text", "text": `{"category":"` + category + `"}`}}})
	}))
	t.Cleanup(srv.Close)
	h.in.model = modelAt(t, srv.URL)
	cfg := replyPolicyTestConfig()
	cfg.RequireIndependentReview = true
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = cfg })
	it := actionItem("lark:review-disagreement", time.Now())
	it.Source = "lark"
	it.Agent = &InboxAgent{State: AgentQueued}
	h.inbox.items[it.ID] = it
	h.classifyInboxReplyPolicies(context.Background(), cfg)
	if calls.Load() != 2 {
		t.Fatalf("independent review made %d model calls, want 2", calls.Load())
	}
	if it.ReplyCategory != "unknown" || it.ReplyMode != "owner_only" || it.ReplyReviewed {
		t.Fatalf("review disagreement did not fail closed: %+v", it)
	}
	before := len(conversationStarts(h.f))
	h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
	checkPolicyReleased(t, it, h.inbox.userLang())
	if got := len(conversationStarts(h.f)); got != before {
		t.Fatalf("review disagreement started %d handling conversations", got-before)
	}
}
