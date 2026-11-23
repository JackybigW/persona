import test from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { stripTypeScriptTypes } from 'node:module';
import { projectToolTurn } from '../entry/src/main/ets/core/Domain.ts';
import { workingTurnLabel } from '../entry/src/main/ets/core/PhoneDate.ts';
import { stepsToItems } from '../../integrations/antigravity/lib/map.js';

const source = readFileSync(new URL('../entry/src/main/ets/models/ClientModel.ets', import.meta.url), 'utf8');

function method(name, file = source) {
  const match = new RegExp('  private ' + name + '\\(').exec(file);
  assert.ok(match, name);
  const start = match.index, body = file.indexOf('{', start);
  let depth = 1, quote = '', escape = false, index = body + 1;
  for (; index < file.length && depth; index += 1) {
    const character = file[index];
    if (quote) {
      if (escape) escape = false;
      else if (character === '\\') escape = true;
      else if (character === quote) quote = '';
    } else if (character === '"' || character === "'" || character === '`') quote = character;
    else if (character === '{') depth += 1;
    else if (character === '}') depth -= 1;
  }
  return file.slice(start, index);
}

class ChatMessage {
  constructor(id, role, text = '') { Object.assign(this, { id, role, text }); }
}

const ProjectionHarness = new Function(
  'projectToolTurn', 'ChatMessage', 'messageDisplay', 'parseMessageContent',
  'return ' + stripTypeScriptTypes('class ProjectionHarness {' + method('projectMessages') + method('itemTime') + '}'),
)(
  projectToolTurn,
  ChatMessage,
  text => ({ text, sources: [] }),
  text => ({ text, images: [], files: [] }),
);

const configs = [
  { reading_case: true },
  { reading_case: true, reading_prepend: true },
  { reading_case: true, reading_prepend: true, reading_append: true },
];
const fixturePath = fileURLToPath(new URL('./native-ui-fixture.py', import.meta.url));
const python = [
  'import importlib.util, json, sys',
  "spec = importlib.util.spec_from_file_location('native_ui_fixture', sys.argv[1])",
  'fixture = importlib.util.module_from_spec(spec)',
  'spec.loader.exec_module(fixture)',
  'controls = json.loads(sys.argv[2])',
  'print(json.dumps([fixture.transcript(control=control)["items"] for control in controls]))',
].join('; ');
const transcripts = JSON.parse(execFileSync(
  'python3', ['-c', python, fixturePath, JSON.stringify(configs)], { encoding: 'utf8' },
));

function project(items, running = false) {
  const model = Object.assign(new ProjectionHarness(), {
    target: () => 'fixture-reading-target',
    workText: step => step.item?.text ?? '',
  });
  return model.projectMessages(items, running);
}

test('a live final turn keeps its work line between the real input and answer without fabricating tool activity', () => {
  const at='2026-11-23T01:00:00Z';
  const items=[{id:'input',role:'user',kind:'text',text:'Convert to JPEG',ts:at},
               {id:'answer',role:'assistant',kind:'text',text:'The plan is ready',ts:'2026-11-23T01:04:00Z'}];
  const rows=project(items,true);
  assert.deepEqual(rows.map(row=>row.role),['user','work','assistant']);
  assert.equal(rows[1].at,Date.parse(at));assert.equal(rows[1].work,'');assert.equal(rows[1].running,true);
  assert.equal(rows[1].toolCount,0);assert.equal(rows[1].thoughtCount,0);
  assert.deepEqual(project(items).map(row=>row.role),['user','assistant']);
  const strings={working:'Working',phone_working_seconds:'Working {s} s',phone_working_minutes:'Working {m} min',
                 phone_working_minutes_seconds:'Working {m}m {s}s',phone_working_hours_minutes:'Working {h} hr {m} min'};
  const label=(delta,start=rows[1].at)=>workingTurnLabel(start,rows[1].at+delta*1000,key=>strings[key]);
  assert.equal(label(9),'Working 9 s');assert.equal(label(60),'Working 1 min');
  assert.equal(label(91),'Working 1m 31s');assert.equal(label(3661),'Working 1 hr 1 min');
  assert.equal(label(-1),'Working');assert.equal(label(20,0),'Working');
});

