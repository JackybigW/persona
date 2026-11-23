import test from 'node:test';
import assert from 'node:assert/strict';
import { registerHooks, stripTypeScriptTypes } from 'node:module';
import { readFileSync } from 'node:fs';

registerHooks({
  resolve(specifier, context, next) {
    if (/\/Health(Workspace|Intake)\.ets$/.test(context.parentURL || '') && ['./Wire','./HealthIntake'].includes(specifier))
      return { url: new URL(specifier + '.ets', context.parentURL).href, shortCircuit: true };
    if (context.parentURL?.endsWith('/HealthIntake.ets') && specifier === '@kit.ArkTS')
      return {url:'data:text/javascript,'+encodeURIComponent("import {randomUUID} from 'node:crypto'; export const util={generateRandomUUID:randomUUID};"), shortCircuit:true};
    return next(specifier, context);
  },
  load(url, context, next) {
    if (url.endsWith('.ets')) {
      const source = readFileSync(new URL(url), 'utf8').replace(/@ObservedV2\s*/g, '').replace(/@Trace\s+/g, '');
      return { format: 'module', source: stripTypeScriptTypes(source), shortCircuit: true };
    }
    return next(url, context);
  }
});

const { HEALTH_SUGGESTIONS, HealthWorkspace, HealthMemoryFile } = await import('../entry/src/main/ets/models/HealthWorkspace.ets');
const { ResourceRow } = await import('../entry/src/main/ets/models/Wire.ets');
const { healthGoalWidth } = await import('../entry/src/main/ets/core/HealthGoalLabel.ts');

test('health ring captions fit complete localized words and preserve narrow no-break number grouping', () => {
  const measure = text => Array.from(text).reduce((w, c) => w + (/\p{Script=Han}/u.test(c) ? 18 : c === ' ' ? 5 : 10), 0);
  const chinese = '目标：2,000 毫升';
  assert.equal(healthGoalWidth(chinese, measure(chinese) - 3, measure), measure('目标：2,000'));
  const english = 'Goal: 2,000 ml';
  assert.equal(healthGoalWidth(english, 200, measure), measure(english));
  const japanese = '目標：2,000 ml';
  assert.equal(healthGoalWidth(japanese, measure(japanese) - 1, measure), measure('目標：2,000'));
  const grouped = 'Goal: 2\u202f000 ml';
  assert.equal(healthGoalWidth(grouped, measure('2\u202f000 ml'), measure), measure('2\u202f000 ml'));
  assert.equal(healthGoalWidth('unbrokenword', 25, measure), 25);
  assert.equal(healthGoalWidth('unavailable-font', 105, () => 0), 105);
});

function clone(value) { return structuredClone(value); }

function deferred() {
  let resolve;
  const promise = new Promise((yes) => { resolve = yes; });
  return { promise, resolve };
}

function failure(status, code, resultUnknown = false, resultPending = false) {
  const error = new Error(code);
  error.status = status; error.resultUnknown = resultUnknown; error.resultPending = resultPending;
  return error;
}

function chat(id, space = 'health', fields = {}) {
  return { id, session_id: id, machine_id: 'health-machine', title: 'Health ' + id, cwd: '/home/user/.persona/health',
    space, last_text: 'A private health question', ...fields };
}

