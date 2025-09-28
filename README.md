# TabCtl

Control browser tabs from the command line.

## Features

- List, close, activate, and open tabs across multiple browsers
- Works with Firefox, Chrome, Chromium, and Brave
- Rofi integration for quick tab switching
- Virtual desktop support with wmctrl
- Multiple output formats (TSV, JSON, simple)
- Automatic mediator cleanup when browser closes

## Installation

### Quick Start

1. Build the binaries:
```bash
git clone https://github.com/slastra/tabctl.git
cd tabctl
go build -o tabctl ./cmd/tabctl
go build -o tabctl-mediator ./cmd/tabctl-mediator
```

2. Install native messaging:
```bash
./tabctl install
```

3. Load browser extension:
   - Open `brave://extensions/` or `chrome://extensions/`
   - Enable "Developer mode"
   - Click "Load unpacked"
   - Select `extensions/chrome/` directory

For Firefox:
   - Open `about:debugging`
   - Click "This Firefox"
   - Click "Load Temporary Add-on"
   - Select `extensions/firefox/manifest.json`

## Commands

### Basic Usage

```bash
# List all tabs
tabctl list

# Activate a tab (Firefox)
tabctl activate f.1.2

# Activate a tab (Chrome/Brave)
tabctl activate c.1874583011.1874583012

# Close tabs
tabctl close f.1.2 f.1.3

# Open URLs in new tabs
echo "https://example.com" | tabctl open

```

### Query and Filter

```bash
# Show active tabs
tabctl active

# Query tabs with filters
tabctl query --active --current-window

# List windows
tabctl windows
```

### Output Formats

```bash
# JSON output
tabctl list --format json

# Simple format (just URLs)
tabctl list --format simple

# Custom delimiter
tabctl list --delimiter ","

# No headers
tabctl list --no-headers
```

## Rofi Integration

Included rofi scripts for quick tab switching:

### For X11 with wmctrl:
```bash
# Make script executable
chmod +x scripts/rofi-wmctrl.sh

# Run with rofi
./scripts/rofi-wmctrl.sh
```

### For Wayland with Hyprland:
```bash
# Make script executable
chmod +x scripts/rofi-hyprctl.sh

# Run with rofi
./scripts/rofi-hyprctl.sh
```

Both scripts will:
- List all open tabs
- Allow fuzzy searching
- Switch to selected tab
- Handle workspace/desktop switching automatically

### Bind to a hotkey

**i3/Sway:**
```
bindsym $mod+Tab exec ~/path/to/tabctl/scripts/rofi-wmctrl.sh
```

**Hyprland:**
```
bind = $mainMod, Tab, exec, ~/path/to/tabctl/scripts/rofi-hyprctl.sh
```

## Architecture

TabCtl uses native messaging to communicate with browser extensions:

```
tabctl CLI → Unix Socket → tabctl-mediator → Native Messaging → Browser Extension
```

The mediator runs on different ports for each browser:
- 4625: Firefox (prefix: f.)
- 4626: Chrome/Chromium (prefix: c.)
- 4627: Brave (prefix: c.)

### Tab ID Format

Tab IDs include a browser prefix and window/tab numbers:
- Firefox: `f.1.2` (f.windowID.tabID) - uses simple sequential numbers
- Chrome/Brave: `c.1874583011.1874583012` - uses large integer IDs

## Configuration

### Environment Variables

- `TABCTL_TARGET`: Default mediator host (default: "localhost:4625")
- `TABCTL_DEBUG`: Enable debug logging
- `TABCTL_PORT`: Override mediator port

## Dependencies

### Required
- Go 1.19+ (for building)
- Browser extension loaded

### Optional
- `rofi` - For interactive tab switching
- `wmctrl` or `xdotool` - For window management in rofi script

## Troubleshooting

### Browser doesn't detect mediator
- Restart browser after running `tabctl install`
- Check if mediator is registered: `ls ~/.mozilla/native-messaging-hosts/`
- Reload extension after browser restart

### Mediator processes accumulating
- Fixed in latest version - mediators now auto-exit when browser closes
- Manual cleanup: `pkill -f tabctl-mediator`


## Building from Source

```bash
# Clone repository
git clone https://github.com/slastra/tabctl.git
cd tabctl

# Build both binaries
go build -o tabctl ./cmd/tabctl
go build -o tabctl-mediator ./cmd/tabctl-mediator

# Run tests
go test ./...

# Install
./tabctl install
```

## License

MIT