func TestTaskSpawnUnknownStartReconcilesLateChildIdentity(t *testing.T) {
	s, f, _, h := spawnWindowFixture(t, context.DeadlineExceeded)
	h.others = nil
	h.duringStart = func(child proto.Session) {
		// Adoption must not find the child before the failed end: it is still being built and its answer/id are lost.
		h.f.mu.Lock()
		h.f.sessions["box"] = slices.DeleteFunc(h.f.sessions["box"], func(row proto.Session) bool { return row.SessionID == child.SessionID })
		h.f.mu.Unlock()
	}
	text, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result."))
	if err == nil || h.child == "" {
		t.Fatalf("fixture: unknown start returned success or no child id: text=%q err=%v child=%q", text, err, h.child)
	}
	late := proto.Session{MachineID: "box", SessionID: h.child, SpaceName: "app", Cwd: "/code/app", Launcher: "codex",
		Agent: "codex", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true}
	if err := spawnWindowLinkForRace(t.Context(), h, late); err != nil {
		t.Fatal(err)
	}
	if agent, ok := activeTodoAgent(onlyTodo(s), "box", h.child); !ok || agent.Source != "space" {
		t.Fatalf("late unknown child did not arrive as space: %+v", agent)
	}
	startsBeforeCall := len(f.starts)
	seen := reached(t, "late unknown child", spawnWindowNode(h, h.child, "Only inspect one bounded assertion; edit nothing and return its result."))
	if seen.listed || !seen.failed || !strings.Contains(seen.text, "one supervised handoff layer") {
		t.Fatalf("late child from an unknown start escaped ancestry tracking: %+v", seen)
	}
	if len(f.starts) != startsBeforeCall {
		t.Fatalf("late unknown child started %d new sessions", len(f.starts)-startsBeforeCall)
	}
}

// Heuristic adoption is not proof of the child. A same-shaped concurrent session must not be stopped or linked as
// spawned; the real child still has to reconcile through unresolved identity tracking.
func TestTaskSpawnDoesNotConfirmHeuristicAdoption(t *testing.T) {
	s, f, _, h := spawnWindowFixture(t, context.DeadlineExceeded)
	h.others = nil
	h.duringStart = func(child proto.Session) {
		h.f.mu.Lock()
		h.f.sessions["box"] = slices.DeleteFunc(h.f.sessions["box"], func(row proto.Session) bool { return row.SessionID == child.SessionID })
		h.f.sessions["box"] = append(h.f.sessions["box"], proto.Session{MachineID: "box", SessionID: "wrong-adoptee", SpaceName: "app", Cwd: "/code/app",
			Launcher: "codex", Agent: "codex", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true, UpdatedAt: time.Now()})
		h.f.mu.Unlock()
		h.syncDevs()
	}
	disabled := false
	s.admission.mu.Lock()
	s.admission.cached = &AdmissionSettings{Enabled: &disabled}
	s.admission.at = time.Now()
	s.admission.mu.Unlock()
	text, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result."))
	if err == nil {
		t.Fatalf("heuristic adoption was confirmed as the child: %s", text)
	}
	s.spawns.mu.Lock()
	unresolved := len(s.spawns.unresolved)
	for identity := range s.spawns.unresolved {
		t.Logf("identity: expected=%q launcher=%q cwd=%q machine=%q until=%v known=%v", identity.expected, identity.launcher, identity.cwd, identity.machine, identity.until, identity.known)
	}
	s.spawns.mu.Unlock()
	if unresolved != 1 {
		t.Fatalf("heuristic adoption left %d unresolved identities, want 1", unresolved)
	}
	for _, agent := range onlyTodo(s).Agents {
		if agent.SessionID == "wrong-adoptee" && agent.Source == sourceSpawned {
			t.Fatalf("heuristic adoptee was linked as spawned: %+v", agent)
		}
	}
	if slices.Contains(f.stopped, "wrong-adoptee") {
		t.Fatal("heuristic adoptee was stopped without proof")
	}
	late := proto.Session{MachineID: "box", SessionID: h.child, SpaceName: "app", Cwd: "/code/app", Launcher: "codex",
		Agent: "codex", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true}
	for identity := range s.spawns.unresolved {
		t.Logf("late identity match=%v session=%+v", identity.matches(late), late)
	}
	if err := spawnWindowLinkForRace(t.Context(), h, late); err != nil {
		t.Fatal(err)
	}
	t.Logf("suspect after late link: %+v", s.spawns.suspect)
	startsBeforeCall := len(f.starts)
	seen := reached(t, "real late child", spawnWindowNode(h, h.child, "Only inspect one bounded assertion; edit nothing and return its result."))
	if seen.listed || !seen.failed || !strings.Contains(seen.text, "one supervised handoff layer") {
		t.Fatalf("real child escaped after heuristic adoption: %+v", seen)
	}
	if len(f.starts) != startsBeforeCall {
		t.Fatalf("real child started %d new sessions", len(f.starts)-startsBeforeCall)
	}
}