function fixture(options = {}) {
  const shared = { calls: [], views: new Map(), defaultManager: options.defaultManager || 'health-owner', managers: options.managers || ['health-owner'],
    request: options.request, childClients: new Map(), rootState: undefined };
  const roster = () => ({ default: shared.defaultManager, managers: shared.managers.map((value) => {
    const id = typeof value === 'string' ? value : value.id;
    return { id, name: id, online: true, default: id === shared.defaultManager, ...(typeof value === 'string' ? {} : value) };
  }) });

  function clientState(manager, scope) {
    return { manager, scope, generation: 5, connected: true, machine: 'health-machine', calls: [], blocked: new Set(),
      conversations: options.conversations || [chat('health-1'), chat('main-leak', 'main')],
      memories: options.memories || [
        { name: 'MEMORY.md', content: '- [Sleep](sleep.md) — sleep goal', updated_at: '2026-11-10T10:00:00Z' },
        { name: 'sleep.md', content: '# Sleep\nAim for seven hours.', updated_at: '2026-11-10T11:00:00Z' }
      ],
      reports: options.reports || [{ name: 'checkup.pdf', title: 'Annual Checkup', path: '/health/reports/checkup.pdf',
        record_path: '/health/reports/checkup.pdf.md', size: 2048, modified_at: '2026-11-11T10:00:00Z' }],
      machineTrust: options.machineTrust || 'personal',
      machineOnline: options.machineOnline !== false,
      beforeUploadAdmit: options.beforeUploadAdmit,
      beforeCallAdmit: options.beforeCallAdmit,
      onCall: options.childOnCall
    };
  }

  function makeClient(state, isRoot = false) {
    const repo = {
      async call(path, method = 'GET', body = {}, op = '', admit) {
        if (method !== 'GET') {
          if (state.beforeCallAdmit) await state.beforeCallAdmit(state, path);
          if (admit && !admit()) throw failure(0, 'request_updated');
        }
        const call = { path, method, body: clone(body), op };
        shared.calls.push(call); state.calls.push(call);
        if (isRoot && path === '/api/v1/managers') return clone(roster());
        if (state.onCall) {
          try {
            const result = await state.onCall(call, state, shared);
            if (result !== undefined) return clone(result);
          } catch (error) {
            if (error.resultUnknown || error.resultPending) state.blocked.add(path);
            throw error;
          }
        }
        if (path === '/api/v1/assistant?space=health' && method === 'GET')
          return { machine_id: state.machine, conversations: clone(state.conversations) };
        if (path === '/api/v1/machines' && method === 'GET')
          return { machines: [{ id: state.machine, manager: state.manager, online: state.machineOnline, trust: state.machineTrust }] };
        if (path === '/api/v1/assistant/memory?space=health' && method === 'GET') return { files: clone(state.memories) };
        if (path.startsWith('/api/v1/machines/') && path.endsWith('/health/reports') && method === 'GET')
          return { reports: clone(state.reports) };
        if (path === '/api/v1/assistant/conversations' && method === 'POST') {
          const session = chat('created-' + (state.calls.filter((item) => item.method === 'POST' && item.path === path).length), 'health', {
            title: body.prompt, session_id: 'created-1'
          });
          state.conversations.unshift(session); return { session };
        }
        if (path.startsWith('/api/v1/assistant/conversations/') && path.endsWith('/archive') && method === 'POST') {
          const id = decodeURIComponent(path.split('/')[5]); state.conversations = state.conversations.filter((item) => item.id !== id);
          return { ok: true, archived: true };
        }
        if (path.startsWith('/api/v1/assistant/memory/') && method === 'PUT') {
          const name = decodeURIComponent(path.split('/')[5].split('?')[0]);
          const file = state.memories.find((item) => item.name === name);
          if (file) file.content = body.content;
          return { ok: true };
        }
        if (path.startsWith('/api/v1/assistant/memory/') && method === 'DELETE') {
          const name = decodeURIComponent(path.split('/')[5].split('?')[0]);
          state.memories = state.memories.filter((item) => item.name !== name); return { ok: true };
        }
        if (path.includes('/health/reports/') && method === 'DELETE') {
          const name = decodeURIComponent(path.split('/').at(-1)); state.reports = state.reports.filter((item) => item.name !== name);
          return { ok: true };
        }
        return { ok: true };
      }
    };
    repo.upload = async (path, bytes, admit) => {
      if (state.beforeUploadAdmit) await state.beforeUploadAdmit(state);
      if (admit && !admit()) throw failure(0, 'request_updated');
      const call = { path, method: 'UPLOAD', bytes: bytes.byteLength };
      shared.calls.push(call); state.calls.push(call);
      if (state.onCall) {
        try {
          const result = await state.onCall(call, state, shared);
          if (result !== undefined) return clone(result);
        } catch (error) {
          if (error.resultUnknown || error.resultPending) state.blocked.add(path);
          throw error;
        }
      }
      const name = decodeURIComponent(path.split('name=')[1].split('&')[0]);
      const extension = name.includes('.') ? name.slice(name.lastIndexOf('.')) : '';
      const stem = extension ? name.slice(0, -extension.length) : name;
      const remotePath = '/home/' + state.manager + '/.persona/health/reports/' + stem + '-20261111-140000' + extension;
      return { path: remotePath, name: remotePath.split('/').at(-1), size: bytes.byteLength };
    };
    const client = {
      connected: true, demo: false, profile: { id: 'hub-profile', manager: state.manager }, features: [],
      repo() { return repo; }, scope() { return state.scope; }, connectionGeneration() { return state.generation; },
      owns(generation, scope) { return state.connected && generation === state.generation && scope === state.scope; },
      operationBlocked(path) { return state.blocked.has(path); },
      async loadView(key) { return shared.views.get(state.scope + ':' + key) || ''; },
      viewRevision(){return state.epoch||0;},
      async saveView(key, value) { shared.views.set(state.scope + ':' + key, value); },
      async viewFor(row) {
        const next = shared.childClients.get(row.manager);
        if (!next) throw new Error('manager_unavailable');
        return next.client;
      }
    };
    state.repo = repo; state.client = client;
    return { client, state, repo };
  }

  shared.rootState = clientState(options.rootManager || 'work', 'hub:work');
  const root = makeClient(shared.rootState, true);
  for (const value of shared.managers) {
    const id = typeof value === 'string' ? value : value.id;
    const state = clientState(id, 'hub:' + id);
    const child = makeClient(state, false);
    shared.childClients.set(id, child);
  }
  return { root: root.client, rootState: root.state, shared, child: (id = shared.defaultManager) => shared.childClients.get(id) };
}

