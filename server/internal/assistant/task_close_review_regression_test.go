package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/proto"
)

func TestTaskJudgmentDefersAndRetriesOwnerActivity(t *testing.T) {
	for _, supervised := range []bool{false, true} {
		for _, replay := range []bool{false, true} {
			for _, activity := range []string{"message", "view"} {
				t.Run(fmt.Sprintf("supervised=%t/replay=%t/%s", supervised, replay, activity), func(t *testing.T) {
					shortTurnRetries(t, time.Nanosecond)
					s, f, todo := newSupervisedService(t)
					ctx := context.Background()
					if !supervised {
						s.todos.mu.Lock()
						s.todos.todos[todo.ID].Supervision = nil
						s.todos.mu.Unlock()
						todo = todoNow(s, todo.ID)
					}
					items := turnPage("finished-turn", "The change is finished and the PR is merged.", true)
					f.setItems(items...)
					row := turnRow(proto.StatusDone, 3, "The change is finished and the PR is merged.")
					nodeListed(s, f, row)
					end, _ := turnEndOf(items)
					work := todo.CreatedAt.Add(time.Millisecond)
					s.takeTurn(ctx, todo, row, items, end, true, work, turnDone)
					check := taskDoneCheck{Verdict: verdictDone, WouldClose: true, Reason: "Finished"}
					if replay {
						s.turnJudged(ctx, todo.ID, check, nil, "")
					}
					if activity == "view" {
						setTaskViewer(t, s, "box", "s1", true)
					} else {
						s.rememberSessionActivity(ctx, "box", "s1")
					}
					if replay {
						saved, _ := s.turnOf(ctx, todo.ID)
						work = saved.Judged.At.Add(time.Millisecond)
						s.turnAgain(ctx, todo, saved, row, row.LastText, work, true)
					} else {
						s.applyTaskDone(ctx, todo, work, true, check, nil)
					}
					// More ticks than the failed-judgment limit must still leave this retryable.
					sweeps(t, s, todo.ID, turnJudgeTries+2)
					if e, _ := s.turnOf(ctx, todo.ID); e.Judged != nil {
						t.Fatalf("owner activity settled the pending turn: %+v", e.Judged)
					}
					if now := todoNow(s, todo.ID); now.DoneAt != nil || now.Check != nil ||
						now.Supervision != nil && now.Supervision.State == reviewPending {
						t.Fatalf("owner activity applied the judgment: %+v", now)
					}
					f.mu.Lock()
					judgments := f.judged
					f.mu.Unlock()
					if judgments != 0 {
						t.Fatalf("activity-only retries called the model %d times", judgments)
					}

					if activity == "view" {
						setTaskViewer(t, s, "box", "s1", false)
					}
					s.taskActivity.mu.Lock()
					s.taskActivity.times[runtimeListID("box", "s1")] = time.Now().Add(-taskUserIdle - time.Second)
					s.taskActivity.mu.Unlock()
					// No row change or new turn: the existing sweep must retry the deferred judgment.
					sweeps(t, s, todo.ID, 2)
					now := todoNow(s, todo.ID)
					if supervised {
						if e, _ := s.turnOf(ctx, todo.ID); e.Judged == nil {
							t.Fatal("inactivity left the turn pending forever")
						}
						if now.DoneAt != nil || now.Supervision.State != reviewPending {
							t.Fatalf("inactivity did not restore supervised review: %+v", now)
						}
					} else if now.DoneAt == nil {
						t.Fatalf("inactivity did not close the unsupervised task: %+v", now)
					}
				})
			}
		}
	}
}

type taskRetirementBarrier struct {
	Hub
	entered chan struct{}
	release chan struct{}
}

func (h *taskRetirementBarrier) DeviceCall(ctx context.Context, p DeviceCall) (json.RawMessage, error) {
	if p.Method == deviceSessionStop {
		close(h.entered)
		select {
		case <-h.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return h.Hub.DeviceCall(ctx, p)
}

func TestClosedTaskRetirementRechecksAfterStop(t *testing.T) {
	for _, change := range []string{"new task", "reopen", "reclose", "view", "message"} {
		t.Run(change, func(t *testing.T) {
			ft, row := closeTestService(t, nil)
			ft.f.setSessions("mac", row)
			s, ctx := ft.s, context.Background()
			gate := &taskRetirementBarrier{Hub: s.hub, entered: make(chan struct{}), release: make(chan struct{})}
			s.hub = gate
			todo, err := s.createTodo(ctx, "Disposable task", "Check the result", "conv1", "")
			if err != nil {
				t.Fatal(err)
			}
			if err := s.linkTodoAgent(ctx, todo.ID, "mac", row.SessionID, row.Title, "/code", "claude"); err != nil {
				t.Fatal(err)
			}
			if ok, err := s.reviewDoneTodo(ctx, todo.ID, "Closed", true); err != nil || !ok {
				t.Fatalf("close: %v %v", ok, err)
			}
			s.todos.mu.Lock()
			old := time.Now().Add(-time.Hour)
			s.todos.todos[todo.ID].DoneAt = &old
			s.todos.mu.Unlock()
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				s.closeDoneTaskAgents(ctx, taskCloseRefused{})
			}()
			defer func() {
				close(gate.release)
				select {
				case <-finished:
				case <-time.After(5 * time.Second):
					t.Error("retirement did not finish")
				}
			}()
			select {
			case <-gate.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("session.stop was not reached")
			}
			switch change {
			case "new task":
				next, err := s.createTodo(ctx, "Next task", "Keep talking", "conv1", "")
				if err != nil {
					t.Fatal(err)
				}
				if err := s.linkTodoAgent(ctx, next.ID, "mac", row.SessionID, row.Title, "/code", "claude"); err != nil {
					t.Fatal(err)
				}
			case "reopen", "reclose":
				if err := s.reopenTodo(ctx, todo.ID); err != nil {
					t.Fatal(err)
				}
				if change == "reclose" {
					if ok, err := s.reviewDoneTodo(ctx, todo.ID, "Closed again", true); err != nil || !ok {
						t.Fatalf("reclose: %v %v", ok, err)
					}
				}
			case "view":
				setTaskViewer(t, s, "mac", row.SessionID, true)
			case "message":
				s.rememberSessionActivity(ctx, "mac", row.SessionID)
			}
			// Let session.stop succeed after the concurrent change, before checking archive state.
			select {
			case gate.release <- struct{}{}:
			case <-finished:
				t.Fatal("session.stop ended before the fixture released it")
			case <-time.After(5 * time.Second):
				t.Fatal("session.stop did not wait for the fixture")
			}
			select {
			case <-finished:
			case <-time.After(5 * time.Second):
				t.Fatal("session.stop did not return")
			}
			if len(retiredOf(ft)) != 1 {
				t.Fatal("fixture did not complete its stop RPC")
			}
			if archivedIDs(t, s)[row.SessionID] {
				t.Fatal("archived an Agent after ownership or activity changed during session.stop")
			}
		})
	}
}
