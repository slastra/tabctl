# TabCtl

Control browser tabs from the command line using D-Bus IPC.

## Features

- **D-Bus Architecture** - Fast, reliable inter-process communication
- **Multi-Browser Support** - Firefox, Zen, Chrome, Chromium, Brave, Brave Origin, and Helium work simultaneously
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
make build
```

Both binaries land in `build/`. Use `make build` rather than a bare `go build`:
it stamps the version, which `tabctl status` reports, and it keeps every build
in one place. The native messaging manifest records an absolute path to the
mediator, so binaries in two locations means the browser can keep launching an
old one after you rebuild.

2. **Install native messaging host:**
```bash
./build/tabctl install
```

3. **Install the browser extension:**

- Firefox / Zen: <https://addons.mozilla.org/en-US/firefox/addon/tabctl1/>
- Chrome / Chromium / Brave / Brave Origin / Helium: <https://chromewebstore.google.com/detail/tabctl/baomblllgemcgbignhpbipgiofmjdhpn>

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
tabctl activate firefox.1.2          # Firefox tab
tabctl activate brave.1234.5678      # Brave tab
tabctl activate helium.1874583011.1874583012  # Helium tab

# Close tabs
tabctl close firefox.1.2 firefox.1.3
echo "brave.1234.5678" | tabctl close

# Open URLs in new tabs (prints the new tab IDs)
tabctl open https://example.com https://github.com
echo "https://example.com" | tabctl open
tabctl open --browser Firefox https://example.com  # when multiple browsers are connected

# Show mediator/extension versions and protocol compatibility
tabctl status
```

### Tab ID Format

Tab IDs are prefixed with the lowercased browser name so multiple
browsers (e.g. Brave + Helium) stay distinguishable:

- `<browser>.<window_id>.<tab_id>`
- Examples: `firefox.1.2`, `helium.1874583011.1874583012`,
  `brave.999.42`, `chrome.123.45`

Tab IDs are ephemeral. `list` prints them and `activate` or `close` consume
them. Do not store them.

### Multiple Browser Profiles

Each browser profile with the extension enabled connects independently, so all
of their tabs appear together. The first profile to connect uses the plain
browser name. Each additional profile takes a numbered suffix.

```bash
$ tabctl list
chrome.1.234    Inbox (Work Gmail)
chrome2.5.678   Reddit

tabctl list --browser chrome     # every Chrome profile
tabctl list --browser chrome2    # just the second one
```

The suffix follows connection order, so it can change when you restart a
browser. Do not hardcode a profile suffix in a script. List the tabs and act
on the result instead.

### Output Formats

```bash
# JSON output
tabctl list --format json

# Simple format (titles only; cannot be mapped back to tab IDs)
tabctl list --format simple

# Custom delimiter
tabctl list --delimiter ","
```

### Favicons

`--format json` includes `favIconUrl`. This is the icon the browser already
resolved for the tab. A consumer therefore does not need to guess one from the
domain or call a third-party icon service.

```json
{
  "id": "firefox.1.2",
  "title": "GitHub",
  "url": "https://github.com",
  "favIconUrl": "https://github.com/favicon.ico"
}
```

The value is an empty string when the tab has no icon, on browser-internal
pages, and when the extension is older than 2.2.0. It is usually an `http(s)`
URL. A page that declares an inline favicon reports a `data:` URI instead, so
handle both. The `tsv` and `simple` formats are positional and unchanged.

## Rofi Integration

Quick tab switching with rofi (includes desktop switching):

```bash
# From a source checkout
./scripts/rofi-tabctl-wmctrl.sh     # X11 (wmctrl)
./scripts/rofi-tabctl-hyprland.sh   # Hyprland (Wayland), with favicons

# AUR installs ship the scripts here
/usr/share/tabctl/scripts/rofi-tabctl-wmctrl.sh
/usr/share/tabctl/scripts/rofi-tabctl-hyprland.sh
```

Bind one to a key for instant access. Both scripts use your default rofi
theme, so style them in your normal rofi config.

The Hyprland script also shows a favicon per tab. It uses each tab's own
`favIconUrl` and never calls a third-party icon service, so it only contacts
sites you already have open. It needs `jq`, `curl` and `imagemagick`, and
caches chips in `$XDG_CACHE_HOME/tabctl/favicons`.

In that script, **Enter** switches to the highlighted tab. **Ctrl+w** closes
it and reopens the menu, so you can clear several tabs in a row.

## Related Projects

Tools built on tabctl:

- [tabstrip](https://github.com/slastra/tabstrip). A waybar tab strip for
  Hyprland. It shows the tabs of every browser window on the current
  workspace as clickable chips with favicons. It reads tabctl over D-Bus and
  updates live on the `TabsUpdated` signal.
- [vicinae-tabctl](https://github.com/brpaz/vicinae-tabctl). A Vicinae
  launcher extension for switching tabs through tabctl.

## Architecture

```
Browser Extension ← Native Messaging → tabctl-mediator ← D-Bus → tabctl CLI
```

### Components

- **tabctl** - Command-line interface
- **tabctl-mediator** - Native messaging host with D-Bus server
- **Browser Extensions** - Firefox (Manifest V2) and Chrome (Manifest V3) extensions
- **D-Bus Services** - `dev.slastra.TabCtl.Firefox`, `dev.slastra.TabCtl.Brave`

The native-messaging protocol is a JSON-RPC 2.0 subset with a version
handshake. Because the extension updates through the browser stores and the
`tabctl` binaries update through the AUR, the two can briefly be out of
sync after a release. **Update both to matching versions together.**

Both sides surface a mismatch so it never fails silently:
- If the **mediator** is newer, `tabctl status` reports it and any tab
  command fails with a clear "update" error.
- If the **extension** is newer (it handshakes but the mediator is too old
  to answer), the extension shows a red **!** badge on its toolbar icon.
  Hover it for the "update the tabctl package" hint.

## Troubleshooting

### "No browsers found on D-Bus"

The most common failure: the CLI reached the session bus, but no mediator
is registered on it. In order of likelihood:

1. **The extension isn't installed or is disabled.** The mediator is
   launched by the browser through the extension. No extension means no
   mediator. Install it from the store:
   - Firefox / Zen: <https://addons.mozilla.org/en-US/firefox/addon/tabctl1/>
   - Chrome / Chromium / Brave / Brave Origin / Helium: <https://chromewebstore.google.com/detail/tabctl/baomblllgemcgbignhpbipgiofmjdhpn>

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
session bus itself is unreachable. Check `DBUS_SESSION_BUS_ADDRESS`.

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
```

A bare `go build` works too, but it writes wherever you point `-o` and skips the
version stamping, so `tabctl status` reports `dev`. Keep every build in `build/`
if you have a mediator installed: the native messaging manifest holds an
absolute path, so a second copy elsewhere means the browser can keep launching
the old one.

### Test

```bash
go test ./...
```

## License

MIT - See LICENSE file for details

## Acknowledgments

Inspired by [BroTab](https://github.com/balta2ar/brotab), rewritten in Go with a D-Bus architecture and a Manifest v3 Chrome extension.
