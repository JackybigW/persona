package assistant

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

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

// replyPolicySourceUnion is the durable set of sources whose text a row still carries. It includes the empty source:
// an enabled policy with no source list applies to it too. Sorting makes the authorization content version stable.
func replyPolicySourceUnion(groups ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, group := range groups {
		for _, source := range group {
			if !seen[source] {
				seen[source] = true
				out = append(out, source)
			}
		}
	}
	sort.Strings(out)
	return out
}

func replyPolicySourceSet(it *InboxItem) []string {
	if it == nil {
		return nil
	}
	sources := []string{it.Source}
	sources = append(sources, it.ReplyContentSources...)
	for _, ref := range it.MergedFrom {
		sources = append(sources, ref.Source)
	}
	return replyPolicySourceUnion(sources)
}

func replyPolicySourcesApply(cfg *assistantwire.InboxReplyPolicyConfig, sources []string) bool {
	for _, source := range sources {
		if replyPolicyApplies(cfg, source) {
			return true
		}
	}
	return false
}

// replyPolicyItemApplies follows merged text, not just the survivor's current source. ReplyContentSources survives Undo
// because Undo does not remove the member text already copied into the survivor's context.
func replyPolicyItemApplies(cfg *assistantwire.InboxReplyPolicyConfig, it *InboxItem) bool {
	return it != nil && replyPolicySourcesApply(cfg, replyPolicySourceSet(it))
}