// An exact-id start must not confirm a same-shaped heuristic adoptee when its expected session is absent. The wrong
// session stays running and unconfirmed; the real expected session remains inside task_spawn ancestry tracking.
func TestTaskSpawnExactIDAdoptionRejectsSameShapeSession(t *testing.T) {
	s, f, _, h := spawnWindowFixtureLauncher(t, context.DeadlineExceeded, "claude")
	expected := ""
	f.startReply = func(params map[string]any, sess *proto.Session) *proto.StartInstead {
		expected, _ = params["session_id"].(string)
		sess.SessionID = expected
		return nil
	}
	wrong := proto.Session{MachineID: "box", SessionID: "wrong-exact-adoptee", SpaceName: "app", Cwd: "/code/app",
		Launcher: "claude", Agent: "claude", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true, UpdatedAt: time.Now()}
	h.duringStart = func(child proto.Session) {
		if expected == "" || child.SessionID != expected {
			t.Errorf("fixture: expected id was not used: expected=%q child=%q", expected, child.SessionID)
		}
		h.f.mu.Lock()
		h.f.sessions["box"] = slices.DeleteFunc(h.f.sessions["box"], func(row proto.Session) bool { return row.SessionID == child.SessionID })
		h.f.sessions["box"] = append(h.f.sessions["box"], wrong)
		h.f.mu.Unlock()
		h.syncDevs()
	}
	disabled := false
	s.admission.mu.Lock()
	s.admission.cached = &AdmissionSettings{Enabled: &disabled}
	s.admission.at = time.Now()
	s.admission.mu.Unlock()
	text, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result."))
	if err == nil {
		t.Fatalf("same-shaped heuristic adoptee was confirmed for exact id %q: %s", expected, text)
	}
	if expected == "" || h.child != expected {
		t.Fatalf("fixture: expected id mismatch: expected=%q child=%q", expected, h.child)
	}
	for _, agent := range onlyTodo(s).Agents {
		if agent.SessionID == wrong.SessionID && agent.Source == sourceSpawned {
			t.Fatalf("wrong session was linked as the spawned child: %+v", agent)
		}
	}
	if slices.Contains(f.stopped, wrong.SessionID) {
		t.Fatal("wrong heuristic adoptee was stopped as though it were the exact child")
	}
	s.spawns.mu.Lock()
	unresolved := len(s.spawns.unresolved)
	s.spawns.mu.Unlock()
	if unresolved != 1 {
		t.Fatalf("exact-id start left %d unresolved identities, want 1", unresolved)
	}
	startsAfterStart := len(f.starts)
	if err := spawnWindowLinkForRace(t.Context(), h, wrong); err != nil {
		t.Fatal(err)
	}
	if seen := reached(t, "wrong exact adoptee", spawnWindowNode(h, wrong.SessionID, "Only inspect one bounded assertion; edit nothing and return its result.")); seen.listed || !seen.failed {
		t.Fatalf("wrong exact adoptee escaped ancestry tracking: %+v", seen)
	}
	real := proto.Session{MachineID: "box", SessionID: h.child, SpaceName: "app", Cwd: "/code/app",
		Launcher: "claude", Agent: "claude", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true, UpdatedAt: time.Now()}
	if err := spawnWindowLinkForRace(t.Context(), h, real); err != nil {
		t.Fatal(err)
	}
	if seen := reached(t, "real exact child", spawnWindowNode(h, real.SessionID, "Only inspect one bounded assertion; edit nothing and return its result.")); seen.listed || !seen.failed {
		t.Fatalf("real exact child became an ordinary first-layer Agent: %+v", seen)
	}
	if len(f.starts) != startsAfterStart {
		t.Fatalf("wrong or real late sessions started another layer: before=%d after=%d", startsAfterStart, len(f.starts))
	}
}

