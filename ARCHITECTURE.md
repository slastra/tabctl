# TabCtl Architecture

## Overview

TabCtl uses a multi-process architecture to bridge command-line operations with browser tab management.

```
┌──────────┐    HTTP    ┌──────────────┐   Native    ┌─────────────┐
│  tabctl  │ ─────────► │   Mediator   │ ◄────────► │   Browser   │
│   CLI    │            │  (HTTP+NM)   │  Messaging  │  Extension  │
└──────────┘            └──────────────┘             └─────────────┘
     ▲                         │                           │
     │                         ▼                           ▼
     │                  ┌──────────────┐            ┌─────────────┐
     └──────────────────│   SQLite     │            │ Browser API │
       Search Results   │   FTS5 DB    │            │   (Tabs)    │
                        └──────────────┘            └─────────────┘
```

## Component Communication

### 1. CLI → Mediator (HTTP)
- **Protocol**: HTTP REST API
- **Ports**: 4625-4627 (one per browser)
- **Why HTTP**: Enables multiple CLI instances, stateless operations
- **Format**: JSON request/response

### 2. Mediator ↔ Browser Extension (Native Messaging)
- **Protocol**: Binary (4-byte length header + JSON payload)
- **Channel**: stdin/stdout
- **Lifecycle**: Browser launches mediator on startup
- **Format**: JSON messages with command/response structure

### 3. Browser Extension → Browser APIs
- **APIs**: chrome.tabs.*, browser.tabs.*
- **Permissions**: tabs, activeTab, nativeMessaging
- **Operations**: List, close, activate, move, query tabs

## Key Design Decisions

### Why HTTP Between CLI and Mediator?

1. **Multiple CLI Instances**: Many terminals can connect to one mediator
2. **Browser Lifecycle**: Mediator lives as long as browser is open
3. **Port Discovery**: Each browser gets unique port for identification
4. **Debugging**: Easy to test with curl/wget
5. **Language Agnostic**: Any language can implement a client

### Why Not Direct Native Messaging?

- **Limitation**: Browser controls mediator lifecycle
- **Limitation**: Can't multiplex stdin/stdout between browser and CLI
- **Limitation**: One process per connection model doesn't scale

## Dual-Mode Mediator Operation

The mediator supports two operational modes:

### Stdio Mode (Browser-Initiated)
```
Browser Extension --[launches]--> tabctl-mediator
                                   (stdio mode)
```
- Launched via native messaging manifest
- Communicates over stdin/stdout
- Lifetime tied to browser

### HTTP Mode (Standalone)
```
User --[launches]--> tabctl-mediator --port 4625
                     (HTTP server mode)
```
- Launched manually or via systemd/launchd
- Provides HTTP API on specified port
- Can serve multiple CLI clients

## Port Assignment Strategy

| Browser  | Default Port | Purpose                    |
|----------|-------------|----------------------------|
| Firefox  | 4625        | Primary browser            |
| Chrome   | 4626        | Secondary browser          |
| Chromium | 4627        | Development browser        |
| Brave    | 4627        | Shares with Chromium       |

## Connection Resilience

### Retry Strategy
- Exponential backoff: 100ms, 200ms, 400ms, 800ms
- Max retries: 3
- Circuit breaker after 3 consecutive failures

### Connection Pooling
- Keep-alive connections per mediator
- Max idle connections: 10
- Connection timeout: 10 seconds

### Health Checks
- Endpoint: GET /
- Frequency: Every 30 seconds when idle
- Automatic rediscovery on failure

## Security Model

### Network Security
- **Binding**: localhost only (127.0.0.1)
- **No external access**: Firewall-safe by design
- **No authentication**: Localhost-only trust model

### Native Messaging Security
- **Manifest-based**: Only allowed extensions can connect
- **Browser validation**: Browser verifies mediator binary
- **Path restrictions**: Absolute paths in manifests

## Data Flow Examples

### List Tabs
```
1. CLI: tabctl list
2. CLI → HTTP GET /list_tabs → Mediator (port 4625,4626,4627)
3. Mediator → Native Message {command: "list_tabs"} → Extension
4. Extension → chrome.tabs.query({}) → Browser
5. Browser → Tab array → Extension
6. Extension → Response → Mediator
7. Mediator → HTTP Response → CLI
8. CLI: Display formatted tab list
```

### Close Tab
```
1. CLI: tabctl close a.1.123
2. CLI → HTTP GET /close_tabs/a.1.123 → Mediator (port 4625)
3. Mediator → Native Message {command: "close_tabs", args: {tabIds: [123]}} → Extension
4. Extension → chrome.tabs.remove(123) → Browser
5. Browser → Success → Extension
6. Extension → Response → Mediator
7. Mediator → HTTP Response "OK" → CLI
```

## Performance Characteristics

### Latency
- HTTP round-trip: ~5-10ms (localhost)
- Native messaging: ~1-2ms
- Browser API: ~10-50ms (depends on operation)
- Total operation: ~20-70ms

### Throughput
- List tabs: ~1000 tabs in 200ms
- Close tabs: ~100 tabs/second
- Text extraction: ~50 tabs/second

### Scalability
- Concurrent CLI operations: Unlimited (HTTP-based)
- Max tabs handled: ~5000 per browser
- Memory usage: ~20MB baseline, +1MB per 100 tabs

## Error Handling

### Connection Errors
- Port not available → Try next port
- Mediator not responding → Retry with backoff
- Browser not running → Report clearly to user

### Protocol Errors
- Invalid JSON → Log and return error
- Unknown command → Return error response
- Timeout → Retry once, then fail

### Browser Errors
- Tab doesn't exist → Ignore silently
- Permission denied → Report to user
- Extension not installed → Installation instructions

## Future Enhancements

### Planned
- WebSocket support for real-time updates
- Batch operations for performance
- Tab content caching layer
- Multi-browser synchronization

### Under Consideration
- gRPC instead of HTTP
- Direct browser DevTools protocol integration
- Cloud sync capability
- Plugin architecture for extensibility

## Development Guidelines

### Adding New Commands
1. Define command in `pkg/types/`
2. Add CLI command in `internal/cli/`
3. Add HTTP endpoint in `internal/mediator/server.go`
4. Add native messaging handler in extension
5. Test with curl first, then CLI

### Debugging
1. Enable debug logging: `tabctl --debug`
2. Check mediator logs: `~/.cache/tabctl/mediator.log`
3. Test HTTP directly: `curl http://localhost:4625/list_tabs`
4. Verify extension: Browser developer console

### Testing Strategy
- Unit tests: Each component in isolation
- Integration tests: CLI → Mediator → Mock browser
- End-to-end tests: Full stack with real browser
- Performance tests: Load testing with many tabs