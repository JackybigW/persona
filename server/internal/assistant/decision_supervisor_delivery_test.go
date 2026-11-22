package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/assistantwire"
)

// Exact choices typed in a supervisor / direction conversation settle only once the hub says their send was
// delivered (#13041). These tests drive the manager the way the hub does: session.user_spoke as a bus event (raw
// JSON, so a client_msg_id an older manager does not know is simply ignored), then session.assistant_sent.

// spokeOnBus delivers session.user_spoke as the hub sends it; id "" leaves client_msg_id out, as an older hub does.
func spokeOnBus(t *testing.T, s *Service, r SessionRef, text, id string) {
	t.Helper()
	params := map[string]any{"machine_id": r.MachineID, "session_id": r.SessionID, "text": text}
	if id != "" {
		params["client_msg_id"] = id
	}
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	b := &BusHub{svc: s}
	if err := b.event(context.Background(), busEvent{name: assistantwire.EventUserSpoke, params: raw}); err != nil {
		t.Fatal(err)
	}
}

// sentNotice is the hub's session.assistant_sent for the message id sent to r.
func sentNotice(s *Service, r SessionRef, id, outcome string) {
	s.ConversationSent(context.Background(), ConversationSent{MachineID: r.MachineID, SessionID: r.SessionID,
		Outcome: outcome, ClientMsgID: id})
}

// deliveredSupervisorChoice is an owner's choice typed in conversation r whose send was delivered.
func deliveredSupervisorChoice(t *testing.T, s *Service, r SessionRef, text, id string) {
	t.Helper()
	spokeOnBus(t, s, r, text, id)
	sentNotice(s, r, id, assistantwire.ConversationDelivered)
}

// stillWaiting fails unless to-do id's decision is still open: not accepted, nothing relayed since sent, needs_user.
func stillWaiting(t *testing.T, s *Service, f *fakeHub, id string, sent int, when string) {
	t.Helper()
	if got := askOf(s, id); got == nil || !got.AcceptedAt.IsZero() || got.Chosen != "" || got.ByDefault {
		t.Fatalf("%s: the decision closed: %+v", when, got)
	}
	if delivered := sentToAgent(f)[sent:]; len(delivered) != 0 {
		t.Fatalf("%s: the worker was told a choice: %q", when, delivered)
	}
	if todoNow(s, id).Supervision.State != reviewNeedsUser {
		t.Fatalf("%s: the decision left needs_user", when)
	}
}

// The spoke alone settles nothing; the matching delivered notice settles the owner's option and relays it.
func TestSupervisorChoiceSettlesOnlyWhenDelivered(t *testing.T) {
	s, f, todo := decisionAgentReplyService(t)
	want := askOf(s, todo.ID).Options[1].Label
	sent := len(sentToAgent(f))
	spokeOnBus(t, s, supervisorRef(), want, "msg-delivered")
	stillWaiting(t, s, f, todo.ID, sent, "before delivery")
	if got := askOf(s, todo.ID); got.AgentReply != nil {
		t.Fatalf("an exact choice waiting for delivery was handed back: %+v", got.AgentReply)
	}
	// Another message's outcome, or the same id in another conversation, is not this one's.
	sentNotice(s, supervisorRef(), "msg-other", assistantwire.ConversationDelivered)
	sentNotice(s, SessionRef{MachineID: "server", SessionID: "conv2"}, "msg-delivered", assistantwire.ConversationDelivered)
	stillWaiting(t, s, f, todo.ID, sent, "after another message's outcome")
	sentNotice(s, supervisorRef(), "msg-delivered", assistantwire.ConversationDelivered)
	got := askOf(s, todo.ID)
	if got == nil || got.AcceptedAt.IsZero() || got.Chosen != want || got.ByDefault {
		t.Fatalf("the delivered choice did not settle: %+v", got)
	}
	if delivered := sentToAgent(f)[sent:]; len(delivered) != 1 || !strings.Contains(delivered[0], want) {
		t.Fatalf("the delivered choice was not relayed to the worker: %q", delivered)
	}
	if s.supervisorSettlePending(todo.ID, time.Now()) {
		t.Fatal("the settled choice still waits")
	}
}

