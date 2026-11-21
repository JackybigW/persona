package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/prompts"
)

const (
	ReplyRouteOwnerDraft = "owner_draft"
	ReplyRouteBot        = "bot"
	ReplyRouteOwnerOnly  = "owner_only"
	ReplyRouteNoReply    = "no_reply"
)

var errReplyPolicyDenied = errCannotReply

var replyPolicyCategories = map[string]bool{
	"personal_transaction": true, "security_compliance": true, "approval_permission": true,
	"decision_delegation": true, "technical_qa": true, "research": true, "alert_triage": true,
	"ppe_deploy": true, "mr_precheck": true, "no_reply": true, "ordinary_coordination": true,
	"logistical_followup": true, "acknowledgement": true, "other_owner_reply": true, "unknown": true,
}

var forcedReplyRoutes = map[string]string{
	"personal_transaction": ReplyRouteOwnerOnly, "security_compliance": ReplyRouteOwnerOnly,
	"approval_permission": ReplyRouteOwnerOnly, "decision_delegation": ReplyRouteOwnerOnly,
	"technical_qa": ReplyRouteBot, "research": ReplyRouteBot, "alert_triage": ReplyRouteBot,
	"ppe_deploy": ReplyRouteBot, "mr_precheck": ReplyRouteBot, "no_reply": ReplyRouteNoReply,
	"unknown": ReplyRouteOwnerOnly, "": ReplyRouteOwnerOnly,
}

func validReplyRoute(route string) bool {
	switch route {
	case ReplyRouteOwnerDraft, ReplyRouteBot, ReplyRouteOwnerOnly, ReplyRouteNoReply:
		return true
	default:
		return false
	}
}

// validReplyPolicyConfig rejects the whole enabled specification, rather than letting one safe-looking category hide
// a malformed route elsewhere. A disabled or absent policy is the legacy path and needs no validation here.
func validReplyPolicyConfig(cfg *assistantwire.InboxReplyPolicyConfig) bool {
	if cfg == nil || !cfg.Enabled {
		return true
	}
	if strings.TrimSpace(cfg.Revision) == "" || cfg.UnknownMode != "" && !validReplyRoute(cfg.UnknownMode) {
		return false
	}
	for category, route := range cfg.Routes {
		if !replyPolicyCategories[category] || !validReplyRoute(route) {
			return false
		}
	}
	return true
}

// replyPolicyApplies selects sources explicitly, or every source when an enabled policy has no source list. A disabled
// or nil policy is the compatibility path: none of the gates changes existing behavior.
func replyPolicyApplies(cfg *assistantwire.InboxReplyPolicyConfig, source string) bool {
	if cfg == nil || !cfg.Enabled {
		return false
	}
	if len(cfg.Sources) == 0 {
		return true
	}
	for _, candidate := range cfg.Sources {
		if candidate == source {
			return true
		}
	}
	return false
}

func replyRoute(category string, cfg *assistantwire.InboxReplyPolicyConfig) string {
	if cfg == nil || !cfg.Enabled {
		return ""
	}
	category = strings.TrimSpace(category)
	if !validReplyPolicyConfig(cfg) || !replyPolicyCategories[category] {
		return ReplyRouteOwnerOnly
	}
	if route, forced := forcedReplyRoutes[category]; forced {
		return route
	}
	if route := cfg.Routes[category]; validReplyRoute(route) {
		return route
	}
	return ReplyRouteOwnerOnly
}

// replyPolicyContentVersion covers the exact text the classifier saw, not only the latest source version. Merges can
// append a restricted member's context without changing Last, so Last alone is not an authorization version.
func replyPolicyContentVersion(it *InboxItem) string {
	if it == nil {
		return ""
	}
	sum := sha256.Sum256([]byte(strconv.Itoa(len(it.Last)) + "\x00" + it.Last + "\x00" + it.Context))
	return hex.EncodeToString(sum[:])
}

func replyPolicyClassificationCurrent(it *InboxItem, cfg *assistantwire.InboxReplyPolicyConfig) bool {
	if it == nil {
		return false
	}
	if !replyPolicyApplies(cfg, it.Source) {
		return true
	}
	if it.ReplyClassifiedOf == "" || it.ReplyClassifiedOf != replyPolicyContentVersion(it) || it.ReplyPolicyRev == "" || it.ReplyPolicyRev != cfg.Revision {
		return false
	}
	// A definitively non-draft result is already fail closed, including an unknown produced by a failed review. Only an
	// owner-draft route needs the independent review bit; turning that requirement on reclassifies an existing draft.
	if replyRoute(it.ReplyCategory, cfg) != ReplyRouteOwnerDraft {
		return true
	}
	return !cfg.RequireIndependentReview || it.ReplyReviewed
}

func replyPolicySnapshotCurrent(it *InboxItem, cfg *assistantwire.InboxReplyPolicyConfig) bool {
	if it == nil || !replyPolicyApplies(cfg, it.Source) {
		return true
	}
	if !replyPolicyClassificationCurrent(it, cfg) {
		return false
	}
	route := replyRoute(it.ReplyCategory, cfg)
	return route == ReplyRouteOwnerDraft && it.ReplyMode == route
}

