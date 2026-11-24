package lark

import (
	"strings"
	"testing"

	"github.com/nwverse/persona/server/internal/channel"
)

// A /read's answer is held for the owner unless the run was asked with the switch to answer in the group directly
// (#14437): the core gets the cards the group would get (no process, no web link), each under the key it goes into
// the group by, nothing goes into the group, and the request is marked DONE as before. With the switch the answer goes
// under the request in the group, as it did before.
func TestReadAnswerIsHeldForTheOwner(t *testing.T) {
	for _, direct := range []bool{false, true} {
		name := "held"
		if direct {
			name = "direct"
		}
		t.Run(name, func(t *testing.T) {
			s, posts, _ := cotHub(t, false)
			withWebBot(s.hub.(*fakeHub), "https://hub.example.com")
			r, _ := cotRun(s)
			s.setRunExt(r, readChatKey, testChat)
			if direct {
				s.setRunExt(r, readDirectKey, "1")
			}
			s.finishChannelRun(r, &channel.Result{Answer: "结论：同一次发布"})
			var group []map[string]any
			for _, p := range *posts {
				if p["chat_id"] == testChat {
					group = append(group, p)
				}
			}
			reacts := s.hub.(*fakeHub).appParams(AppFeishu, FeishuReact)
			if len(reacts) != 1 || reacts[0]["message_id"] != "om_m1" || reacts[0]["emoji"] != "DONE" {
				t.Fatalf("reactions %v", reacts)
			}
			if direct {
				if len(s.held) != 0 || len(group) != 1 || group[0]["reply_to"] != "om_m1" || group[0]["no_detour"] != true {
					t.Fatalf("held %+v, group posts %v", s.held, group)
				}
				return
			}
			if len(group) != 0 || len(*posts) != 0 {
				t.Fatalf("posted while held: %v", *posts)
			}
			if len(s.held) != 1 {
				t.Fatalf("held %+v", s.held)
			}
			h := s.held[0]
			if h.run != r.ID || h.quiet || len(h.cards) != 1 || len(h.keys) != 1 || h.keys[0] != readAnswerKey(r.ID, r.WaitFrom(), 0) {
				t.Fatalf("held %+v", h)
			}
			if c := h.cards[0]; !strings.Contains(c, "同一次发布") || strings.Contains(c, "hub.example.com") || strings.Contains(c, "ls") {
				t.Fatalf("the held card has the process or the web link: %s", c)
			}
		})
	}
}

// A held /read whose answer could not be handed over because the hub was unreachable is tried again, with the same
// keys; once it is, the run ends.
func TestReadAnswerHeldRetriesWhenUnreachable(t *testing.T) {
	s, _, _ := cotHub(t, false)
	r, _ := cotRun(s)
	s.setRunExt(r, readChatKey, testChat)
	s.heldErr = errOffline
	s.finishChannelRun(r, &channel.Result{Answer: "结论"})
	if got, _ := s.channelRunByID(r.ID); got == nil {
		t.Fatal("the run ended although its answer was not held")
	}
	if reacts := s.hub.(*fakeHub).appParams(AppFeishu, FeishuReact); len(reacts) != 0 {
		t.Fatalf("marked done before the answer was held: %v", reacts)
	}
	s.heldErr = nil
	s.finishChannelRun(r, &channel.Result{Answer: "结论"})
	if got, _ := s.channelRunByID(r.ID); got != nil {
		t.Fatal("the run did not end once its answer was held")
	}
	if len(s.held) != 2 || s.held[0].keys[0] != s.held[1].keys[0] {
		t.Fatalf("held %+v", s.held)
	}
}

// A recalled /read says nothing, held or not.
func TestReadAnswerHeldRecalledSaysNothing(t *testing.T) {
	s, posts, _ := cotHub(t, false)
	r, _ := cotRun(s)
	s.setRunExt(r, readChatKey, testChat)
	s.finishChannelRun(r, &channel.Result{Quiet: true, End: channel.EndRecalled})
	if len(s.held) != 0 || len(*posts) != 0 {
		t.Fatalf("held %+v, posts %v", s.held, *posts)
	}
}
