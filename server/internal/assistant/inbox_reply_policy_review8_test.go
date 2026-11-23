package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/connector"
	"github.com/nwverse/persona/server/internal/feishubot"
)

func replyPolicyReview8Action(raw, op string) map[string]any {
	var decoded any
	if json.Unmarshal([]byte(raw), &decoded) != nil {
		return nil
	}
	var visit func(any) map[string]any
	visit = func(v any) map[string]any {
		switch x := v.(type) {
		case map[string]any:
			if x["op"] == op {
				return x
			}
			for _, child := range x {
				if found := visit(child); found != nil {
					return found
				}
			}
		case []any:
			for _, child := range x {
				if found := visit(child); found != nil {
					return found
				}
			}
		}
		return nil
	}
	return visit(decoded)
}

func replyPolicyReview8Version(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		n, _ := strconv.Atoi(x)
		return n
	default:
		return 0
	}
}

func replyPolicyReview8DraftPosts(h *inboxHub) chan string {
	posts := make(chan string, 8)
	h.f.answerApp(AppFeishu, FeishuPost, func(raw json.RawMessage) (any, error) {
		var in map[string]any
		_ = json.Unmarshal(raw, &in)
		if card, ok := in["card"].(string); ok {
			posts <- card
		} else {
			card, _ := json.Marshal(in["card"])
			posts <- string(card)
		}
		return map[string]any{"message_id": "synthetic-review8-draft-card"}, nil
	})
	return posts
}

func replyPolicyReview8WorkbenchAction(h *inboxHub, value map[string]any, form map[string]any, mid string) {
	action := &feishubot.CardAction{}
	action.Action.Value = value
	action.Action.FormValue = form
	action.Context.MessageID = mid
	h.workbenchInboxAction(context.Background(), action)
}

func replyPolicyReview8OrdinaryItem(h *inboxHub, it *InboxItem) {
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy = replyPolicyTestConfig() })
	it.ReplyCategory, it.ReplyMode = "ordinary_coordination", "owner_draft"
	it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
}

func TestInboxReplyPolicyWorkbenchDraftCardBindsGeneratedTextAndVersion(t *testing.T) {
	h, lark, it := handledHub(t)
	replyPolicyReview8OrdinaryItem(h, it)
	_ = replyPolicyEnableWorkbench(h)
	posts := replyPolicyReview8DraftPosts(h)
	const generated = "Synthetic generation whose text must stay bound to its own version"
	calls := replyPolicyDraftSequenceModel(t, h, nil, generated)
	h.inbox.workbenchDraftHook = func() {
		h.inbox.workbenchDraftHook = nil
		it.Lark.FeedID = "synthetic-new-recipient"
		finalReviewInstallMirror(h, it, "A newer replacement installed before Workbench renders")
	}
	replyPolicyReview8WorkbenchAction(h, map[string]any{"a": "ib", "op": "draft", "id": it.ID}, nil, "synthetic-review8-source-card")
	var card string
	select {
	case card = <-posts:
	case <-time.After(3 * time.Second):
		t.Fatal("Workbench did not publish its draft card")
	}
	if got := len(calls()); got != 1 {
		t.Fatalf("Workbench draft made %d model calls, want one generation", got)
	}
	if action := replyPolicyReview8Action(card, "send"); action != nil {
		if version := replyPolicyReview8Version(action["version"]); version != 1 || !strings.Contains(card, generated) {
			t.Fatalf("Workbench combined one draft's text with another draft's version: version=%d card=%s", version, card)
		}
	}
	if g := currentSuggestion(it); g == nil || g.Version != 2 || g.Text == generated {
		t.Fatalf("test setup did not retain the newer replacement: %+v", g)
	}
	if len(lark.got) != 0 {
		t.Fatalf("Workbench version race reached the physical sender: %+v", lark.got)
	}
}