// replyPolicyQueuePending leaves a not-yet-classified item queued without starting work. The next pass classifies it;
// only a current, definitively non-owner-draft route is released like TwinNever.
func replyPolicyQueuePending(it *InboxItem, cfg *assistantwire.InboxReplyPolicyConfig) bool {
	return replyPolicyApplies(cfg, it.Source) && validReplyPolicyConfig(cfg) && !replyPolicyClassificationCurrent(it, cfg)
}

func replyPolicyQueueBlocked(it *InboxItem, cfg *assistantwire.InboxReplyPolicyConfig) bool {
	if !replyPolicyApplies(cfg, it.Source) {
		return false
	}
	if !validReplyPolicyConfig(cfg) {
		return true
	}
	if !replyPolicyClassificationCurrent(it, cfg) {
		return false
	}
	route := replyRoute(it.ReplyCategory, cfg)
	return route != ReplyRouteOwnerDraft || it.ReplyMode != route
}

func replyPolicySuggestionGate(it *InboxItem, kind string, cfg *assistantwire.InboxReplyPolicyConfig) error {
	if (kind != SuggestReply && kind != SuggestAgent) || !replyPolicyApplies(cfg, it.Source) {
		return nil
	}
	if !replyPolicySnapshotCurrent(it, cfg) {
		return errReplyPolicyDenied
	}
	return nil
}

// replyPolicyActionGate binds both reply and Agent confirmations to the classification the owner saw. In particular,
// an Agent suggestion created before a policy was enabled, or before its category became restricted, cannot execute.
func replyPolicyActionGate(it *InboxItem, g *InboxSuggestion, cfg *assistantwire.InboxReplyPolicyConfig) error {
	if g == nil || g.Kind != SuggestReply && g.Kind != SuggestAgent || !replyPolicyApplies(cfg, it.Source) {
		return nil
	}
	if !replyPolicySnapshotCurrent(it, cfg) || g.ReplyCategory != it.ReplyCategory || g.ReplyMode != it.ReplyMode ||
		g.ReplyClassifiedOf != it.ReplyClassifiedOf || g.ReplyPolicyRev != it.ReplyPolicyRev || g.ReplyReviewed != it.ReplyReviewed {
		return errReplyPolicyDenied
	}
	return nil
}

func replyPolicySendGate(it *InboxItem, g *InboxSuggestion, cfg *assistantwire.InboxReplyPolicyConfig) error {
	if g != nil && g.Kind == SuggestReply && replyPolicyApplies(cfg, it.Source) {
		return replyPolicyActionGate(it, g, cfg)
	}
	return nil
}

// mirroredLegacyReply identifies the AI draft exposed to old clients through agent.draft. A direct owner-authored reply
// without this mirror stays on the legacy /reply path; a reply that edits the mirrored draft is still an AI suggestion
// confirmation and must pass the same edit veto and snapshot authorization as the versioned suggestion API.
func mirroredLegacyReply(it *InboxItem) *InboxSuggestion {
	if it == nil || it.Agent == nil || it.Agent.State != AgentDrafted {
		return nil
	}
	g := currentSuggestion(it)
	if g == nil || g.Kind != SuggestReply || g.Handback || it.Agent.Draft == "" || it.Agent.Draft != g.Text || suggestionAttempt(it, g.Version) != nil {
		return nil
	}
	return g
}

func snapshotReplyPolicy(g *InboxSuggestion, it *InboxItem, cfg *assistantwire.InboxReplyPolicyConfig) {
	if g == nil || g.Kind != SuggestReply && g.Kind != SuggestAgent || !replyPolicyApplies(cfg, it.Source) {
		return
	}
	g.ReplyCategory, g.ReplyMode = it.ReplyCategory, it.ReplyMode
	g.ReplyClassifiedOf, g.ReplyPolicyRev, g.ReplyReviewed = it.ReplyClassifiedOf, it.ReplyPolicyRev, it.ReplyReviewed
}

// classifyReplyPolicyInput returns only a category. The server, not the model, maps it to a route. Independent review
// is a second invocation that must reach the same category without seeing the first answer. An owner edit includes the
// original conversation: words such as "yes, go ahead" derive their authority from that context, not from the edit alone.
func (s *Service) classifyReplyPolicyInput(ctx context.Context, cfg *assistantwire.InboxReplyPolicyConfig, in prompts.InboxReplyPolicyInput) (string, error) {
	if !validReplyPolicyConfig(cfg) {
		return "unknown", errors.New("invalid reply policy config")
	}
	if strings.TrimSpace(in.Text) == "" {
		return "unknown", errors.New("empty reply policy input")
	}
	input := prompts.RenderInboxReplyPolicyInput(in)
	classify := func(system string) (string, error) {
		cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		var out struct {
			Category string `json:"category"`
		}
		if err := s.inboxModelJSON(cctx, purposeTriage, system, input, 0, &out); err != nil {
			return "", err
		}
		category := strings.TrimSpace(out.Category)
		if !replyPolicyCategories[category] {
			return "", errors.New("invalid reply policy category")
		}
		return category, nil
	}
	category, err := classify(prompts.RenderInboxReplyPolicy())
	if err != nil {
		return "unknown", err
	}
	if cfg.RequireIndependentReview {
		reviewed, err := classify(prompts.RenderInboxReplyPolicyReview())
		if err != nil {
			return "unknown", err
		}
		if reviewed != category {
			return "unknown", errors.New("reply policy review disagreed")
		}
	}
	return category, nil
}

