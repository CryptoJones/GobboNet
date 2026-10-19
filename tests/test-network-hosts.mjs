// Exercise the actual CONFIG handlers with server responses, including failed
// saves. No DNS lookup or external connection is needed for these regressions.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import vm from 'node:vm';
const source = fs.readFileSync(new URL('../js/15-cards.js', import.meta.url), 'utf8');
const html = fs.readFileSync(new URL('../chat.html', import.meta.url), 'utf8');
function panel({ served = true, code = 200, reply = { allowed_hosts: [] }, offline = false } = {}) {
  const elements = Object.fromEntries(['set-network-hosts', 'network-hosts-save', 'network-hosts-status'].map(id => [id, { value: '', disabled: true, textContent: '' }]));
  const requests = [];
  const ctx = vm.createContext({
    IS_SERVED: served, window: { location: { origin: 'http://127.0.0.1:9066' } },
    document: { getElementById: id => elements[id] },
    fetch: async (url, options) => {
      requests.push({ url, options });
      if (offline) throw Error('offline');
      return { ok: code === 200, status: code, json: async () => reply };
    },
  });
  vm.runInContext(source, ctx);
  return { ctx, elements, requests };
}
assert.match(source.slice(source.indexOf('function openSettings()'), source.indexOf('function closeSettings')), /loadNetworkHostsPanel\(\)/);
for (const id of ['set-network-hosts', 'network-hosts-save', 'network-hosts-status']) assert.ok(html.includes(`id="${id}"`));
{
  const p = panel({ reply: { allowed_hosts: ['my-goblin.duckdns.org', 'desktop.tail123.ts.net'] } });
  await p.ctx.loadNetworkHostsPanel();
  assert.equal(p.elements['set-network-hosts'].value, 'my-goblin.duckdns.org\ndesktop.tail123.ts.net');
  assert.equal(p.elements['network-hosts-save'].disabled, false);
  p.elements['set-network-hosts'].value = 'https://My-Goblin.duckdns.org:9066/ , desktop.tail123.ts.net';
  await p.ctx.saveNetworkHosts();
  assert.deepEqual(JSON.parse(p.requests[1].options.body), { allowed_hosts: ['https://My-Goblin.duckdns.org:9066/', 'desktop.tail123.ts.net'] });
  assert.equal(p.requests[1].url, 'http://127.0.0.1:9066/network/hosts');
  assert.match(p.elements['network-hosts-status'].textContent, /^Saved/);
  assert.equal(p.elements['set-network-hosts'].value, 'my-goblin.duckdns.org\ndesktop.tail123.ts.net');
  p.elements['set-network-hosts'].value = '';
  await p.ctx.saveNetworkHosts();
  assert.deepEqual(JSON.parse(p.requests[2].options.body), { allowed_hosts: [] });
}
for (const config of [{ code: 401 }, { code: 404 }, { offline: true }, { reply: {} }, { served: false }]) {
  const p = panel(config);
  await p.ctx.loadNetworkHostsPanel();
  assert.equal(p.elements['set-network-hosts'].disabled, true);
  assert.equal(p.elements['network-hosts-save'].disabled, true);
  assert.ok(p.elements['network-hosts-status'].textContent.length);
}
for (const config of [{ code: 400, reply: { error: 'Invalid hostname' } }, { code: 500, reply: { error: 'Could not save config.toml' } }, { offline: true }]) {
  const p = panel(config);
  p.elements['set-network-hosts'].disabled = p.elements['network-hosts-save'].disabled = false;
  p.elements['set-network-hosts'].value = 'my-goblin.duckdns.org';
  await p.ctx.saveNetworkHosts();
  assert.equal(p.elements['set-network-hosts'].value, 'my-goblin.duckdns.org');
  assert.equal(p.elements['network-hosts-save'].disabled, false);
  assert.doesNotMatch(p.elements['network-hosts-status'].textContent, /^Saved/);
}
console.log('PASS: CONFIG network names, load/save/clear, normalization response, unsupported/offline/auth and save errors');
