package lark

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/nwverse/persona/server/internal/botcard"
	"github.com/nwverse/persona/server/internal/channel"
)

// The fake core's side of a held /read (Host.HoldRead) is kept in this file, in a book of its own for each fake
// (heldOf), not in the fake's fields: service_test.go stays as the other turn tests have it.

// heldRead is one HoldRead call: the cards as JSON, their fallbacks and keys.
type heldRead struct {
	run       string
	cards     []string
	fallbacks []string
	keys      []string
	quiet     bool
}

// heldBook is what one fake core was handed to hold for the owner (HoldRead), and what HoldRead answers.
type heldBook struct {
	mu   sync.Mutex
	held []heldRead
	err  error
}

// heldBooks are the fakes' books (*Service -> *heldBook).
var heldBooks sync.Map

// heldOf is s's book, made on first use and dropped when its test ends.
func heldOf(s *Service) *heldBook {
	if b, ok := heldBooks.Load(s); ok {
		return b.(*heldBook)
	}
	b, loaded := heldBooks.LoadOrStore(s, &heldBook{})
	if !loaded {
		s.t.Cleanup(func() { heldBooks.Delete(s) })
	}
	return b.(*heldBook)
}

// list is what was handed over so far.
func (b *heldBook) list() []heldRead {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.held)
}

// answer sets what HoldRead answers from now on (nil: the answer is held).
func (b *heldBook) answer(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.err = err
}

// HoldRead keeps what the turn handed over, as the core would before posting it to the owner's private chat.
func (s *Service) HoldRead(_ context.Context, r *channel.Run, cards []*botcard.Card, fallbacks, keys []string, quiet bool) error {
	h := heldRead{run: r.ID, fallbacks: slices.Clone(fallbacks), keys: slices.Clone(keys), quiet: quiet}
	for _, c := range cards {
		h.cards = append(h.cards, c.JSON())
	}
	b := heldOf(s)
	b.mu.Lock()
	defer b.mu.Unlock()
	b.held = append(b.held, h)
	return b.err
}

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
			held := heldOf(s).list()
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
				if len(held) != 0 || len(group) != 1 || group[0]["reply_to"] != "om_m1" || group[0]["no_detour"] != true {
					t.Fatalf("held %+v, group posts %v", held, group)
				}
				return
			}
			if len(group) != 0 || len(*posts) != 0 {
				t.Fatalf("posted while held: %v", *posts)
			}
			if len(held) != 1 {
				t.Fatalf("held %+v", held)
			}
			h := held[0]
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
	heldOf(s).answer(errOffline)
	s.finishChannelRun(r, &channel.Result{Answer: "结论"})
	if got, _ := s.channelRunByID(r.ID); got == nil {
		t.Fatal("the run ended although its answer was not held")
	}
	if reacts := s.hub.(*fakeHub).appParams(AppFeishu, FeishuReact); len(reacts) != 0 {
		t.Fatalf("marked done before the answer was held: %v", reacts)
	}
	heldOf(s).answer(nil)
	s.finishChannelRun(r, &channel.Result{Answer: "结论"})
	if got, _ := s.channelRunByID(r.ID); got != nil {
		t.Fatal("the run did not end once its answer was held")
	}
	if held := heldOf(s).list(); len(held) != 2 || held[0].keys[0] != held[1].keys[0] {
		t.Fatalf("held %+v", held)
	}
}

// A recalled /read says nothing, held or not.
func TestReadAnswerHeldRecalledSaysNothing(t *testing.T) {
	s, posts, _ := cotHub(t, false)
	r, _ := cotRun(s)
	s.setRunExt(r, readChatKey, testChat)
	s.finishChannelRun(r, &channel.Result{Quiet: true, End: channel.EndRecalled})
	if held := heldOf(s).list(); len(held) != 0 || len(*posts) != 0 {
		t.Fatalf("held %+v, posts %v", held, *posts)
	}
}