async function opened(options = {}) {
  const env = fixture(options), workspace = new HealthWorkspace();
  await workspace.open(env.root);
  return { ...env, workspace };
}

function pathMethod(env, method, prefix) {
  return env.shared.calls.filter((call) => call.method === method && call.path.startsWith(prefix));
}

test('the four iPhone Health suggestions retain distinct controls and send their displayed prompt to Health', async () => {
  const english = ['How did I sleep this week?', 'How do my steps over the last 7 days compare with last week?',
    'Log my weight today: 62.5 kg', 'Log my weight today: 62.5 kg'];
  const questions = {
    base: english, en_US: english,
    zh_CN: ['这周我睡得怎么样？', '最近 7 天步数和上周比怎么样', '我的静息心率最近有变化吗', '帮我记一下今天体重 62.5 kg'],
    ja_JP: ['今週の睡眠はどうでしたか？', '直近7日間の歩数は先週と比べてどうですか？', '今日の体重62.5kgを記録して', '今日の体重62.5kgを記録して']
  };
  assert.equal(new Set(HEALTH_SUGGESTIONS).size, 4);
  for (const [locale, expected] of Object.entries(questions)) {
    const file = new URL(`../entry/src/main/resources/${locale}/element/string.json`, import.meta.url);
    const resources = new Map(JSON.parse(readFileSync(file, 'utf8')).string.map(v => [v.name, v.value]));
    const displayed = HEALTH_SUGGESTIONS.map(id => resources.get(id));
    assert.deepEqual(displayed, expected, locale);
    const env = await opened();
    for (const prompt of displayed) {
      const row = await env.workspace.startConversation(prompt);
      assert.equal(row.manager, 'health-owner');
      assert.equal(row.data().space, 'health');
    }
    const sent = pathMethod(env, 'POST', '/api/v1/assistant/conversations');
    assert.deepEqual(sent.map(call => call.body.prompt), expected, locale);
    assert.ok(sent.every(call => call.body.space === 'health'));
  }
});

test('Health resolves the roster default and keeps only health-space conversations from the health endpoint', async () => {
  const { workspace, root, rootState, shared, child } = await opened();
  assert.equal(root.profile.manager, 'work', 'opening Health does not change the selected manager');
  assert.equal(workspace.canStartConversation(), true);
  assert.equal(workspace.machine, 'health-machine');
  assert.deepEqual(workspace.conversations.map((row) => row.id), ['health-1']);
  assert.equal(workspace.conversations[0].manager, 'health-owner');
  assert.equal(workspace.conversations[0].data().space, 'health');
  assert.deepEqual(workspace.memoryEntries().map((file) => file.name), ['sleep.md']);
  assert.equal(workspace.indexMemory().name, 'MEMORY.md');
  assert.deepEqual(rootState.calls.map((call) => call.path), ['/api/v1/managers']);
  assert.ok(child().state.calls.some((call) => call.path === '/api/v1/assistant?space=health'));
  assert.ok(child().state.calls.some((call) => call.path === '/api/v1/assistant/memory?space=health'));
  assert.ok(child().state.calls.some((call) => call.path === '/api/v1/machines/health-machine/health/reports'));
  assert.equal(shared.calls.some((call) => call.path === '/api/v1/assistant'), false);
});

test('creating a Health chat uses the exact health space body and returns a default-manager route', async () => {
  const { workspace, child } = await opened();
  workspace.setDraft('How did I sleep this week?');
  const row = await workspace.startConversation();
  assert.deepEqual(pathMethod({ shared: { calls: child().state.calls } }, 'POST', '/api/v1/assistant/conversations').map((call) => [call.path, call.body]), [
    ['/api/v1/assistant/conversations', { prompt: 'How did I sleep this week?', space: 'health' }]
  ]);
  assert.ok(row);
  assert.equal(row.manager, 'health-owner');
  assert.equal(row.machine, 'health-machine');
  assert.equal(row.session, 'created-1');
  assert.equal(workspace.isHealthConversation(row), true);
  assert.equal(workspace.draft, '');
  assert.ok(child().state.calls.find((call) => call.method === 'POST').op);
});

