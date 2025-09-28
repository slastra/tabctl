# TabCtl Architecture

## System Overview

TabCtl provides command-line control of browser tabs through a multi-component architecture that bridges native browser APIs with Unix command-line tools.

```
┌─────────────┐     ┌──────────────┐     ┌────────────────┐     ┌──────────────┐
│   tabctl    │────▶│ Unix Socket  │────▶│ tabctl-mediator│────▶│   Browser    │
│     CLI     │◀────│/tmp/tabctl-* │◀────│    (Native)    │◀────│  Extension   │
└─────────────┘     └──────────────┘     └────────────────┘     └──────────────┘
                                               │                         │
                                               ▼                         ▼
                                        Native Messaging            Browser APIs
                                           Protocol                (tabs, windows)
```

## Component Details

### 1. Browser Extension (`extensions/`)

Browser-specific extensions that provide access to tab APIs:

- **Firefox** (`extensions/firefox/`)
  - Uses WebExtensions API
  - Communicates via native messaging
  - Tab IDs: Simple sequential numbers (e.g., `f.1.2`)

- **Chrome/Brave** (`extensions/chrome/`)
  - Uses Chrome Extensions API
  - Compatible with Chromium-based browsers
  - Tab IDs: Large integers (e.g., `c.1874583011.1874583012`)

**Key responsibilities:**
- Listen for commands from mediator
- Execute browser API calls (list, activate, close tabs)
- Return results to mediator
- Handle browser-specific quirks

### 2. TabCtl Mediator (`cmd/tabctl-mediator/`)

Native messaging host that bridges browser extensions and CLI:

```go
// Mediator lifecycle
Browser launches mediator → stdin/stdout for native messaging
                        → Unix socket server for CLI connections
                        → EOF detection for auto-cleanup
```

**Key features:**
- **Dual communication modes:**
  - Native messaging (stdin/stdout) with browser
  - Unix socket server for CLI connections
- **Port allocation by browser:**
  - 4625: Firefox
  - 4626: Chrome/Chromium
  - 4627: Brave
- **Auto-cleanup:** Exits when browser closes (EOF on stdin)
- **Socket paths:** `/tmp/tabctl-{port}.sock` (or `$XDG_RUNTIME_DIR`)

### 3. TabCtl CLI (`cmd/tabctl/`)

Command-line interface using Cobra framework:

**Core commands:**
```
tabctl list                 # List all tabs
tabctl activate <tab_id>    # Activate a tab
tabctl close <tab_ids...>   # Close tabs
tabctl open                 # Open URLs from stdin
tabctl window-id <tab_id>   # Get system window ID
tabctl query                # Filter tabs
tabctl active              # Show active tabs
tabctl windows             # List browser windows
```

**Client architecture:**
- `ProcessClient`: Connects to Unix socket
- `ParallelClient`: Manages multiple browser connections
- Response caching for performance

### 4. System Window Integration

Bridge between browser windows and window manager:

```
Browser Tab → Tab Title → wmctrl -l → System Window ID
                ↓
          Activate Tab
                ↓
          Focus Window
```

**Window ID detection strategy:**
1. Activate the target tab
2. Get tab title from browser
3. Use `wmctrl -l` to list windows
4. Match by title (exact or partial)
5. Extract system window ID

**Limitations:**
- X11 only (wmctrl/xdotool dependency)
- Wayland requires compositor-specific tools
- Relies on unique window titles

## Communication Protocols

### Native Messaging Protocol

Browser ↔ Mediator communication:

```json
// Request (stdin)
{
  "id": 1,
  "command": "list",
  "args": {}
}

// Response (stdout)
{
  "id": 1,
  "result": [
    {"id": "f.1.1", "title": "Example", "url": "https://example.com"}
  ]
}
```

Message format:
- 4-byte message length (native endianness)
- JSON payload
- Synchronous request/response

### Unix Socket Protocol

CLI ↔ Mediator communication:

```json
// Request
{
  "name": "list_tabs",
  "args": {}
}

// Response
[
  "f.1.1\tExample\thttps://example.com\tfalse"
]
```

Features:
- JSON-encoded commands
- TSV responses for compatibility
- Connection per request
- Automatic socket cleanup

## Tab ID Format

Tab IDs encode browser, window, and tab information:

```
Format: <prefix>.<window_id>.<tab_id>

Firefox:    f.1.2      (window 1, tab 2)
Chrome:     c.1874583011.1874583012
Brave:      c.2094732819.2094732820
```