// The reconnect adoption pass has the same exact-id rule: B appears only after the first lookup has failed and the
// link is coming back, so this exercises adoptTaskStartSessionUntil rather than the first adoption branch.
func TestTaskSpawnExactIDReconnectAdoptionRejectsSameShapeSession(t *testing.T) {
	s, f, _, h := spawnWindowFixtureLauncher(t, rpc.ErrDropped, "claude")
	oldWait, oldPoll, oldRetry := taskStartAdoptWait, taskStartAdoptPoll, agentStartAdoptRetry
	taskStartAdoptWait, taskStartAdoptPoll, agentStartAdoptRetry = 20*time.Millisecond, 100*time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() {
		taskStartAdoptWait, taskStartAdoptPoll, agentStartAdoptRetry = oldWait, oldPoll, oldRetry
	})
	expected := ""
	f.startReply = func(params map[string]any, sess *proto.Session) *proto.StartInstead {
		expected, _ = params["session_id"].(string)
		sess.SessionID = expected
		return nil
	}
	wrong := proto.Session{MachineID: "box", SessionID: "wrong-reconnect-adoptee", SpaceName: "app", Cwd: "/code/app",
		Launcher: "claude", Agent: "claude", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true, UpdatedAt: time.Now()}
	h.duringStart = func(child proto.Session) {
		h.f.mu.Lock()
		h.f.sessions["box"] = slices.DeleteFunc(h.f.sessions["box"], func(row proto.Session) bool { return row.SessionID == child.SessionID })
		h.f.mu.Unlock()
		lists := 0
		h.f.onList = func(f *fakeHub, machine string) { // the second lookup is the reconnect adoption pass
			lists++
			if lists == 2 {
				f.sessions[machine] = append(f.sessions[machine], wrong)
			}
		}
		h.syncDevs()
	}
	disabled := false
	s.admission.mu.Lock()
	s.admission.cached = &AdmissionSettings{Enabled: &disabled}
	s.admission.at = time.Now()
	s.admission.mu.Unlock()
	s.devs = nil // force adoption through the fake Hub's sessions.list, including the staged second lookup
	text, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result."))
	if err == nil {
		t.Fatalf("reconnect heuristic adoptee was confirmed for exact id %q: %s", expected, text)
	}
	for _, agent := range onlyTodo(s).Agents {
		if agent.SessionID == wrong.SessionID && agent.Source == sourceSpawned {
			t.Fatalf("reconnect wrong session was linked as spawned: %+v", agent)
		}
	}
	if slices.Contains(f.stopped, wrong.SessionID) {
		t.Fatal("reconnect wrong heuristic adoptee was stopped")
	}
	s.spawns.mu.Lock()
	unresolved := len(s.spawns.unresolved)
	s.spawns.mu.Unlock()
	if unresolved != 1 {
		t.Fatalf("reconnect exact-id start left %d unresolved identities, want 1", unresolved)
	}
}

// A definite pre-launch refusal must not poison later ordinary Agents. Only genuinely unknown starts can leave an
// unresolved identity, and that entry must have a bounded lifecycle.
func TestTaskSpawnDefiniteRefusalDoesNotPoisonLaterAgents(t *testing.T) {
	s, f, _, h := spawnWindowFixture(t, nil)
	h.armed = false
	f.failStart = errors.New("definite start refusal")
	if _, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result.")); err == nil {
		t.Fatal("definite start refusal unexpectedly succeeded")
	}
	f.failStart = nil // the next ordinary start must not inherit the injected one-shot refusal
	ordinary := proto.Session{MachineID: "box", SessionID: "ordinary-after-refusal", SpaceName: "app", Cwd: "/code/app",
		Launcher: "codex", Agent: "codex", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true}
	if err := spawnWindowLinkForRace(t.Context(), h, ordinary); err != nil {
		t.Fatal(err)
	}
	before := len(f.starts)
	seen := reached(t, "ordinary after refusal", spawnWindowNode(h, ordinary.SessionID, "Only inspect one bounded assertion; edit nothing and return its result."))
	if !seen.listed || seen.failed {
		t.Fatalf("definite refusal poisoned an ordinary Agent: %+v", seen)
	}
	if len(f.starts) != before+1 {
		t.Fatalf("ordinary Agent did not start exactly one child: before=%d after=%d", before, len(f.starts))
	}
}

