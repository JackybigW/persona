package node

import (
	"os"
	"sync"
	"time"

	"github.com/nwverse/persona/server/internal/proto"
)

// A discovered home can be on an offline mount or waiting for a folder permission.
// The OS may leave its open pending indefinitely. Keep one scan per home and bound
// the number of blocked filesystem workers across connections.
var codexHomeScanSlots = make(chan struct{}, 16)

const (
	codexHomeScanWait = 100 * time.Millisecond
	codexHomeListWait = 300 * time.Millisecond
)

type codexHomeScan struct {
	done    chan struct{}
	started time.Time
	fresh   time.Time
}

type codexHomeIndex struct {
	mu    sync.Mutex
	homes map[string]*codexHomeScan
}

// indexCodexHome registers both the rollouts and the App's sidebar threads (including
// threads without a rollout). Neither file source can hold up a different home.
func (n *Node) indexCodexHome(home string, deadline time.Time) bool {
	return n.codices.index.available(home, deadline, func() time.Duration {
		if st, err := os.Stat(home); err != nil || !st.IsDir() {
			return time.Minute
		}
		n.registerRollouts(home)
		n.registerCodexAppThreads(home, nil)
		return time.Second
	})
}

// A live pane is decorated before the history rows, sometimes between host refreshes.
// Do not re-enter a filesystem scan the index has already timed out or another reader holds.
func (n *Node) liveCodexHomeReady(id string, deadline time.Time) bool {
	src, ok := n.codexSource(id)
	if !ok {
		// A pane can already have its own transcript before its home is in
		// the Codex registry. The normal transcript lookup revalidates that
		// path; it needs no inferred home or new public identity here.
		n.mu.Lock()
		path := n.paths[id]
		n.mu.Unlock()
		return path != ""
	}
	if !n.indexCodexHome(src.home, deadline) {
		return false
	}
	c := rolloutCacheFor(src.home)
	if !c.mu.TryLock() {
		return false
	}
	c.mu.Unlock()
	return true
}

// firstMessageRollouts reads the completed index. A start polls until its first
// message appears, so it can pass a busy home and check again on the next look.
func (n *Node) firstMessageRollouts(home string, deadline time.Time) []codexRollout {
	if n.codices == nil {
		return rollouts(home)
	}
	c := rolloutCacheFor(home)
	if !c.mu.TryLock() {
		return nil
	}
	if time.Since(c.at) < rolloutScanInterval {
		list := c.rollouts
		c.mu.Unlock()
		return list
	}
	c.mu.Unlock()
	// The rollout cache may have been invalidated since the last catalog check.
	// Reusing an older, incomplete list could turn two candidate threads into one.
	i := &n.codices.index
	i.mu.Lock()
	if s := i.homes[home]; s != nil && s.done == nil {
		s.fresh = time.Time{}
	}
	i.mu.Unlock()
	if !n.indexCodexHome(home, deadline) {
		return nil
	}
	if !c.mu.TryLock() {
		return nil
	}
	defer c.mu.Unlock()
	return c.rollouts
}

// available waits only for the first short look. Once a scan overruns its budget,
// later lists pass it immediately. A completed scan is used by the next list.
func (i *codexHomeIndex) available(home string, deadline time.Time, scan func() time.Duration) bool {
	i.mu.Lock()
	if i.homes == nil {
		i.homes = make(map[string]*codexHomeScan)
	}
	s := i.homes[home]
	if s == nil {
		s = &codexHomeScan{}
		i.homes[home] = s
	}
	now := time.Now()
	if now.Before(s.fresh) {
		i.mu.Unlock()
		return true
	}
	if s.done == nil {
		select {
		case codexHomeScanSlots <- struct{}{}:
		default:
			i.mu.Unlock()
			return false
		}
		s.done, s.started = make(chan struct{}), now
		go func() {
			defer func() { <-codexHomeScanSlots }()
			freshFor := scan()
			i.mu.Lock()
			s.fresh = time.Now().Add(freshFor)
			close(s.done)
			s.done = nil
			i.mu.Unlock()
		}()
	}
	done, started := s.done, s.started
	i.mu.Unlock()
	return waitCodexScan(done, started, deadline)
}

func waitCodexScan(done <-chan struct{}, started, deadline time.Time) bool {
	now := time.Now()
	wait := min(codexHomeScanWait-now.Sub(started), deadline.Sub(now))
	if wait <= 0 {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}

// A home which is slow to scan retains the identity and context the node last confirmed.
// The current process, permissions and actions are unknown until the next successful read.
func unavailableCodexRow(s proto.Session) proto.Session {
	return proto.Session{
		SessionID: s.SessionID, Agent: s.Agent, AgentName: s.AgentName, AgentSurface: s.AgentSurface,
		Title: s.Title, Cwd: s.Cwd, WorkspaceCwd: s.WorkspaceCwd, RepoRoot: s.RepoRoot, Worktree: s.Worktree,
		Branch: s.Branch, Model: s.Model, Effort: s.Effort, UpdatedAt: s.UpdatedAt,
		SourceKind: s.SourceKind, SourceInstanceID: s.SourceInstanceID, SourceThreadID: s.SourceThreadID,
		ParentThreadID: s.ParentThreadID, ParentSessionID: s.ParentSessionID, RootSessionID: s.RootSessionID,
		IsSubagent: s.IsSubagent, Status: proto.StatusUnknown, Capabilities: &proto.SessionCapabilities{},
	}
}
