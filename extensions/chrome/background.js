const GET_WORDS_SCRIPT = '[...new Set(document.documentElement.innerText.match(#match_regex#))].sort().join(#join_with#);';
const GET_TEXT_SCRIPT = 'document.documentElement.innerText.replace(#delimiter_regex#, #replace_with#);';
const GET_HTML_SCRIPT = 'document.documentElement.innerHTML.replace(#delimiter_regex#, #replace_with#);';

// Global port variable for native messaging
let port = undefined;
let browserTabs = undefined;
const NATIVE_APP_NAME = 'tabctl_mediator';

/**
 * Send a standardized success response to the mediator
 */
function sendResponse(data) {
  // Ensure connection exists before sending
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
 * Extract numeric tab ID from full tab ID format (e.g., "a.1234.5678" -> 5678)
 */
function parseTabId(fullTabId) {
  if (typeof fullTabId === 'string' && fullTabId.includes('.')) {
    const parts = fullTabId.split('.');
    return parseInt(parts[parts.length - 1], 10);
  }
  return parseInt(fullTabId, 10);
}

/**
 * Format a tab object into TSV format with all fields
 */
function formatTabToTSV(tab) {
  const TAB = '\t';
  return `c.${tab.windowId}.${tab.id}${TAB}${tab.title}${TAB}${tab.url}${TAB}${tab.index}${TAB}${tab.active}${TAB}${tab.pinned}`;
}


// Chrome V3 implementation - using chrome APIs directly since we only support Chrome for V3
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

// Initialize browser tabs handler immediately
browserTabs = new ChromeTabs();

// Establish connection when service worker starts
function connect() {
  if (port) {
    return; // Already connected
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

function compareWindowIdTabId(tabA, tabB) {
  if (tabA.windowId != tabB.windowId) {
    return tabA.windowId - tabB.windowId;
  }
  return tabA.index - tabB.index;
}

function listTabsOnSuccess(tabs) {
  try {
    if (!tabs || !Array.isArray(tabs)) {
      sendError('Invalid tabs data received');
      return;
    }

    // Make sure tabs are sorted by their index within a window
    tabs.sort(compareWindowIdTabId);
    const lines = tabs.map(tab => formatTabToTSV(tab));
    sendResponse(lines);
  } catch (error) {
    sendError('Failed to process tabs list');
  }
}

function listTabs() {
  browserTabs.list({}, listTabsOnSuccess);
}

function queryTabsOnSuccess(tabs) {
  try {
    if (!tabs || !Array.isArray(tabs)) {
      sendResponse([]);
      return;
    }

    tabs.sort(compareWindowIdTabId);
    const lines = tabs.map(tab => formatTabToTSV(tab));
    sendResponse(lines);
  } catch (error) {
    sendError('Failed to process query results');
  }
}

function queryTabsOnFailure(error) {
  sendResponse([]);
}

function queryTabs(query_info) {
  try {
    let query = atob(query_info)
    query = JSON.parse(query)

    integerKeys = { 'windowId': null, 'index': null };
    booleanKeys = {
      'active': null, 'pinned': null, 'audible': null, 'muted': null, 'highlighted': null,
      'discarded': null, 'autoDiscardable': null, 'currentWindow': null, 'lastFocusedWindow': null, 'windowFocused': null
    };

    query = Object.entries(query).reduce((o, [k, v]) => {
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
  catch (error) {
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
  browserTabs.move(tabId, { index: index, windowId: windowId },
    (tab) => moveTabs(move_triplets.slice(1))
  );
}

function closeTabs(tab_ids) {
  try {
    if (!tab_ids || !Array.isArray(tab_ids)) {
      sendError('Invalid tab_ids parameter');
      return;
    }

    // Parse full tab IDs to extract just the numeric tab ID
    const numericIds = tab_ids.map(id => parseTabId(id));
    browserTabs.close(numericIds, () => sendResponse('OK'));
  } catch (error) {
    sendError('Failed to close tabs');
  }
}

function openUrls(urls, window_id, first_result = "") {
  if (urls.length == 0) {
    sendResponse([]);
    return;
  }

  if (window_id === 0) {
    browserTabs.create({ 'url': urls[0], windowId: 0 }, (window) => {
      result = `${window.id}.${window.tabs[0].id}`;
      urls = urls.slice(1);
      openUrls(urls, window.id, result);
    });
    return;
  }

  var promises = [];
  for (let url of urls) {
    promises.push(new Promise((resolve, reject) => {
      browserTabs.create({ 'url': url, windowId: window_id },
        (tab) => resolve(`${tab.windowId}.${tab.id}`)
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
}

function createTab(url) {
  browserTabs.create({ 'url': url },
    (tab) => {
      sendResponse([`${tab.windowId}.${tab.id}`]);
    });
}

function updateTabs(updates) {
  if (updates.length == 0) {
    sendResponse([]);
    return;
  }

  var promises = [];
  for (let update of updates) {
    promises.push(new Promise((resolve, reject) => {
      browserTabs.update(update.tab_id, update.properties,
        (tab) => { resolve(`${tab.windowId}.${tab.id}`) },
        (error) => {
          // Could not update tab
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
  // Convert string tab ID to integer for Chrome API
  const tabIdInt = parseTabId(tab_id);
  browserTabs.activate(tabIdInt, focused);
  sendResponse('OK');
}

function getActiveTabs() {
  browserTabs.getActive(tabs => {
    var result = tabs.map(tab => tab.windowId + "." + tab.id).toString()
    sendResponse(result);
  });
}

function getActiveScreenshot() {
  browserTabs.getActiveScreenshot(data => {
    sendResponse(data);
  });
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
  const script = getWordsScript(match_regex, join_with);

  for (let tab of tabs) {
    var promise = new Promise(
      (resolve, reject) => browserTabs.runScript(tab.id, script, null,
        (words, _payload) => {
          words = listOr(words, []);
          resolve(words);
        },
        (error, _payload) => resolve([])
      )
    );
    promises.push(promise);
  }
  Promise.all(promises).then(
    (all_words) => {
      const result = Array.prototype.concat(...all_words);
      sendResponse(result);
    }
  )
}

function getWords(tab_id, match_regex, join_with) {
  if (tab_id == null) {
    browserTabs.getActive(
      (tabs) => getWordsFromTabs(tabs, match_regex, join_with),
    );
  } else {
    const script = getWordsScript(match_regex, join_with);
    browserTabs.runScript(tab_id, script, null,
      (words, _payload) => sendResponse(listOr(words, []))
    );
  }
}

function getTextOrHtmlFromTabs(tabs, scriptGetter, delimiter_regex, replace_with, onSuccess) {
  var promises = [];
  const script = scriptGetter(delimiter_regex, replace_with)

  lines = [];
  for (let tab of tabs) {
    var promise = new Promise(
      (resolve, reject) => browserTabs.runScript(tab.id, script, tab,
        (text, current_tab) => {
          // Array of one item is sent here, so take the first item
          if (text && text[0]) {
            resolve({ tab: current_tab, text: text[0] });
          } else {
            resolve({ tab: current_tab, text: '' });
          }
        },
        (error, current_tab) => resolve({ tab: current_tab, text: '' })
      )
    );
    promises.push(promise);
  }

  Promise.all(promises).then(onSuccess);
}

function getTextOnRunScriptSuccess(all_results) {
  lines = [];
  for (let result of all_results) {
    tab = result['tab'];
    text = result['text'];
    let line = tab.windowId + "." + tab.id + "\t" + tab.title + "\t" + tab.url + "\t" + text;
    lines.push(line);
  }
  sendResponse(lines);
}

function getTextOnListSuccess(tabs, delimiter_regex, replace_with) {
  // Make sure tabs are sorted by their index within a window
  tabs.sort(compareWindowIdTabId);
  getTextOrHtmlFromTabs(tabs, getTextScript, delimiter_regex, replace_with, getTextOnRunScriptSuccess);
}

function getText(delimiter_regex, replace_with) {
  browserTabs.list({ 'discarded': false },
    (tabs) => getTextOnListSuccess(tabs, delimiter_regex, replace_with),
  );
}

function getHtmlOnListSuccess(tabs, delimiter_regex, replace_with) {
  // Make sure tabs are sorted by their index within a window
  tabs.sort(compareWindowIdTabId);
  getTextOrHtmlFromTabs(tabs, getHtmlScript, delimiter_regex, replace_with, getTextOnRunScriptSuccess);
}

function getHtml(delimiter_regex, replace_with) {
  browserTabs.list({ 'discarded': false },
    (tabs) => getHtmlOnListSuccess(tabs, delimiter_regex, replace_with),
  );
}

function getBrowserName() {
  const name = browserTabs.getBrowserName();
  sendResponse(name);
}

function handleMessage(command) {
  // Ensure connection on every message (in case service worker was dormant)
  if (!port) {
    connect();
  }

  if (command['name'] == 'list_tabs') {
    listTabs();
  }
  else if (command['name'] == 'query_tabs') {
    queryTabs(command['args']['query_info']);
  }

  else if (command['name'] == 'close_tabs') {
    closeTabs(command['args']['tab_ids']);
  }

  else if (command['name'] == 'move_tabs') {
    moveTabs(command['args']['move_triplets']);
  }

  else if (command['name'] == 'open_urls') {
    openUrls(command['args']['urls'], command['args']['window_id']);
  }

  else if (command['name'] == 'new_tab') {
    createTab(command['args']['url']);
  }

  else if (command['name'] == 'update_tabs') {
    updateTabs(command['args']['updates']);
  }

  else if (command['name'] == 'activate_tab') {
    activateTab(command['args']['tab_id'], !!command['args']['focused']);
  }

  else if (command['name'] == 'get_active_tabs') {
    getActiveTabs();
  }

  else if (command['name'] == 'get_screenshot') {
    getActiveScreenshot();
  }

  else if (command['name'] == 'get_words') {
    getWords(command['tab_id'], command['match_regex'], command['join_with']);
  }

  else if (command['name'] == 'get_text') {
    getText(command['delimiter_regex'], command['replace_with']);
  }

  else if (command['name'] == 'get_html') {
    getHtml(command['delimiter_regex'], command['replace_with']);
  }

  else if (command['name'] == 'get_browser') {
    getBrowserName();
  }
}

function handleDisconnect() {
  // Clear the port reference
  port = null;

  // Try to reconnect after a short delay
  setTimeout(() => {
    connect();
  }, 1000);
}

