// Firefox/Zen adapter (Manifest V2, persistent background page).
// Defines the adapter contract consumed by core.js: TAB_PREFIX, runtimeApi,
// browserTabs, onConnected. browser.* APIs are natively promise-based;
// rejections propagate to core's sendError.
// Adapters must not call core functions at top level (const TDZ under
// concatenation) — listeners only; core.js calls connect() at the end of
// the concatenated script.

const TAB_PREFIX = 'f.';
const runtimeApi = browser.runtime;

const browserTabs = {
  list: (queryInfo) => browser.tabs.query(queryInfo),
  close: (tabIds) => browser.tabs.remove(tabIds),
  create: (opts) => (opts.windowId === 0)
    ? browser.windows.create({ url: opts.url })
    : browser.tabs.create(opts),
  // Unlike Chrome, only touch window focus when focused was requested
  activate: (tabId, focused) =>
    browser.tabs.update(tabId, { active: true }).then(tab =>
      focused ? browser.windows.update(tab.windowId, { focused: true }) : tab),
};

// Post-connect ping; the mediator replies {type:"pong"} (ignored in core).
function onConnected(p) {
  setTimeout(() => {
    try {
      if (port === p) {
        p.postMessage({ type: 'ping', timestamp: Date.now() });
      }
    } catch (e) {
      // Port already dead; the disconnect handler owns recovery
    }
  }, 1000);
}

// --- core.js follows ---
