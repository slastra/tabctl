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

  // Captured tab-event listeners (watchTabEvents registers these); tests
  // fire them via port.fireTab(name, ...args) to drive tabs_changed.
  const tabListeners = {};
  const mkEvent = (name) => ({ addListener: (fn) => { (tabListeners[name] ||= []).push(fn); } });

  // Controllable timers: record pending callbacks so tests can fire them.
  const timers = new Map();
  let timerSeq = 0;

  // Captured toolbar-badge state.
  const badge = { text: null, color: null, title: null };
  const badgeApi = {
    setBadgeText: (o) => { badge.text = o.text; },
    setBadgeBackgroundColor: (o) => { badge.color = o.color; },
    setTitle: (o) => { badge.title = o.title; },
  };

  const port = {
    postMessage: (m) => sent.push(m),
    onMessage: { addListener: (fn) => { onMessage = fn; } },
    onDisconnect: { addListener: (fn) => { onDisconnect = fn; } },
    sent,
    badge,
    emit: (m) => onMessage(m),
    fireTab: (name, ...args) => (tabListeners[name] || []).forEach((fn) => fn(...args)),
    disconnect: () => onDisconnect && onDisconnect(),
    fireTimers: () => {
      const fns = [...timers.values()];
      timers.clear();
      fns.forEach((fn) => fn());
    },
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
    onCreated: mkEvent('onCreated'),
    onRemoved: mkEvent('onRemoved'),
    onMoved: mkEvent('onMoved'),
    onActivated: mkEvent('onActivated'),
    onAttached: mkEvent('onAttached'),
    onDetached: mkEvent('onDetached'),
    onUpdated: mkEvent('onUpdated'),
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
    setTimeout: (fn) => { const id = ++timerSeq; timers.set(id, fn); return id; },
    clearTimeout: (id) => { timers.delete(id); },
  };
  if (kind === 'chrome') {
    g.chrome = {
      runtime,
      tabs: tabsApi,
      windows: windowsApi,
      action: badgeApi,
      alarms: { create: () => {}, onAlarm: { addListener: () => {} } },
    };
  } else {
    g.browser = { runtime, tabs: tabsApi, windows: windowsApi, browserAction: badgeApi };
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

  test(`${kind}: no hello reply badges the icon out-of-date`, () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    // hello was sent on connect; no reply arrives. Fire the pending timer.
    port.fireTimers();
    assert.equal(port.badge.text, '!');
    assert.match(port.badge.title, /out of date/i);
  });

  test(`${kind}: compatible hello reply leaves no badge`, () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    port.emit({ jsonrpc: '2.0', id: 0, result: { mediatorVersion: '2.0.0', protocolVersion: 2 } });
    port.fireTimers(); // handshake done -> timeout must not badge
    assert.notEqual(port.badge.text, '!');
  });

  test(`${kind}: protocol mismatch reply badges out-of-date`, () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    port.emit({ jsonrpc: '2.0', id: 0, result: { mediatorVersion: '9.0.0', protocolVersion: 99 } });
    assert.equal(port.badge.text, '!');
  });

  test(`${kind}: disconnect cancels the handshake timer (no false badge)`, () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    port.disconnect(); // mediator absent, not out of date
    port.fireTimers();
    assert.notEqual(port.badge.text, '!');
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

  test(`${kind}: list_tabs carries the browser-resolved favicon`, async () => {
    const port = loadBundle(kind, {
      query: () => Promise.resolve([
        { windowId: 1, id: 42, title: 'GitHub', url: 'https://github.com', index: 0,
          active: true, pinned: false, favIconUrl: 'https://github.com/favicon.ico' },
        // Inline data: URI favicons are reported verbatim, not fetched.
        { windowId: 1, id: 43, title: 'Inline', url: 'https://inline.example', index: 1,
          active: false, pinned: false, favIconUrl: 'data:image/png;base64,iVBORw0KGgo=' },
        // No icon (internal page, or not loaded yet) must not become undefined.
        { windowId: 1, id: 44, title: 'Blank', url: 'about:blank', index: 2,
          active: false, pinned: false },
      ]),
    });
    port.emit({ jsonrpc: '2.0', id: 20, method: 'list_tabs' });
    await tick();
    const resp = port.sent.find((m) => m.id === 20);
    assert.ok(resp, 'expected a response for id 20');
    assert.equal(resp.result[0].favIconUrl, 'https://github.com/favicon.ico');
    assert.equal(resp.result[1].favIconUrl, 'data:image/png;base64,iVBORw0KGgo=');
    assert.equal(resp.result[2].favIconUrl, '', 'missing favicon must serialize as an empty string');
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

  test(`${kind}: a tab event emits a debounced tabs_changed notification`, () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    port.fireTab('onCreated', { id: 1 });
    assert.ok(!port.sent.some((m) => m.method === 'tabs_changed'), 'must debounce, not emit synchronously');
    port.fireTimers();
    const note = port.sent.find((m) => m.method === 'tabs_changed');
    assert.ok(note, 'expected a tabs_changed notification after the debounce');
    assert.equal(note.jsonrpc, '2.0');
    assert.equal('id' in note, false, 'notification carries no id (no reply expected)');
  });

  test(`${kind}: a burst of tab events coalesces into one notification`, () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    port.fireTab('onCreated', { id: 1 });
    port.fireTab('onMoved', 1, {});
    port.fireTab('onActivated', { tabId: 1 });
    port.fireTab('onRemoved', 2, {});
    port.fireTimers();
    const notes = port.sent.filter((m) => m.method === 'tabs_changed');
    assert.equal(notes.length, 1, 'a burst must coalesce into a single notification');
  });

  test(`${kind}: favicon/progress-only onUpdated does not notify`, () => {
    const port = loadBundle(kind, { query: () => Promise.resolve([]) });
    port.fireTab('onUpdated', 1, { favIconUrl: 'x' });
    port.fireTab('onUpdated', 1, { status: 'loading' });
    port.fireTimers();
    assert.ok(!port.sent.some((m) => m.method === 'tabs_changed'), 'churn-only updates must not notify');
  });
}
