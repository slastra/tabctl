/*
On startup, connect to the "tabctl_mediator" app.
*/

const GET_WORDS_SCRIPT = '[...new Set(document.documentElement.innerText.match(#match_regex#))].sort().join(#join_with#);';
const GET_TEXT_SCRIPT = 'document.documentElement.innerText.replace(#delimiter_regex#, #replace_with#);';
const GET_HTML_SCRIPT = 'document.documentElement.innerHTML.replace(#delimiter_regex#, #replace_with#);';

// Global port variable for native messaging
let port = undefined;
let browserTabs = undefined;
const NATIVE_APP_NAME = 'tabctl_mediator';

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

/**
 * Send a standardized success response to the mediator
 */
function sendResponse(data) {
  console.log('Sending response with', Array.isArray(data) ? data.length : 'non-array', 'items');

  // Debug: Log the actual data being sent
  if (Array.isArray(data)) {
    console.log('Sending array with items:', data);
    console.log('JSON.stringify of data:', JSON.stringify(data));
  }

  if (port) {
    const message = {result: data};
    console.log('Full message being sent:', message);
    console.log('Stringified message:', JSON.stringify(message));
    port.postMessage(message);
    console.log('Response sent successfully');
  } else {
    console.error('ERROR: Port is undefined, cannot send response!');
  }
}

/**
 * Send a standardized error response to the mediator
 */