func resetReplyPolicyClassification(it *InboxItem) {
	it.ReplyCategory, it.ReplyMode, it.ReplyClassifiedOf, it.ReplyPolicyRev = "", "", "", ""
	it.ReplyReviewed = false
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
// replyPolicyContextTruncated also recognizes rows clipped before ReplyContextTruncated existed. A still-merged row at
// the clipping cap cannot prove that its retained prefix was complete; fail closed and persist the boundary on new writes.
func replyPolicyContextTruncated(it *InboxItem) bool {
	if it == nil {
		return false
	}
	// A mail snippet is not its body. Even after BodyOf catches up, readMailBody retains at most 3000 characters; a
	// prefix at that boundary cannot authorize text that was never presented to the classifier.
	mailIncomplete := it.Mail != nil && (it.BodyOf != it.Last || utf8.RuneCountInString(it.Context) >= 3000)
	return it.ReplyContextTruncated || mailIncomplete || len(it.MergedFrom) > 0 && utf8.RuneCountInString(it.Context) >= mergeCtxMax
}

func replyPolicyContentVersion(it *InboxItem) string {
	if it == nil {
		return ""
	}
	version := strconv.Itoa(len(it.Last)) + "\x00" + it.Last + "\x00" + it.Context
	sources := replyPolicySourceSet(it)
	truncated := replyPolicyContextTruncated(it)
	if len(sources) > 1 || truncated {
		// A clipped append can leave Last and Context byte-for-byte unchanged. Use a separate domain and length-prefix
		// both text fields: Context itself may contain the source/truncation marker bytes, so suffixing the legacy form
		// would let an ordinary item collide with a different merged authorization state.
		version = "reply-policy-merged\x00" + strconv.Itoa(len(it.Last)) + "\x00" + it.Last + "\x00" + strconv.Itoa(len(it.Context)) + "\x00" + it.Context
		version += "\x00sources\x00" + strconv.Itoa(len(sources))
		for _, source := range sources {
			version += "\x00" + strconv.Itoa(len(source)) + "\x00" + source
		}
		version += "\x00truncated\x00" + strconv.FormatBool(truncated)
	}
	sum := sha256.Sum256([]byte(version))
	return hex.EncodeToString(sum[:])
}

func replyPolicyClassificationCurrent(it *InboxItem, cfg *assistantwire.InboxReplyPolicyConfig) bool {
	if it == nil {
		return false
	}
	if !replyPolicyItemApplies(cfg, it) {
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
	if it == nil || !replyPolicyItemApplies(cfg, it) {
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
	return replyPolicyItemApplies(cfg, it) && validReplyPolicyConfig(cfg) && !replyPolicyClassificationCurrent(it, cfg)
}

func replyPolicyQueueBlocked(it *InboxItem, cfg *assistantwire.InboxReplyPolicyConfig) bool {
	if !replyPolicyItemApplies(cfg, it) {
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
	if (kind != SuggestReply && kind != SuggestAgent) || !replyPolicyItemApplies(cfg, it) {
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
	if g == nil || g.Kind != SuggestReply && g.Kind != SuggestAgent || !replyPolicyItemApplies(cfg, it) {
		return nil
	}
	if !replyPolicySnapshotCurrent(it, cfg) || g.ReplyCategory != it.ReplyCategory || g.ReplyMode != it.ReplyMode ||
		g.ReplyClassifiedOf != it.ReplyClassifiedOf || g.ReplyPolicyRev != it.ReplyPolicyRev || g.ReplyReviewed != it.ReplyReviewed {
		return errReplyPolicyDenied
	}
	return nil
}

func replyPolicySendGate(it *InboxItem, g *InboxSuggestion, cfg *assistantwire.InboxReplyPolicyConfig) error {
	if g != nil && g.Kind == SuggestReply && replyPolicyItemApplies(cfg, it) {
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

// replyInstructionForSuggestion returns the latest owner rewrite instruction not consumed by a later owner decision.
// Reported progress and other unrelated chain entries do not consume it. A published replacement carries the chain ID,
// but that ID authorizes only the identical generated text; every changed replacement needs its own classification.
func chainReplyInstruction(chain []InboxChainEntry) *InboxChainEntry {
	for i := len(chain) - 1; i >= 0; i-- {
		switch chain[i].Kind {
		case ChainConfirmed, ChainRejected:
			return nil
		case ChainInstructed:
			instruction := chain[i]
			return &instruction
		}
	}
	return nil
}

func replyInstructionForSuggestion(it *InboxItem, proposedText string, cfg *assistantwire.InboxReplyPolicyConfig) (*InboxChainEntry, bool) {
	if it == nil {
		return nil, false
	}
	instruction := it.ReplyInstruction
	if instruction == nil {
		instruction = chainReplyInstruction(it.Chain)
	}
	if instruction == nil {
		return nil, false
	}
	g := currentSuggestion(it)
	needsClassification := instruction.ID == "" || g == nil || g.ReplyInstruction != instruction.ID || g.Text != proposedText ||
		replyPolicyActionGate(it, g, cfg) != nil
	return instruction, needsClassification
}

func snapshotReplyPolicy(g *InboxSuggestion, it *InboxItem, cfg *assistantwire.InboxReplyPolicyConfig) {
	if g == nil || g.Kind != SuggestReply && g.Kind != SuggestAgent || !replyPolicyItemApplies(cfg, it) {
		return
	}
	g.ReplyCategory, g.ReplyMode = it.ReplyCategory, it.ReplyMode
	g.ReplyClassifiedOf, g.ReplyPolicyRev, g.ReplyReviewed = it.ReplyClassifiedOf, it.ReplyPolicyRev, it.ReplyReviewed
}

// strictReplyPolicyCategory accepts exactly one flat JSON object with no duplicate keys. Authorization must never
// depend on JSON's first-object-wins extraction or last-duplicate-wins unmarshalling.
func strictReplyPolicyCategory(raw string) (string, error) {
	dec := json.NewDecoder(strings.NewReader(strings.TrimSpace(raw)))
	opening, err := dec.Token()
	if err != nil || opening != json.Delim('{') {
		return "", errors.New("reply policy response is not one JSON object")
	}
	seen := map[string]bool{}
	category := ""
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return "", err
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return "", errors.New("ambiguous duplicate reply policy key")
		}
		if key != "category" {
			return "", errors.New("unexpected reply policy key")
		}
		seen[key] = true
		var value json.RawMessage
		if err := dec.Decode(&value); err != nil {
			return "", err
		}
		if key == "category" {
			if err := json.Unmarshal(value, &category); err != nil {
				return "", errors.New("invalid reply policy category")
			}
		}
	}
	if _, err := dec.Token(); err != nil {
		return "", err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return "", errors.New("multiple reply policy JSON values")
	}
	return strings.TrimSpace(category), nil
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
		raw, err := s.inboxModelComplete(cctx, purposeTriage, prompts.RenderJSONOnly(system), input, 1024)
		if err != nil {
			return "", err
		}
		category, err := strictReplyPolicyCategory(raw)
		if err != nil {
			return "", err
		}
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
		if it.VisibilityExcluded || !it.open() || !replyPolicyItemApplies(cfg, it) {
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
		if err := s.classifyInboxReplyPolicyItem(ctx, cfg, it); err != nil {
			s.logf("inbox: reply policy classification for %s: %v", it.ID, err)
		}
	}
}

// classifyInboxReplyPolicyForDraft gives an explicitly requested on-demand draft the same classification fence as a
// queue pass, including an open FYI item that has no TwinFirst rule or previous handling session.
func (s *Service) classifyInboxReplyPolicyForDraft(ctx context.Context, id string, cfg *assistantwire.InboxReplyPolicyConfig) error {
	b := &s.inbox
	b.mu.Lock()
	it := b.items[id]
	b.mu.Unlock()
	if it == nil {
		return nil // draftReply returns the ordinary not-found error below.
	}
	return s.classifyInboxReplyPolicyItem(ctx, cfg, it)
}

// classifyInboxReplyPolicyItem classifies one current item and persists the exact content/policy snapshot. Model
// failures become the durable unknown/owner_only result; only a failed persistence write is returned to the caller.
func (s *Service) classifyInboxReplyPolicyItem(ctx context.Context, cfg *assistantwire.InboxReplyPolicyConfig, it *InboxItem) error {
	if cfg == nil || !cfg.Enabled || !validReplyPolicyConfig(cfg) {
		return nil
	}
	b := &s.inbox
	b.mu.Lock()
	if it.VisibilityExcluded || !it.open() || !replyPolicyItemApplies(cfg, it) || replyPolicyClassificationCurrent(it, cfg) {
		b.mu.Unlock()
		return nil
	}
	version, text, truncated := replyPolicyContentVersion(it), it.Context, replyPolicyContextTruncated(it)
	// A slower earlier model response must not overwrite a newer classification of the same content. The serial is
	// process-local concurrency evidence; the durable category/content/revision snapshot remains the restart fence.
	it.replyClassifySerial++
	serial := it.replyClassifySerial
	b.mu.Unlock()
	allowed, visibilityErr := s.inboxVisibilityModelAllowed(ctx, it)
	if visibilityErr == nil && !allowed {
		return nil
	}
	category, err := "unknown", visibilityErr
	if visibilityErr == nil {
		if truncated {
			err = errors.New("merged reply context was truncated")
		} else {
			category, err = s.classifyReplyPolicyText(ctx, cfg, text)
		}
	}
	if err != nil {
		category = "unknown"
	}
	b.mu.Lock()
	if !it.open() || replyPolicyContentVersion(it) != version || it.replyClassifySerial != serial {
		b.mu.Unlock()
		return nil
	}
	w := beginWrite(it)
	it.ReplyCategory = category
	it.ReplyMode = replyRoute(category, cfg)
	it.ReplyClassifiedOf, it.ReplyPolicyRev = version, cfg.Revision
	it.ReplyReviewed = err == nil && cfg.RequireIndependentReview
	b.settle(w)
	b.mu.Unlock()
	if failed := s.saveWrites(ctx, []*inboxWrite{w}, "its reply policy classification"); failed[it.ID] {
		return errInboxNotPersisted
	}
	return nil
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
	if it == nil || g == nil || (g.Kind != SuggestReply && g.Kind != SuggestAgent) || g.Version != in.Version || !replyPolicyItemApplies(cfg, it) {
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
