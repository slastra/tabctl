/*
On startup, connect to the "tabctl_mediator" app.
*/

const GET_WORDS_SCRIPT = '[...new Set(document.documentElement.innerText.match(#match_regex#))].sort().join(#join_with#);';
const GET_TEXT_SCRIPT = 'document.documentElement.innerText.replace(#delimiter_regex#, #replace_with#);';
const GET_HTML_SCRIPT = 'document.documentElement.innerHTML.replace(#delimiter_regex#, #replace_with#);';

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

/**
 * Send a standardized success response to the mediator
 */
function sendResponse(data) {
  if (!port) {
    console.error("Cannot send response - port is undefined");
    return;
  }

  try {
    port.postMessage({result: data});
  } catch (error) {
    console.error("Error sending response:", error);
  }
}

/**
 * Send a standardized error response to the mediator
 */
function sendError(message) {
  if (!port) {
    console.error("Cannot send error - port is undefined");
    return;
  }

  try {
    port.postMessage({error: message});
  } catch (error) {
    console.error("Error sending error message:", error);
  }
}

/**
 * Extract numeric tab ID from full tab ID format (e.g., "f.1234.5678" -> 5678)
 */
function parseTabId(fullTabId) {
  if (typeof fullTabId === 'string' && fullTabId.includes('.')) {
    const parts = fullTabId.split('.');
    return parseInt(parts[parts.length - 1], 10);
  }
  return parseInt(fullTabId, 10);
}

/**
 * Format a tab object into TSV format with all fields (Firefox prefix)
 */
function formatTabToTSV(tab) {
  return `f.${tab.windowId}.${tab.id}\t${tab.title}\t${tab.url}\t${tab.index}\t${tab.active}\t${tab.pinned}`;
}

/**
 * Validate required command arguments
 */
function validateArgs(command, requiredFields) {
  if (!command || !command.args) {
    return 'Missing command arguments';
  }

  for (const field of requiredFields) {
    if (command.args[field] === undefined || command.args[field] === null) {
      return `Missing required argument: ${field}`;
    }
  }

  return null;
}


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
      (error) => console.log(`Error listing tabs: ${error}`)
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
        (error) => console.log(`Error executing queryTabs: ${error}`)
      );
    } else {
      this._browser.tabs.query(queryInfo).then(
        onSuccess,
        (error) => console.log(`Error executing queryTabs: ${error}`)
      );
    }
  }

  close(tab_ids, onSuccess) {
    this._browser.tabs.remove(tab_ids).then(
      onSuccess,
      (error) => console.log(`Error removing tab: ${error}`)
    );
  }

  move(tabId, moveOptions, onSuccess) {
    this._browser.tabs.move(tabId, moveOptions).then(
      onSuccess,
      (error) => console.log(`Error moving tab: ${error}`)
    );
  }

  update(tabId, options, onSuccess, onError) {
    this._browser.tabs.update(tabId, options).then(
      onSuccess,
      (error) => {
        console.log(`Error updating tab ${tabId}: ${error}`)
        onError(error)
      }
    );
  }

  create(createOptions, onSuccess) {
    if (createOptions.windowId === 0) {
      this._browser.windows.create({ url: createOptions.url }).then(
        onSuccess,
        (error) => console.log(`Error: ${error}`)
      );
    } else {
      this._browser.tabs.create(createOptions).then(
        onSuccess,
        (error) => console.log(`Error: ${error}`)
      );
    }
  }

  getActive(onSuccess) {
    this._browser.tabs.query({active: true}).then(
      onSuccess,
      (error) => console.log(`Error: ${error}`)
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
          (error) => console.log(`Error: ${error}`)
        );
      },
      (error) => console.log(`Error: ${error}`)
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
    console.log(`[TabCtl] FirefoxTabs.activate: tab_id=${tab_id}, focused=${focused}`);

    this._browser.tabs.update(tab_id, {'active': true}).then(
      (tab) => {
        console.log(`[TabCtl] Tab ${tab_id} activated successfully`);
        if (focused) {
          this._browser.windows.update(tab.windowId, {focused: true}).then(
            () => console.log(`[TabCtl] Window ${tab.windowId} focused`),
            (error) => console.error(`[TabCtl] Error focusing window: ${error}`)
          );
        }
      },
      (error) => {
        console.error(`[TabCtl] Error activating tab ${tab_id}:`, error);
        console.error(`[TabCtl] Error details:`, error.message);
      }
    );
  }
}

