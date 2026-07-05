# TabCtl

Control browser tabs from the command line using D-Bus IPC.

## Features

- **D-Bus Architecture** - Fast, reliable inter-process communication
- **Multi-Browser Support** - Firefox, Zen, Chrome, Brave work simultaneously
- **Core Commands** - List, close, activate, and open tabs across browsers
- **Desktop Switching** - Automatic window focus across virtual desktops
- **Rofi Integration** - Quick tab switching with rofi scripts
- **Multiple Output Formats** - TSV, JSON, simple
- **Clean Architecture** - Minimal dependencies, production ready

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

3. **Install the browser extension:**

- Firefox / Zen: <https://addons.mozilla.org/en-US/firefox/addon/tabctl1/>
- Chrome / Chromium / Brave / Helium: <https://chromewebstore.google.com/detail/tabctl/baomblllgemcgbignhpbipgiofmjdhpn>

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
tabctl activate firefox.f.1.2          # Firefox tab
tabctl activate brave.c.1234.5678      # Brave tab
tabctl activate helium.c.1874583011.1874583012  # Helium tab

# Close tabs
tabctl close firefox.f.1.2 firefox.f.1.3
echo "brave.c.1234.5678" | tabctl close

# Open URLs in new tabs (prints the new tab IDs)
tabctl open https://example.com https://github.com
echo "https://example.com" | tabctl open
tabctl open --browser Firefox https://example.com  # when multiple browsers are connected
```

### Tab ID Format

Tab IDs are prefixed with the lowercased browser name so multiple
browsers in the same family (e.g. Brave + Helium) stay distinguishable:

- `<browser>.<family>.<window_id>.<tab_id>`
- `<family>` is `f` for Firefox/Zen, `c` for Chromium-family browsers
- Examples: `firefox.f.1.2`, `helium.c.1874583011.1874583012`,
  `brave.c.999.42`, `chrome.c.123.45`

### Output Formats

```bash
# JSON output
tabctl list --format json

# Simple format (titles only — display-only, cannot be mapped back to tab IDs)
tabctl list --format simple

# Custom delimiter
tabctl list --delimiter ","

# No headers
tabctl list --no-headers
```

## Rofi Integration

Quick tab switching with rofi (includes desktop switching):

```bash
# From a source checkout
./scripts/rofi-tabctl-wmctrl.sh   # X11 (wmctrl)
./scripts/rofi-tabctl-niri.sh     # Niri (Wayland)

# AUR installs ship the scripts here
/usr/share/tabctl/scripts/rofi-tabctl-wmctrl.sh
/usr/share/tabctl/scripts/rofi-tabctl-niri.sh
```

Add to your window manager keybindings for instant access. If
`~/.config/rofi/browser-tabs.rasi` exists it is used as the rofi theme;
otherwise the scripts fall back to your default theme.

## Architecture

```
Browser Extension ← Native Messaging → tabctl-mediator ← D-Bus → tabctl CLI
```

### Components

- **tabctl** - Command-line interface
- **tabctl-mediator** - Native messaging host with D-Bus server
- **Browser Extensions** - Firefox (Manifest V2) and Chrome (Manifest V3) extensions
- **D-Bus Services** - `dev.slastra.TabCtl.Firefox`, `dev.slastra.TabCtl.Brave`

## Troubleshooting

### "No browsers found on D-Bus"

The most common failure: the CLI reached the session bus, but no mediator
is registered on it. In order of likelihood:

1. **The extension isn't installed or is disabled.** The mediator is
   launched by the browser through the extension — no extension, no
   mediator. Install it from the store:
   - Firefox / Zen: <https://addons.mozilla.org/en-US/firefox/addon/tabctl1/>
   - Chrome / Chromium / Brave / Helium: <https://chromewebstore.google.com/detail/tabctl/baomblllgemcgbignhpbipgiofmjdhpn>

   Note: loading the extension as a *temporary add-on* (about:debugging)
   only lasts until the browser restarts.
2. **The browser wasn't restarted** after `tabctl install` wrote the
   native messaging manifest.
3. **The mediator crashed.** Check its log (see below).

Diagnostics:
```bash
# Is a mediator registered on the bus?
busctl --user list | grep -i tabctl

# Is the mediator process alive?
pgrep -af tabctl-mediator

# What does the mediator log say?
tail ~/.local/state/tabctl/mediator-*.log
```

A different error, `cannot connect to D-Bus session bus`, means the
session bus itself is unreachable — check `DBUS_SESSION_BUS_ADDRESS`.

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

2. Check logs (one file per browser):
   ```bash
   tail -f ~/.local/state/tabctl/mediator-firefox.log
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

Inspired by [BroTab](https://github.com/balta2ar/brotab), rewritten in Go with a D-Bus architecture and a Manifest v3 Chrome extension.
