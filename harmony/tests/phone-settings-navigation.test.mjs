import test from 'node:test';
import assert from 'node:assert/strict';
import { phoneAppearancePage, phoneAppearancePages, phoneSettingsLink, pushPhoneSetting } from '../entry/src/main/ets/core/PhoneSettingsNavigation.ts';

test('the phone pops each appearance child to its real origin', () => {
  const directory = [];
  const appearance = pushPhoneSetting(directory, 'appearance');
  for (const child of phoneAppearancePages) {
    const fromHome = pushPhoneSetting(appearance, child);
    assert.deepEqual(fromHome, ['appearance', child]);
    assert.deepEqual(fromHome.slice(0, -1), appearance);
    assert.deepEqual(pushPhoneSetting(directory, child).slice(0, -1), directory);
    assert.equal(pushPhoneSetting(fromHome, child), fromHome);
  }
  assert.equal(phoneSettingsLink('phone_agent_layout'), 'phone_look_layout');
  assert.equal(phoneAppearancePage('phone_look_avatar'), true);
  assert.equal(phoneSettingsLink('machines'), '');
  assert.deepEqual(appearance, ['appearance']);
});
