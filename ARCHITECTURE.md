---
layout: page
title: Architecture
---

## Overview

TabCtl uses a D-Bus-based architecture to enable command-line control of browser tabs. The system consists of three main components that communicate through well-defined interfaces.

## Component Architecture

```
Browser Extension ← Native Messaging → tabctl-mediator ← D-Bus → tabctl CLI
```

## Components

### 1. Browser Extension (`extensions/`)

**Purpose:** Interface with browser's tab API and communicate with mediator.

**Key Files:**
- `background.js` - Main extension logic
- `manifest.json` - Extension configuration

**Responsibilities:**
- Listen for native messaging connections (JSON-RPC 2.0 subset)
- Execute tab operations (list, activate, close)
- Return tab data as structured JSON
- Handle browser-specific APIs (Chrome vs Firefox)

**Communication:**
- **Input:** JSON commands via stdin from mediator
- **Output:** JSON responses via stdout to mediator

### 2. Mediator (`cmd/tabctl-mediator/`)

**Purpose:** Bridge between browser extension (native messaging) and CLI (D-Bus).

**Key Files:**
- `main.go` - Entry point, lifecycle management
- `internal/mediator/mediator.go` - Core orchestration
- `internal/mediator/browser_api.go` - Extension communication
- `internal/mediator/browser_handler.go` - D-Bus interface adapter
- `internal/mediator/transport.go` - Native messaging protocol

**Responsibilities:**
- Claim the first free D-Bus instance name for its browser
- Translate between native messaging and D-Bus protocols
- Handle browser lifecycle (exit when browser closes)
- Log errors to `$XDG_STATE_HOME/tabctl/mediator-<browser>.log`
  (default `~/.local/state/tabctl/`). Profiles of one browser share a file,
  so each line is prefixed with the instance that wrote it.

**Communication:**
- **Stdin/Stdout:** Native messaging with browser extension
- **D-Bus:** Service at `dev.slastra.TabCtl.<Instance>`, where instance is the
  browser name for the first profile to connect and a numbered variant for
  each one after it

### 3. CLI (`cmd/tabctl/`)

**Purpose:** User interface for tab control commands.

**Key Files:**
- `main.go` - Entry point
- `internal/cli/*.go` - Command implementations
- `internal/client/browser_manager.go` - Multi-browser orchestration
- `internal/client/dbus_client.go` - D-Bus communication
- `internal/dbus/client.go` - Low-level D-Bus operations

**Responsibilities:**
- Parse command-line arguments
- Discover available browsers via D-Bus
- Route commands to appropriate browser
- Format output (TSV, JSON, simple)

## Data Flow

### List Tabs Example

```
1. User executes: tabctl list
2. CLI discovers D-Bus services (Firefox, Chrome, Chrome2)
3. For each instance:
   - CLI calls ListTabsWithIcons() via D-Bus, falling back to ListTabs()
     against a mediator that predates it
   - Mediator receives D-Bus call
   - Mediator sends {"jsonrpc":"2.0","id":N,"method":"list_tabs"} to extension
   - Extension queries browser tabs API
   - Extension returns structured tab objects, each with favIconUrl
   - Mediator maps to TabInfoWithIcon[] for D-Bus
   - CLI composes IDs and formats output
4. CLI displays combined results to user
```

### Activate Tab Example

```
1. User executes: tabctl activate firefox.1.2
2. CLI parses tab ID: browser=firefox, tab=2
3. CLI sends ActivateTab(2, false) via D-Bus
4. Mediator receives call and forwards to extension:
   {"jsonrpc":"2.0","id":N,"method":"activate_tab","params":{"tab_id":2,"focused":false}}
5. Extension activates tab:
   - tabs.update(2, {active: true})
   - windows.update(windowId, {focused: true})  // only when focused is set
6. Window focus behavior:
   - Firefox: Window manager switches desktop and focuses
   - Chrome/Brave: Focuses only if on current desktop
7. Extension returns success to mediator
8. Mediator returns success via D-Bus
9. CLI reports: "Activated tab firefox.1.2"
```

## D-Bus Interface

