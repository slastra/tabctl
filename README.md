# TabCtl

Control browser tabs from the command line using D-Bus IPC.

## Features

- 🚀 **D-Bus Architecture** - Fast, reliable inter-process communication
- 🌐 **Multi-Browser Support** - Firefox, Zen, Chrome, Brave work simultaneously
- 📋 **Core Commands** - List, close, and activate tabs across browsers
- 🖥️ **Desktop Switching** - Automatic window focus across virtual desktops
- 🔧 **Rofi Integration** - Quick tab switching with rofi scripts
- 📊 **Multiple Output Formats** - TSV, JSON, simple
- 🧹 **Clean Architecture** - Minimal dependencies, production ready

## Installation

### Arch Linux (AUR)

```bash
yay -S tabctl
# or
paru -S tabctl
```

After installation, set up native messaging:
```bash
tabctl install
```

### From Source

1. **Build the binaries:**
```bash
git clone https://github.com/slastra/tabctl.git
cd tabctl
go build -o tabctl ./cmd/tabctl
go build -o tabctl-mediator ./cmd/tabctl-mediator
```

2. **Install native messaging host:**
```bash
./tabctl install
```

3. **Install browser extensions:**

**Firefox:**
- Download `tabctl-firefox-1.1.0.xpi` from releases
- Open Firefox → `about:addons`
- Click gear icon → "Install Add-on From File..."
- Select the XPI file

**Chrome/Brave:**
- Open `chrome://extensions/` or `brave://extensions/`
- Enable "Developer mode"
- Click "Load unpacked"
- Select `extensions/chrome/` directory

4. **Restart browser** to activate native messaging

## Usage

### Basic Commands

```bash
# List all tabs from all browsers
tabctl list

# List tabs from specific browser
tabctl list --browser Firefox
tabctl list --browser Brave

# Activate a tab (switches desktop if needed!)
tabctl activate f.1.2        # Firefox tab
tabctl activate c.1234.5678  # Chrome/Brave tab

# Close tabs
tabctl close f.1.2 f.1.3
echo "c.1234.5678" | tabctl close
```

### Tab ID Format

- Firefox: `f.<window_id>.<tab_id>` (e.g., `f.1.2`)
- Chrome/Brave: `c.<window_id>.<tab_id>` (e.g., `c.1874583011.1874583012`)

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

Quick tab switching with rofi (includes desktop switching):

```bash
# For X11 (wmctrl)
./scripts/rofi-wmctrl.sh

# For Hyprland
./scripts/rofi-hyprctl.sh
```

Add to your window manager keybindings for instant access.

## Architecture

```
Browser Extension ← Native Messaging → tabctl-mediator ← D-Bus → tabctl CLI
```

### Components

- **tabctl** - Command-line interface
- **tabctl-mediator** - Native messaging host with D-Bus server
- **Browser Extensions** - Firefox (v1.1.0) and Chrome extensions
- **D-Bus Services** - `dev.slastra.TabCtl.Firefox`, `dev.slastra.TabCtl.Brave`

## Troubleshooting

### Extension Not Connecting

1. Check extension is enabled in browser
2. Verify native messaging host:
   ```bash
   ls ~/.mozilla/native-messaging-hosts/tabctl_mediator.json
   ls ~/.config/*/NativeMessagingHosts/tabctl_mediator.json
   ```
3. Check mediator is running:
   ```bash
   ps aux | grep tabctl-mediator
   ```

### Commands Not Working

1. Check D-Bus registration:
   ```bash
   dbus-send --session --print-reply --dest=org.freedesktop.DBus \
     /org/freedesktop/DBus org.freedesktop.DBus.ListNames | grep TabCtl
   ```

2. Enable debug mode:
   ```bash
   TABCTL_DEBUG=1 tabctl list
   ```

3. Check logs:
   ```bash
   tail -f /tmp/tabctl-mediator.log
   ```

## Building from Source

### Requirements

- Go 1.19+
- D-Bus session bus
- Browser with native messaging support

### Build

```bash
make build
# or
go build -o tabctl ./cmd/tabctl
go build -o tabctl-mediator ./cmd/tabctl-mediator
```

### Test

```bash
go test ./...
```

## License

MIT - See LICENSE file for details

## Acknowledgments

Inspired by [BroTab](https://github.com/balta2ar/brotab), rewritten in Go with D-Bus architecture for better performance and reliability.
