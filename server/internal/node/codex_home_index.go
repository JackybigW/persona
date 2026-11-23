package node

import (
	"os"

	"github.com/nwverse/persona/server/internal/proto"
	"sync"
	"time"
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
	done, wait := s.done, min(codexHomeScanWait-now.Sub(s.started), deadline.Sub(now))
	i.mu.Unlock()
	if wait <= 0 {
		return false
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