func TestInboxReplyPolicyIdenticalReplacementRequiresFreshIndependentReview(t *testing.T) {
	h, lark, it := handledHub(t)
	replyPolicyReview8OrdinaryItem(h, it)
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Original draft before owner instruction"}`); err != nil {
		t.Fatal(err)
	}
	instruction := "Synthetic owner instruction whose replacement later needs independent review"
	if _, err := h.actOnSuggestion(t.Context(), it.ID, suggestionReq{Version: 1, Action: suggestInstruct, Text: &instruction}, true); err != nil {
		t.Fatal(err)
	}
	toldConv(t, h, instruction)
	calls := replyPolicyDraftSequenceModel(t, h, nil,
		`{"category":"ordinary_coordination"}`,
		`{"category":"ordinary_coordination"}`, `{"category":"ordinary_coordination"}`,
		`{"category":"ordinary_coordination"}`, `{"category":"ordinary_coordination"}`)
	const replacement = "Identical instructed replacement under a newly strengthened review policy"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"`+replacement+`"}`); err != nil {
		t.Fatal(err)
	}
	h.setConfig(func(c *assistantwire.InboxConfig) { c.ReplyPolicy.RequireIndependentReview = true })
	h.classifyInboxReplyPolicies(t.Context(), h.in.cfg.ReplyPolicy)
	if !it.ReplyReviewed {
		t.Fatal("test setup did not reclassify the original item with independent review")
	}
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"`+replacement+`"}`); err != nil {
		t.Fatalf("identical replacement was rejected after its fresh independent review: %v", err)
	}
	if got := len(calls()); got != 5 {
		t.Fatalf("identical replacement skipped newly required independent review: model calls=%d want=5", got)
	}
	if g := currentSuggestion(it); g == nil || !g.ReplyReviewed || g.Text != replacement {
		t.Fatalf("reviewed identical replacement was not published: %+v", g)
	}
	if len(lark.got) != 0 {
		t.Fatalf("replacement review test reached the physical sender: %+v", lark.got)
	}
}