// A failed or unknown send leaves the card waiting, and the default works as before; a late delivered notice for the
// dropped message settles nothing.
func TestSupervisorChoiceUndeliveredLeavesCardWaiting(t *testing.T) {
	for _, outcome := range []string{assistantwire.ConversationFailed, assistantwire.ConversationUnknown} {
		t.Run(outcome, func(t *testing.T) {
			s, f, todo := decisionAgentReplyService(t)
			want := askOf(s, todo.ID).Options[1].Label
			sent := len(sentToAgent(f))
			spokeOnBus(t, s, supervisorRef(), want, "msg-"+outcome)
			sentNotice(s, supervisorRef(), "msg-"+outcome, outcome)
			stillWaiting(t, s, f, todo.ID, sent, outcome)
			sentNotice(s, supervisorRef(), "msg-"+outcome, assistantwire.ConversationDelivered)
			stillWaiting(t, s, f, todo.ID, sent, "a late delivered notice after "+outcome)
			if s.supervisorSettlePending(todo.ID, time.Now()) {
				t.Fatal("the dropped choice still holds the default")
			}
			pastDeadline(t, s, todo.ID)
			s.sweepDecisionDeadlines(time.Now())
			if got := askOf(s, todo.ID); got == nil || !got.ByDefault {
				t.Fatalf("the default no longer works after an undelivered choice: %+v", got)
			}
		})
	}
}

// While the choice is on its way, the deadline default does not run: the owner answered, the send just has not
// finished. Delivery then settles the owner's option, not the recommendation.
func TestSupervisorChoiceHoldsDefaultUntilDelivered(t *testing.T) {
	s, f, todo := decisionAgentReplyService(t)
	a := askOf(s, todo.ID)
	want := a.Options[1].Label
	if want == a.Recommend {
		t.Fatalf("fixture: option %q is the recommendation", want)
	}
	sent := len(sentToAgent(f))
	spokeOnBus(t, s, supervisorRef(), want, "msg-slow")
	pastDeadline(t, s, todo.ID)
	s.sweepDecisionDeadlines(time.Now())
	stillWaiting(t, s, f, todo.ID, sent, "the deadline passing while the send runs")
	sentNotice(s, supervisorRef(), "msg-slow", assistantwire.ConversationDelivered)
	if got := askOf(s, todo.ID); got == nil || got.Chosen != want || got.ByDefault {
		t.Fatalf("delivery did not settle the owner's choice: %+v", got)
	}
}

// A choice whose notice never comes (a refused or kept send has none) stops holding the default after its time, and
// the book never grows past its bound.
func TestSupervisorChoiceExpiresAndStaysBounded(t *testing.T) {
	s, _, todo := decisionAgentReplyService(t)
	spokeOnBus(t, s, supervisorRef(), askOf(s, todo.ID).Options[1].Label, "msg-never-told")
	if !s.supervisorSettlePending(todo.ID, time.Now()) {
		t.Fatal("the exact choice is not waiting for its send")
	}
	later := time.Now().Add(supervisorSettleTTL + time.Second)
	if s.supervisorSettlePending(todo.ID, later) {
		t.Fatal("the choice holds the default past its time")
	}
	pastDeadline(t, s, todo.ID)
	s.sweepDecisionDeadlines(later)
	if got := askOf(s, todo.ID); got == nil || !got.ByDefault {
		t.Fatalf("an expired choice kept holding the default: %+v", got)
	}

	for i := range supervisorSettleMax + 10 {
		s.deferSupervisorSettle(supervisorRef(), fmt.Sprintf("flood-%d", i), "todo_other", time.Now(), "x")
	}
	s.supervisorSettles.mu.Lock()
	n := len(s.supervisorSettles.pending)
	s.supervisorSettles.mu.Unlock()
	if n > supervisorSettleMax {
		t.Fatalf("the book grew to %d entries", n)
	}
}

