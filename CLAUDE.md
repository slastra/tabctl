# TabCtl Project Context

## Overview
TabCtl is a standalone Go implementation of a browser tab controller, inspired by BroTab but completely independent. It allows controlling browser tabs from the command line with excellent rofi integration.

## Current Status
- Core functionality implemented and working
- Browser extensions updated to use `tabctl_mediator` instead of `brotab_mediator`
- Native messaging registration via `tabctl install` command
- Repository published at https://github.com/slastra/tabctl

## Architecture
```
Browser Extension → Native Messaging → tabctl-mediator → HTTP → tabctl CLI
```

## Key Components
- **tabctl**: CLI binary (uses Cobra framework)
- **tabctl-mediator**: Native messaging host and HTTP server
- **extensions/**: Browser extensions for Firefox and Chrome/Brave

## Working Commands
- `tabctl list` - List all tabs (✓ working)
- `tabctl close <tab_ids>` - Close tabs (✓ working)
- `tabctl activate <tab_id>` - Activate tab (has HTTP 404 issue with brotab mediator)
- `tabctl query` - Filter tabs
- `tabctl open` - Open URLs
- `tabctl active` - Show active tabs
- `tabctl windows` - List windows

## Recent Fixes
- Fixed prefix comparison bug where `ParseTabID` returns "a" but `GetPrefix` returns "a."
- Close command now works correctly
- Native messaging manifests properly configured

## Installation Steps
1. Build: `go build -o tabctl ./cmd/tabctl && go build -o tabctl-mediator ./cmd/tabctl-mediator`
2. Install native messaging: `./tabctl install`
3. Restart Brave (required for native messaging host detection)
4. Load extension: brave://extensions/ → Developer mode → Load unpacked → select `extensions/chrome/`

## Known Issues
- Mediator crashes when run standalone (stdio detection issue)
- Activate command returns HTTP 404 with brotab mediator
- Brave needs restart after `tabctl install` for native messaging to work

## Testing
- Use `./tabctl list --target localhost:4625` to test with brotab mediator
- After loading extension, tabctl should auto-detect port 4627 for Brave

## Rofi Integration
Scripts in `scripts/`:
- `tabctl-rofi-switch.sh` - Main rofi tab switcher with virtual desktop support
- Uses wmctrl for window activation and desktop switching

## Next Steps
- Fix mediator stdio detection for standalone operation
- Debug activate command HTTP request format
- Create GitHub release with binaries
- Submit PKGBUILD to AUR