test('native transcript projection keeps a middle stable ID across prepend and append phases', () => {
  const middleId = 'visible-fixture-reading-50#0';
  const initial = project(transcripts[0]);
  assert.equal(initial.length, 20);
  assert.ok(initial.some(message => message.id === middleId));

  const prepended = project(transcripts[1]);
  assert.equal(prepended.length, 40);
  assert.equal(prepended[31].id, middleId);

  const appended = project(transcripts[2]);
  assert.equal(appended.length, 45);
  assert.equal(appended[31].id, middleId);
  assert.equal(appended[31].itemId, 'fixture-reading-51#0');
});

test('an Antigravity wake keeps the finished reply and its attachment in the conversation', () => {
  const raw = readFileSync(new URL('../../integrations/antigravity/test/fixtures/autonomous-wake.jsonl', import.meta.url), 'utf8');
  const steps = raw.trim().split('\n').map(line => JSON.parse(line));
  const prior = project(stepsToItems(steps.slice(0, 3)).items);
  const history = stepsToItems(steps).items;
  const after = project(history);
  const replies = after.filter(row => row.role === 'assistant');
  assert.deepEqual(replies.map(row => row.text), [steps[2].content, steps[5].content, steps[8].content]);
  assert.deepEqual(replies[0], prior.find(row => row.role === 'assistant'));
  assert.match(replies[0].text, /file: \/synthetic\/chart\.png/);
  assert.equal(after.filter(row => row.role === 'user').length, 2);
  assert.ok(after.every(row => !row.work?.includes(steps[2].content)));

  const wake = history.findIndex(item => item.wake === true);
  assert.ok(wake >= 0);
  const recap = { id: 'before-wake-recap', role: 'system', kind: 'recap', text: 'The chart is ready.' };
  const withRecap = project([...history.slice(0, wake), recap, ...history.slice(wake)]);
  assert.deepEqual(withRecap.filter(row => row.role === 'assistant').map(row => row.text), replies.map(row => row.text));
  assert.equal(withRecap.find(row => row.role === 'recap').terminalRecap, false);
});

test('real recap entries retain original answers, time ordering and all consecutive source texts', () => {
  const inputs = [
    { id: 'ask', seq: 1, role: 'user', kind: 'text', text: '看一下 PR #42' },
    { id: 'answer', seq: 2, role: 'assistant', kind: 'text', text: '两条命名建议' },
    { id: 'recap1', seq: 3, role: 'system', kind: 'recap', text: '旧回顾' },
    { id: 'recap2', seq: 4, role: 'system', kind: 'recap', text: '最新回顾：`fetchAll`' }
  ];
  const legacy = project(inputs.slice(0, 2));
  const withRecap = project(inputs);
  assert.deepEqual(withRecap.slice(0, 2), legacy);
  const last = withRecap.at(-1);
  assert.equal(last.role, 'recap');
  assert.equal(last.terminalRecap, true);
  assert.equal(last.ownerTarget, 'fixture-reading-target');
  assert.deepEqual(last.recaps.map(item => [item.id, item.text]), [['recap1', '旧回顾'], ['recap2', '最新回顾：`fetchAll`']]);
  const next = project(inputs.concat({ id: 'next', seq: 5, role: 'assistant', kind: 'text', text: '已补测试' }));
  assert.equal(next.find(item => item.role === 'recap').terminalRecap, false);
  assert.equal(next.at(-1).text, '已补测试');
  assert.ok(project([{ id: 'pretend', seq: 1, role: 'user', kind: 'recap', text: '文档' }]).every(item => item.role !== 'recap'));
});