test('memory index is read only; memory edits and deletion retain the health query and local draft scope', async () => {
  const { workspace, child, shared } = await opened();
  const index = workspace.indexMemory();
  workspace.openMemory(index);
  assert.equal(workspace.canEditMemory(), false);
  assert.equal(await workspace.deleteMemory(), false);
  assert.equal(pathMethod({ shared: { calls: child().state.calls } }, 'DELETE', '/api/v1/assistant/memory/').length, 0);

  const file = workspace.memoryEntries()[0];
  workspace.openMemory(file);
  await new Promise((resolve) => setImmediate(resolve));
  workspace.setMemoryDraft('# Sleep\nTry 8 hours.');
  await workspace.memoryWrite;
  const draftKey = 'hub:health-owner:health-memory-draft:["health-owner","sleep.md"]';
  assert.equal(JSON.parse(shared.views.get(draftKey)).content, '# Sleep\nTry 8 hours.');
  assert.equal(await workspace.saveMemory(), true);
  assert.deepEqual(pathMethod({ shared: { calls: child().state.calls } }, 'PUT', '/api/v1/assistant/memory/').map((call) => [call.path, call.body]), [
    ['/api/v1/assistant/memory/sleep.md?space=health', { content: '# Sleep\nTry 8 hours.' }]
  ]);
  workspace.openMemory(workspace.memoryEntries()[0]);
  assert.equal(await workspace.deleteMemory(), true);
  assert.deepEqual(pathMethod({ shared: { calls: child().state.calls } }, 'DELETE', '/api/v1/assistant/memory/').map((call) => call.path), [
    '/api/v1/assistant/memory/sleep.md?space=health'
  ]);
});

test('archive accepts only a listed Health conversation owned by the roster default', async () => {
  const { workspace, child } = await opened();
  const healthRow = workspace.conversations[0];
  const mainRow = ResourceRow.from(chat('other', 'main'), 'conversation'); mainRow.manager = 'health-owner'; mainRow.machine = 'health-machine';
  const foreign = ResourceRow.from(chat('foreign', 'health'), 'conversation'); foreign.manager = 'work'; foreign.machine = 'health-machine';
  assert.equal(await workspace.archive(mainRow), false);
  assert.equal(await workspace.archive(foreign), false);
  assert.equal(await workspace.archive(healthRow), true);
  assert.deepEqual(pathMethod({ shared: { calls: child().state.calls } }, 'POST', '/api/v1/assistant/conversations/').map((call) => call.path), [
    '/api/v1/assistant/conversations/health-1/archive'
  ]);
  assert.equal(workspace.canArchiveConversation(healthRow), false);
});

test('opening and deleting a report preserve its manager, machine, record and exact delete target', async () => {
  const { workspace, child } = await opened();
  const report = workspace.reports[0];
  assert.equal(workspace.openReport(report, true).path, '/health/reports/checkup.pdf.md');
  assert.equal(workspace.openReport(report, false).path, '/health/reports/checkup.pdf');
  assert.equal(workspace.openReport(report, true).manager, 'health-owner');
  assert.equal(workspace.openReport(report, true).machine, 'health-machine');
  assert.equal(await workspace.deleteReport(report), true);
  assert.deepEqual(pathMethod({ shared: { calls: child().state.calls } }, 'DELETE', '/api/v1/machines/').map((call) => call.path), [
    '/api/v1/machines/health-machine/health/reports/checkup.pdf'
  ]);
});

test('report upload targets the trusted default manager machine and returns the health reports path', async () => {
  const { workspace, shared, child } = await opened();
  assert.equal(workspace.canUploadReports(), true);
  const path = await workspace.uploadReport('checkup.pdf', new ArrayBuffer(12));
  assert.equal(path, '/home/health-owner/.persona/health/reports/checkup-20261111-140000.pdf');
  assert.deepEqual(shared.calls.filter((call) => call.method === 'UPLOAD').map((call) => [call.path, call.bytes]), [
    ['/api/v1/machines/health-machine/files?name=checkup.pdf&dest=health', 12]
  ]);
  assert.equal(workspace.appendReportPath(path, 'Review this checkup report item by item.'), true);
  await workspace.draftWrite;
  assert.equal(workspace.draft, 'Review this checkup report item by item.\n' + path);
  assert.equal(shared.calls.some((call) => call.method === 'POST' && call.path === '/api/v1/assistant/conversations'), false);
  assert.ok(child().state.calls.some((call) => call.path === '/api/v1/machines' && call.method === 'GET'));
  assert.equal(workspace.reportUploadState, 'done');
});