var port = undefined;
var browserTabs = undefined;
var reconnectTimer = null;
const NATIVE_APP_NAME = 'tabctl_mediator';

reconnect();

function reconnect() {
  // Clear any pending reconnect timer
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }

  // Avoid reconnecting if already connected
  if (port) {
    console.log("[TabCtl] Already connected, skipping reconnect");
    return;
  }

  console.log("[TabCtl] Connecting to native messaging host...");
  port = browser.runtime.connectNative(NATIVE_APP_NAME);
  browserTabs = new FirefoxTabs(browser);

  // Add message listener
  port.onMessage.addListener(handleMessage);

  // Add disconnect listener
  port.onDisconnect.addListener(handleDisconnect);

  // Send a test ping after connection
  setTimeout(() => {
    try {
      if (port) {
        port.postMessage({type: 'ping', timestamp: Date.now()});
      }
    } catch (e) {
      console.error("Failed to send ping:", e);
    }
  }, 1000);
}


function compareWindowIdTabId(tabA, tabB) {
  if (tabA.windowId != tabB.windowId) {
    return tabA.windowId - tabB.windowId;
  }
  return tabA.index - tabB.index;
}

function listTabsOnSuccess(tabs) {
  try {
    console.log(`[TabCtl] listTabsOnSuccess: Processing ${tabs ? tabs.length : 0} tabs`);

    if (!tabs || !Array.isArray(tabs)) {
      console.error('[TabCtl] Invalid tabs data received:', tabs);
      sendError('Invalid tabs data received');
      return;
    }

    // Make sure tabs are sorted by their index within a window
    tabs.sort(compareWindowIdTabId);
    const lines = tabs.map(tab => {
      const line = formatTabToTSV(tab);
      console.log(`[TabCtl] Tab formatted: ${line.substring(0, 80)}...`);
      return line;
    });

    console.log(`[TabCtl] Sending ${lines.length} formatted tab lines`);
    sendResponse(lines);
  } catch (error) {
    console.error('[TabCtl] Error in listTabsOnSuccess:', error);
    console.error('[TabCtl] Stack:', error.stack);
    sendError('Failed to process tabs list');
  }
}

function listTabs() {
  browserTabs.list({}, listTabsOnSuccess);
}

function queryTabsOnSuccess(tabs) {
  try {
    if (!tabs || !Array.isArray(tabs)) {
      console.error('queryTabsOnSuccess received invalid tabs:', tabs);
      sendResponse([]);
      return;
    }

    tabs.sort(compareWindowIdTabId);
    const lines = tabs.map(tab => formatTabToTSV(tab));
    sendResponse(lines);
  } catch (error) {
    console.error('Error in queryTabsOnSuccess:', error);
    sendError('Failed to process query results');
  }
}

function queryTabsOnFailure(error) {
  console.error('Query tabs failed:', error);
  sendResponse([]);
}

function queryTabs(query_info) {
  try {
    let query = atob(query_info)
    query = JSON.parse(query)

    integerKeys = {'windowId': null, 'index': null};
    booleanKeys = {'active': null, 'pinned': null, 'audible': null, 'muted': null, 'highlighted': null,
      'discarded': null, 'autoDiscardable': null, 'currentWindow': null, 'lastFocusedWindow': null, 'windowFocused': null};

    query = Object.entries(query).reduce((o, [k,v]) => {
      if (booleanKeys.hasOwnProperty(k) && typeof v != 'boolean') {
        if (v.toLowerCase() == 'true')
          o[k] = true;
        else if (v.toLowerCase() == 'false')
          o[k] = false;
        else
          o[k] = v;
      }
      else if (integerKeys.hasOwnProperty(k) && typeof v != 'number')
        o[k] = Number(v);
      else
        o[k] = v;
      return o;
    }, {})

    browserTabs.query(query, queryTabsOnSuccess);
  }
  catch(error) {
    queryTabsOnFailure(error);
  }
}


function moveTabs(move_triplets) {
  // move_triplets is a tuple of (tab_id, window_id, new_index)
  if (move_triplets.length == 0) {
    // this post is only required to make bt move command synchronous. mediator
    // is waiting for any reply
    sendResponse('OK');
    return
  }

  // we request a move of a single tab and when it happens, we call ourselves
  // again with the remaining tabs (first omitted)
  const [tabId, windowId, index] = move_triplets[0];
  browserTabs.move(tabId, {index: index, windowId: windowId},
    (tab) => moveTabs(move_triplets.slice(1))
  );
}