test('full recap Markdown grows past interface type, with the body-relative reading metric and spacing between lines', () => {
  const markdown = readFileSync(new URL('../entry/src/main/ets/components/MarkdownText.ets', import.meta.url), 'utf8');
  const environment = {fontScale: 1};
  const Renderer = new Function('nativeEnvironment', 'hasHanProse', 'return ' + stripTypeScriptTypes(
    'class Renderer {' + ['readingPoints','readingFont','inlineSpacing','inlineLineHeight'].map(name=>method(name,markdown)).join('\n') + '}',
  ))(environment, () => true);
  const renderer = Object.assign(new Renderer(), {recapReading: true});
  assert.equal(renderer.readingFont(17), '17vp');
  environment.fontScale = 2;
  const intermediate = renderer.readingPoints(17);
  assert.ok(intermediate > 23 && intermediate < 48);
  environment.fontScale = 3.2;
  assert.equal(renderer.readingFont(17), '48vp');
  assert.equal(renderer.inlineSpacing(17,0.26),48*0.26);
  renderer.recapProse = true;
  assert.equal(renderer.inlineSpacing(17,0.26),0);
  assert.equal(parseFloat(renderer.inlineLineHeight(17,true,0.26,[])),66.48);
  renderer.recapProse = false;
  renderer.recapReading = false;
  assert.equal(renderer.readingFont(17), 17);
  renderer.promptBodyVp = 33;
  assert.equal(renderer.inlineSpacing(17,0.26),0);
  assert.equal(renderer.inlineLineHeight(17,false,0.26,[],true),'47.85vp');
  renderer.scaleAgentProse=true;
  for(const [points,advance] of [[26,38],[31,45],[48,69+1/3]]) {
    renderer.agentBodyVp=points;
    assert.ok(Math.abs(parseFloat(renderer.inlineLineHeight(17,true,0.26,[]))-advance)<0.001);
  }
  assert.match(markdown, /\.lineSpacing\(LengthMetrics\.vp\(this\.inlineSpacing\(size, lineSpacingEm, readingBody\)\), \{ onlyBetweenLines: true \}/);
});

test('recap history announces its count and identifies the newest entry only in a group', () => {
  const file = readFileSync(new URL('../entry/src/main/ets/components/ConversationRecap.ets', import.meta.url), 'utf8');
  const strings = JSON.parse(readFileSync(new URL('../entry/src/main/resources/en_US/element/string.json', import.meta.url), 'utf8'));
  const labels = new Map(strings.string.map(row => [row.name, row.value]));
  const methods = ['sheetTitle', 'written', 'historyLabel'].map(name => method(name, file)).join('\n');
  const Recap = new Function('tr', 'return ' + stripTypeScriptTypes('class Recap {' + methods + '}'))(key => labels.get(key));
  const recap = new Recap();
  recap.relative = date => date ? '10m ago' : '';
  recap.sheetItems = [{ id: 'newest' }];
  assert.equal(recap.sheetTitle(), 'While You Were Away');
  assert.equal(recap.historyLabel(1), 'Recap while you were away, tap to view');
  assert.equal(recap.written(1, 0), 'Written 10m ago');
  recap.sheetItems.push({ id: 'earlier' });
  assert.equal(recap.sheetTitle(), 'While You Were Away · 2');
  assert.equal(recap.historyLabel(2), '2 recaps while you were away, tap to view');
  assert.equal(recap.written(1, 0), 'Latest · Written 10m ago');
  assert.equal(recap.written(1, 1), 'Written 10m ago');
  assert.equal(recap.written(0, 0), '');
});

test('a recap dismissal belongs to its hub and session and an older storage read cannot close a new recap', async () => {
  const recapSource = readFileSync(new URL('../entry/src/main/ets/components/ConversationRecap.ets', import.meta.url), 'utf8');
  const Recap = new Function('return ' + stripTypeScriptTypes('class Recap {' +
    ['current', 'newest', 'syncOwner', 'open', 'dismiss'].map(name => method(name, recapSource)).join('\n') + '}'))();
  const reads = new Map(), writes = [], stored = new Map();
  let scope = 'hub-1', target = 'macbook/review';
  const v = Object.assign(new Recap(), {
    active: true, closed: '', ready: false, ownerKey: '', shown: false,
    model: { selected: {kind:'session'}, target: () => target, scope: () => scope, connectionGeneration: () => 1,
      captureView: key => {
        const address = scope + ':' + key, read = Promise.withResolvers(); reads.set(address, read);
        return { valid: () => true, read: () => read.promise, write: async value => { writes.push([address,value]);stored.set(address,value); } };
      }},
    message: {id:'recap:a', ownerTarget:target, terminalRecap:true, recaps:[{id:'a',text:'A'}]}
  });
  v.syncOwner();
  const oldAddress = 'hub-1:closed-recap:macbook/review';
  scope='hub-2';v.syncOwner();
  const address = 'hub-2:closed-recap:macbook/review';
  reads.get(address).resolve('');await Promise.resolve();
  assert.equal(v.open(),true);
  v.dismiss();
  assert.equal(v.open(),false);
  assert.deepEqual(writes,[[address,'a']]);
  reads.get(oldAddress).resolve('foreign-id');await Promise.resolve();
  assert.equal(v.closed,'a');
  target='macbook/other';v.message={id:'recap:b',ownerTarget:target,terminalRecap:true,recaps:[{id:'b'}]};v.syncOwner();
  reads.get('hub-2:closed-recap:macbook/other').resolve('');await Promise.resolve();assert.equal(v.open(),true);
  target='macbook/review';v.message={id:'recap:a',ownerTarget:target,terminalRecap:true,recaps:[{id:'a'}]};v.syncOwner();
  reads.get(address).resolve(stored.get(address));await Promise.resolve();assert.equal(v.open(),false);
  v.message={...v.message,recaps:[{id:'a'},{id:'new'}]};v.syncOwner();assert.equal(v.open(),true);
  v.active=false;v.syncOwner();v.dismiss();assert.equal(writes.length,1);
});

test('a recording profile retains its standard body and uses the measured mixed-script face and single line advance at display sizes',async()=>{
 const {agentReplyBodySize,hasHanProse,markdownProseRuns}=await import('../entry/src/main/ets/core/Markdown.ts');
 const markdown=readFileSync(new URL('../entry/src/main/ets/components/MarkdownText.ets',import.meta.url),'utf8');
 const model=readFileSync(new URL('../entry/src/main/ets/components/KnowledgeWorkspacePage.ets',import.meta.url),'utf8');
 const getters=['scaleAgentProse','recapProse','agentBodyVp','agentHanVp','profileDisplayProse']
   .map(name=>markdown.match(new RegExp('  @Computed (get '+name+'\\(\\)[^\\n]+)'))?.[1]);
 assert.ok(getters.every(Boolean));
 const environment={fontScale:1};
 const Renderer=new Function('agentReplyBodySize','hasHanProse','nativeEnvironment','Tokens',
   'return '+stripTypeScriptTypes('class Renderer {\n'+getters.concat(['inlineLineHeight','inlineSpacing','readingPoints','readingFont','proseFont']
     .map(name=>method(name,markdown))).join('\n')+'\n}'))(agentReplyBodySize,hasHanProse,environment,{body:17});
 const p=Object.assign(new Renderer(),{profileReading:true,recapReading:true,agentFontScale:1});
 const nodes=[{kind:'text',text:'[0:00] 核对项目工作树、知识库分类以及会议摘要。'}];
 assert.deepEqual(markdownProseRuns(nodes[0].text).map(r=>r.han),[false,true]);
 assert.equal(p.profileDisplayProse,false);
 assert.equal(p.readingFont(17),'17vp');
 assert.equal(p.inlineSpacing(17,0.45,false),17*0.45);
 p.agentFontScale=environment.fontScale=3.2;
 assert.equal(p.profileDisplayProse,true);assert.equal(p.agentBodyVp,48);
 assert.equal(p.proseFont(true),'45.293vp');assert.equal(p.proseFont(false),undefined);
 assert.equal(p.inlineSpacing(17,0.26,true),0);
 assert.ok(Math.abs(parseFloat(p.inlineLineHeight(17,true,0.26,nodes))-(69+1/3))<0.001);
 assert.equal(Math.floor((330-140)/p.agentHanVp),4);assert.equal(Math.floor(330/p.agentHanVp),7);
 assert.equal(Math.floor((330-140)/48),3);assert.equal(Math.floor(330/48),6);
 assert.equal(p.inlineSpacing(17,0.45,false),48*0.45,'headings, inline code and other block layouts retain their existing reading metrics');
 p.profileReading=false;assert.equal(p.profileDisplayProse,false);
 assert.match(model,/MarkdownText\(\{text:Wire\.text\(this\.workspace\.detail,'summary'\),recapReading:true,profileReading:true,\s*agentFontScale:nativeEnvironment.fontScale/);
 assert.match(model,/\.height\(this\.recordingMetric\(52\.3,59\.3\)\)[^\n]+id\('knowledge\.facts\.open'\)/);
});