test('a retired owner is rejected after durable upload admission and before sending bytes', async () => {
  const { workspace, shared } = await opened({ beforeUploadAdmit: async (state) => { state.generation++; } });
  assert.equal(await workspace.uploadReport('checkup.pdf', new ArrayBuffer(12)), undefined);
  assert.equal(shared.calls.filter((call) => call.method === 'UPLOAD').length, 0);
});

test('a changed roster default cannot receive a report from the previous Health owner', async () => {
  const { workspace, shared } = await opened({ managers: ['health-owner', 'next-owner'] });
  shared.defaultManager = 'next-owner';
  assert.equal(await workspace.uploadReport('checkup.pdf', new ArrayBuffer(12)), undefined);
  assert.equal(shared.calls.filter((call) => call.method === 'UPLOAD').length, 0);
});

test('an unknown report upload cannot be retried for the same exact file target', async () => {
  const { workspace, shared } = await opened({ childOnCall: async (call) => {
    if (call.method === 'UPLOAD') throw failure(0, 'result_unknown', true);
    return undefined;
  } });
  assert.equal(await workspace.uploadReport('checkup.pdf', new ArrayBuffer(12)), undefined);
  assert.equal(workspace.reportUploadState, 'unknown');
  assert.equal(await workspace.uploadReport('checkup.pdf', new ArrayBuffer(12)), undefined);
  assert.equal(shared.calls.filter((call) => call.method === 'UPLOAD').length, 1);
});

test('report uploads stay unavailable when the default assistant machine is not trusted', async () => {
  for (const options of [{ machineTrust: 'unknown' }, { machineOnline: false }]) {
    const { workspace, shared } = await opened(options);
    assert.equal(workspace.canUploadReports(), false);
    assert.equal(workspace.canStartConversation(), false);
    assert.equal(await workspace.startConversation('An unsafe target?'), undefined);
    assert.equal(await workspace.uploadReport('checkup.pdf', new ArrayBuffer(12)), undefined);
    assert.equal(workspace.error, 'health_default_manager_unavailable');
    assert.equal(shared.calls.filter((call) => call.method === 'UPLOAD').length, 0);
  }
});

test('pending and unknown conversation writes are durable and cannot be sent twice', async () => {
  for (const resultFailure of [failure(202, 'result_unknown', true, true), failure(0, 'result_unknown', true, false)]) {
    const { workspace, child } = await opened({ childOnCall: async (call, state) => {
      if (call.method === 'POST' && call.path === '/api/v1/assistant/conversations') throw resultFailure;
      return undefined;
    } });
    workspace.setDraft('Log my weight today: 62.5 kg');
    await workspace.draftWrite;
    assert.equal(await workspace.startConversation(), undefined);
    const firstWrites = pathMethod({ shared: { calls: child().state.calls } }, 'POST', '/api/v1/assistant/conversations').length;
    assert.equal(firstWrites, 1);
    assert.ok(['pending', 'unknown'].includes(workspace.conversationWriteState));
    await workspace.open(workspace.client);
    assert.ok(['pending', 'unknown'].includes(workspace.conversationWriteState));
    assert.equal(await workspace.startConversation(), undefined);
    assert.equal(pathMethod({ shared: { calls: child().state.calls } }, 'POST', '/api/v1/assistant/conversations').length, 1);
  }
});

test('an absent or offline roster default reports Health unavailable without querying another space', async () => {
  const env = fixture();
  env.shared.defaultManager = '';
  const workspace = new HealthWorkspace();
  await workspace.open(env.root);
  assert.equal(workspace.error, 'health_default_manager_unavailable');
  assert.equal(env.shared.calls.some((call) => call.path === '/api/v1/assistant?space=health'), false);

  const offline = fixture({ managers: ['health-owner'] });
  offline.shared.managers = [{ id: 'health-owner', online: false }];
  offline.shared.defaultManager = 'health-owner';
  const blocked = new HealthWorkspace();
  await blocked.open(offline.root);
  assert.equal(blocked.error, 'health_default_manager_unavailable');
  assert.equal(offline.shared.calls.some((call) => call.path === '/api/v1/assistant?space=health'), false);
});

