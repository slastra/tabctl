# TabCtl Architecture

## Overview

TabCtl uses a D-Bus-based architecture to enable command-line control of browser tabs. The system consists of three main components that communicate through well-defined interfaces.

## Component Architecture

```
┌─────────────┐     Native      ┌──────────────┐      D-Bus       ┌────────┐
│   Browser   │    Messaging     │   Mediator   │     Session      │ TabCtl │
│  Extension  │ <────stdio────>  │   Process    │ <─────Bus────>  │   CLI  │
└─────────────┘                  └──────────────┘                  └────────┘
```

## Components

### 1. Browser Extension (`extensions/`)

**Purpose:** Interface with browser's tab API and communicate with mediator.

**Key Files:**
- `background.js` - Main extension logic
- `manifest.json` - Extension configuration

**Responsibilities:**
- Listen for native messaging connections
- Execute tab operations (list, activate, close)
- Format responses in TSV format
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
- Register on D-Bus with browser-specific name
- Translate between native messaging and D-Bus protocols
- Handle browser lifecycle (exit when browser closes)
- Log errors to `/tmp/tabctl-mediator.log`

**Communication:**
- **Stdin/Stdout:** Native messaging with browser extension
- **D-Bus:** Service at `dev.slastra.TabCtl.<Browser>`

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

1. **User:** `tabctl list`
2. **CLI:** Discovers browsers on D-Bus
3. **CLI → D-Bus:** Calls `ListTabs()` on each browser service
4. **Mediator:** Receives D-Bus call
5. **Mediator → Extension:** Sends `{"name": "list_tabs"}` via stdout
6. **Extension:** Queries browser tabs API
7. **Extension → Mediator:** Returns TSV-formatted tab data
8. **Mediator → D-Bus:** Returns structured tab array
9. **CLI:** Formats and displays results

### Activate Tab Example

1. **User:** `tabctl activate f.1.2`
2. **CLI:** Parses tab ID, identifies Firefox prefix
3. **CLI → D-Bus:** Calls `ActivateTab("f.1.2")` on Firefox service
4. **Mediator → Extension:** `{"name": "activate_tab", "args": {"tab_id": "2"}}`
5. **Extension:**
   - `browser.tabs.update(2, {active: true})`
   - `browser.windows.update(windowId, {focused: true})`
6. **Window Manager:** Switches desktop if needed (automatic!)
7. **Extension → Mediator:** Returns success
8. **CLI:** Displays "Activated tab f.1.2"

## D-Bus Interface

### Service Names
- `dev.slastra.TabCtl.Firefox`
- `dev.slastra.TabCtl.Brave`
- `dev.slastra.TabCtl.Chrome`

### Object Path
`/dev/slastra/TabCtl/Browser/<BrowserName>`

### Interface
`dev.slastra.TabCtl.Browser`

### Methods

```go
type BrowserHandler interface {
    ListTabs() ([]TabInfo, error)
    ActivateTab(tabID string) (bool, error)
    CloseTab(tabID string) (bool, error)
    OpenTab(url string) (string, error)
}
```

### TabInfo Structure

```go
type TabInfo struct {
    ID     string
    Title  string
    URL    string
    Index  int32
    Active bool
    Pinned bool
}
```

## Tab ID Format

Tab IDs encode browser, window, and tab information:

- **Firefox:** `f.<window_id>.<tab_id>`
  - Example: `f.1.2` (window 1, tab 2)
- **Chrome/Brave:** `c.<window_id>.<tab_id>`
  - Example: `c.1874583011.1874583012`

The prefix allows routing commands to the correct browser.

## Native Messaging Protocol

### Message Format

**Request:**
```json
{
  "name": "command_name",
  "args": {
    "arg1": "value1",
    "arg2": "value2"
  }
}
```

**Response:**
```json
{
  "result": "data" | ["array"],
  "error": "error message if failed"
}
```

### Message Framing

Native messaging uses length-prefixed JSON:
1. 4-byte little-endian integer (message length)
2. JSON message body

## Process Lifecycle

### Mediator Startup
1. Browser launches mediator via native messaging
2. Mediator detects browser from command-line args
3. Registers D-Bus service with browser-specific name
4. Logs startup to `/tmp/tabctl-mediator.log` (debug mode only)

### Mediator Shutdown
1. Browser closes → stdin EOF
2. Mediator detects EOF in polling loop
3. Unregisters from D-Bus
4. Process exits cleanly

### Multi-Browser Support
- Each browser gets its own mediator process
- Mediators run independently
- CLI discovers all via D-Bus name listing
- Commands can target specific browser or all

## Directory Structure

```
tabctl/
├── cmd/
│   ├── tabctl/           # CLI entry point
│   └── tabctl-mediator/   # Mediator entry point
├── internal/
│   ├── cli/              # Command implementations
│   ├── client/           # D-Bus client & browser manager
│   ├── dbus/             # D-Bus primitives
│   ├── mediator/         # Mediator core logic
│   ├── platform/         # OS-specific code
│   └── utils/            # Shared utilities
├── pkg/
│   ├── api/              # Public interfaces
│   └── types/            # Shared types
├── extensions/
│   ├── firefox/          # Firefox extension
│   └── chrome/           # Chrome/Brave extension
└── scripts/              # Rofi integration scripts
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

## Window Focus Magic

The `activate` command achieves cross-desktop switching through browser APIs:

```javascript
// Extension calls browser's window.update API
browser.windows.update(windowId, {focused: true})
```

Modern window managers (KDE, GNOME, etc.) respond to this by:
1. Switching to the virtual desktop containing the window
2. Raising the window to the top
3. Giving it keyboard focus

This works without TabCtl needing to know about window managers!

## Future Enhancements

- WebSocket support for remote control
- Tab search and filtering in mediator
- Batch operations optimization
- Extension UI for status indication
- Persistent mediator mode for faster operations