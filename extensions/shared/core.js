// Shared core logic for tabctl browser extensions
// Requires adapter to define: TAB_PREFIX, RECONNECT_DELAY, NATIVE_APP_NAME,
//   port, browserTabs, sendResponse, sendError, connect

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

// ============================================================================
// TAB OPERATIONS
// ============================================================================

function listTabsOnSuccess(tabs) {
  try {
    if (!tabs || !Array.isArray(tabs)) {
      sendError('Invalid tabs data received');
      return;
    }

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

function closeTabs(tab_ids) {
  try {
    if (!tab_ids || !Array.isArray(tab_ids)) {
      sendError('Invalid tab_ids parameter');
      return;
    }

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
      let result = makeTabId(window.id, window.tabs[0].id);
      urls = urls.slice(1);
      openUrls(urls, window.id, result);
    });
    return;
  }

  var promises = [];
  for (let url of urls) {
    promises.push(new Promise((resolve, reject) => {
      browserTabs.create({ 'url': url, windowId: window_id },
        (tab) => resolve(makeTabId(tab.windowId, tab.id))
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

function activateTab(tab_id, focused) {
  const tabIdInt = parseTabId(tab_id);

  if (isNaN(tabIdInt)) {
    sendError(`Invalid tab ID: ${tab_id}`);
    return;
  }

  browserTabs.activate(tabIdInt, focused);
  sendResponse('OK');
}

// ============================================================================
// MESSAGE HANDLING
// ============================================================================

function handleMessage(command) {
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

function handleDisconnect() {
  port = null;

  setTimeout(() => {
    connect();
  }, RECONNECT_DELAY);
}