function closeTabs(tab_ids) {
  console.log(`[TabCtl] closeTabs called with tab_ids:`, tab_ids);
  try {
    if (!tab_ids || !Array.isArray(tab_ids)) {
      console.error('[TabCtl] closeTabs: tab_ids is undefined or not an array:', tab_ids);
      sendError('Invalid tab_ids parameter');
      return;
    }

    // Parse full tab IDs to extract just the numeric tab ID
    const numericIds = tab_ids.map(id => {
      const parsed = parseTabId(id);
      console.log(`[TabCtl] Parsed tab ID for closing: '${id}' -> ${parsed}`);
      return parsed;
    });

    console.log(`[TabCtl] Closing tabs with numeric IDs:`, numericIds);
    browserTabs.close(numericIds, () => {
      console.log(`[TabCtl] Tabs closed successfully`);
      sendResponse('OK');
    });
  } catch (error) {
    console.error('[TabCtl] Error closing tabs:', error);
    console.error('[TabCtl] Stack:', error.stack);
    sendError('Failed to close tabs: ' + error.message);
  }
}

function openUrls(urls, window_id, first_result="") {
  try {
    if (urls.length == 0) {
      sendResponse([]);
      return;
    }

  if (window_id === 0) {
    browserTabs.create({'url': urls[0], windowId: 0}, (window) => {
      result = `f.${window.id}.${window.tabs[0].id}`;
      console.log(`Opened first window: ${result}`);
      urls = urls.slice(1);
      openUrls(urls, window.id, result);
    });
    return;
  }

  var promises = [];
  for (let url of urls) {
    console.log(`Opening another one url ${url}`);
    promises.push(new Promise((resolve, reject) => {
      browserTabs.create({'url': url, windowId: window_id},
        (tab) => resolve(`f.${tab.windowId}.${tab.id}`)
      );
    }))
  };
  Promise.all(promises).then(result => {
    if (first_result !== "") {
      result.unshift(first_result);
    }
    const data = Array.prototype.concat(...result)
    sendResponse(data);
  });
  } catch (error) {
    console.error('Error opening URLs:', error);
    sendError('Failed to open URLs');
  }
}

function createTab(url) {
  try {
    browserTabs.create({'url': url},
      (tab) => {
        sendResponse([`f.${tab.windowId}.${tab.id}`]);
    });
  } catch (error) {
    console.error('Error creating tab:', error);
    sendError('Failed to create tab');
  }
}

function updateTabs(updates) {
  if (updates.length == 0) {
    sendResponse([]);
    return;
  }

  var promises = [];
  for (let update of updates) {
    console.log(`Updating tab ${JSON.stringify(update)}`);
    promises.push(new Promise((resolve, reject) => {
      browserTabs.update(update.tab_id, update.properties,
        (tab) => { resolve(`f.${tab.windowId}.${tab.id}`) },
        (error) => {
          console.error(`Could not update tab: ${error}, update=${JSON.stringify(update)}`)
          resolve()
        }
      );
    }))
  };
  Promise.all(promises).then(result => {
    const data = Array.prototype.concat(...result).filter(x => !!x)
    sendResponse(data);
  });
}

function activateTab(tab_id, focused) {
  console.log(`[TabCtl] activateTab called with tab_id: '${tab_id}', focused: ${focused}`);
  try {
    // Convert string tab ID to integer for Firefox API
    const tabIdInt = parseTabId(tab_id);
    console.log(`[TabCtl] Parsed tab ID: '${tab_id}' -> ${tabIdInt}`);

    if (isNaN(tabIdInt)) {
      console.error(`[TabCtl] Invalid tab ID after parsing: ${tabIdInt}`);
      sendError(`Invalid tab ID: ${tab_id}`);
      return;
    }

    browserTabs.activate(tabIdInt, focused);
    console.log(`[TabCtl] Tab activation initiated for ID ${tabIdInt}`);
    sendResponse('OK');
  } catch (error) {
    console.error('[TabCtl] Error activating tab:', error);
    console.error('[TabCtl] Stack:', error.stack);
    sendError('Failed to activate tab: ' + error.message);
  }
}

function getActiveTabs() {
  try {
    browserTabs.getActive(tabs => {
        var result = tabs.map(tab => `f.${tab.windowId}.${tab.id}`).toString()
        sendResponse(result);
    });
  } catch (error) {
    console.error('Error getting active tabs:', error);
    sendError('Failed to get active tabs');
  }
}