func TestInboxReplyPolicyReportedProgressCannotEvictRewriteAuthorization(t *testing.T) {
	h, lark, it := handledHub(t)
	replyPolicyReview8OrdinaryItem(h, it)
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Original draft before owner instruction"}`); err != nil {
		t.Fatal(err)
	}
	instruction := "Synthetic owner rewrite instruction that progress must not evict"
	if _, err := h.actOnSuggestion(t.Context(), it.ID, suggestionReq{Version: 1, Action: suggestInstruct, Text: &instruction}, true); err != nil {
		t.Fatal(err)
	}
	toldConv(t, h, instruction)
	calls := replyPolicyDraftSequenceModel(t, h, nil, `{"category":"ordinary_coordination"}`, `{"category":"approval_permission"}`)
	const ordinary = "First replacement retaining the instruction ID"
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"`+ordinary+`"}`); err != nil {
		t.Fatal(err)
	}
	h.inbox.mu.Lock()
	for i := 0; i < chainMax; i++ {
		h.inbox.chainAdd(it, InboxChainEntry{Kind: ChainReported, Detail: "synthetic-progress-" + strconv.Itoa(i)})
	}
	h.inbox.mu.Unlock()
	if _, err := suggestTool(t, h, `{"kind":"reply","text":"Changed replacement after rewrite instruction eviction"}`); !errors.Is(err, errReplyPolicyDenied) {
		t.Fatalf("evicted rewrite instruction authorized an unclassified replacement: %v", err)
	}
	if g := currentSuggestion(it); g == nil || g.Text != ordinary {
		t.Fatalf("unclassified post-eviction replacement displaced the authorized draft: %+v", g)
	}
	if got := len(calls()); got > 2 {
		t.Fatalf("post-eviction replacement made unexpected model calls: %d", got)
	}
	if len(lark.got) != 0 {
		t.Fatalf("post-eviction replacement reached the physical sender: %+v", lark.got)
	}
}

func TestInboxReplyPolicyFailedPublicationMarkerSurvivesConcurrentMetadataReload(t *testing.T) {
	c := &recoveryReplier{}
	h, _, it := handledHub(t)
	h.in.mu.Lock()
	h.in.conns[0] = c
	h.in.mu.Unlock()
	replyPolicyReview8OrdinaryItem(h, it)
	h.in.model = fakeModel(t, "Synthetic draft whose failed-publication marker must persist")
	st := &replyPolicyDraftOneShotFailStore{inboxMemStore: h.st, fail: true}
	st.before = func() {
		h.inbox.mu.Lock()
		h.inbox.touched(it.ID)
		h.inbox.mu.Unlock()
	}
	h.store = func() Store { return st }
	h.inbox.onDemandDraftSaveHook = func() {
		h.inbox.onDemandDraftSaveHook = nil
		if _, err := suggestTool(t, h, `{"recap":"Concurrent metadata persisted before the failure tombstone"}`); err != nil {
			t.Errorf("concurrent metadata write failed: %v", err)
		}
	}
	if text, err := h.draftReply(t.Context(), it.ID, ""); err == nil || text != "" {
		t.Fatalf("failed publication returned text: text=%q err=%v", text, err)
	}
	restarted := New(Options{Hub: h.f, Store: func() Store { return h.st }, Logf: testLogf(t)})
	if err := restarted.inboxReady(t.Context()); err != nil {
		t.Fatal(err)
	}
	restarted.inbox.mu.Lock()
	loaded := restarted.inbox.items[it.ID]
	g := mirroredLegacyReply(loaded)
	restarted.inbox.mu.Unlock()
	if g == nil {
		t.Fatal("reload lost the unpublished mirror")
	}
	if !g.ReplyPublicationFailed {
		t.Errorf("failed-publication marker was lost by a concurrent persisted metadata change")
	}
	if _, err := restarted.replyInboxChecked(t.Context(), it.ID, g.Text, recoveryFreshID, false, nil, nil); err == nil {
		t.Errorf("lost failed-publication marker authorized an old-client reply after reload")
	}
	if got := len(c.calls()); got != 0 {
		t.Fatalf("lost failed-publication marker caused %d physical sends after reload", got)
	}
}

func TestInboxReplyPolicyUnversionedWorkbenchRedraftFailsClosed(t *testing.T) {
	c := &recoveryReplier{}
	h, _, it := handledHub(t)
	h.in.mu.Lock()
	h.in.conns[0] = c
	h.in.mu.Unlock()
	replyPolicyReview8OrdinaryItem(h, it)
	updated := replyPolicyEnableWorkbench(h)
	const oldText = "Old unversioned Workbench draft"
	finalReviewInstallMirror(h, it, oldText)
	value := replyPolicyWorkbenchActionValue(t, h.draftCard(context.Background(), *it, oldText, nil, "").JSON(), "redraft")
	delete(value, "version")
	const current = "Current replacement that an old card must not supersede"
	finalReviewInstallMirror(h, it, current)
	calls := replyPolicyDraftSequenceModel(t, h, nil, "Forbidden generation from an unversioned card", `{"category":"ordinary_coordination"}`)
	replyPolicyReview8WorkbenchAction(h, value, map[string]any{"text": oldText, "instruction": "Synthetic old-card redraft"}, "synthetic-unversioned-redraft")
	select {
	case <-updated:
	case <-time.After(3 * time.Second):
		t.Fatal("unversioned Workbench redraft did not finish")
	}
	if got := len(calls()); got != 0 {
		t.Fatalf("unversioned Workbench redraft bypassed the current suggestion version: model calls=%d", got)
	}
	if g := currentSuggestion(it); g == nil || g.Version != 2 || g.Text != current {
		t.Fatalf("unversioned Workbench redraft superseded the current suggestion: %+v", g)
	}
	if len(c.calls()) != 0 {
		t.Fatalf("unversioned Workbench redraft reached the physical sender: %+v", c.calls())
	}
}

type replyPolicyReview8Connector struct{ *recoveryReplier }

func (*replyPolicyReview8Connector) ID() string   { return "synthetic" }
func (*replyPolicyReview8Connector) Name() string { return "Synthetic connector" }
func (*replyPolicyReview8Connector) Manifest() connector.Manifest {
	return connector.Manifest{Capabilities: []connector.Capability{connector.CapInbox}, Kinds: []string{KindDM}}
}

func replyPolicyReview8NonLark(t *testing.T) (*inboxHub, *replyPolicyReview8Connector, *InboxItem) {
	t.Helper()
	h, _, it := handledHub(t)
	c := &replyPolicyReview8Connector{&recoveryReplier{}}
	h.addConnector(c, true)
	cfg := replyPolicyTestConfig()
	cfg.Sources = nil
	h.setConfig(func(out *assistantwire.InboxConfig) { out.ReplyPolicy = cfg })
	h.inbox.mu.Lock()
	delete(h.inbox.items, it.ID)
	it.ID = "synthetic:omitted-sources"
	it.Source = "synthetic"
	it.Lark = nil
	it.Connector = &InboxConnector{Thread: "omitted-sources", ReplyTo: "synthetic-target", CanReply: true}
	h.inbox.items[it.ID] = it
	h.inbox.mu.Unlock()
	it.ReplyCategory, it.ReplyMode = "approval_permission", "owner_only"
	it.ReplyClassifiedOf, it.ReplyPolicyRev = replyPolicyContentVersion(it), "test-v1"
	return h, c, it
}

func TestInboxReplyPolicyOmittedSourcesNonLarkCoverAllStages(t *testing.T) {
	t.Run("queue", func(t *testing.T) {
		h, _, it := replyPolicyReview8NonLark(t)
		it.Agent = &InboxAgent{State: AgentQueued}
		it.Handling = nil
		before := len(conversationStarts(h.f))
		h.startInboxAgents(context.Background(), InboxAgentConfig{Enabled: true})
		if got := len(conversationStarts(h.f)); got != before {
			t.Errorf("omitted sources excluded a non-Lark restricted queue item: %d conversations started", got-before)
		}
		checkPolicyReleased(t, it, h.inbox.userLang())
	})
	t.Run("suggestion", func(t *testing.T) {
		h, _, it := replyPolicyReview8NonLark(t)
		if _, err := suggestTool(t, h, `{"kind":"reply","text":"Restricted non-Lark draft with omitted sources"}`); !errors.Is(err, errReplyPolicyDenied) {
			t.Errorf("omitted sources excluded non-Lark suggestion authorization: %v", err)
		}
		if currentSuggestion(it) != nil {
			t.Fatalf("restricted non-Lark suggestion was published with omitted sources: %+v", currentSuggestion(it))
		}
	})
	t.Run("send", func(t *testing.T) {
		h, c, it := replyPolicyReview8NonLark(t)
		g := finalReviewInstallMirror(h, it, "Restricted non-Lark mirror with omitted sources")
		if _, err := h.replyInboxChecked(t.Context(), it.ID, g.Text, recoveryFreshID, false, nil, nil); !errors.Is(err, errReplyPolicyDenied) {
			t.Errorf("omitted sources excluded non-Lark final-send authorization: %v", err)
		}
		if len(c.calls()) != 0 {
			t.Fatalf("omitted sources allowed a restricted non-Lark physical send: %+v", c.calls())
		}
	})
}

func TestInboxReplyPolicyWorkbenchPublishedCardSend(t *testing.T) {
	c := &recoveryReplier{}
	h, _, it := handledHub(t)
	h.in.mu.Lock()
	h.in.conns[0] = c
	h.in.mu.Unlock()
	replyPolicyReview8OrdinaryItem(h, it)
	updated := replyPolicyEnableWorkbench(h)
	posts := replyPolicyReview8DraftPosts(h)
	const generated = "Synthetic draft emitted by the real Workbench publication path"
	replyPolicyDraftSequenceModel(t, h, nil, generated)
	replyPolicyReview8WorkbenchAction(h, map[string]any{"a": "ib", "op": "draft", "id": it.ID}, nil, "synthetic-published-draft-source")
	var card string
	select {
	case card = <-posts:
	case <-time.After(3 * time.Second):
		t.Fatal("real Workbench draft publication did not emit a card")
	}
	value := replyPolicyWorkbenchActionValue(t, card, "send")
	if version := replyPolicyReview8Version(value["version"]); version == 0 {
		t.Fatalf("published Workbench card was born stale: version=0 card=%s", card)
	}
	replyPolicyReview8WorkbenchAction(h, value, map[string]any{"text": generated}, "synthetic-published-draft-card")
	select {
	case <-updated:
	case <-time.After(3 * time.Second):
		t.Fatal("published Workbench card send did not finish")
	}
	if calls := c.calls(); len(calls) != 1 || calls[0].Text != generated {
		t.Fatalf("published Workbench card did not send through its emitted version: %+v", calls)
	}
}

func TestInboxReplyPolicyWorkbenchFeishuActionDispatch(t *testing.T) {
	c := &recoveryReplier{}
	h, _, it := handledHub(t)
	h.in.mu.Lock()
	h.in.conns[0] = c
	h.in.mu.Unlock()
	replyPolicyReview8OrdinaryItem(h, it)
	updated := replyPolicyEnableWorkbench(h)
	const text = "Synthetic rendered card submitted through FeishuAction"
	finalReviewInstallMirror(h, it, text)
	value := replyPolicyWorkbenchActionValue(t, h.draftCard(context.Background(), *it, text, nil, "").JSON(), "send")
	action := &feishubot.CardAction{}
	action.Action.Value = value
	action.Action.FormValue = map[string]any{"text": text}
	action.Context.MessageID = "synthetic-real-feishu-action-card"
	event, err := json.Marshal(action)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.FeishuAction(t.Context(), FeishuAction{Event: event, Action: "oa_option"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-updated:
	case <-time.After(3 * time.Second):
		t.Fatal("real FeishuAction did not dispatch the rendered Inbox send")
	}
	if calls := c.calls(); len(calls) != 1 || calls[0].Text != text {
		t.Fatalf("real FeishuAction did not send the rendered Inbox card: %+v", calls)
	}
}
