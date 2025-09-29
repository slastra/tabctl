# TabCtl Project Context

## Overview
TabCtl is a standalone Go implementation of a browser tab controller that allows controlling browser tabs from the command line. It uses D-Bus for inter-process communication and supports Firefox and Chrome-based browsers.

## Current Status (v1.1.1)
- ✅ **D-Bus Architecture** - Fully migrated from Unix sockets to D-Bus
- ✅ **Core Commands Working** - list, close, activate, install
- ✅ **Multi-Browser Support** - Firefox and Chrome/Brave work simultaneously
- ✅ **Window Focus** - Firefox has automatic desktop switching, Chrome focuses on current desktop
- ✅ **Clean Logging** - Verbose logs removed, debug mode with TABCTL_DEBUG=1
- ✅ **Simplified Codebase** - ~600+ lines removed, cleaner architecture
- ✅ **Production Ready** - Released on GitHub and AUR
- ✅ **Firefox Extension v1.1.2** - All manifest warnings fixed, packaged as XPI

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
- `tabctl install` - Interactive native messaging setup
- `tabctl list` - List all tabs from all browsers
- `tabctl list --browser Firefox` - List tabs from specific browser
- `tabctl close <tab_ids>` - Close tabs by ID
- `tabctl activate <tab_id>` - Switch to tab (desktop switching on Firefox)

## Recent Changes (v1.1.1)
- **Critical Fix**: Restored missing `tabctl install` command
- **Firefox Extension v1.1.2**: Fixed all manifest warnings
  - Correct data_collection_permissions format
  - Updated strict_min_version to 58.0
- **Documentation**: Jekyll site with Minima theme
- **Rofi Script**: Fixed window focusing logic
- **AUR Package**: Available at https://aur.archlinux.org/packages/tabctl

## Installation

### Arch Linux
```bash
yay -S tabctl
tabctl install
```

### From Source
1. Build: `go build -o tabctl ./cmd/tabctl && go build -o tabctl-mediator ./cmd/tabctl-mediator`
2. Install native messaging: `./tabctl install`
3. Load browser extensions:
   - Firefox: Install `extensions/firefox/tabctl-firefox-1.1.2.xpi`
   - Chrome/Brave: Load unpacked from `extensions/chrome/`
4. Restart browser for native messaging to work

## Known Issues
- Chrome/Brave window focus only works on current desktop (browser API limitation)
- Browser restart required after `tabctl install` for native messaging

## Rofi Integration
Scripts in `scripts/`:
- `rofi-wmctrl.sh` - Rofi tab switcher for X11 (uses wmctrl)
- `rofi-hyprctl.sh` - Rofi tab switcher for Hyprland (uses hyprctl)
- Both handle workspace/desktop switching automatically

## Future Enhancements
- Browser name prefixes (e.g., f.zen.123 instead of f.1.123)
- Chrome extension callback improvements for window focus
- Additional window manager integrations
- WebSocket support for remote control