function getActiveScreenshot() {
  try {
    browserTabs.getActiveScreenshot(data => {
      sendResponse(data);
    });
  } catch (error) {
    console.error('Error getting screenshot:', error);
    sendError('Failed to get screenshot');
  }
}

function getWordsScript(match_regex, join_with) {
  return GET_WORDS_SCRIPT
    .replace('#match_regex#', match_regex)
    .replace('#join_with#', join_with);
}

function getTextScript(delimiter_regex, replace_with) {
  return GET_TEXT_SCRIPT
    .replace('#delimiter_regex#', delimiter_regex)
    .replace('#replace_with#', replace_with);
}

function getHtmlScript(delimiter_regex, replace_with) {
  return GET_HTML_SCRIPT
    .replace('#delimiter_regex#', delimiter_regex)
    .replace('#replace_with#', replace_with);
}

function listOr(list, default_value) {
  if ((list.length == 1) && (list[0] == null)) {
    return default_value;
  }
  return list;
}

function getWordsFromTabs(tabs, match_regex, join_with) {
  var promises = [];
  console.log(`Getting words from tabs: ${tabs}`);
  const script = getWordsScript(match_regex, join_with);

  for (let tab of tabs) {
    var promise = new Promise(
      (resolve, reject) => browserTabs.runScript(tab.id, script, null,
        (words, _payload) => {
          words = listOr(words, []);
          console.log(`Got ${words.length} words from another tab`);
          resolve(words);
        },
        (error, _payload) => {
          console.log(`Could not get words from tab: ${error}`);
          resolve([]);
        }
      )
    );
    promises.push(promise);
  }
  Promise.all(promises).then(
    (all_words) => {
      const result = Array.prototype.concat(...all_words);
      console.log(`Total number of words: ${result.length}`);
      sendResponse(result);
    }
  )
}

function getWords(tab_id, match_regex, join_with) {
  if (tab_id == null) {
    console.log(`Getting words for active tabs`);
    browserTabs.getActive(
      (tabs) => getWordsFromTabs(tabs, match_regex, join_with),
    );
  } else {
    const script = getWordsScript(match_regex, join_with);
    console.log(`Getting words, running a script: ${script}`);
    browserTabs.runScript(tab_id, script, null,
      (words, _payload) => sendResponse(listOr(words, [])),
      (error, _payload) => console.log(`getWords: tab_id=${tab_id}, could not run script (${script})`),
    );
  }
}

function getTextOrHtmlFromTabs(tabs, scriptGetter, delimiter_regex, replace_with, onSuccess) {
  var promises = [];
  const script = scriptGetter(delimiter_regex, replace_with)
  console.log(`Getting text from tabs: ${tabs.length}, script (${script})`);

  lines = [];
  for (let tab of tabs) {
    // console.log(`Processing tab ${tab.id}`);
    var promise = new Promise(
      (resolve, reject) => browserTabs.runScript(tab.id, script, tab,
        (text, current_tab) => {
          // let as_text = JSON.stringify(text);
          // I don't know why, but an array of one item is sent here, so I take
          // the first item.
          if (text && text[0]) {
            console.log(`Got ${text.length} chars of text from another tab: ${current_tab.id}`);
            resolve({tab: current_tab, text: text[0]});
          } else {
            console.log(`Got empty text from another tab: ${current_tab.id}`);
            resolve({tab: current_tab, text: ''});
          }
        },
        (error, current_tab) => {
          console.log(`Could not get text from tab: ${error}: ${current_tab.id}`);
          resolve({tab: current_tab, text: ''});
        }
      )
    );
    promises.push(promise);
  }

  Promise.all(promises).then(onSuccess);
}

function getTextOnRunScriptSuccess(all_results) {
  console.log(`Ready`);
  console.log(`Text promises are ready: ${all_results.length}`);
  // console.log(`All results: ${JSON.stringify(all_results)}`);
  lines = [];
  for (let result of all_results) {
    // console.log(`result: ${result}`);
    tab = result['tab'];
    text = result['text'];
    // console.log(`Result: ${tab.id}, ${text.length}`);
    let line = `f.${tab.windowId}.${tab.id}\t${tab.title}\t${tab.url}\t${text}`;
    lines.push(line);
  }
  // lines = lines.sort(naturalCompare);
  sendResponse(lines);
}

