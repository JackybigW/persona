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
// the hub keeps for its manager to send later (undelivered.go) settles when that kept message goes
// in or is found in the conversation; one that is never kept, fails or is deleted leaves the card
// waiting. A spoke without an id (an older hub, or a client that names no
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
		s.holdSupervisorChoice(ctx, r, clientMsgID, matched[0].ID, at, text)
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
// this one is released (the locks are not reentrant). asked is the round the owner answered: a round asked since is
// another question, which their words never settle (chooseOption checks it again under the lock).
func (s *Service) settleSupervisorChoice(ctx context.Context, id string, asked, at time.Time, text string) {
	s.reviewDismissals.actions.acquire(id)
	cur, ok := s.todoNow(id)
	if !ok || cur.DoneAt != nil || cur.Supervision == nil || cur.Supervision.State != reviewNeedsUser {
		s.reviewDismissals.actions.release(id)
		return
	}
	a, ok := s.askCopy(id)
	s.reviewDismissals.actions.release(id)
	if !ok || !decisionOpen(&a, cur) || !a.RecheckAt.IsZero() || ownerDecisionReturned(cur.Supervision) ||
		!at.After(a.AskedAt) || !a.AskedAt.Equal(asked) {
		return
	}
	label := supervisorOptionLabel(text, a.Options)
	if label == "" {
		return
	}
	if _, err := s.chooseOption(ctx, id, asked.UTC().Format(timeKeyLayout), label); err != nil {
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
// notice, or a kept message going in, that the new process cannot match settles nothing.
//
// A send that does not go in at once is not always over: for a failed send the conversation refused before typing
// (busy, a 5xx) or one whose outcome is unknown, the hub hands the same message to keepUndelivered right after the
// notice, and the manager sends it later (undeliveredSend → deliverKept) or finds it in the conversation
// (undeliveredLook). Neither tells session.assistant_sent again, so the entry stays, marked kept: it holds the default
// while the kept message waits and settles where that message goes in (takeKeptSupervisorSettleLocked). It is
// dropped when the kept message fails (given up, the conversation gone, the device signed out, not found after a try
// in doubt), the owner deletes it, or the hub never keeps it (supervisorKeepGrace after the notice).
//
// Locks, outermost first: the card's action lock (s.reviewDismissals.actions), then s.undelivered.mu, then the book's
// own mutex. The accept ledger's mutex (s.accept.mu) may be taken under the action lock, and this file never holds
// it together with the other two. A choice is registered under its card's action lock (holdSupervisorChoice), the
// lock the deadline default holds while it checks the book once more and takes the recommendation (acceptDefault):
// a choice either is in the book before the default looks, or finds the round answered and is not registered.
const (
	// supervisorSettleTTL outlives the hub's longest wait for a send (five minutes to a session still starting, plus
	// the manager's preparation): an entry whose notice never comes (a send the manager refused, a hub restart)
	// stops holding the default after this. A kept entry lives as long as its kept message may still be sent.
	supervisorSettleTTL = 10 * time.Minute
	// supervisorSettleMax bounds the book: each entry is one owner message waiting for its own send.
	supervisorSettleMax = 64
	// supervisorKeepGrace is how long after a failed or unknown notice the kept message may still arrive: the hub's
	// keepUndelivered call follows the notice and waits up to ten seconds, and the two travel separately.
	supervisorKeepGrace = 30 * time.Second
)

type supervisorSettle struct {
	todo    string
	asked   time.Time // the round answered (its ask's AskedAt): it holds no other round's default and settles no other
	at      time.Time // when the owner spoke: a round asked after it is not the one answered
	text    string
	expires time.Time
	// kept: the send did not go in at once, and the hub keeps (or is about to keep) the message to send later; the
	// entry then lives only while that kept message waits. keptSeen: the keep reached this manager. Until then, and
	// until keepBy, a missing kept message may only be on its way.
	kept     bool
	keptSeen bool
	keepBy   time.Time
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

// holdSupervisorChoice registers the owner's exact choice of to-do id's decision, written in r as message
// clientMsgID, to settle when that message is delivered. It runs under the card's action lock, which the deadline
// default holds too (acceptDefault): once this returns the default finds the choice, and a default that got there
// first has answered the round, so nothing is registered for it.
func (s *Service) holdSupervisorChoice(ctx context.Context, r SessionRef, clientMsgID, id string, at time.Time, text string) {
	s.reviewDismissals.actions.acquire(id)
	defer s.reviewDismissals.actions.release(id)
	cur, ok := s.todoNow(id)
	if !ok || cur.DoneAt != nil || cur.Supervision == nil || cur.Supervision.State != reviewNeedsUser {
		return
	}
	a, ok := s.askCopy(id)
	if !ok || !decisionOpen(&a, cur) || !a.RecheckAt.IsZero() || ownerDecisionReturned(cur.Supervision) ||
		!at.After(a.AskedAt) || supervisorOptionLabel(text, a.Options) == "" {
		return
	}
	s.deferSupervisorSettle(ctx, r, clientMsgID, id, a.AskedAt, at, text)
}

// deferSupervisorSettle keeps an exact choice of the round asked until the hub says how its send went. While it
// waits, that round's deadline default holds (supervisorSettlePending): the owner has answered, and the answer may
// only be in flight.
//
// The hub tells session.user_spoke for every try of a message, also a retry it then answers from its own record of
// the send (delivered, or kept by the manager), which tells no session.assistant_sent again. So the same message
// told again changes nothing: the entry keeps what is known of it (kept, and for how long it may wait). A message
// with no entry that the manager already keeps (the entry was lost, or the choice only matches now) waits on that
// kept message from the start, not on a notice that will not come.
func (s *Service) deferSupervisorSettle(ctx context.Context, r SessionRef, clientMsgID, todo string, asked, at time.Time, text string) {
	key := supervisorSettleKey(r, clientMsgID)
	// s.undelivered.mu stays held over the registration: the kept message cannot go in between the look and the entry
	// (it takes the entry under this lock, takeKeptSupervisorSettleLocked).
	u := &s.undelivered
	u.mu.Lock()
	defer u.mu.Unlock()
	kept := false
	if s.undeliveredLocked(ctx) {
		m := s.keptLocked(r.SessionID, clientMsgID)
		kept = m != nil && m.Machine == r.MachineID && m.State == proto.UndeliveredWaiting
	}
	b := &s.supervisorSettles
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending == nil {
		b.pending = map[[3]string]supervisorSettle{}
	}
	now := time.Now()
	b.pruneLocked(now, 0)
	if _, ok := b.pending[key]; ok {
		return
	}
	b.pruneLocked(now, 1)
	p := supervisorSettle{todo: todo, asked: asked, at: at, text: text, expires: now.Add(supervisorSettleTTL)}
	if kept {
		p.kept, p.keptSeen, p.expires = true, true, now.Add(undeliveredGiveUp+supervisorSettleTTL)
	}
	b.pending[key] = p
}

// supervisorSettlePending says whether an exact choice of the round to-do id's decision asks now still waits for its
// send's outcome. A choice of an earlier round answers a question that is no longer asked: it is dropped here, and
// never holds the default of the round that replaced it.
func (s *Service) supervisorSettlePending(id string, now time.Time) bool {
	s.reapSupervisorSettles(now)
	cur, asking := s.askCopy(id) // before the book's lock: s.accept.mu and the book are never held together
	b := &s.supervisorSettles
	b.mu.Lock()
	defer b.mu.Unlock()
	pending := false
	for k, p := range b.pending {
		if p.todo != id || !now.Before(p.expires) {
			continue
		}
		// Rounds only move on: an entry of a round asked after the one read here is the current one's.
		if asking && p.asked.Before(cur.AskedAt) {
			delete(b.pending, k)
			continue
		}
		pending = true
	}
	return pending
}

// markSupervisorSettleKept marks the waiting choice clientMsgID in r as kept for a later send: keepBy is until when
// its kept message may still be missing (the notice can come before the keep); zero, the keep itself reached this
// manager. Nothing waits under that id: nothing.
func (s *Service) markSupervisorSettleKept(r SessionRef, clientMsgID string, keepBy time.Time) {
	if clientMsgID == "" {
		return
	}
	b := &s.supervisorSettles
	b.mu.Lock()
	defer b.mu.Unlock()
	key := supervisorSettleKey(r, clientMsgID)
	p, ok := b.pending[key]
	now := time.Now()
	if !ok || !now.Before(p.expires) {
		return
	}
	p.kept = true
	if keepBy.IsZero() {
		p.keptSeen = true
	} else if keepBy.After(p.keepBy) {
		p.keepBy = keepBy
	}
	// A kept message is tried for a day before it is given up; the reap ends the entry when it fails sooner.
	if e := now.Add(undeliveredGiveUp + supervisorSettleTTL); e.After(p.expires) {
		p.expires = e
	}
	b.pending[key] = p
}

// reapSupervisorSettles drops expired entries and kept ones whose kept message no longer waits: it failed, the owner
// deleted it, or the hub never kept it. One that went in was taken when it went in. This book's mutex is never held
// while taking the undelivered side's (that side takes this book under its own lock), so each is read on its own; a
// kept message that goes in between the two reads was taken with its drop, so nothing it settles is lost here.
func (s *Service) reapSupervisorSettles(now time.Time) {
	b := &s.supervisorSettles
	b.mu.Lock()
	b.pruneLocked(now, 0)
	type keptEntry struct {
		key     [3]string
		waiting bool // the kept message may still be on its way: missing is not gone yet
	}
	var kept []keptEntry
	for k, p := range b.pending {
		if p.kept {
			kept = append(kept, keptEntry{k, !p.keptSeen && now.Before(p.keepBy)})
		}
	}
	b.mu.Unlock()
	if len(kept) == 0 {
		return
	}
	u := &s.undelivered
	u.mu.Lock()
	var gone [][3]string
	if s.undeliveredLocked(context.Background()) {
		for _, e := range kept {
			m := s.keptLocked(e.key[1], e.key[2])
			if m != nil && m.Machine != e.key[0] {
				m = nil
			}
			if (m == nil && !e.waiting) || (m != nil && m.State != proto.UndeliveredWaiting) {
				gone = append(gone, e.key)
			}
		}
	}
	u.mu.Unlock()
	if len(gone) == 0 {
		return
	}
	b.mu.Lock()
	for _, k := range gone {
		if p, ok := b.pending[k]; ok {
			delete(b.pending, k)
			s.logf("task %s: decision: supervisor choice not settled, its kept message did not go in", p.todo)
		}
	}
	b.mu.Unlock()
}

// takeSupervisorSettle removes and returns the live entry for key.
func (s *Service) takeSupervisorSettle(key [3]string) (supervisorSettle, bool) {
	b := &s.supervisorSettles
	b.mu.Lock()
	defer b.mu.Unlock()
	p, ok := b.pending[key]
	delete(b.pending, key)
	return p, ok && time.Now().Before(p.expires)
}

// takeKeptSupervisorSettleLocked takes the choice waiting on kept message k, which just went in (sent, or found in
// the conversation); s.undelivered.mu is held, so the take and the kept message's drop are one step for the reap.
func (s *Service) takeKeptSupervisorSettleLocked(k keptMessage) (supervisorSettle, bool) {
	return s.takeSupervisorSettle(supervisorSettleKey(SessionRef{MachineID: k.Machine, SessionID: k.Session}, k.ID))
}

// settleKeptSupervisorChoice settles a choice taken from a kept message that went in.
func (s *Service) settleKeptSupervisorChoice(ctx context.Context, p supervisorSettle, ok bool) {
	if ok {
		s.settleSupervisorChoice(context.WithoutCancel(ctx), p.todo, p.asked, p.at, p.text)
	}
}

// supervisorSettleOutcome settles a waiting exact choice whose send was delivered. A failed or unknown send leaves
// the card waiting; its entry stays, marked kept, in case the hub keeps the message to send later
// (markSupervisorSettleKept), and goes when the hub does not.
func (s *Service) supervisorSettleOutcome(ctx context.Context, r ConversationSent) {
	if r.ClientMsgID == "" {
		return
	}
	ref := SessionRef{MachineID: r.MachineID, SessionID: r.SessionID}
	if r.Outcome != assistantwire.ConversationDelivered {
		s.markSupervisorSettleKept(ref, r.ClientMsgID, time.Now().Add(supervisorKeepGrace))
		return
	}
	if p, ok := s.takeSupervisorSettle(supervisorSettleKey(ref, r.ClientMsgID)); ok {
		s.settleSupervisorChoice(ctx, p.todo, p.asked, p.at, p.text)
	}
}