test('a changed default manager refuses writes until refresh binds Health to the new owner', async () => {
  const { workspace, shared, child } = await opened({ managers: ['health-owner', 'next-owner'] });
  shared.defaultManager = 'next-owner';
  workspace.setDraft('A question');
  assert.equal(await workspace.startConversation(), undefined);
  assert.equal(pathMethod({ shared: { calls: child('health-owner').state.calls } }, 'POST', '/api/v1/assistant/conversations').length, 0);
  await workspace.refresh();
  assert.equal(workspace.conversations[0].manager, 'next-owner');
  const row = await workspace.startConversation('A question');
  assert.equal(row.manager, 'next-owner');
  assert.equal(pathMethod({ shared: { calls: child('next-owner').state.calls } }, 'POST', '/api/v1/assistant/conversations').length, 1);
});

test('a delayed response from a retired health connection cannot change state, and the persisted intent blocks a duplicate', async () => {
  const response = deferred(), entered = deferred();
  const { workspace, child } = await opened({ childOnCall: async (call) => {
    if (call.method === 'POST' && call.path === '/api/v1/assistant/conversations') { entered.resolve(); return response.promise; }
    return undefined;
  } });
  workspace.setDraft('Will this remain private?'); await workspace.draftWrite;
  const pending = workspace.startConversation(); await entered.promise;
  const state = child().state, oldGeneration = state.generation;
  state.generation++;
  response.resolve({ session: chat('late', 'health') });
  assert.equal(await pending, undefined);
  assert.equal(workspace.conversations.some((row) => row.id === 'late'), false);
  assert.notEqual(state.generation, oldGeneration);
  assert.equal(workspace.conversationWriteState, 'sending');
  await workspace.open(workspace.client);
  assert.equal(workspace.conversationWriteState, 'unknown');
  assert.equal(await workspace.startConversation(), undefined);
  assert.equal(pathMethod({ shared: { calls: state.calls } }, 'POST', '/api/v1/assistant/conversations').length, 1);
});


test('Health local chat and memory drafts persist while the same owner is offline',async()=>{
 const {workspace,shared}=await opened(),child=shared.childClients.get('health-owner');
 await workspace.openMemory(workspace.memories.find(file=>file.name==='sleep.md'));child.client.connected=false;child.state.generation++;
 workspace.setDraft('Offline private health draft');workspace.setMemoryDraft('Offline sleep notes');await workspace.draftWrite;await workspace.memoryWrite;
 assert.ok([...shared.views.values()].some(raw=>raw.includes('Offline private health draft')));assert.ok([...shared.views.values()].some(raw=>raw.includes('Offline sleep notes')));
});

const {intakeAmount, intakeDay} = await import('../entry/src/main/ets/models/HealthIntake.ets');
const intakePath = '/api/v1/assistant/health/intake';
const snapshotPath = intakePath + '?space=health';

function intakeFixture() {
  const env = fixture({managers:['health-owner','next-owner']});
  const snapshot = {date:'2026-09-25',zone:'Asia/Tokyo',water:{today:100,goal:1650},protein:{today:7.5,goal:42}};
  const entries = new Map(), writes = [], listeners = new Map();
  const current = {readFail:null, loseOnce:false, reply:null, readOverride:null};
  function value() {
    const byKind = kind => [...entries.values()].filter(e=>e.kind===kind && !e.deleted_at);
    const portion = kind => ({today:snapshot[kind].today+byKind(kind).reduce((sum,e)=>sum+e.amount,0),goal:snapshot[kind].goal,entries:byKind(kind)});
    return {zone:snapshot.zone,days:[{date:snapshot.date}],water:portion('water'),protein:portion('protein')};
  }
  env.root.features=['health_intake'];
  for (const [manager,child] of env.shared.childClients) {
    child.client.features=['health_intake'];
    child.client.observeEvents=fn=>{listeners.set(manager,fn);return ()=>listeners.delete(manager);};
    child.state.onCall=async call=>{
      if (call.path!==snapshotPath) return;
      if (current.readFail) throw current.readFail;
      return current.readOverride ? await current.readOverride() : value();
    };
    child.repo.transport={async request(path,method,raw){
      const b=raw && JSON.parse(raw), id=b ? b.id : path.split('/').at(-1).split('?')[0];
      writes.push({manager,path,method,body:b});
      if (current.reply) return await current.reply();
      if(method==='POST' && !entries.has(id))entries.set(id,{...b,amount:b.amount??181.5});
      if(method==='DELETE')entries.get(id).deleted_at='2026-09-25T00:13:00Z';
      if(current.loseOnce){current.loseOnce=false;throw failure(0,'disconnected');}
      const v=value();return JSON.stringify({today:{water:v.water.today,protein:v.protein.today},zone:v.zone,...(b?{entry:entries.get(id)}:{})});
    }};
  }
  return {...env,value,entries,writes,listeners,current,workspace:new HealthWorkspace()};
}