### Service Names
- `dev.slastra.TabCtl.Firefox`
- `dev.slastra.TabCtl.Brave`
- `dev.slastra.TabCtl.Chrome`

One mediator process runs per browser *profile*, and the profile is invisible
from the mediator's side: Chrome passes only the extension origin in argv, all
profiles share one browser process, and the native-host manifest directory is
per-user rather than per-profile. So each mediator claims the first free
instance name for its browser instead of insisting on one: `…TabCtl.Chrome`,
then `…TabCtl.Chrome2`, up to `MaxInstances`. `RequestName` is atomic on the
bus, so mediators racing for a slot cannot both win it.

The suffix is a bare digit because it has to be legal in a D-Bus object path
element (letters, digits, underscore) and must not contain the `.` that
delimits a user-facing tab ID.

### Object Path
`/dev/slastra/TabCtl/Browser/<BrowserName>`

### Interface
`dev.slastra.TabCtl.Browser`

### Methods

```go
type BrowserHandler interface {
    ListTabs() ([]TabInfoWithIcon, error)
    ActivateTab(tabID int32, focused bool) error
    CloseTabs(tabIDs []int32) error
    OpenTab(url string) (windowID, tabID int32, err error)
    GetInfo() Info
}
```

Methods return only an error (no redundant success bool). `GetInfo` reports
mediator/extension versions and protocol compatibility for `tabctl status`.

### TabInfo Structure

```go
type TabInfo struct {          // ListTabs -> a(iissibb)
    WindowID int32
    TabID    int32
    Title    string
    URL      string
    Index    int32
    Active   bool
    Pinned   bool
}

type TabInfoWithIcon struct {  // ListTabsWithIcons -> a(iissibbs)
    // ...same fields, plus:
    FavIconURL string
}
```

The D-Bus layer carries raw browser-assigned numeric IDs; the CLI composes
the user-facing token from them.

**`TabInfo`'s field set is frozen.** D-Bus matches struct signatures by arity,
so appending a field to it would break every consumer compiled against
`a(iissibb)` at the call, including out-of-tree ones. New per-tab data goes on
`TabInfoWithIcon` behind a new method instead, and the handler always returns
the richer shape with the server projecting it down for `ListTabs`. The same
rule applies to the next field added: extend `ListTabsWithIcons`'s successor,
don't mutate a published signature.

`ListTabs` and `ListTabsWithIcons` share one round trip to the extension:
`list_tabs` already returns `favIconUrl` for every tab, so the richer method
costs nothing extra and the cheaper one just drops the field.

## Tab ID Format

The user-facing tab ID is `<browser>.<window_id>.<tab_id>`, prefixed with the
lowercased browser name so multiple browsers (e.g. Brave + Helium) can be
addressed unambiguously:

- Examples: `firefox.1.2`, `brave.999.42`, `helium.1874583011.1874583012`

A second profile of the same browser carries its instance suffix into the
token (`chrome2.5.678`), so profiles are addressable separately. `--browser
chrome` matches every Chrome profile; `--browser chrome2` narrows to one.

The CLI's D-Bus client composes this token from the raw `windowId`/`tabId`
fields in `TabInfo`; the mediator and extension speak only the numeric IDs.
IDs are ephemeral (generated by `list`, consumed by `activate`/`close`).

## Native Messaging Protocol

A JSON-RPC 2.0 subset. Message kind is disambiguated by shape: a `method`
field marks a request; a `result`/`error` field marks a response. Requests
and responses correlate by `id`.

### Handshake

On connect the extension announces itself; the mediator replies with its own
version. A protocol-version mismatch is surfaced to the CLI (`tabctl status`
and a clear error on any command) rather than failing silently.

```json
// extension → mediator
{"jsonrpc":"2.0","id":0,"method":"hello",
 "params":{"extensionVersion":"2.0.0","protocolVersion":2}}
// mediator → extension
{"jsonrpc":"2.0","id":0,"result":{"mediatorVersion":"2.0.0","protocolVersion":2}}
```

### Request / Response

