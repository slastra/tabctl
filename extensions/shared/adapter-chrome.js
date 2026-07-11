// Chrome/Brave/Chromium adapter (Manifest V3, service worker).
// Defines the adapter contract consumed by core.js: runtimeApi, browserTabs.
// MV3 chrome.* APIs return promises when the callback is omitted; API errors
// surface as rejections. Listeners only at top level — core.js calls connect().

const runtimeApi = chrome.runtime;
const KEEPALIVE_ALARM = 'tabctl-keepalive';

const browserTabs = {
  list: (queryInfo) => chrome.tabs.query(queryInfo),
  close: (tabIds) => chrome.tabs.remove(tabIds),
  create: (opts) => chrome.tabs.create(opts),
  // Only touch window focus when focus was requested (matches Firefox).
  activate: (tabId, focused) =>
    chrome.tabs.update(tabId, { active: true }).then(tab =>
      focused ? chrome.windows.update(tab.windowId, { focused: true }) : tab),
};

// --- MV3 service worker keepalive / revival ---
// Since Chrome 105 an OPEN native-messaging port keeps the worker alive; when
// the port closes (mediator down), the worker is killed ~30s later and the
// pending setTimeout reconnect is lost. The alarm re-wakes it. Chrome clamps
// periodInMinutes to a 30s minimum, so 0.5 is the floor.
chrome.alarms.create(KEEPALIVE_ALARM, { periodInMinutes: 0.5 });
chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === KEEPALIVE_ALARM) { ensureConnected(); }
});

chrome.runtime.onStartup.addListener(() => ensureConnected());
chrome.runtime.onInstalled.addListener(() => ensureConnected());

// --- core.js follows ---