test('Hub goal, calendar, fractional units and the three genuine Health intake operations round-trip with undo', async () => {
  const env=intakeFixture(), intake=env.workspace.intake;
  await env.workspace.open(env.root);
  assert.deepEqual([intake.data.zone,intake.data.date,intake.data.water.today,intake.data.water.goal,intake.data.protein.today,intake.data.protein.goal],['Asia/Tokyo','2026-09-25',100,1650,7.5,42]);
  for(const [kind,amount] of [['water',undefined],['water',125.5],['protein',0.5]]){
    const readCount=env.child().state.calls.filter(c=>c.path===snapshotPath).length;
    const pending=intake.record(kind,amount);await Promise.all([pending,intake.record(kind,amount)]);
    const record=env.writes.at(-1);
    assert.equal(record.path,snapshotPath);assert.equal(record.manager,'health-owner');assert.equal(record.method,'POST');
    assert.deepEqual(Object.keys(record.body).sort(),['at','id','kind',...(amount===undefined?[]:['amount'])].sort());
    assert.match(record.body.id,/^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/);assert.equal(new Date(record.body.at).toISOString(),record.body.at);
    assert.equal(record.body.kind,kind);assert.equal(record.body.amount,amount);
    assert.equal(intake.data[kind].today,env.value()[kind].today);
    assert.equal(intake.notice,'health_intake_logged');assert.ok(env.child().state.calls.filter(c=>c.path===snapshotPath).length>readCount);
    const id=intake.actionId, previousWrites=env.writes.length;
    await Promise.all([intake.act(id),intake.act(id)]);
    assert.equal(env.writes.length,previousWrites+1);assert.equal(env.writes.at(-1).path,intakePath+'/'+record.body.id+'?space=health');
    assert.equal(intake.notice,'health_intake_undone');assert.equal(intake.actionId,'');
    assert.equal(intake.data.water.today,100);assert.equal(intake.data.protein.today,7.5);
  }
  assert.equal(env.writes.length,6);
  env.workspace.hide();assert.equal(env.listeners.size,0);
});

test('Health intake retries the same authoritative identity after a commit with a lost response; foreign owners and denied accounts cannot replay',async()=>{
  const env=intakeFixture(), intake=env.workspace.intake;await env.workspace.open(env.root);
  env.current.loseOnce=true;await intake.record('protein',4.5);
  const first=env.writes[0],retry=intake.actionId;
  assert.equal(intake.data.protein.today,7.5);assert.equal(intake.error,'health_intake_record_failed');
  await intake.act(retry);assert.deepEqual(env.writes[1],first);assert.equal(intake.data.protein.today,12);assert.equal(env.entries.size,1);
  const undo=intake.actionId;
  env.shared.defaultManager='next-owner';await intake.act(undo);assert.equal(env.writes.length,2);assert.equal(intake.data,undefined);
  await env.workspace.refresh();assert.equal(intake.currentClient(),env.child('next-owner').client);await intake.act(undo);assert.equal(env.writes.length,2);
  env.current.reply=async()=>{throw failure(403,'forbidden')};await intake.record('water',5);assert.equal(intake.error,'permission_denied');assert.equal(intake.data,undefined);assert.equal(intake.actionId,'');
});

test('Health only accepts finite, one-decimal quantities from the two controls and requires a complete Hub snapshot',()=>{
  for (const [kind,text,expected] of [['water','1',1],['water','3000',3000],['water','8,5',8.5],['water','0.9',undefined],['water','3000.1',undefined],['protein','0.1',.1],['protein','.2',.2],['protein','200',200],['protein','200.1',undefined],['protein','0.01',undefined],['water','NaN',undefined],['tea','20',undefined],['water','',undefined]]) assert.equal(intakeAmount(kind,text),expected);
  const entry=intakeFixture().value();
  for(const bad of [{},{...entry,zone:''},{...entry,days:[]},{...entry,water:{today:7}},{...entry,protein:{today:NaN,goal:9}}])assert.throws(()=>intakeDay(bad));
});

