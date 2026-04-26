// Firefox/Zen adapter (Manifest V2)
// Provides: TAB_PREFIX, RECONNECT_DELAY, NATIVE_APP_NAME, port, browserTabs,
//           sendResponse, sendError, connect, FirefoxTabs class

const TAB_PREFIX = 'f.';
const RECONNECT_DELAY = 5000;
const NATIVE_APP_NAME = 'tabctl_mediator';

var port = undefined;
var browserTabs = undefined;

class FirefoxTabs {
  constructor(browser) {
    this._browser = browser;
  }

  list(queryInfo, onSuccess) {
    this._browser.tabs.query(queryInfo).then(
      onSuccess,
      (error) => { /* Error listing tabs */ }
    );
  }

  close(tab_ids, onSuccess) {
    this._browser.tabs.remove(tab_ids).then(
      onSuccess,
      (error) => { /* Error removing tab */ }
    );
  }

  create(createOptions, onSuccess) {
    if (createOptions.windowId === 0) {
      this._browser.windows.create({ url: createOptions.url }).then(
        onSuccess,
        (error) => { /* Error in tab operation */ }
      );
    } else {
      this._browser.tabs.create(createOptions).then(
        onSuccess,
        (error) => { /* Error in tab operation */ }
      );
    }
  }

  activate(tab_id, focused) {
    this._browser.tabs.update(tab_id, {'active': true}).then(
      (tab) => {
        if (focused) {
          this._browser.windows.update(tab.windowId, {focused: true}).then(
            () => { /* Window focused */ },
            (error) => { /* Error focusing window */ }
          );
        }
      },
      (error) => { /* Error updating tab */ }
    );
  }
}

/**
 * Send a standardized success response to the mediator
 */
function sendResponse(data) {
  if (!port) {
    return;
  }

  try {
    port.postMessage({result: data});
  } catch (error) {
    // Send failed
  }
}

/**
 * Send a standardized error response to the mediator
 */
function sendError(message) {
  if (!port) {
    return;
  }

  try {
    port.postMessage({error: message});
  } catch (error) {
    // Send failed
  }
}

/**
 * Establish native messaging connection
 */
function connect() {
  if (port) {
    return;
  }

  port = browser.runtime.connectNative(NATIVE_APP_NAME);
  browserTabs = new FirefoxTabs(browser);

  port.onMessage.addListener(handleMessage);
  port.onDisconnect.addListener(handleDisconnect);

  // Send a test ping after connection
  setTimeout(() => {
    try {
      if (port) {
        port.postMessage({type: 'ping', timestamp: Date.now()});
      }
    } catch (e) {
      // Ping failed
    }
  }, 1000);
}

// Connect immediately
connect();

// --- core.js follows ---
