package assistant

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/connector"
	"github.com/nwverse/persona/server/internal/i18n"
	"github.com/nwverse/persona/server/internal/proto"
)

func inboxVisibilitySource(cfg *assistantwire.InboxVisibilityConfig, source string) bool {
	return cfg.AppliesTo(source)
}

// inboxVisibilityMatch evaluates only stored metadata. Explicit ids match groups and direct chats; the direct-chat
// switch fails closed when a Lark item has no verifiable type. Non-Lark sources are untouched unless named explicitly.
func inboxVisibilityMatch(it *InboxItem, cfg *assistantwire.InboxVisibilityConfig) bool {
	if it == nil || !inboxVisibilitySource(cfg, it.Source) {
		return false
	}
	chatID, direct, known := "", false, false
	if it.Lark != nil {
		chatID, direct, known = it.Lark.FeedID, !it.Lark.IsGroup, true
	} else if it.Connector != nil {
		for _, tag := range it.Connector.Tags {
			if tag.Key == connector.TagChat {
				chatID = tag.Value
				break
			}
		}
		switch it.Kind {
		case "dm", "bot":
			direct, known = true, true
		case "group", "mention":
			known = true
		}
	}
	for _, excluded := range cfg.Chats {
		if chatID != "" && strings.TrimSpace(excluded) == strings.TrimSpace(chatID) {
			return true
		}
	}
	return cfg.DirectChats && (!known || direct)
}

func inboxVisibilityExcluded(it *InboxItem, cfg *assistantwire.InboxVisibilityConfig) bool {
	return it != nil && (it.VisibilityExcluded || inboxVisibilityMatch(it, cfg))
}

// inboxVisibilityModelAllowed authorizes one item immediately before any model prompt is built from its content.
// The display cache is not an authorization input. Existing items are quarantined in the same pass; callers handling
// a not-yet-admitted item persist the excluded verdict themselves. A configuration error never permits a model read.
func (s *Service) inboxVisibilityModelAllowed(ctx context.Context, it *InboxItem) (bool, error) {
	cfg, err := s.inboxConfigFresh(ctx)
	if err != nil {
		return false, err
	}
	if inboxVisibilityExcluded(it, cfg.Visibility) {
		s.quarantineInboxVisibility(ctx, cfg.Visibility)
		return false, nil
	}
	return true, nil
}

// inboxVisibilityToolRefusal is the manager-side veto for an explicitly listed target. It keeps the local manager
// configuration effective even when an older source extension has no visibility preflight. A source that implements
// the all-direct-chats switch still performs its own metadata check, so ordinary groups need not be guessed here.
func (s *Service) inboxVisibilityToolRefusal(ctx context.Context, machine, session, origin, name string, args json.RawMessage) (proto.ToolResult, bool) {
	if !s.isInboxModelCall(ctx, machine, session, origin) {
		return proto.ToolResult{}, false
	}
	source := ""
	switch name {
	case "lark_messages", "lark_search":
		source = "lark"
	case "larkcli_messages", "larkcli_search":
		source = "larkcli"
	default:
		return proto.ToolResult{}, false
	}
	denied := func() (proto.ToolResult, bool) {
		return proto.TextResult("This chat is excluded from AI and knowledge-base reads.", true), true
	}
	// This is an authorization read, not a display read: a newly enabled boundary cannot wait for the cache to expire.
	fullCfg, err := s.inboxConfigFresh(ctx)
	if err != nil {
		return denied()
	}
	cfg := fullCfg.Visibility
	if cfg == nil || !cfg.DirectChats && len(cfg.Chats) == 0 {
		return proto.ToolResult{}, false
	}
	if !inboxVisibilitySource(cfg, source) {
		return proto.ToolResult{}, false
	}
	if name == "lark_search" || name == "larkcli_search" {
		// Result-level filtering cannot prove that a global search omitted an excluded body.
		return denied()
	}
	var target struct {
		ChatID string `json:"chat_id"`
		Chat   string `json:"chat"`
		FeedID string `json:"feed_id"`
	}
	if len(args) == 0 || json.Unmarshal(args, &target) != nil {
		return denied()
	}
	chatID := strings.TrimSpace(target.ChatID)
	if chatID == "" {
		chatID = strings.TrimSpace(target.Chat)
	}
	if chatID == "" {
		chatID = strings.TrimSpace(target.FeedID)
	}
	if chatID == "" {
		// A thread-only request cannot be tied to a verifiable chat without reading it first.
		return denied()
	}
	for _, excluded := range cfg.Chats {
		if chatID == strings.TrimSpace(excluded) {
			return denied()
		}
	}
	return proto.ToolResult{}, false
}

// quarantineInboxVisibility makes the source boundary retroactive without deleting content or sent-attempt audit.
// The persistent marker also closes old suggestions and direct confirmation calls, including if a source later
// redelivers the same stored item. A matched item's unfinished handling returns to the owner in the same pass.
func (s *Service) quarantineInboxVisibility(ctx context.Context, cfg *assistantwire.InboxVisibilityConfig) {
	b := &s.inbox
	b.mu.Lock()
	var writes []*inboxWrite
	now := time.Now()
	for _, it := range b.items {
		if !inboxVisibilityExcluded(it, cfg) {
			continue
		}
		active, pending := inboxAgentReleaseState(it)
		if it.VisibilityExcluded && !active && !pending {
			continue
		}
		w := beginWrite(it)
		it.VisibilityExcluded = true
		if active {
			b.setHandling(it, HandlingDone, "", now)
			b.chainAdd(it, InboxChainEntry{Kind: ChainConcluded, At: now.UTC(), Text: i18n.T(b.userLang(), "assistant-inbox_agent.never")})
		}
		if a := it.Agent; a != nil {
			a.Pending, a.Wake, a.Back, a.RetryAfter = nil, false, "", nil
			a.DeferredAt, a.WakeSlot = nil, false
			// Detach a live handling session as well as releasing its queue state. Its later tool calls can no longer
			// find this item, while the origin-based outbound policy remains fail closed.
			a.MachineID, a.SessionID = "", ""
			if it.Handling != nil {
				it.Handling.MachineID, it.Handling.SessionID = "", ""
			}
		}
		b.settle(w)
		writes = append(writes, w)
	}
	b.mu.Unlock()
	s.saveWrites(ctx, writes, "the chat visibility quarantine")
}