func (s *Service) classifyReplyPolicyText(ctx context.Context, cfg *assistantwire.InboxReplyPolicyConfig, text string) (string, error) {
	return s.classifyReplyPolicyInput(ctx, cfg, prompts.InboxReplyPolicyInput{Text: text})
}

// classifyInboxReplyPolicies classifies actionable items before their queue pass. Failures are persisted as unknown /
// owner_only for this exact content and policy revision, so a model outage cannot turn into a draft on a later gate.
func (s *Service) classifyInboxReplyPolicies(ctx context.Context, cfg *assistantwire.InboxReplyPolicyConfig) {
	if cfg == nil || !cfg.Enabled || !validReplyPolicyConfig(cfg) {
		return
	}
	b := &s.inbox
	b.mu.Lock()
	var todo []*InboxItem
	for _, it := range b.items {
		if it.VisibilityExcluded || !it.open() || !replyPolicyApplies(cfg, it.Source) {
			continue
		}
		// Classification runs before escalation. TwinFirst can enqueue an already-tiered FYI item, and an existing
		// handling session can receive work outside TierAction; both need a snapshot before the queue pass.
		if it.Tier != TierAction && b.policyTwin(it) != TwinFirst && !it.handled() {
			continue
		}
		stale := it.ReplyClassifiedOf != replyPolicyContentVersion(it) || it.ReplyPolicyRev != cfg.Revision
		if replyRoute(it.ReplyCategory, cfg) == ReplyRouteOwnerDraft && cfg.RequireIndependentReview && !it.ReplyReviewed {
			stale = true
		}
		if stale {
			todo = append(todo, it)
		}
	}
	b.mu.Unlock()
	for _, it := range todo {
		allowed, visibilityErr := s.inboxVisibilityModelAllowed(ctx, it)
		if visibilityErr == nil && !allowed {
			continue
		}
		b.mu.Lock()
		version, text := replyPolicyContentVersion(it), it.Context
		b.mu.Unlock()
		category, err := "unknown", visibilityErr
		if visibilityErr == nil {
			category, err = s.classifyReplyPolicyText(ctx, cfg, text)
		}
		if err != nil {
			category = "unknown"
		}
		b.mu.Lock()
		if it.open() && replyPolicyContentVersion(it) == version {
			w := beginWrite(it)
			it.ReplyCategory = category
			it.ReplyMode = replyRoute(category, cfg)
			it.ReplyClassifiedOf, it.ReplyPolicyRev = version, cfg.Revision
			it.ReplyReviewed = err == nil && cfg.RequireIndependentReview
			b.settle(w)
			b.mu.Unlock()
			s.saveWrites(ctx, []*inboxWrite{w}, "its reply policy classification")
		} else {
			b.mu.Unlock()
		}
	}
}

// vetoEditedReply rechecks owner-introduced text before confirmation can reserve a send. Unedited text already carries
// the item's trusted classification; an edited text must independently classify as an owner-draft route.
func (s *Service) vetoEditedReply(ctx context.Context, id string, in suggestionReq, cfg *assistantwire.InboxReplyPolicyConfig) error {
	if in.Action != suggestConfirm || in.Text == nil || strings.TrimSpace(*in.Text) == "" {
		return nil
	}
	b := &s.inbox
	b.mu.Lock()
	it := b.items[id]
	var g *InboxSuggestion
	if it != nil {
		g = currentSuggestion(it)
	}
	if it == nil || g == nil || g.Kind != SuggestReply || g.Version != in.Version || !replyPolicyApplies(cfg, it.Source) {
		b.mu.Unlock()
		return nil
	}
	text := strings.TrimSpace(*in.Text)
	original, conversation := g.Text, it.Context
	b.mu.Unlock()
	if text == original {
		return nil
	}
	category, err := s.classifyReplyPolicyInput(ctx, cfg, prompts.InboxReplyPolicyInput{Text: text, Original: conversation, Edit: true})
	if err != nil || replyRoute(category, cfg) != ReplyRouteOwnerDraft {
		// This veto was required by the enabled policy under which classification just ran. A later disable can restore
		// legacy behavior for a new attempt, but it cannot retroactively turn this rejected edit into an authorized send.
		return errReplyPolicyDenied
	}
	return nil
}
