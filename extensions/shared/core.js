// Shared core logic for tabctl browser extensions.
// Requires the adapter (concatenated above this file) to define:
//   runtimeApi        - chrome.runtime or browser.runtime (must have connectNative)
//   browserTabs       - promise-based tab operations:
//                         list(queryInfo)          -> Promise<Tab[]>
//                         close(tabIds)            -> Promise<void>
//                         create(createOptions)    -> Promise<Tab>
//                         activate(tabId, focused) -> Promise<any>
// Adapters MUST NOT call core functions at top level (const TDZ under
// concatenation); they may only register event listeners. connect() at the
// bottom of this file is the single startup entry point.

// --- Protocol ---------------------------------------------------------------
const NATIVE_APP_NAME = 'tabctl_mediator';
const JSONRPC_VERSION = '2.0';
const PROTOCOL_VERSION = 2;
const EXTENSION_VERSION = runtimeApi.getManifest().version;
const RPC_ERR_BROWSER = -32000;

// --- Reconnect / lifecycle --------------------------------------------------
const RECONNECT_DELAY_BASE = 1000;   // 1s
const RECONNECT_DELAY_MAX = 30000;   // 30s cap
const STABLE_CONNECTION_MS = 10000;  // a connection older than this resets backoff

var port = null;
var reconnectDelay = RECONNECT_DELAY_BASE;
var lastConnectTime = 0;

// ============================================================================
// HELPERS
// ============================================================================

function errMsg(e) {
  return (e && e.message) ? e.message : String(e);
}

// Map a browser tab object to the wire shape: raw numeric IDs, no composed
// string, no TSV.
function toTabData(tab) {
  return {
    windowId: tab.windowId,
    tabId: tab.id,
    title: tab.title || '',
    url: tab.url || '',
    index: tab.index,
    active: !!tab.active,
    pinned: !!tab.pinned,
  };
}

function compareWindowIdTabIndex(a, b) {
  if (a.windowId !== b.windowId) return a.windowId - b.windowId;
  return a.index - b.index;
}

// ============================================================================
// RESPONSE PLUMBING
// ============================================================================

function postToMediator(message) {
  if (!port) { connect(); }
  if (!port) { return; }
  try {
    port.postMessage(message);
  } catch (e) {
    // Dead port; onDisconnect will fire and reschedule.
  }
}

function sendResult(id, result) {
  postToMediator({ jsonrpc: JSONRPC_VERSION, id: id, result: result });
}

function sendError(id, message) {
  postToMediator({ jsonrpc: JSONRPC_VERSION, id: id, error: { code: RPC_ERR_BROWSER, message: message } });
}

// ============================================================================
// COMMAND HANDLERS (mediator -> extension requests)
// ============================================================================

function listTabs(id) {
  browserTabs.list({}).then(tabs => {
    if (!Array.isArray(tabs)) { sendError(id, 'Invalid tabs data received'); return; }
    tabs.sort(compareWindowIdTabIndex);
    sendResult(id, tabs.map(toTabData));
  }, err => sendError(id, 'Failed to list tabs: ' + errMsg(err)));
}

function closeTabs(id, tabIds) {
  if (!Array.isArray(tabIds)) { sendError(id, 'Invalid tab_ids parameter'); return; }
  browserTabs.close(tabIds).then(
    () => sendResult(id, null),
    err => sendError(id, 'Failed to close tabs: ' + errMsg(err)));
}

function activateTab(id, tabId, focused) {
  if (typeof tabId !== 'number') { sendError(id, 'Invalid tab_id: ' + tabId); return; }
  browserTabs.activate(tabId, focused).then(
    () => sendResult(id, null),
    err => sendError(id, 'Failed to activate tab ' + tabId + ': ' + errMsg(err)));
}

function openUrls(id, urls) {
  if (!Array.isArray(urls)) { sendError(id, 'Invalid urls parameter'); return; }
  if (urls.length === 0) { sendResult(id, []); return; }

  Promise.allSettled(urls.map(u => browserTabs.create({ url: u }))).then(settled => {
    const opened = [];
    const failures = [];
    for (const s of settled) {
      if (s.status === 'fulfilled' && s.value) { opened.push(toTabData(s.value)); }
      else { failures.push(errMsg(s.reason)); }
    }
    if (failures.length) { sendError(id, 'Failed to open ' + failures.length + ' url(s): ' + failures[0]); }
    else { sendResult(id, opened); }
  });
}

// ============================================================================
// MESSAGE DISPATCH
// ============================================================================

function handleMessage(msg) {
  // Any inbound traffic proves the link is healthy.
  reconnectDelay = RECONNECT_DELAY_BASE;

  if (!msg || typeof msg !== 'object') { return; }

  // A response to our hello (has result/error, no method) — nothing to do.
  if (msg.method === undefined) { return; }

  const id = msg.id;
  const params = msg.params || {};
  switch (msg.method) {
    case 'list_tabs':    listTabs(id); break;
    case 'close_tabs':   closeTabs(id, params.tab_ids); break;
    case 'activate_tab': activateTab(id, params.tab_id, !!params.focused); break;
    case 'open_urls':    openUrls(id, params.urls); break;
    default:
      if (id !== undefined) { sendError(id, 'Unknown method: ' + msg.method); }
  }
}

// ============================================================================
// CONNECTION LIFECYCLE
// ============================================================================

function sendHello() {
  postToMediator({
    jsonrpc: JSONRPC_VERSION,
    id: 0,
    method: 'hello',
    params: { extensionVersion: EXTENSION_VERSION, protocolVersion: PROTOCOL_VERSION },
  });
}

function connect() {
  if (port) { return; }
  lastConnectTime = Date.now();
  port = runtimeApi.connectNative(NATIVE_APP_NAME);
  port.onMessage.addListener(handleMessage);
  port.onDisconnect.addListener(handleDisconnect);
  sendHello();
}

function ensureConnected() {
  if (!port) { connect(); }
}

function handleDisconnect() {
  port = null;
  // connectNative "succeeds" even when the host is missing; failure arrives as
  // an immediate disconnect. A connection that survived STABLE_CONNECTION_MS
  // was real, so reset backoff before scheduling.
  if (Date.now() - lastConnectTime >= STABLE_CONNECTION_MS) {
    reconnectDelay = RECONNECT_DELAY_BASE;
  }
  const delay = reconnectDelay;
  reconnectDelay = Math.min(reconnectDelay * 2, RECONNECT_DELAY_MAX);
  setTimeout(connect, delay);
}

// Single startup entry point (must stay the last line of the concatenated script).
connect();
