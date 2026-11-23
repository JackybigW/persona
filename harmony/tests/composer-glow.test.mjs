import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { stripTypeScriptTypes } from 'node:module';
import test from 'node:test';

const source = readFileSync(new URL('../entry/src/main/ets/components/PhoneComposerGlow.ets', import.meta.url), 'utf8');
const methods = source.slice(source.indexOf('  private schedule():'), source.indexOf('  build()'));
const bands = source.slice(source.indexOf('const LIGHT:'), source.indexOf('/**'));

function glow(reduceMotion) {
  const phases = [], alpha = [], callbacks = new Map();
  let id = 0, now = 251;
  const ctx = {
    createConicGradient(phase) { phases.push(phase); return { addColorStop() {} }; },
    beginPath() {}, moveTo() {}, arcTo() {}, closePath() {}, clearRect() {},
    stroke() { alpha.push(this.globalAlpha); }
  };
  const Rim = new Function('Date', 'setInterval', 'clearInterval',
    stripTypeScriptTypes(bands + '\nclass Rim {\n' + methods + '\n}') + '\nreturn Rim;')(
    { now: () => now }, fn => { callbacks.set(++id, fn); return id; }, n => callbacks.delete(n));
  const rim = new Rim();
  Object.assign(rim, {ctx, timer:-1, visible:true, ready:true, reduced:reduceMotion, frameWidth:378, frameHeight:52,
    env:{dark:false,foreground:true}});
  return {rim, phases, alpha, callbacks, advance() { now += 370; callbacks.forEach(cb => cb()); }};
}

test('a reduced-motion reply draws a stationary colored rim and schedules no highlight or timer', () => {
  const r=glow(true);
  r.rim.schedule();
  r.advance();
  r.rim.paint();
  assert.deepEqual(r.phases,[0,0]);
  assert.deepEqual(r.alpha,[.035,.04,.055,.065,.09,.24,1,.035,.04,.055,.065,.09,.24,1]);
  assert.equal(r.callbacks.size,0);
});

test('the normal rim advances, stops immediately on reduced motion and releases its timer off screen', () => {
  const r=glow(false);
  r.rim.schedule();
  r.advance();
  assert.notEqual(r.phases[0],r.phases[2]);
  assert.equal(r.callbacks.size,1);
  r.rim.reduced=true;
  r.rim.schedule();
  assert.equal(r.phases.at(-1),0);
  assert.equal(r.callbacks.size,0);
  r.rim.reduced=false;
  r.rim.visible=false;
  r.rim.schedule();
  assert.equal(r.callbacks.size,0);
});
