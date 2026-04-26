// Chrome/Brave/Chromium adapter (Manifest V3)
// Provides: TAB_PREFIX, RECONNECT_DELAY, NATIVE_APP_NAME, port, browserTabs,
//           sendResponse, sendError, connect, ChromeTabs class

const TAB_PREFIX = 'c.';
const RECONNECT_DELAY = 1000;
const NATIVE_APP_NAME = 'tabctl_mediator';

var port = undefined;
var browserTabs = undefined;

class ChromeTabs {
  constructor() {
    this._browser = chrome;
  }

  list(queryInfo, onSuccess) {
    this._browser.tabs.query(queryInfo, (tabs) => {
      onSuccess(tabs || []);
    });
  }

  activate(tab_id, focused) {
    this._browser.tabs.update(tab_id, { 'active': true });
    this._browser.tabs.get(tab_id, function (tab) {
      chrome.windows.update(tab.windowId, { focused: focused });
    });
  }

  close(tab_ids, onSuccess) {
    this._browser.tabs.remove(tab_ids, onSuccess);
  }

  create(createOptions, onSuccess) {
    if (createOptions.windowId === 0) {
      this._browser.windows.create({ url: createOptions.url }, onSuccess);
    } else {
      this._browser.tabs.create(createOptions, onSuccess);
    }
  }
}

browserTabs = new ChromeTabs();

/**
 * Send a standardized success response to the mediator
 */
function sendResponse(data) {
  if (!port) {
    connect();
  }

  if (port) {
    const message = { result: data };
    port.postMessage(message);
  }
}

/**
 * Send a standardized error response to the mediator
 */
function sendError(message) {
  if (port) {
    port.postMessage({ error: message });
  }
}

/**
 * Establish native messaging connection
 */
function connect() {
  if (port) {
    return;
  }

  port = chrome.runtime.connectNative(NATIVE_APP_NAME);
  port.onMessage.addListener(handleMessage);
  port.onDisconnect.addListener(handleDisconnect);
}

// Connect on browser startup
chrome.runtime.onStartup.addListener(() => {
  connect();
});

// Connect on install/update
chrome.runtime.onInstalled.addListener(() => {
  connect();
});

// Connect immediately when service worker starts
connect();

// --- core.js follows ---