// Unresolved identities are scoped to their starting machine, including exact-id collisions. Another machine's
// ordinary same-launcher/cwd Agent must not inherit a failed start's suspect marker.
func TestTaskSpawnUnresolvedIdentityStaysOnStartMachine(t *testing.T) {
	s, f, todo, h := spawnWindowFixture(t, context.DeadlineExceeded)
	h.others = nil
	h.duringStart = func(child proto.Session) {
		h.f.mu.Lock()
		h.f.sessions["box"] = slices.DeleteFunc(h.f.sessions["box"], func(row proto.Session) bool { return row.SessionID == child.SessionID })
		h.f.mu.Unlock()
	}
	if _, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result.")); err == nil {
		t.Fatal("unknown start unexpectedly succeeded")
	}
	other := proto.Session{MachineID: "other", SessionID: "other-ordinary", SpaceName: "app", Cwd: "/code/app",
		Launcher: "codex", Agent: "codex", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true}
	h.devs.add("other", "Other")
	h.devs.setSessions("other", []proto.Session{other})
	if _, err := s.linkTodoSpace(t.Context(), todo.ID, "other", "app"); err != nil {
		t.Fatal(err)
	}
	s.autoLinkTodos("other", []proto.Session{other})
	f.setSessions("other", other)
	before := len(f.starts)
	_, err := s.taskSpawn(t.Context(), "other", other.SessionID, spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result."))
	if err != nil {
		t.Fatalf("unrelated machine inherited suspect: %v", err)
	}
	if len(f.starts) != before+1 {
		t.Fatalf("unrelated machine did not start exactly one child: before=%d after=%d", before, len(f.starts))
	}
	late := proto.Session{MachineID: "box", SessionID: h.child, SpaceName: "app", Cwd: "/code/app", Launcher: "codex",
		Agent: "codex", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true}
	if err := spawnWindowLinkForRace(t.Context(), h, late); err != nil {
		t.Fatal(err)
	}
	seen := reached(t, "originating-machine child", spawnWindowNode(h, h.child, "Only inspect one bounded assertion; edit nothing and return its result."))
	if seen.listed || !seen.failed {
		t.Fatalf("originating machine's late child was not refused: %+v", seen)
	}
}

