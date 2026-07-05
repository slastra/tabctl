// Shared core logic for tabctl browser extensions.
// Requires the adapter (concatenated above this file) to define:
//   TAB_PREFIX   - 'c.' or 'f.'
//   runtimeApi   - chrome.runtime or browser.runtime (must have connectNative)
//   browserTabs  - promise-based tab operations:
//                    list(queryInfo)          -> Promise<Tab[]>
//                    close(tabIds)            -> Promise<void>
//                    create(createOptions)    -> Promise<Tab|Window> (Window when windowId === 0)
//                    activate(tabId, focused) -> Promise<any>
//   onConnected(port) - OPTIONAL hook invoked after each connectNative()
// Adapters MUST NOT call core functions at top level (const TDZ under
// concatenation); they may only register event listeners. The connect()
// call at the bottom of this file is the single startup entry point.

const NATIVE_APP_NAME = 'tabctl_mediator';
const RECONNECT_DELAY_BASE = 1000;   // 1s
const RECONNECT_DELAY_MAX = 30000;   // 30s cap
const STABLE_CONNECTION_MS = 10000;  // connection older than this resets backoff

var port = null;
var reconnectDelay = RECONNECT_DELAY_BASE;
var lastConnectTime = 0;

// ============================================================================
// HELPERS
// ============================================================================

/**
 * Build a prefixed tab ID string: "c.windowId.tabId" or "f.windowId.tabId"
 */
function makeTabId(windowId, tabId) {
  return `${TAB_PREFIX}${windowId}.${tabId}`;
}

/**
 * Extract numeric tab ID from full tab ID format. The CLI may attach an
 * additional browser-name segment (e.g. "helium.c.1234.5678"); we always
 * take the last dot-separated component.
 */
function parseTabId(fullTabId) {
  if (typeof fullTabId === 'string' && fullTabId.includes('.')) {
    const parts = fullTabId.split('.');
    return parseInt(parts[parts.length - 1], 10);
  }
  return parseInt(fullTabId, 10);
}

function formatTabToTSV(tab) {
  return `${TAB_PREFIX}${tab.windowId}.${tab.id}\t${tab.title}\t${tab.url}\t${tab.index}\t${tab.active}\t${tab.pinned}`;
}

function compareWindowIdTabId(tabA, tabB) {
  if (tabA.windowId != tabB.windowId) {
    return tabA.windowId - tabB.windowId;
  }
  return tabA.index - tabB.index;
}

function errMsg(e) {
  return (e && e.message) ? e.message : String(e);
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
    // Dead port; onDisconnect will fire and reschedule
  }
}

function sendResponse(data) {
  postToMediator({ result: data });
}

function sendError(message) {
  postToMediator({ error: message });
}

// ============================================================================
// TAB OPERATIONS
// ============================================================================

function listTabs() {
  browserTabs.list({}).then(tabs => {
    if (!Array.isArray(tabs)) {
      sendError('Invalid tabs data received');
      return;
    }
    tabs.sort(compareWindowIdTabId);
    sendResponse(tabs.map(formatTabToTSV));
  }, err => sendError('Failed to list tabs: ' + errMsg(err)));
}

function closeTabs(tab_ids) {
  if (!Array.isArray(tab_ids)) {
    sendError('Invalid tab_ids parameter');
    return;
  }
  const ids = tab_ids.map(parseTabId);
  if (ids.some(isNaN)) {
    sendError('Invalid tab ID in: ' + tab_ids.join(','));
    return;
  }
  browserTabs.close(ids).then(
    () => sendResponse('OK'),
    err => sendError('Failed to close tabs: ' + errMsg(err)));
}

function activateTab(tab_id, focused) {
  const id = parseTabId(tab_id);
  if (isNaN(id)) {
    sendError(`Invalid tab ID: ${tab_id}`);
    return;
  }
  browserTabs.activate(id, focused).then(
    () => sendResponse('OK'),
    err => sendError('Failed to activate tab ' + tab_id + ': ' + errMsg(err)));
}

function openUrls(urls, window_id) {
  if (!Array.isArray(urls)) {
    sendError('Invalid urls parameter');
    return;
  }
  if (urls.length === 0) {
    sendResponse([]);
    return;
  }

  const results = [];
  let remaining = urls;
  let ready = Promise.resolve(window_id);

  if (window_id === 0) {
    // First URL opens a new window; the rest go into it
    ready = browserTabs.create({ url: urls[0], windowId: 0 }).then(win => {
      if (!win) {
        throw new Error('window creation returned no window');
      }
      if (win.tabs && win.tabs[0]) {
        results.push(makeTabId(win.id, win.tabs[0].id));
      }
      remaining = urls.slice(1);
      return win.id;
    });
  }

  ready.then(winId =>
    Promise.allSettled(remaining.map(u => browserTabs.create({ url: u, windowId: winId })))
  ).then(settled => {
    const failures = [];
    for (const s of settled) {
      if (s.status === 'fulfilled' && s.value) {
        results.push(makeTabId(s.value.windowId, s.value.id));
      } else {
        failures.push(errMsg(s.reason));
      }
    }
    if (failures.length) {
      sendError('Failed to open ' + failures.length + ' url(s): ' + failures[0]);
    } else {
      sendResponse(results);
    }
  }).catch(err => sendError('Failed to open urls: ' + errMsg(err)));
}

// ============================================================================
// MESSAGE HANDLING
// ============================================================================

function handleMessage(command) {
  // Any inbound traffic proves the link is healthy
  reconnectDelay = RECONNECT_DELAY_BASE;

  if (!command) {
    return;
  }

  // Handle pong responses (Firefox sends ping on connect)
  if (command.type === 'pong') {
    return;
  }

  if (!command.name) {
    return;
  }

  if (command['name'] == 'list_tabs') {
    listTabs();
  }

  else if (command['name'] == 'close_tabs') {
    closeTabs(command['args']['tab_ids']);
  }

  else if (command['name'] == 'open_urls') {
    openUrls(command['args']['urls'], command['args']['window_id']);
  }

  else if (command['name'] == 'activate_tab') {
    activateTab(command['args']['tab_id'], !!command['args']['focused']);
  }
}

// ============================================================================
// CONNECTION LIFECYCLE
// ============================================================================

function connect() {
  if (port) {
    return;
  }
  lastConnectTime = Date.now();
  port = runtimeApi.connectNative(NATIVE_APP_NAME);
  port.onMessage.addListener(handleMessage);
  port.onDisconnect.addListener(handleDisconnect);
  if (typeof onConnected === 'function') {
    onConnected(port);
  }
}

function ensureConnected() {
  if (!port) {
    connect();
  }
}

function handleDisconnect() {
  port = null;

  // connectNative "succeeds" even when the host is missing; failure arrives
  // as an immediate disconnect. A connection that survived
  // STABLE_CONNECTION_MS was real, so reset backoff before scheduling.
  if (Date.now() - lastConnectTime >= STABLE_CONNECTION_MS) {
    reconnectDelay = RECONNECT_DELAY_BASE;
  }
  const delay = reconnectDelay;
  reconnectDelay = Math.min(reconnectDelay * 2, RECONNECT_DELAY_MAX);

  // In Chrome MV3 this timer can die with the service worker; the
  // keepalive alarm (adapter) covers that gap.
  setTimeout(connect, delay);
}

// Single startup entry point (must stay the last line of the concatenated script)
connect();
