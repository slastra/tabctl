// Tests for the generated extension bundles. Loads the concatenated
// background.js in a vm sandbox with mocked browser globals, then drives the
// JSON-RPC dispatch and asserts on the messages posted back to the mediator.
//
// Run: node --test extensions/test/
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import vm from 'node:vm';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');

// buildMockApi returns a { api, port } pair. `api` is the global object the
// bundle runs against (chrome or browser); `port` captures postMessage calls
// and lets the test inject inbound messages via port.emit().
function buildMockApi(kind, tabsImpl) {
  const sent = [];
  let onMessage = null;
  let onDisconnect = null;

  const port = {
    postMessage: (m) => sent.push(m),
    onMessage: { addListener: (fn) => { onMessage = fn; } },
    onDisconnect: { addListener: (fn) => { onDisconnect = fn; } },
    sent,
    emit: (m) => onMessage(m),
    disconnect: () => onDisconnect && onDisconnect(),
  };

  const runtime = {
    getManifest: () => ({ version: '2.0.0' }),
    connectNative: () => port,
    onStartup: { addListener: () => {} },
    onInstalled: { addListener: () => {} },
  };

  const tabsApi = {
    query: tabsImpl.query,
    remove: tabsImpl.remove,
    create: tabsImpl.create,
    update: tabsImpl.update,
  };
  const windowsApi = { update: tabsImpl.windowsUpdate || (() => Promise.resolve({})) };

  const g = {
    console,
    JSON,
    Math,
    Date,
    Promise,
    Object,
    Array,
    setTimeout: () => {}, // reconnect timer is inert in tests
  };
  if (kind === 'chrome') {
    g.chrome = {
      runtime,
      tabs: tabsApi,
      windows: windowsApi,
      alarms: { create: () => {}, onAlarm: { addListener: () => {} } },
    };
  } else {
    g.browser = { runtime, tabs: tabsApi, windows: windowsApi };
  }

  return { g, port };
}

function loadBundle(kind, tabsImpl) {
  const { g, port } = buildMockApi(kind, tabsImpl);
  const file = kind === 'chrome' ? 'chrome/background.js' : 'firefox/background.js';
  const code = readFileSync(join(ROOT, file), 'utf8');
  vm.runInNewContext(code, g);
  return port;
}

// Wait a microtask turn so promise-based handlers settle.
const tick = () => new Promise((r) => setImmediate(r));

for (const kind of ['chrome', 'firefox']) {
  test(`${kind}: sends hello on connect`, () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    const hello = port.sent.find((m) => m.method === 'hello');
    assert.ok(hello, 'expected a hello message');
    assert.equal(hello.params.protocolVersion, 2);
    assert.equal(hello.params.extensionVersion, '2.0.0');
  });

  test(`${kind}: list_tabs returns structured tabs`, async () => {
    const port = loadBundle(kind, {
      query: () => Promise.resolve([
        { windowId: 1, id: 42, title: 'GitHub', url: 'https://github.com', index: 0, active: true, pinned: false },
      ]),
    });
    port.emit({ jsonrpc: '2.0', id: 7, method: 'list_tabs' });
    await tick();
    const resp = port.sent.find((m) => m.id === 7);
    assert.ok(resp, 'expected a response for id 7');
    assert.equal(resp.result[0].tabId, 42);
    assert.equal(resp.result[0].windowId, 1);
    assert.equal(resp.result[0].title, 'GitHub');
    // no TSV, no composed string id
    assert.equal(typeof resp.result[0].tabId, 'number');
  });

  test(`${kind}: close_tabs success returns null result`, async () => {
    const port = loadBundle(kind, {
      query: () => Promise.resolve([]),
      remove: () => Promise.resolve(),
    });
    port.emit({ jsonrpc: '2.0', id: 8, method: 'close_tabs', params: { tab_ids: [1, 2] } });
    await tick();
    const resp = port.sent.find((m) => m.id === 8);
    assert.ok(resp && 'result' in resp && resp.result === null);
  });

  test(`${kind}: close failure reports an error, not OK`, async () => {
    const port = loadBundle(kind, {
      query: () => Promise.resolve([]),
      remove: () => Promise.reject(new Error('no such tab')),
    });
    port.emit({ jsonrpc: '2.0', id: 9, method: 'close_tabs', params: { tab_ids: [999] } });
    await tick();
    const resp = port.sent.find((m) => m.id === 9);
    assert.ok(resp && resp.error, 'expected an error response');
    assert.match(resp.error.message, /no such tab/);
  });

  test(`${kind}: activate failure propagates`, async () => {
    const port = loadBundle(kind, {
      query: () => Promise.resolve([]),
      update: () => Promise.reject(new Error('stale tab')),
    });
    port.emit({ jsonrpc: '2.0', id: 10, method: 'activate_tab', params: { tab_id: 123, focused: false } });
    await tick();
    const resp = port.sent.find((m) => m.id === 10);
    assert.ok(resp && resp.error);
  });

  test(`${kind}: unknown method returns an error`, async () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    port.emit({ jsonrpc: '2.0', id: 11, method: 'no_such_method' });
    await tick();
    const resp = port.sent.find((m) => m.id === 11);
    assert.ok(resp && resp.error);
  });

  test(`${kind}: open_urls with one bad url still opens the rest`, async () => {
    let n = 0;
    const port = loadBundle(kind, {
      query: () => Promise.resolve([]),
      create: (opts) => {
        n += 1;
        return opts.url.includes('bad')
          ? Promise.reject(new Error('blocked'))
          : Promise.resolve({ windowId: 1, id: 100 + n });
      },
    });
    port.emit({ jsonrpc: '2.0', id: 12, method: 'open_urls', params: { urls: ['https://ok.example', 'https://bad.example'] } });
    await tick();
    await tick();
    const resp = port.sent.find((m) => m.id === 12);
    // One failed -> error, but the good one was still attempted (n === 2).
    assert.ok(resp && resp.error, 'expected an error naming the failure');
    assert.equal(n, 2, 'both urls should have been attempted');
  });
}
