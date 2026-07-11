// Firefox/Zen adapter (Manifest V2, persistent background page).
// Defines the adapter contract consumed by core.js: runtimeApi, browserTabs.
// browser.* APIs are natively promise-based; rejections propagate to core's
// sendError. Listeners only at top level — core.js calls connect().

const runtimeApi = browser.runtime;
const badgeApi = browser.browserAction;

const browserTabs = {
  list: (queryInfo) => browser.tabs.query(queryInfo),
  close: (tabIds) => browser.tabs.remove(tabIds),
  create: (opts) => browser.tabs.create(opts),
  // Only touch window focus when focus was requested.
  activate: (tabId, focused) =>
    browser.tabs.update(tabId, { active: true }).then(tab =>
      focused ? browser.windows.update(tab.windowId, { focused: true }) : tab),
};

// --- core.js follows ---
