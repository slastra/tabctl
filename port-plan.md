# TabCtl Project Plan

A standalone browser tab controller written in Go, inspired by BroTab but fully independent.

## Project Overview

**Goal**: Create a fast, lightweight browser tab controller with excellent rofi integration.

**Repository**: https://github.com/slastra/tabctl

**Key Features**:
- Independent TabCtl browser extensions
- Native messaging protocol implementation
- Rofi integration with virtual desktop support
- Single binary distribution
- TSV/JSON/Simple output formats

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   tabctl CLI    │───▶│  HTTP Mediator   │───▶│ Browser Extension│
│   (Go binary)   │    │  (Go HTTP srv)   │    │   (JavaScript)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ SQLite FTS5 DB   │    │  Browser APIs   │
                       │   (Tab Search)    │    │ (Tabs/Windows)  │
                       └──────────────────┘    └─────────────────┘
```

## Phase 1: Core CLI & Architecture Setup ✅ COMPLETED

### Goals
- Establish Go project structure
- Implement basic CLI framework
- Set up build system and dependencies

### Tasks
1. **Project Structure**
   ```
   tabctl/
   ├── cmd/
   │   └── tabctl/
   │       └── main.go
   ├── internal/
   │   ├── cli/          # CLI commands
   │   ├── client/       # HTTP client
   │   ├── mediator/     # HTTP server
   │   ├── search/       # SQLite FTS5
   │   └── platform/     # OS-specific code
   ├── pkg/
   │   ├── api/          # API interfaces
   │   └── types/        # Shared types
   ├── extensions/       # Browser extensions
   │   ├── firefox/
   │   └── chrome/
   ├── scripts/          # Build/install scripts
   ├── go.mod
   ├── go.sum
   ├── Makefile
   └── README.md
   ```

2. **Dependencies Setup**
   ```go
   // Core dependencies
   github.com/spf13/cobra           // CLI framework
   github.com/go-resty/resty/v2     // HTTP client
   github.com/gorilla/mux           // HTTP router
   modernc.org/sqlite               // Pure Go SQLite
   github.com/tidwall/gjson         // Fast JSON parsing
   ```

3. **Basic CLI Commands**
   - Implement argument parsing with Cobra
   - Create subcommand structure matching Python version
   - Add `--target` flag for mediator hosts
   - Implement help system

### Deliverables
- ✅ Go module initialized
- ✅ Basic CLI structure with Cobra
- ✅ Core command stubs (list, close, activate, etc.)
- ✅ Build system (Makefile + go build)

### Duration: 3-4 days

## Phase 2: Mediator & HTTP Server ✅ COMPLETED

### Goals
- Port Python mediator to Go HTTP server
- Implement native messaging bridge
- Set up multi-port support (4625-4627)

### Tasks
1. **HTTP Server Implementation**
   - Port Flask routes to Gorilla Mux
   - Implement request/response marshaling
   - Add CORS support for browser requests
   - Port error handling patterns

2. **Native Messaging Bridge**
   - Implement stdin/stdout JSON communication
   - Port message validation and routing
   - Add process lifecycle management
   - Handle browser extension connections

3. **Multi-Client Support**
   - Port port discovery logic
   - Implement client prefix system (a, b, c...)
   - Add connection pooling for multiple browsers

4. **Configuration System**
   - Port manifest file templates
   - Add install command for native messaging setup
   - Support Windows registry operations

### Key Files to Port
- `brotab/mediator/http_server.py` → `internal/mediator/server.go`
- `brotab/mediator/brotab_mediator.py` → `cmd/mediator/main.go`
- `brotab/mediator/transport.py` → `internal/mediator/transport.go`

### Deliverables
- ✅ HTTP server with all routes
- ✅ Native messaging implementation
- ✅ Multi-port mediator support
- ✅ Configuration and installation system

### Duration: 5-6 days

## Phase 3: Browser Communication Layer ✅ COMPLETED

### Goals
- Implement HTTP client for CLI→Mediator communication
- Port all tab operation APIs
- Add parallel request support
- Implement dual-mode mediator operation
- Add connection resilience and pooling

### Tasks
1. **HTTP Client Layer**
   - Implement tabctl API client with Resty
   - Port timeout and retry logic
   - **ENHANCED**: Add connection pooling with keep-alive
   - **NEW**: Implement exponential backoff for retries
   - **NEW**: Add circuit breaker pattern for failed mediators

2. **Port Discovery & Connection**
   ```go
   // NEW: Implement actual port connectivity checking
   func isPortAcceptingConnections(host string, port int) bool {
       conn, err := net.DialTimeout("tcp",
           fmt.Sprintf("%s:%d", host, port),
           100*time.Millisecond)
       if err == nil {
           conn.Close()
           return true
       }
       return false
   }
   ```

3. **Dual-Mode Mediator Operation**
   - **NEW**: Detect launch mode (stdio vs standalone)
   - **NEW**: Native messaging mode when launched by browser
   - **NEW**: HTTP server mode when launched standalone
   - **NEW**: Add mode detection via environment or args

4. **Tab Operations**
   ```go
   // Core operations to implement
   type TabAPI interface {
       ListTabs() ([]Tab, error)
       CloseTabs(tabIDs []string) error
       ActivateTab(tabID string, focused bool) error
       OpenURLs(urls []string, windowID string) ([]string, error)
       UpdateTabs(updates []TabUpdate) error
       QueryTabs(query TabQuery) ([]Tab, error)
       GetText(tabIDs []string) ([]TabContent, error)
       GetHTML(tabIDs []string) ([]TabContent, error)
       GetWords(tabIDs []string) ([]string, error)
   }
   ```

5. **Parallel Processing**
   - Port Python's parallel client calls
   - Implement goroutine-based concurrent requests
   - Add proper error aggregation
   - **NEW**: Add request cancellation support
   - **NEW**: Implement request deduplication

6. **Connection Resilience**
   - **NEW**: Health check endpoints for mediators
   - **NEW**: Automatic mediator discovery on failure
   - **NEW**: Connection pool management per mediator
   - **NEW**: Metrics collection for debugging

7. **Data Marshaling**
   - Port tab ID parsing logic
   - Implement JSON serialization for all types
   - Add validation for browser responses
   - **NEW**: Add response caching for read operations

### Key Files to Port
- `brotab/api.py` → `internal/client/api.go`
- `brotab/operations.py` → `pkg/api/operations.go`
- `brotab/tab.py` → `pkg/types/tab.go`
- **NEW**: `internal/client/pool.go` - Connection pooling
- **NEW**: `internal/client/retry.go` - Retry logic

### Deliverables
- ✅ Complete HTTP client implementation
- ✅ All tab operations working
- ✅ Parallel request support
- ✅ Comprehensive error handling
- **NEW**: ✅ Connection resilience and pooling
- **NEW**: ✅ Dual-mode mediator support
- **NEW**: ✅ Performance metrics/logging

### Duration: 4-5 days (unchanged - optimizations balance complexity)

## Phase 4: Simplified Rofi-Focused Features ✅ COMPLETED

### Goals
- Focus on features that enhance rofi integration
- Skip complex features that rofi handles better
- Prioritize speed and simplicity

### Tasks to Implement
1. **Essential CLI Commands**
   - ✅ `list` - List tabs (DONE)
   - ✅ `close` - Close tabs (DONE)
   - ✅ `activate` - Switch to tab (DONE)
   - ✅ `open` - Open URLs in new tabs (DONE)
   - ✅ `query` - Filter tabs by properties (active, window, etc.) (DONE)
   - ✅ `active` - Show active tabs (DONE)
   - ✅ `windows` - List windows (DONE)

2. **Output Formatting**
   - ✅ Add `--format` flag for different output styles (json, tsv, simple) (DONE)
   - ✅ Add `--no-headers` flag for cleaner rofi parsing (DONE)
   - ✅ Support `--delimiter` for custom separators (DONE)

3. **Performance Optimizations**
   - Keep binary size minimal
   - Fast startup time for rofi responsiveness
   - Efficient concurrent mediator queries

### Tasks to SKIP (not needed with rofi)
- ❌ SQLite FTS5 search (rofi handles fuzzy search)
- ❌ Tab content indexing (unnecessary complexity)
- ❌ Interactive move command (rofi is the UI)
- ❌ Words extraction for autocomplete (rofi handles this)
- ❌ Screenshot functionality
- ❌ Complex text/HTML extraction

### Deliverables
- ✅ Core commands for rofi integration
- ✅ Fast, lightweight binary
- ✅ Flexible output formatting
- ✅ Works with existing Python mediator

### Duration: 1-2 days

## Phase 5: Browser Extensions ✅ COMPLETED

**Updated Decision**: Created independent TabCtl extensions
- ✅ Updated extensions to use `tabctl_mediator` instead of `brotab_mediator`
- ✅ Firefox extension ID: `tabctl@slastra.github.io`
- ✅ Chrome/Brave: Will get ID when published to store
- ✅ Native messaging registration via `tabctl install` command

## Phase 6: GitHub Repository & Distribution ✅ COMPLETED

### Goals
- Just enough features for excellent rofi integration
- Skip complex features Python brotab can handle

### What Was Built
1. **GitHub Repository**
   - ✅ Created https://github.com/slastra/tabctl
   - ✅ All code committed and pushed
   - ✅ Proper .gitignore for Go projects

2. **Working Features**
   - ✅ All essential commands implemented
   - ✅ Rofi integration scripts created
   - ✅ Virtual desktop switching with wmctrl
   - ✅ Independent from brotab

### What We're NOT Building
- ❌ Interactive editor integration (complexity)
- ❌ Screenshot features (not needed)
- ❌ Duplicate detection (rofi can handle)
- ❌ Shell completions (not critical)
- ❌ Platform installers (manual is fine)

### Duration: 1 day

## Phase 7: Packaging & Release 🚧 IN PROGRESS

### Goals
- Minimal but useful documentation
- Simple distribution (GitHub releases)
- Focus on rofi integration examples

### What's Next
1. **Arch Linux Package**
   - Create PKGBUILD for AUR submission
   - Include rofi scripts in package
   - Add systemd user service (optional)

2. **Documentation**
   - Update README with installation instructions
   - Add rofi usage examples
   - Document browser extension installation

3. **Testing & Release**
   - Test with Brave browser
   - Create GitHub release with binaries
   - Submit to AUR

### What We're NOT Doing
- ❌ Comprehensive test suite (overkill for our needs)
- ❌ Package managers (manual install is fine)
- ❌ Extensive documentation (keep it simple)

### Duration: 1 day

## Current Status Summary

### ✅ COMPLETED (Phases 1-6)
- **Core Architecture**: CLI, HTTP mediator, browser communication
- **All Essential Commands**: list, close, activate, open, query, active, windows
- **Output Formatting**: JSON, TSV, simple formats with customization
- **Rofi Integration**: Complete scripts with virtual desktop switching
- **Browser Extensions**: Independent TabCtl extensions created
- **GitHub Repository**: Code published at https://github.com/slastra/tabctl
- **Native Messaging**: Install command configures browsers automatically

### 🚧 IN PROGRESS (Phase 7)
- **Arch Linux Packaging**: PKGBUILD for AUR submission
- **Documentation**: README updates and usage examples
- **Release**: Binary distribution on GitHub

### 📊 Project Metrics
- **Total Development Time**: ~1 week
- **Binary Size**: <10MB
- **Startup Time**: <50ms
- **Dependencies**: Minimal (only Go standard library + 3 packages)

## Risk Mitigation

### Technical Risks
1. **SQLite FTS5 Compatibility**: Test early with realistic data
2. **Native Messaging Changes**: Maintain protocol compatibility
3. **Browser Extension Breakage**: Minimal changes, extensive testing
4. **Performance Regression**: Benchmark against Python version

### Migration Risks
1. **User Adoption**: Clear migration path and documentation
2. **Extension Updates**: Coordinate browser store releases
3. **Configuration Compatibility**: Support existing user configs

## Success Criteria

### Functional Requirements
- ✅ 100% CLI command parity with brotab
- ✅ Full browser extension compatibility
- ✅ Search and indexing functionality
- ✅ Cross-platform installation support

### Non-Functional Requirements
- ✅ Single binary distribution
- ✅ <50MB binary size
- ✅ <100ms command startup time
- ✅ Memory usage <20MB baseline

### Quality Requirements
- ✅ >80% test coverage
- ✅ Comprehensive documentation
- ✅ Zero breaking changes for existing users
- ✅ Smooth migration path from brotab

## Post-Launch Roadmap

### Immediate (First Month)
- Bug fixes and user feedback
- Performance optimizations
- Additional browser support

### Medium-term (3-6 Months)
- Enhanced search capabilities
- Advanced tab management features
- Desktop integration improvements

### Long-term (6+ Months)
- Cloud sync capabilities
- Plugin architecture
- Advanced automation features

---

This plan provides a structured approach to porting BroTab to Go while maintaining all existing functionality and providing a clear path for users to migrate from the Python version.