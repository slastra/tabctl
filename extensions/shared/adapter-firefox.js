// Firefox/Zen adapter (Manifest V2)
// Provides: TAB_PREFIX, RECONNECT_DELAY, NATIVE_APP_NAME, port, browserTabs,
//           sendResponse, sendError, connect, BrowserTabs/FirefoxTabs classes

const TAB_PREFIX = 'f.';
const RECONNECT_DELAY = 5000;
const NATIVE_APP_NAME = 'tabctl_mediator';

var port = undefined;
var browserTabs = undefined;

class BrowserTabs {
  constructor(browser) {
    this._browser = browser;
  }

  runtime() {
    return this._browser.runtime;
  }

  list(queryInfo, onSuccess) {
    throw new Error('list is not implemented');
  }

  query(queryInfo, onSuccess) {
    throw new Error('query is not implemented');
  }

  close(tab_ids, onSuccess) {
    throw new Error('close is not implemented');
  }

  move(tabId, moveOptions, onSuccess) {
    throw new Error('move is not implemented');
  }

  update(tabId, options, onSuccess, onError) {
    throw new Error('update is not implemented');
  }

  create(createOptions, onSuccess) {
    throw new Error('create is not implemented');
  }

  activate(tab_id) {
    throw new Error('activate is not implemented');
  }

  getActive(onSuccess) {
    throw new Error('getActive is not implemented');
  }

  getActiveScreenshot(onSuccess) {
    throw new Error('getActiveScreenshot is not implemented');
  }

  runScript(tab_id, script, payload, onSuccess, onError) {
    throw new Error('runScript is not implemented');
  }

  getBrowserName() {
    throw new Error('getBrowserName is not implemented');
  }
}

class FirefoxTabs extends BrowserTabs {
  list(queryInfo, onSuccess) {
    this._browser.tabs.query(queryInfo).then(
      onSuccess,
      (error) => { /* Error listing tabs */ }
    );
  }

  query(queryInfo, onSuccess) {
    if (queryInfo.hasOwnProperty('windowFocused')) {
      let keepFocused = queryInfo['windowFocused']
      delete queryInfo.windowFocused;
      this._browser.tabs.query(queryInfo).then(
        tabs => {
          Promise.all(tabs.map(tab => {
            return new Promise(resolve => {
              this._browser.windows.get(tab.windowId, {populate: false}, window => {
                resolve(window.focused === keepFocused ? tab : null);
              });
            });
          })).then(result => {
            tabs = result.filter(tab => tab !== null);
            onSuccess(tabs);
          });
        },
        (error) => { /* Error querying tabs */ }
      );
    } else {
      this._browser.tabs.query(queryInfo).then(
        onSuccess,
        (error) => { /* Error querying tabs */ }
      );
    }
  }

  close(tab_ids, onSuccess) {
    this._browser.tabs.remove(tab_ids).then(
      onSuccess,
      (error) => { /* Error removing tab */ }
    );
  }

  move(tabId, moveOptions, onSuccess) {
    this._browser.tabs.move(tabId, moveOptions).then(
      onSuccess,
      (error) => { /* Error moving tab */ }
    );
  }

  update(tabId, options, onSuccess, onError) {
    this._browser.tabs.update(tabId, options).then(
      onSuccess,
      (error) => {
        onError(error)
      }
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

  getActive(onSuccess) {
    this._browser.tabs.query({active: true}).then(
      onSuccess,
      (error) => { /* Error in tab operation */ }
    );
  }

  getActiveScreenshot(onSuccess) {
    let queryOptions = { active: true, lastFocusedWindow: true };
    this._browser.tabs.query(queryOptions).then(
      (tabs) => {
        let tab = tabs[0];
        let windowId = tab.windowId;
        let tabId = tab.id;
        this._browser.tabs.captureVisibleTab(windowId, { format: 'png' }).then(
          function(data) {
            const message = {
              tab: tabId,
              window: windowId,
              data: data
            };
            onSuccess(message);
          },
          (error) => { /* Error in tab operation */ }
        );
      },
      (error) => { /* Error in tab operation */ }
    );
  }

  runScript(tab_id, script, payload, onSuccess, onError) {
    this._browser.tabs.executeScript(tab_id, {code: script}).then(
      (result) => onSuccess(result, payload),
      (error) => onError(error, payload)
    );
  }

  getBrowserName() {
    return "firefox";
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
