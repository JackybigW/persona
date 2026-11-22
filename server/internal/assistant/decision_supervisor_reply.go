package assistant

import (
	"context"
	"sync"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
	"github.com/nwverse/persona/server/internal/lazyre"
	"github.com/nwverse/persona/server/internal/prompts"
	"github.com/nwverse/persona/server/internal/proto"
)

// Side-conversation markers, kept local so this file does not depend on task_completion.go's
// package vars (a parallel change extracts a shared prompts classifier). They match the two
// envelopes RenderSideConversation emits (prompts/assistant/side_conversation.md).
var (
	supervisorSideConversationContext = lazyre.New(`<persona-context\b[^>]*\bkind\s*=\s*["']side-conversation["'][^>]*>`)
	supervisorSideConversationMarker  = lazyre.New(`<persona-side-conversation\s*/>`)
)

// supervisorDecisionSpoke handles an owner message written in a supervisor / direction
// conversation itself. The Agent-chat path (agentDecisionSpoke) anchors on the to-do's worker
// session via activeTodoOf, which a supervisor conversation never is; this is the reverse
// lookup: the open needs_user to-dos supervised by the conversation the owner wrote in.
//
// An exact option label settles the single to-do it uniquely matches, but only once the hub
// says the message was delivered (#13041): the hub tells session.user_spoke before it sends, so
// the choice waits by the send's client_msg_id (deferSupervisorSettle) and settles on a
// session.assistant_sent reporting delivered (supervisorSettleOutcome). A failed or unknown send
// leaves the card waiting. A spoke without an id (an older hub, or a client that names no
// message) cannot be matched to its outcome, so it is handed back like free text instead of
// settling. A lone option letter
// ("A", "b") never settles here: unlike an Agent session, a supervisor / direction conversation
// is usually the owner's main chat and carries unrelated replies (its own lettered or yes/no
// questions), so a one-letter message cannot be tied to the card and is treated as free text
// instead. Free text never guesses an option: each affected to-do pauses its default and hands
// back to this conversation's supervisor through the existing decision_reply mechanism
// (#11852), with the reply sourced from the supervisor conversation. Provenance-marked relay /
// side-conversation input is not the owner answering and changes nothing. When one message
// could settle several to-dos at once, none is settled silently: every candidate is handed back
// for an explicit choice.
func (s *Service) supervisorDecisionSpoke(ctx context.Context, r SessionRef, at time.Time, text, clientMsgID string) {
	if !supervisorOwnerText(text) {
		return
	}
	if err := s.ensureTodosLoaded(ctx); err != nil {
		s.logf("decision: supervisor reply: load tasks: %v", err)
		return
	}
	candidates := s.supervisedDecisionCandidates(r.SessionID)
	if len(candidates) == 0 {
		return
	}
	matched := make([]Todo, 0, len(candidates))
	for _, t := range candidates {
		a, ok := s.askCopy(t.ID)
		if !ok || supervisorOptionLabel(text, a.Options) == "" {
			continue
		}
		matched = append(matched, t)
	}
	switch {
	case len(matched) == 1 && len(candidates) == 1 && clientMsgID != "":
		s.deferSupervisorSettle(r, clientMsgID, matched[0].ID, at, text)
	case len(matched) == 1 && len(candidates) == 1:
		// No id to learn the send's outcome by: never close the card irreversibly on a message that may not
		// have gone in; the supervisor reads the words and confirms with task_decided.
		s.handBackSupervisorReplies(ctx, r, at, candidates, text)
	case len(matched) == 1:
		// One exact match among several open questions is still ambiguous: the same words may be
		// meant for another card, so do not settle; hand every candidate back to the supervisor.
		s.handBackSupervisorReplies(ctx, r, at, candidates, text)
	default:
		// No unique exact answer (zero or several matches): never guess, hand them all back.
		s.handBackSupervisorReplies(ctx, r, at, candidates, text)
	}
}

// supervisorOwnerText is the provenance gate for a message observed on a supervisor
// conversation: a relay, a message-source envelope or a side-conversation wrapper is another
// conversation's input, not the owner answering a card. Empty text (a hook without payload) has
// no owner words to settle with, so it is ignored here rather than treated as an answer.
func supervisorOwnerText(text string) bool {
	if text == "" {
		return false
	}
	if supervisorSideConversationContext.MatchString(text) || supervisorSideConversationMarker.MatchString(text) {
		return false
	}
	if src := prompts.ItemSource(proto.Item{Role: "user", Kind: proto.KindText, Text: text}); src != nil && src.Kind != "" {
		return false
	}
	return prompts.ParseRelaySource(text) == nil
}

