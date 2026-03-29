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

  runtime() {
    return this._browser.runtime;
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

  query(queryInfo, onSuccess) {
    if (queryInfo.hasOwnProperty('windowFocused')) {
      let keepFocused = queryInfo['windowFocused']
      delete queryInfo.windowFocused;
      this._browser.tabs.query(queryInfo, tabs => {
        Promise.all(tabs.map(tab => {
          return new Promise(resolve => {
            this._browser.windows.get(tab.windowId, { populate: false }, window => {
              resolve(window.focused === keepFocused ? tab : null);
            });
          });
        })).then(result => {
          tabs = result.filter(tab => tab !== null);
          onSuccess(tabs);
        });
      });
    } else {
      this._browser.tabs.query(queryInfo, onSuccess);
    }
  }

  close(tab_ids, onSuccess) {
    this._browser.tabs.remove(tab_ids, onSuccess);
  }

  move(tabId, moveOptions, onSuccess) {
    this._browser.tabs.move(tabId, moveOptions, onSuccess);
  }

  update(tabId, options, onSuccess, onError) {
    this._browser.tabs.update(tabId, options, tab => {
      if (this._browser.runtime.lastError) {
        let error = this._browser.runtime.lastError.message;
        onError(error)
      } else {
        onSuccess(tab)
      }
    });
  }

  create(createOptions, onSuccess) {
    if (createOptions.windowId === 0) {
      this._browser.windows.create({ url: createOptions.url }, onSuccess);
    } else {
      this._browser.tabs.create(createOptions, onSuccess);
    }
  }

  getActive(onSuccess) {
    this._browser.tabs.query({ active: true }, onSuccess);
  }

  getActiveScreenshot(onSuccess) {
    let queryOptions = { active: true, lastFocusedWindow: true };
    this._browser.tabs.query(queryOptions, (tabs) => {
      let tab = tabs[0];
      let windowId = tab.windowId;
      let tabId = tab.id;
      this._browser.tabs.captureVisibleTab(windowId, { format: 'png' }, function (data) {
        const message = {
          tab: tabId,
          window: windowId,
          data: data
        };
        onSuccess(message);
      });
    });
  }

  async runScript(tab_id, script, payload, onSuccess, onError) {
    try {
      const results = await this._browser.scripting.executeScript({
        target: { tabId: tab_id },
        func: (scriptCode) => {
          try {
            return eval(scriptCode);
          } catch (e) {
            return null;
          }
        },
        args: [script]
      });

      const result = results && results[0] ? results[0].result : null;
      onSuccess(result ? [result] : [], payload);
    } catch (error) {
      onError(error, payload);
    }
  }

  getBrowserName() {
    return "chrome/chromium";
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