function sendError(message) {
  if (port) {
    port.postMessage({error: message});
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
  // Use actual tab character instead of \t in template literal for service worker
  const TAB = '\t';
  const formatted = `c.${tab.windowId}.${tab.id}${TAB}${tab.title}${TAB}${tab.url}${TAB}${tab.index}${TAB}${tab.active}${TAB}${tab.pinned}`;

  // Debug: log the formatted string and check for tabs
  console.log('Formatted tab string:', formatted);
  console.log('Tab character count:', (formatted.match(/\t/g) || []).length, 'expected: 5');
  console.log('String length:', formatted.length);
  console.log('URL ends at:', formatted.indexOf(tab.url) + tab.url.length);
  console.log('Next char after URL:', formatted.charCodeAt(formatted.indexOf(tab.url) + tab.url.length));

  return formatted;
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

// Chrome V3 implementation - using chrome APIs directly since we only support Chrome for V3
class ChromeTabs {
  constructor() {
    this._browser = chrome;
  }

  runtime() {
    return this._browser.runtime;
  }

  list(queryInfo, onSuccess) {
    console.log('ChromeTabs.list called with:', queryInfo);
    this._browser.tabs.query(queryInfo, (tabs) => {
      console.log('tabs.query callback triggered, found', tabs?.length || 0, 'tabs');
      if (tabs && tabs.length > 0) {
        console.log('First tab:', tabs[0]);
      }
      onSuccess(tabs || []);
    });
  }

  activate(tab_id, focused) {
    this._browser.tabs.update(tab_id, {'active': true});
    this._browser.tabs.get(tab_id, function(tab) {
      chrome.windows.update(tab.windowId, {focused: focused});
    });
  }

  query(queryInfo, onSuccess) {
    if (queryInfo.hasOwnProperty('windowFocused')) {
      let keepFocused = queryInfo['windowFocused']
      delete queryInfo.windowFocused;
      this._browser.tabs.query(queryInfo, tabs => {
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
        console.error(`Could not update tab: ${error}, tabId=${tabId}, options=${JSON.stringify(options)}`)
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
    this._browser.tabs.query({active: true}, onSuccess);
  }

  getActiveScreenshot(onSuccess) {
    let queryOptions = { active: true, lastFocusedWindow: true };
    this._browser.tabs.query(queryOptions, (tabs) => {
      let tab = tabs[0];
      let windowId = tab.windowId;
      let tabId = tab.id;
      this._browser.tabs.captureVisibleTab(windowId, { format: 'png' }, function(data) {
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
      // For V3, we need to inject a function, not arbitrary code
      // We'll create a wrapper function that executes the script
      const results = await this._browser.scripting.executeScript({
        target: { tabId: tab_id },
        func: (scriptCode) => {
          try {
            // Execute the script in the page context
            return eval(scriptCode);
          } catch (e) {
            console.error('Script execution error:', e);
            return null;
          }
        },
        args: [script]
      });

      const result = results && results[0] ? results[0].result : null;
      onSuccess(result ? [result] : [], payload);
    } catch (error) {
      console.error(`Could not run script on tab ${tab_id}:`, error);
      onError(error, payload);
    }
  }

  getBrowserName() {
      return "chrome/chromium";
  }
}

console.log("Detecting browser - Chrome V3 Service Worker");
reconnect();

function reconnect() {
  console.log("Connecting to native app");
  port = chrome.runtime.connectNative(NATIVE_APP_NAME);
  console.log("Connected to native app: " + port);
  browserTabs = new ChromeTabs();

  // Set up port listeners
  port.onMessage.addListener(handleMessage);
  port.onDisconnect.addListener(handleDisconnect);
}

// see https://stackoverflow.com/a/15479354/258421

function compareWindowIdTabId(tabA, tabB) {
  if (tabA.windowId != tabB.windowId) {
    return tabA.windowId - tabB.windowId;
  }
  return tabA.index - tabB.index;
}

function listTabsOnSuccess(tabs) {
  console.log('listTabsOnSuccess received', tabs?.length || 0, 'tabs');
  try {
    if (!tabs || !Array.isArray(tabs)) {
      sendError('Invalid tabs data received');
      return;
    }

    // Make sure tabs are sorted by their index within a window
    tabs.sort(compareWindowIdTabId);
    const lines = tabs.map(tab => formatTabToTSV(tab));

    // Debug: Check the lines array
    console.log('Lines array has', lines.length, 'items');
    lines.forEach((line, i) => {
      console.log(`Line ${i}:`, line.substring(0, 100) + '...');
      console.log(`Line ${i} tab count:`, (line.match(/\t/g) || []).length);
    });

    sendResponse(lines);
  } catch (error) {
    console.error('Error in listTabsOnSuccess:', error);
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
  try {
    if (!tab_ids || !Array.isArray(tab_ids)) {
      console.error('closeTabs: tab_ids is undefined or not an array:', tab_ids);
      sendError('Invalid tab_ids parameter');
      return;
    }

    // Parse full tab IDs to extract just the numeric tab ID
    const numericIds = tab_ids.map(id => parseTabId(id));
    browserTabs.close(numericIds, () => sendResponse('OK'));
  } catch (error) {
    console.error('Error closing tabs:', error);
    sendError('Failed to close tabs');
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
      result = `${window.id}.${window.tabs[0].id}`;
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
  } catch (error) {
    console.error('Error opening URLs:', error);
    sendError('Failed to open URLs');
  }
}

function createTab(url) {
  try {
    browserTabs.create({'url': url},
      (tab) => {
        sendResponse([`${tab.windowId}.${tab.id}`]);
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
        (tab) => { resolve(`${tab.windowId}.${tab.id}`) },
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
  try {
    // Convert string tab ID to integer for Chrome API
    const tabIdInt = parseTabId(tab_id);
    browserTabs.activate(tabIdInt, focused);
    sendResponse('OK');
  } catch (error) {
    console.error('Error activating tab:', error);
    sendError('Failed to activate tab');
  }
}

function getActiveTabs() {
  try {
    browserTabs.getActive(tabs => {
        var result = tabs.map(tab => tab.windowId + "." + tab.id).toString()
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
    let line = tab.windowId + "." + tab.id + "\t" + tab.title + "\t" + tab.url + "\t" + text;
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

/*
Listen for messages from the app.
*/
function handleMessage(command) {
  console.log("Received: " + JSON.stringify(command, null, 4));

  if (command['name'] == 'list_tabs') {
    console.log('Listing tabs...');
    listTabs();
  }

  else if (command['name'] == 'query_tabs') {
    console.log('Querying tabs...');
    queryTabs(command['args']['query_info']);
  }

  else if (command['name'] == 'close_tabs') {
    console.log('Closing tabs:', command['args']['tab_ids']);
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
    console.log('Activating tab:', command['args']['tab_id']);
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
    console.log('Getting browser name');
    getBrowserName();
  }
}

function handleDisconnect() {
  console.log("Disconnected");
  if(chrome.runtime.lastError) {
    console.warn("Reason: " + chrome.runtime.lastError.message);
  } else {
    console.warn("lastError is undefined");
  }
  // Try to reconnect after a delay
  console.log("Trying to reconnect in 1 second");
  setTimeout(reconnect, 1000);
}

console.log("Connected to native app " + NATIVE_APP_NAME);