// supervisedDecisionCandidates returns the open to-dos that conversation supervisorID
// supervises and whose decide ask is still waiting, as the caller's snapshot reads them. A
// supervisor conversation is never an active worker session, so activeTodoOf cannot find these;
// the Supervision.Supervisor relationship is the reverse index.
func (s *Service) supervisedDecisionCandidates(supervisorID string) []Todo {
	s.todos.mu.RLock()
	todos := make([]Todo, 0, len(s.todos.todos))
	for _, t := range s.todos.todos {
		if t.DoneAt != nil || t.Supervision == nil || t.Supervision.Supervisor != supervisorID ||
			t.Supervision.State != reviewNeedsUser {
			continue
		}
		todos = append(todos, *t)
	}
	s.todos.mu.RUnlock()
	out := todos[:0]
	for _, t := range todos {
		a, ok := s.askCopy(t.ID)
		if !ok || !decisionOpen(&a, t) || !a.RecheckAt.IsZero() || ownerDecisionReturned(t.Supervision) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// supervisorOptionLabel matches a full option label only, never a lone letter. The Agent-chat
// path accepts letters because that session is scoped to one task; the supervisor conversation
// carries every conversation the owner has, so an isolated "A" there usually answers something
// else and must not settle a card irreversibly.
func supervisorOptionLabel(text string, options []DecisionOption) string {
	return matchDecisionOption(text, options, false)
}

// settleSupervisorChoice records a unique exact option label typed in the supervisor
// conversation, once its send was delivered, and delivers it to the worker Agent. chooseOption is the same delivery path the
// card and task_decided (told_agent=false) use: it relays the choice to the task's Agent before
// recording it, and askRound revalidates the whole round under the lock (acceptAskEnded, the
// latest review's staleness, RecheckAt and ownerDecisionReturned), so neither an undelivered
// record-only close nor a superseded round can settle. The action lock here only fences a
// concurrent deadline/sweep; chooseOption takes the same per-card lock itself, so it runs after
// this one is released (the locks are not reentrant).
func (s *Service) settleSupervisorChoice(ctx context.Context, id string, at time.Time, text string) {
	s.reviewDismissals.actions.acquire(id)
	cur, ok := s.todoNow(id)
	if !ok || cur.DoneAt != nil || cur.Supervision == nil || cur.Supervision.State != reviewNeedsUser {
		s.reviewDismissals.actions.release(id)
		return
	}
	a, ok := s.askCopy(id)
	s.reviewDismissals.actions.release(id)
	if !ok || !decisionOpen(&a, cur) || !a.RecheckAt.IsZero() || ownerDecisionReturned(cur.Supervision) ||
		!at.After(a.AskedAt) {
		return
	}
	label := supervisorOptionLabel(text, a.Options)
	if label == "" {
		return
	}
	if _, err := s.chooseOption(ctx, id, "", label); err != nil {
		s.logf("task %s: decision: deliver supervisor choice: %v", id, err)
	}
}

// handBackSupervisorReplies pauses each candidate's default and emits a decision_reply signal
// that routes back to this conversation's supervisor digest. No option is inferred: the
// supervisor reads the owner's words and confirms with task_decided or clarifies.
func (s *Service) handBackSupervisorReplies(ctx context.Context, r SessionRef, at time.Time, todos []Todo, text string) {
	for _, t := range todos {
		s.handBackOneSupervisorReply(ctx, r, at, t.ID, text)
	}
}

func (s *Service) handBackOneSupervisorReply(ctx context.Context, r SessionRef, at time.Time, id, text string) {
	s.reviewDismissals.actions.acquire(id)
	defer s.reviewDismissals.actions.release(id)
	cur, ok := s.todoNow(id)
	if !ok || cur.DoneAt != nil || cur.Supervision == nil || cur.Supervision.State != reviewNeedsUser {
		return
	}
	b := &s.accept
	b.mu.Lock()
	if !s.loadAcceptLocked(ctx) {
		b.mu.Unlock()
		return
	}
	a := b.asks[id]
	if a == nil || !decideAsk(a) || a.AskedAt.IsZero() || !a.AcceptedAt.IsZero() ||
		!at.After(a.AskedAt) || acceptAskStale(a, cur) || acceptAskEnded(a, cur) || !a.RecheckAt.IsZero() ||
		ownerDecisionReturned(cur.Supervision) {
		b.mu.Unlock()
		return
	}
	if a.AgentReply != nil { // a prior reply already paused this round; keep the first hand-off
		b.mu.Unlock()
		return
	}
	a.AgentReply = &decisionAgentReply{MachineID: r.MachineID, SessionID: r.SessionID, At: at.UTC(), Text: clip(text, 2000)}
	a.DueAt = at.UTC()
	// Even a failed write must not let this process take the default; the in-memory pause stands and the sweep retries.
	if err := s.saveAcceptErrLocked(ctx); err != nil {
		s.logf("task %s: decision: persist supervisor reply: %v", id, err)
	}
	cp := *a
	b.mu.Unlock()
	s.askChanged(ctx)
	s.signalSupervisorDecisionReply(ctx, id, cp)
	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		s.updateAskCards(ctx, id, cp)
	}()
}

// Exact choices typed in a supervisor conversation wait here for their send's outcome (#13041). The book is memory
// only on purpose: nothing about the decision is persisted until delivery settles it, so a manager restart while one
// waits leaves the card exactly as the accept ledger has it (still waiting, its default as before), and a delivered
// notice the new process cannot match settles nothing.
const (
	// supervisorSettleTTL outlives the hub's longest wait for a send (five minutes to a session still starting, plus
	// the manager's preparation): an entry whose notice never comes (a send the manager refused or kept to send
	// later, a hub restart) stops holding the default after this.
	supervisorSettleTTL = 10 * time.Minute
	// supervisorSettleMax bounds the book: each entry is one owner message waiting for its own send.
	supervisorSettleMax = 64
)

type supervisorSettle struct {
	todo    string
	at      time.Time // when the owner spoke: a round asked after it is not the one answered
	text    string
	expires time.Time
}

type supervisorSettleBook struct {
	mu      sync.Mutex
	pending map[[3]string]supervisorSettle // machine, session, client_msg_id
}

func supervisorSettleKey(r SessionRef, clientMsgID string) [3]string {
	return [3]string{r.MachineID, r.SessionID, clientMsgID}
}

// pruneLocked drops expired entries and, when the book is full, the ones closest to expiring.
func (b *supervisorSettleBook) pruneLocked(now time.Time, room int) {
	for k, p := range b.pending {
		if !now.Before(p.expires) {
			delete(b.pending, k)
		}
	}
	for len(b.pending) > supervisorSettleMax-room {
		var oldest [3]string
		var at time.Time
		for k, p := range b.pending {
			if at.IsZero() || p.expires.Before(at) {
				oldest, at = k, p.expires
			}
		}
		delete(b.pending, oldest)
	}
}

// deferSupervisorSettle keeps an exact choice until the hub says how its send went. While it waits, the decision's
// deadline default holds (supervisorSettlePending): the owner has answered, and the answer may only be in flight.
func (s *Service) deferSupervisorSettle(r SessionRef, clientMsgID, todo string, at time.Time, text string) {
	b := &s.supervisorSettles
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending == nil {
		b.pending = map[[3]string]supervisorSettle{}
	}
	now := time.Now()
	b.pruneLocked(now, 1)
	b.pending[supervisorSettleKey(r, clientMsgID)] = supervisorSettle{todo: todo, at: at, text: text,
		expires: now.Add(supervisorSettleTTL)}
}

// supervisorSettlePending says whether an exact choice for to-do id still waits for its send's outcome.
func (s *Service) supervisorSettlePending(id string, now time.Time) bool {
	b := &s.supervisorSettles
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, p := range b.pending {
		if p.todo == id && now.Before(p.expires) {
			return true
		}
	}
	return false
}

// supervisorSettleOutcome settles a waiting exact choice whose send was delivered; a failed or unknown send drops it
// and leaves the card waiting (the owner's words may not be in the conversation, so they decide nothing).
func (s *Service) supervisorSettleOutcome(ctx context.Context, r ConversationSent) {
	if r.ClientMsgID == "" {
		return
	}
	b := &s.supervisorSettles
	b.mu.Lock()
	key := supervisorSettleKey(SessionRef{MachineID: r.MachineID, SessionID: r.SessionID}, r.ClientMsgID)
	p, ok := b.pending[key]
	delete(b.pending, key)
	b.mu.Unlock()
	if !ok || !time.Now().Before(p.expires) {
		return
	}
	if r.Outcome != assistantwire.ConversationDelivered {
		s.logf("task %s: decision: supervisor choice not settled, its send was %s", p.todo, r.Outcome)
		return
	}
	s.settleSupervisorChoice(ctx, p.todo, p.at, p.text)
}