// An older hub's spoke has no client_msg_id: the outcome can never be matched, so an exact label is handed back like
// free text (the default pauses, the supervisor confirms) instead of closing the card on a send that may fail.
func TestSupervisorChoiceFromOlderHubHandsBack(t *testing.T) {
	s, f, todo := decisionAgentReplyService(t)
	sent := len(sentToAgent(f))
	spokeOnBus(t, s, supervisorRef(), askOf(s, todo.ID).Options[1].Label, "")
	stillWaiting(t, s, f, todo.ID, sent, "an id-less spoke")
	if got := askOf(s, todo.ID); got.AgentReply == nil || got.DueAt.IsZero() {
		t.Fatalf("an id-less exact choice did not pause the default: %+v", got)
	}
	waitFor(t, "the id-less hand-off", func() bool {
		return signalled(s, f, todo.ID, sigDecisionReply, "task_decided")
	})
	sentNotice(s, supervisorRef(), "", assistantwire.ConversationDelivered)
	stillWaiting(t, s, f, todo.ID, sent, "an id-less delivered notice")
}

// The Agent-chat path keeps pausing the default on the spoke alone, before any delivery result, whether or not the
// hub names the message.
func TestAgentChatSpokePausesDefaultBeforeDelivery(t *testing.T) {
	for _, id := range []string{"", "agent-msg-1"} {
		t.Run("id="+id, func(t *testing.T) {
			s, f, todo := decisionAgentReplyService(t)
			a := askOf(s, todo.ID)
			sent := len(sentToAgent(f))
			spokeOnBus(t, s, SessionRef{MachineID: a.MachineID, SessionID: a.SessionID}, "let me check the dashboards first", id)
			if got := askOf(s, todo.ID); got.AgentReply == nil || got.DueAt.IsZero() {
				t.Fatalf("the Agent-chat spoke did not pause the default: %+v", got)
			}
			pastDeadline(t, s, todo.ID)
			s.sweepDecisionDeadlines(time.Now().Add(48 * time.Hour))
			stillWaiting(t, s, f, todo.ID, sent, "the deadline after an Agent-chat reply")
		})
	}
}

// A manager restart while a choice waits for its send: nothing was written, the reloaded manager has no record of
// it, and the delivered notice reaching the new process settles nothing. The card stays waiting.
func TestSupervisorChoiceRestartWhileWaitingKeepsCard(t *testing.T) {
	presenceTimes(t, 0, 0)
	ownerTurnTimes(t, 0, 0)
	database := todoTestDB(t)
	s, f, todo := newSupervisedServiceOn(t, database)
	waitsForReview(t, s, f, todo.ID, "The agent needs a choice between two rollout options.")
	reviewTask(t, s, todo.ID, rolloutDecision())
	spokeOnBus(t, s, supervisorRef(), askOf(s, todo.ID).Options[1].Label, "msg-restart")
	var stored map[string]*acceptAsk
	if _, err := database.Setting(context.Background(), acceptanceKey, &stored); err != nil {
		t.Fatal(err)
	}
	if a := stored[todo.ID]; a == nil || !a.AcceptedAt.IsZero() || a.Chosen != "" {
		t.Fatalf("a choice waiting for delivery was persisted as settled: %+v", a)
	}
	reloaded, nextHub := newQuietServiceOn(t, database)
	if err := reloaded.ensureTodosLoaded(context.Background()); err != nil {
		t.Fatal(err)
	}
	if a, ok := reloaded.askCopy(todo.ID); !ok || !a.AcceptedAt.IsZero() { // loads the persisted ledger
		t.Fatalf("the reloaded ledger lost the waiting decision: %+v", a)
	}
	sent := len(sentToAgent(nextHub))
	sentNotice(reloaded, supervisorRef(), "msg-restart", assistantwire.ConversationDelivered)
	stillWaiting(t, reloaded, nextHub, todo.ID, sent, "after a restart")
}