function getTextOnListSuccess(tabs, delimiter_regex, replace_with) {
  // Make sure tabs are sorted by their index within a window
  tabs.sort(compareWindowIdTabId);
  getTextOrHtmlFromTabs(tabs, getTextScript, delimiter_regex, replace_with, getTextOnRunScriptSuccess);
}

function getText(delimiter_regex, replace_with) {
  browserTabs.list({'discarded': false},
      (tabs) => getTextOnListSuccess(tabs, delimiter_regex, replace_with),
  );
}

function getHtmlOnListSuccess(tabs, delimiter_regex, replace_with) {
  // Make sure tabs are sorted by their index within a window
  tabs.sort(compareWindowIdTabId);
  getTextOrHtmlFromTabs(tabs, getHtmlScript, delimiter_regex, replace_with, getTextOnRunScriptSuccess);
}

function getHtml(delimiter_regex, replace_with) {
  browserTabs.list({'discarded': false},
      (tabs) => getHtmlOnListSuccess(tabs, delimiter_regex, replace_with),
  );
}

function getBrowserName() {
  try {
    const name = browserTabs.getBrowserName();
    sendResponse(name);
  } catch (error) {
    console.error('Error getting browser name:', error);
    sendError('Failed to get browser name');
  }
}

function handleMessage(command) {
  if (!command) {
    return;
  }

  // Handle ping response
  if (command.type === 'pong') {
    return;
  }

  if (!command.name) {
    return;
  }

  if (command['name'] == 'list_tabs') {
    // For Firefox, immediately return tab data instead of waiting for CLI request
    browserTabs.list({}, (tabs) => {
      if (!tabs || !Array.isArray(tabs)) {
        sendError('Invalid tabs data received');
        return;
      }
      tabs.sort(compareWindowIdTabId);
      const lines = tabs.map(tab => formatTabToTSV(tab));
      sendResponse(lines);
    });
  }

  else if (command['name'] == 'query_tabs') {
    queryTabs(command['args']['query_info']);
  }

  else if (command['name'] == 'close_tabs') {
    closeTabs(command['args']['tab_ids']);
  }

  else if (command['name'] == 'move_tabs') {
    console.log('Moving tabs:', command['args']['move_triplets']);
    moveTabs(command['args']['move_triplets']);
  }

  else if (command['name'] == 'open_urls') {
    console.log('Opening URLs:', command['args']['urls'], command['args']['window_id']);
    openUrls(command['args']['urls'], command['args']['window_id']);
  }

  else if (command['name'] == 'new_tab') {
    console.log('Creating tab:', command['args']['url']);
    createTab(command['args']['url']);
  }

  else if (command['name'] == 'update_tabs') {
    console.log('Updating tabs:', command['args']['updates']);
    updateTabs(command['args']['updates']);
  }

  else if (command['name'] == 'activate_tab') {
    activateTab(command['args']['tab_id'], !!command['args']['focused']);
  }

  else if (command['name'] == 'get_active_tabs') {
    console.log('Getting active tabs');
    getActiveTabs();
  }

  else if (command['name'] == 'get_screenshot') {
    console.log('Getting visible screenshot');
    getActiveScreenshot();
  }

  else if (command['name'] == 'get_words') {
    console.log('Getting words from tab:', command['tab_id']);
    getWords(command['tab_id'], command['match_regex'], command['join_with']);
  }

  else if (command['name'] == 'get_text') {
    console.log('Getting texts from all tabs');
    getText(command['delimiter_regex'], command['replace_with']);
  }

  else if (command['name'] == 'get_html') {
    console.log('Getting HTML from all tabs');
    getHtml(command['delimiter_regex'], command['replace_with']);
  }

  else if (command['name'] == 'get_browser') {
    getBrowserName();
  } else {
    console.warn(`[TabCtl] Unknown command: ${command['name']}`);
  }
}

function handleDisconnect() {
  console.log("[TabCtl] Native messaging host disconnected");
  if (browser.runtime.lastError) {
    console.error("[TabCtl] Disconnect error:", browser.runtime.lastError);
  }
  port = undefined;
  // Reconnect after a delay
  console.log("[TabCtl] Will attempt reconnection in 5 seconds...");
  reconnectTimer = setTimeout(reconnect, 5000);
}


/*
On a click on the browser action, send the app a message.
*/
// browser.browserAction.onClicked.addListener(() => {
//   // console.log("Sending:  ping");
//   // port.postMessage("ping");
//
//   console.log('Listing tabs');
//   listTabs();
// });