test('Health switches and delayed reads cannot transfer personal totals; old Hubs perform zero intake requests',async()=>{
  const env=intakeFixture(),i=env.workspace.intake;env.root.features=[];await env.workspace.open(env.root);
  assert.equal(i.supported,false);assert.equal(i.data,undefined);assert.equal(env.shared.calls.some(c=>c.path===snapshotPath),false);await i.record('water');assert.equal(env.writes.length,0);
  env.root.features=['health_intake'];await env.workspace.refresh();
  const late=deferred(), began=deferred();env.current.readOverride=()=>{began.resolve();return late.promise};const outstanding=i.refresh();await began.promise;
  env.workspace.hide();late.resolve(env.value());await outstanding;assert.equal(i.data,undefined);assert.equal(i.currentClient(),undefined);
  env.current.readOverride=null;env.current.readFail=failure(500,'server_error');await env.workspace.open(env.root);
  assert.equal(i.data,undefined);assert.equal(i.error,'health_intake_load_failed');assert.equal(env.workspace.conversations.length,1);
  env.current.readFail=null;await i.refresh();assert.equal(i.currentClient(),env.child().client);
});

test('Health observes only the current manager’s intake and resnapshots after a write without racing its receipt',async()=>{
  const env=intakeFixture(),i=env.workspace.intake;await env.workspace.open(env.root);
  const cb=env.listeners.get('health-owner'), count=()=>env.child().state.calls.filter(c=>c.path===snapshotPath).length;
  const first=count();cb('{"type":"health_intake","manager":"elsewhere"}');cb('{"type":"notice","x":"health_intake"}');cb('{"type":"health_intake"');assert.equal(count(),first);
  cb('{"type":"health_intake"}');await new Promise(resolve=>setImmediate(resolve));assert.equal(count(),first+1);
  const enter=deferred(),resume=deferred();env.current.reply=()=>{enter.resolve();return resume.promise};
  const writing=i.record('water',25);await enter.promise;const active=count();cb('{"type":"health_intake","manager":"health-owner"}');assert.equal(count(),active);
  resume.resolve(JSON.stringify({today:{water:125,protein:7.5},zone:'Asia/Tokyo'}));await writing;assert.equal(count(),active+1);
  assert.equal(i.data.water.today,100,'authoritative follow-up read takes precedence over a transient receipt');
});

test('the official omitted or empty machine trust means confirmed; missing/offline/pending or foreign machines stay unavailable',async()=>{
  for (const [shape,allowed] of [[{},true],[{trust:''},true],[{trust:'personal'},true],[{trust:'work'},true],[{trust:'pending'},false],[{trust:'unknown'},false],[{trust:false},false],[{manager:'elsewhere'},false],[{online:false},false],[{pending_confirm:true},false]]) {
    const {workspace,shared}=await opened({childOnCall:async c=>c.path==='/api/v1/machines'?{machines:[{id:'health-machine',manager:'health-owner',online:true,...shape}]}:undefined});
    assert.equal(workspace.canUploadReports(),allowed);assert.equal(workspace.canStartConversation(),allowed);
    if(allowed) {assert.equal(workspace.error,'');assert.ok(await workspace.uploadReport('fixture.pdf',new ArrayBuffer(5)));}
    else assert.equal(shared.calls.some(c=>c.method==='UPLOAD'),false);
  }
});

test('a failed readback keeps an acknowledged undo explicitly separate from a safe GET retry; revoked access drops both',async()=>{
  const env=intakeFixture(),i=env.workspace.intake;await env.workspace.open(env.root);
  env.current.readFail=failure(503,'unavailable');await i.record('protein',5);
  const undo=i.actionId;assert.equal(i.notice,'health_intake_logged');assert.equal(i.error,'');assert.equal(i.refreshError,'health_intake_load_failed');
  assert.equal(i.data.protein.today,12.5);assert.equal(env.writes.length,1);await i.refresh();assert.equal(env.writes.length,1);assert.equal(i.actionId,undo);
  env.current.readFail=null;await i.refresh();assert.equal(i.refreshError,'');assert.equal(i.actionId,undo);
  env.current.readFail=failure(403,'forbidden');await i.refresh();assert.equal(i.data,undefined);assert.equal(i.notice,'');assert.equal(i.actionId,'');
  await i.act(undo);assert.equal(env.writes.length,1);
});

test('Health upload and suggestion controls subscribe to machine readiness on their first native render',()=>{
  const workspace=new HealthWorkspace(),observed=[];
  Object.defineProperty(workspace,'reportUploadAllowed',{get(){observed.push('reportUploadAllowed');return false;},configurable:true});
  assert.equal(workspace.canUploadReports(),false);assert.deepEqual(observed,['reportUploadAllowed']);
  observed.length=0;assert.equal(workspace.canStartConversation(),false);assert.deepEqual(observed,['reportUploadAllowed']);
});