// Suspect marking and unresolved retirement are different decisions. A heuristic B at batch N must not retire the
// start before the real A arrives at batch N+1. Exact identities retire only on A; heuristic identities stay bounded
// until expiry, while suspect entries keep both A and B inside the one-layer rule.
func TestTaskSpawnUnresolvedRetirementRequiresExactIdentity(t *testing.T) {
	for _, tc := range []struct {
		name     string
		launcher string
	}{{"exact", "claude"}, {"non-exact", "codex"}} {
		t.Run(tc.name, func(t *testing.T) {
			s, f, _, h := spawnWindowFixtureLauncher(t, context.DeadlineExceeded, tc.launcher)
			if tc.launcher == "claude" {
				f.startReply = func(params map[string]any, sess *proto.Session) *proto.StartInstead {
					sess.SessionID, _ = params["session_id"].(string)
					return nil
				}
			}
			h.duringStart = func(child proto.Session) {
				h.f.mu.Lock()
				h.f.sessions["box"] = slices.DeleteFunc(h.f.sessions["box"], func(row proto.Session) bool { return row.SessionID == child.SessionID })
				h.f.mu.Unlock()
				h.syncDevs()
			}
			disabled := false
			s.admission.mu.Lock()
			s.admission.cached = &AdmissionSettings{Enabled: &disabled}
			s.admission.at = time.Now()
			s.admission.mu.Unlock()
			if text, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result.")); err == nil {
				t.Fatalf("unknown start unexpectedly succeeded: %s", text)
			}
			wrong := proto.Session{MachineID: "box", SessionID: "wrong-" + tc.name, SpaceName: "app", Cwd: "/code/app",
				Launcher: tc.launcher, Agent: tc.launcher, PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true, UpdatedAt: time.Now()}
			if err := spawnWindowLinkForRace(t.Context(), h, wrong); err != nil { // batch N: B only
				t.Fatal(err)
			}
			startsAfterStart := len(f.starts)
			if seen := reached(t, "batch N wrong session", spawnWindowNode(h, wrong.SessionID, "Only inspect one bounded assertion; edit nothing and return its result.")); seen.listed || !seen.failed {
				t.Fatalf("batch N wrong session escaped suspect: %+v", seen)
			}
			real := proto.Session{MachineID: "box", SessionID: h.child, SpaceName: "app", Cwd: "/code/app",
				Launcher: tc.launcher, Agent: tc.launcher, PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true, UpdatedAt: time.Now()}
			if err := spawnWindowLinkForRace(t.Context(), h, real); err != nil { // batch N+1: the actual child
				t.Fatal(err)
			}
			if seen := reached(t, "batch N+1 real child", spawnWindowNode(h, real.SessionID, "Only inspect one bounded assertion; edit nothing and return its result.")); seen.listed || !seen.failed {
				t.Fatalf("real child escaped after heuristic retirement at batch N: %+v", seen)
			}
			if seen := reached(t, "batch N+1 wrong session", spawnWindowNode(h, wrong.SessionID, "Only inspect one bounded assertion; edit nothing and return its result.")); seen.listed || !seen.failed {
				t.Fatalf("wrong session lost its suspect marker at batch N+1: %+v", seen)
			}
			s.spawns.mu.Lock()
			unresolved := len(s.spawns.unresolved)
			s.spawns.mu.Unlock()
			want := 1
			if tc.launcher == "claude" {
				want = 0 // the exact child was seen; its suspect marker still remains for this process
			}
			if unresolved != want {
				t.Fatalf("unresolved retirement after exact/heuristic batches = %d, want %d", unresolved, want)
			}
			if len(f.starts) != startsAfterStart {
				t.Fatalf("late sessions started another layer: before=%d after=%d", startsAfterStart, len(f.starts))
			}
		})
	}
}

