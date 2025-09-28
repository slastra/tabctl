# TabCtl Project Context

## Overview
TabCtl is a standalone Go implementation of a browser tab controller, inspired by BroTab but completely independent. It allows controlling browser tabs from the command line with excellent rofi integration.

## Current Status
- ✅ All core commands working (list, close, activate, open, query, etc.)
- ✅ Browser extensions working for Firefox and Chrome/Brave
- ✅ Automatic mediator cleanup when browser closes (EOF detection)
- ✅ Browser-specific prefixes (f. for Firefox, c. for Chrome/Brave)
- ✅ Socket conflict detection prevents duplicate mediators
- ✅ Native messaging registration via `tabctl install` command
- ✅ Repository published at https://github.com/slastra/tabctl

## Architecture
```
Browser Extension ← Native Messaging (stdio) → tabctl-mediator ← Unix Socket → tabctl CLI
```

## Key Components
- **tabctl**: CLI binary (uses Cobra framework)
- **tabctl-mediator**: Native messaging host with Unix socket server
- **extensions/**: Browser extensions for Firefox and Chrome/Brave

## Working Commands
- `tabctl list` - List all tabs
- `tabctl close <tab_ids>` - Close tabs
- `tabctl activate <tab_id>` - Activate tab (changes window title)
- `tabctl query` - Filter tabs with conditions
- `tabctl open` - Open URLs from stdin
- `tabctl active` - Show active tabs
- `tabctl windows` - List browser windows

## Recent Improvements
- Mediator auto-exits when browser closes (EOF detection on stdin)
- Socket conflict detection prevents multiple mediators on same port
- Firefox extension handles reconnection gracefully
- Browser-specific tab ID prefixes (f. for Firefox, c. for Chrome/Brave)
- Fixed activate command by correcting prefix handling
- Removed unnecessary timeout delays in mediator shutdown
- Simplified architecture: removed wmctrl dependency from core tabctl

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