**Prefix mapping:**
- `f.` - Firefox (port 4625)
- `c.` - Chrome/Chromium/Brave (ports 4626-4627)

## Installation Flow

1. **Build binaries:**
   ```bash
   go build -o tabctl ./cmd/tabctl
   go build -o tabctl-mediator ./cmd/tabctl-mediator
   ```

2. **Register native messaging host:**
   ```bash
   ./tabctl install
   ```

   Creates manifests in:
   - Firefox: `~/.mozilla/native-messaging-hosts/`
   - Chrome: `~/.config/google-chrome/NativeMessagingHosts/`
   - Brave: `~/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts/`

3. **Load browser extension:**
   - Developer mode required
   - Point to `extensions/firefox/` or `extensions/chrome/`

## Process Management

### Mediator Lifecycle

```
Browser starts
    ↓
Extension loaded
    ↓
First native message → Spawn mediator
    ↓
Mediator runs (stdin/stdout connected)
    ↓
Browser closes → EOF on stdin
    ↓
Mediator exits → Socket cleaned up
```

### Socket Management

- **Location:** `/tmp/tabctl-{port}.sock` or `$XDG_RUNTIME_DIR/tabctl-{port}.sock`
- **Permissions:** User-only (0700)
- **Cleanup:** Automatic on mediator exit
- **Conflict detection:** Check existing socket before binding

## Data Flow Examples

### List Tabs
```
1. User: tabctl list
2. CLI: Connect to Unix socket
3. CLI→Mediator: {"name": "list_tabs"}
4. Mediator→Extension: {"command": "list"}
5. Extension: chrome.tabs.query({})
6. Extension→Mediator: [tabs array]
7. Mediator→CLI: TSV formatted tabs
8. CLI: Display to user
```

### Activate Tab with Window ID
```
1. User: tabctl activate --window-id f.1.2
2. CLI→Mediator: {"name": "activate_tab", "args": {"tab_id": "f.1.2"}}
3. Mediator→Extension: Activate tab
4. Extension: browser.tabs.update(tabId, {active: true})
5. CLI: Get tab title from list
6. CLI: Run wmctrl -l
7. CLI: Match window by title
8. CLI: Output window ID
```

## Error Handling

### Connection Failures
- Mediator not running → Start instructions
- Socket permission denied → Check user/permissions
- Browser not responding → Extension not loaded

### Browser-Specific Issues
- Firefox: Requires browser restart after install
- Chrome: Security policy may block native messaging
- Brave: Shields may interfere with extension

## Performance Optimizations

1. **Response Caching:**
   - Cache list results for 10 seconds
   - Invalidate on write operations

2. **Parallel Queries:**
   - Query all browsers simultaneously
   - Merge results for unified view

3. **Unix Socket:**
   - Lower latency than HTTP
   - No network overhead
   - Direct process communication

## Security Considerations

1. **Unix Socket Permissions:**
   - User-only access (0700)
   - No network exposure

2. **Native Messaging:**
   - Browser-enforced manifest validation
   - Limited to registered extensions

3. **Command Validation:**
   - Input sanitization in mediator
   - Tab ID format validation

## Future Enhancements

### Planned Features
- Wayland window management support
- Tab content extraction (text, HTML)
- Tab grouping and workspace management
- Remote browser control
- WebSocket support for persistent connections

### Architecture Improvements
- Plugin system for window managers
- Configurable mediator ports
- Multi-user socket namespacing
- D-Bus integration for Linux desktop

## Debugging

### Enable Debug Logging
```bash
export TABCTL_DEBUG=1
tabctl list
```

### Check Mediator Status
```bash
# See if mediator is running
ps aux | grep tabctl-mediator

# Check socket exists
ls -la /tmp/tabctl-*.sock

# Test socket connection
nc -U /tmp/tabctl-4627.sock
```

### Browser Extension Debugging
- Firefox: `about:debugging` → This Firefox
- Chrome/Brave: `chrome://extensions/` → Developer mode

### Common Issues

**"No mediator found"**
- Browser not running
- Extension not loaded
- Native messaging not registered

**"Connection refused"**
- Mediator crashed
- Socket file exists but process dead
- Wrong port for browser

**"Window ID not found"**
- wmctrl not installed
- Running on Wayland
- Window title changed