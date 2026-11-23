package node

import (
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nwverse/persona/server/internal/proto"
)

func TestCodexIndexListsHealthyHomeWhileAnotherScanIsBlocked(t *testing.T) {
	n := remoteTestNode(t)
	slowRoot := t.TempDir()
	slowHome := canonicalCodexHome(filepath.Join(slowRoot, ".codex"), n.home)
	const slowID = "aaaa0000-0000-4000-8000-000000000001"
	const healthyID = "bbbb0000-0000-4000-8000-000000000002"
	writeThread(t, slowRoot, slowID, n.home, "first")
	writeThread(t, n.home, healthyID, n.home, "second")
	n.configureCodex(proto.CodexConfig{Homes: []string{slowHome}})
	slow := rolloutCacheFor(slowHome)
	slow.mu.Lock() // simulate a filesystem read that never returns on its own
	var release sync.Once
	unblock := func() { release.Do(slow.mu.Unlock) }
	defer unblock()
	go n.indexCodexHome(slowHome, time.Now().Add(codexHomeListWait)) // the background scan can be holding it before a list arrives
	has := func(rows []proto.Session, id string) bool {
		return slices.ContainsFunc(rows, func(s proto.Session) bool { return s.SourceThreadID == id })
	}
	done := make(chan []proto.Session, 1)
	go func() { done <- n.codexSessions(map[string]bool{}) }()
	select {
	case rows := <-done:
		if !has(rows, healthyID) || has(rows, slowID) {
			t.Fatalf("listing while a home is blocked: %d rows, healthy=%v, blocked=%v", len(rows), has(rows, healthyID), has(rows, slowID))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a blocked Codex home stopped the session list")
	}
	unblock()
	deadline := time.After(2 * time.Second)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		rows := n.codexSessions(map[string]bool{})
		if has(rows, slowID) && has(rows, healthyID) {
			return
		}
		select {
		case <-deadline:
			t.Fatal("the released home did not return to the list")
		case <-tick.C:
		}
	}
}

func TestCodexIndexSharesBlockedWork(t *testing.T) {
	var index codexHomeIndex
	var calls atomic.Int32
	release := make(chan struct{})
	finished := make(chan struct{})
	read := func() time.Duration {
		calls.Add(1)
		<-release
		close(finished)
		return time.Second
	}
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	for range 4 {
		if index.available("slow", time.Now().Add(codexHomeListWait), read) {
			t.Fatal("an unfinished scan was reported ready")
		}
	}
	if calls.Load() != 1 || !index.available("healthy", time.Now().Add(codexHomeListWait), func() time.Duration { return time.Second }) {
		t.Fatalf("blocked scan ran %d times or prevented an unrelated scan", calls.Load())
	}
	unblock()
	<-finished
}

func TestCodexIndexUsesOneWaitBudgetForAList(t *testing.T) {
	var index codexHomeIndex
	release := make(chan struct{})
	var stop sync.Once
	unblock := func() { stop.Do(func() { close(release) }) }
	defer unblock()
	scan := func() time.Duration {
		<-release
		return time.Second
	}
	done := make(chan struct{})
	go func() {
		deadline := time.Now().Add(codexHomeScanWait)
		for j := range 8 {
			index.available(strconv.Itoa(j), deadline, scan)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * codexHomeScanWait):
		unblock()
		<-done
		t.Fatal("the list paid the wait budget separately for each blocked home")
	}
}

func TestCodexIndexRetainsObservedRowsWhenWorkersAreBusy(t *testing.T) {
	n := remoteTestNode(t)
	const id = "cccc0000-0000-4000-8000-000000000003"
	writeThread(t, n.home, id, n.home, "keep this history")
	rows := n.codexSessions(map[string]bool{})
	if len(rows) != 1 || rows[0].SessionID != id {
		t.Fatalf("initial list = %v", rows)
	}
	home := canonicalCodexHome(n.codexHome(), n.home)
	n.codices.index.mu.Lock()
	n.codices.index.homes[home].fresh = time.Time{}
	n.codices.index.mu.Unlock()
	acquired := 0
fill:
	for {
		select {
		case codexHomeScanSlots <- struct{}{}:
			acquired++
		default:
			break fill
		}
	}
	defer func() {
		for range acquired {
			<-codexHomeScanSlots
		}
	}()
	seen := map[string]bool{}
	got := n.codexSessions(seen)
	if len(got) != 1 || got[0].SessionID != id || got[0].Title != rows[0].Title || got[0].Status != proto.StatusUnknown ||
		got[0].Capabilities == nil || got[0].Capabilities.Send || !seen[id] {
		t.Fatalf("a busy index lost the last observation or offered stale controls: %v", got)
	}
	if dupe := n.codexSessions(map[string]bool{id: true}); len(dupe) != 0 {
		t.Fatalf("a live row was duplicated with stale history: %v", dupe)
	}
}
