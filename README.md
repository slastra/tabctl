# TabCtl

Control browser tabs from the command line.

## Features

- List, close, activate, and open tabs across multiple browsers
- Works with Firefox, Chrome, Chromium, and Brave
- Rofi integration for quick tab switching
- Virtual desktop support with wmctrl
- Multiple output formats (TSV, JSON, simple)
- Single binary with no dependencies

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

# Activate a tab
tabctl activate a.123.456

# Close tabs
tabctl close a.123.456 a.123.457

# Open URLs in new tabs
echo "https://example.com" | tabctl open a.0
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

Use the included rofi script for quick tab switching:

```bash
# Make script executable
chmod +x scripts/tabctl-rofi-switch.sh

# Run with rofi
./scripts/tabctl-rofi-switch.sh
```

This script will:
- List all open tabs
- Allow fuzzy searching
- Switch to selected tab
- Handle virtual desktop switching automatically

### Bind to a hotkey

Add to your window manager config (e.g., i3):
```
bindsym $mod+Tab exec ~/path/to/tabctl/scripts/tabctl-rofi-switch.sh
```

## Architecture

TabCtl uses native messaging to communicate with browser extensions:

```
tabctl CLI → HTTP → tabctl-mediator → Native Messaging → Browser Extension
```

The mediator runs on ports:
- 4625: Firefox
- 4626: Chrome/Chromium
- 4627: Brave

## Configuration

### Environment Variables

- `TABCTL_TARGET`: Default mediator host (default: "localhost:4625")
- `TABCTL_DEBUG`: Enable debug logging
- `TABCTL_PORT`: Override mediator port

## Building from Source

```bash
# Build for current platform
make build

# Run tests
make test

# Format code
make fmt
```

## License

MIT