// Workspace fanout can publish a late child before its sessions.changed event. Discovery and direct calls must
// reconcile that caller's typed identity even when the workspace membership arrived first.
func TestTaskSpawnLateSpaceFanoutReconcilesUnknownStart(t *testing.T) {
	oldWait, oldPoll, oldRetry := taskStartAdoptWait, taskStartAdoptPoll, agentStartAdoptRetry
	taskStartAdoptWait, taskStartAdoptPoll, agentStartAdoptRetry = 20*time.Millisecond, 5*time.Millisecond, time.Millisecond
	t.Cleanup(func() { taskStartAdoptWait, taskStartAdoptPoll, agentStartAdoptRetry = oldWait, oldPoll, oldRetry })
	for _, tc := range []struct {
		name     string
		launcher string
	}{{"exact", "claude"}, {"non-exact", "codex"}} {
		for _, inventoryFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/inventory-first-%v", tc.name, inventoryFirst), func(t *testing.T) {
				s, f, todo, h := spawnWindowFixtureLauncher(t, context.DeadlineExceeded, tc.launcher)
				if err := s.unlinkTodoSpace(t.Context(), todo.ID, "box", "app"); err != nil {
					t.Fatal(err)
				}
				if tc.launcher == "claude" {
					f.startReply = func(params map[string]any, sess *proto.Session) *proto.StartInstead {
						sess.SessionID, _ = params["session_id"].(string)
						return nil
					}
				}
				h.duringStart = func(child proto.Session) {
					h.f.mu.Lock()
					h.f.sessions["box"] = slices.DeleteFunc(h.f.sessions["box"], func(row proto.Session) bool { return row.SessionID == child.SessionID })
					h.f.mu.Unlock()
					h.syncDevs()
				}
				disabled := false
				s.admission.mu.Lock()
				s.admission.cached, s.admission.at = &AdmissionSettings{Enabled: &disabled}, time.Now()
				s.admission.mu.Unlock()
				if text, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result.")); err == nil {
					t.Fatalf("fixture: uncertain start unexpectedly succeeded: %s", text)
				}
				s.spawns.mu.Lock()
				unresolved := len(s.spawns.unresolved)
				s.spawns.mu.Unlock()
				if unresolved != 1 || h.child == "" {
					t.Fatalf("fixture: unresolved=%d child=%q", unresolved, h.child)
				}
				late := proto.Session{MachineID: "box", SessionID: h.child, SpaceName: "app", Cwd: "/code/app",
					Launcher: tc.launcher, Agent: tc.launcher, PermissionMode: "plan", Status: proto.StatusWorking,
					StartedByPersona: true, UpdatedAt: time.Now()}
				spawnWindowSetSessionForRace(h, late) // lookup sees A before any sessions.changed event
				if inventoryFirst {
					h.f.mu.Lock()
					list := slices.Clone(h.f.sessions["box"])
					h.f.mu.Unlock()
					s.SessionsChanged(t.Context(), DeviceSessions{MachineID: "box", Sessions: list})
				}
				if _, err := s.linkTodoSpace(t.Context(), todo.ID, "box", "app"); err != nil {
					t.Fatal(err)
				}
				if agent, ok := activeTodoAgent(onlyTodo(s), "box", h.child); !ok || agent.Source != "space" {
					t.Fatalf("fixture: fanout did not publish the child as space: %+v", agent)
				}
				before := len(f.starts)
				seen := reached(t, "late fanout child", spawnWindowNode(h, h.child, "Only inspect a second bounded assertion; edit nothing and return its result."))
				if seen.listed || !seen.failed || len(f.starts) != before {
					t.Fatalf("late workspace fanout escaped unresolved ancestry: launcher=%s inventoryFirst=%v listed=%v failed=%v starts_delta=%d text=%q", tc.launcher, inventoryFirst, seen.listed, seen.failed, len(f.starts)-before, seen.text)
				}
			})
		}
	}
}

// The unresolved registry is bounded: after the node's own start budget expires, an ordinary same-shaped late Agent
// is no longer held back by an old unknown start.
func TestTaskSpawnUnresolvedIdentityExpires(t *testing.T) {
	s, f, _, h := spawnWindowFixture(t, context.DeadlineExceeded)
	h.others = nil
	h.duringStart = func(child proto.Session) {
		h.f.mu.Lock()
		h.f.sessions["box"] = slices.DeleteFunc(h.f.sessions["box"], func(row proto.Session) bool { return row.SessionID == child.SessionID })
		h.f.mu.Unlock()
	}
	if _, err := s.taskSpawn(t.Context(), "box", "s1", spawnPromptArgs("Only inspect one bounded assertion; edit nothing and return its result.")); err == nil {
		t.Fatal("unknown start unexpectedly succeeded")
	}
	s.spawns.mu.Lock()
	for identity := range s.spawns.unresolved {
		identity.until = time.Now().Add(-time.Second)
	}
	s.spawns.mu.Unlock()
	ordinary := proto.Session{MachineID: "box", SessionID: "ordinary-after-expiry", SpaceName: "app", Cwd: "/code/app",
		Launcher: "codex", Agent: "codex", PermissionMode: "plan", Status: proto.StatusWorking, StartedByPersona: true}
	if err := spawnWindowLinkForRace(t.Context(), h, ordinary); err != nil {
		t.Fatal(err)
	}
	before := len(f.starts)
	seen := reached(t, "ordinary after expiry", spawnWindowNode(h, ordinary.SessionID, "Only inspect one bounded assertion; edit nothing and return its result."))
	if !seen.listed || seen.failed {
		t.Fatalf("expired unresolved identity still refused an ordinary Agent: %+v", seen)
	}
	if len(f.starts) != before+1 {
		t.Fatalf("ordinary Agent after expiry did not start exactly one child: before=%d after=%d", before, len(f.starts))
	}
}