```json
// mediator → extension (request)
{"jsonrpc":"2.0","id":42,"method":"activate_tab","params":{"tab_id":123,"focused":false}}
// extension → mediator (response)
{"jsonrpc":"2.0","id":42,"result":null}
// or, on failure
{"jsonrpc":"2.0","id":42,"error":{"code":-32000,"message":"no tab with id 123"}}
```

`list_tabs` returns an array of structured tab objects (`windowId`, `tabId`,
`title`, `url`, `index`, `active`, `pinned`), not TSV.

### Message Framing

Native messaging uses length-prefixed JSON:
1. 4-byte little-endian integer (message length)
2. JSON message body

## Process Lifecycle

### Mediator Startup
1. Browser launches mediator via native messaging
2. Mediator detects browser from command-line args
3. Claims the first free instance name for that browser, so a second profile
   takes the next one rather than failing on the collision
4. Logs startup to `$XDG_STATE_HOME/tabctl/mediator-<browser>.log`

### Mediator Shutdown
1. Browser closes → stdin EOF
2. Transport reader goroutine reports the disconnect on its error channel
3. Unregisters from D-Bus
4. Process exits cleanly

### Multi-Browser Support
- One mediator process per browser *profile*, not per browser. A browser with
  three profiles runs three mediators, each on its own instance name.
- Mediators run independently
- CLI discovers all via D-Bus name listing
- Commands can target one instance, every profile of a browser, or all

## Directory Structure

```
tabctl/
├── cmd/
│   ├── tabctl/              # CLI entry point
│   └── tabctl-mediator/     # Mediator entry point
├── internal/
│   ├── cli/                 # Command implementations
│   ├── client/               # D-Bus client & browser manager
│   ├── config/               # Timeouts & extension IDs
│   ├── dbus/                 # D-Bus primitives
│   ├── errors/               # Error types
│   ├── mediator/             # Mediator core logic
│   └── utils/                # Shared utilities
├── pkg/
│   ├── api/                  # Public interfaces
│   └── types/                # Shared types
├── extensions/
│   ├── firefox/              # Firefox extension (generated background.js)
│   ├── chrome/               # Chrome/Brave extension (generated background.js)
│   └── shared/               # Extension source (core.js + adapters)
└── scripts/
    ├── build-extensions.sh     # Assemble extensions from shared source
    ├── integration-test.py      # End-to-end mediator+CLI test over D-Bus
    ├── rofi-tabctl-wmctrl.sh    # X11 integration
    └── rofi-tabctl-hyprland.sh  # Hyprland (Wayland) integration
```

## Error Handling

### Connection Errors
- D-Bus registration conflicts logged and handled
- Browser disconnection triggers clean shutdown
- Network errors bubble up to CLI with context

### Command Errors
- Invalid tab IDs return error to CLI
- Browser API failures logged in mediator
- Timeout protection on all operations

## Security Considerations

1. **Native Messaging:** Only registered extensions can launch mediator
2. **D-Bus:** Session bus only (user isolation)
3. **No Network:** All communication is local IPC
4. **No Elevated Privileges:** Runs as user process
5. **Minimal Permissions:** Extensions use only necessary browser APIs

## Performance

- **D-Bus:** ~1ms latency for local calls
- **Tab List:** <50ms for 100 tabs
- **Tab Activation:** <100ms including window focus
- **Memory:** ~10MB per mediator process
- **CPU:** Near zero when idle

## Window Focus Behavior

The `activate` command's window focusing behavior varies by browser:

### Firefox
Firefox's `browser.windows.update()` API with `{focused: true}` triggers automatic desktop switching:

```javascript
// Firefox extension calls
browser.windows.update(windowId, {focused: true})
```

Modern window managers (KDE, GNOME, etc.) respond by:
1. Switching to the virtual desktop containing the window
2. Raising the window to the top
3. Giving it keyboard focus

### Chrome/Chromium/Brave
Chrome-based browsers have more limited window focus capabilities:
- `chrome.windows.update()` focuses the window if on current desktop
- Does not trigger automatic desktop/workspace switching
- User must manually switch to the appropriate desktop first

This difference is due to browser API implementations, not TabCtl limitations.

## Future Enhancements

- WebSocket support for remote control
- Tab search and filtering in mediator
- Batch operations optimization
- Persistent mediator mode for faster operations
