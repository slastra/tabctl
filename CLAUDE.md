# TabCtl Project Context

## Overview
TabCtl is a standalone Go implementation of a browser tab controller that allows controlling browser tabs from the command line. It uses D-Bus for inter-process communication and supports Firefox and Chrome-based browsers.

## Current Status (v1.1.0)
- ✅ **D-Bus Architecture** - Fully migrated from Unix sockets to D-Bus
- ✅ **Core Commands Working** - list, close, activate
- ✅ **Multi-Browser Support** - Firefox and Chrome/Brave work simultaneously
- ✅ **Automatic Window Focus** - Browser APIs handle desktop switching
- ✅ **Clean Logging** - Verbose logs removed, debug mode available
- ✅ **Simplified Codebase** - ~600+ lines removed, cleaner architecture
- ✅ **Production Ready** - Firefox extension packaged as XPI

## Architecture
```
Browser Extension ← Native Messaging → tabctl-mediator ← D-Bus → tabctl CLI
```

### D-Bus Service Names
- `dev.slastra.TabCtl.Firefox` - Firefox mediator
- `dev.slastra.TabCtl.Brave` - Brave/Chrome mediator

## Key Components
- **tabctl**: CLI binary (Cobra framework)
- **tabctl-mediator**: Native messaging host with D-Bus server
- **extensions/**: Browser extensions (Firefox 1.1.0, Chrome)
- **internal/dbus/**: D-Bus client/server implementation
- **internal/client/**: BrowserManager for multi-browser support

## Working Commands
- `tabctl list` - List all tabs from all browsers
- `tabctl list --browser Firefox` - List tabs from specific browser
- `tabctl close <tab_ids>` - Close tabs by ID
- `tabctl activate <tab_id>` - Switch to tab (with desktop switching!)

## Recent Major Changes
- **Phase 5 Completed**: Removed all Unix socket code
- **Refactoring Done**:
  - RemoteAPI → BrowserAPI
  - ParallelClient → BrowserManager
  - Removed unused: cache, editor, search, config cruft
- **Logging Cleaned**: Only errors shown, debug with TABCTL_DEBUG=1
- **Firefox Extension**: v1.1.0 packaged as XPI

## Installation Steps
1. Build: `go build -o tabctl ./cmd/tabctl && go build -o tabctl-mediator ./cmd/tabctl-mediator`
2. Install native messaging: `./tabctl install`
3. Restart Brave (required for native messaging host detection)
4. Load extension: brave://extensions/ → Developer mode → Load unpacked → select `extensions/chrome/`

## Known Issues
- Brave needs restart after `tabctl install` for native messaging to work

## Testing
- Use `./tabctl list --target localhost:4625` to test with brotab mediator
- After loading extension, tabctl should auto-detect port 4627 for Brave

## Rofi Integration
Scripts in `scripts/`:
- `rofi-wmctrl.sh` - Rofi tab switcher for X11 (uses wmctrl)
- `rofi-hyprctl.sh` - Rofi tab switcher for Hyprland (uses hyprctl)
- Both handle workspace/desktop switching automatically

## Next Steps
- Create GitHub release with binaries
- Submit PKGBUILD to AUR
- Consider adding more window manager integrations