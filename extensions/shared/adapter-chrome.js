// Chrome/Brave/Chromium adapter (Manifest V3, service worker).
// Defines the adapter contract consumed by core.js: TAB_PREFIX, runtimeApi,
// browserTabs. MV3 chrome.* APIs return promises when the callback is
// omitted; API errors surface as rejections (no chrome.runtime.lastError
// handling needed).
// Adapters must not call core functions at top level (const TDZ under
// concatenation) — listeners only; core.js calls connect() at the end of
// the concatenated script.

const TAB_PREFIX = 'c.';
const runtimeApi = chrome.runtime;
const KEEPALIVE_ALARM = 'tabctl-keepalive';

const browserTabs = {
  list: (queryInfo) => chrome.tabs.query(queryInfo),
  close: (tabIds) => chrome.tabs.remove(tabIds),
  create: (opts) => (opts.windowId === 0)
    ? chrome.windows.create({ url: opts.url })
    : chrome.tabs.create(opts),
  activate: (tabId, focused) =>
    chrome.tabs.update(tabId, { active: true })
      .then(() => chrome.tabs.get(tabId)) // rejects on stale tab id
      .then(tab => chrome.windows.update(tab.windowId, { focused: focused })),
};

// --- MV3 service worker keepalive / revival ---
// Since Chrome 105 an OPEN native-messaging port keeps the worker alive; the
// failure mode is: port closed (mediator down) -> worker killed ~30s later ->
// pending setTimeout reconnect lost -> extension dead until browser restart.
// The repeating alarm re-wakes the worker and re-establishes the port.
// Chrome clamps periodInMinutes to a 30s minimum, so 0.5 is the floor.
chrome.alarms.create(KEEPALIVE_ALARM, { periodInMinutes: 0.5 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === KEEPALIVE_ALARM) {
    ensureConnected();
  }
});

chrome.runtime.onStartup.addListener(() => ensureConnected());
chrome.runtime.onInstalled.addListener(() => ensureConnected());

// --- core.js follows ---
