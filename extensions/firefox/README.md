# TabCtl Firefox Extension

Control your Firefox tabs from the command line using the `tabctl` CLI tool.

## Installation

### From XPI File
1. Download `tabctl-firefox-1.1.0.xpi`
2. Open Firefox and navigate to `about:addons`
3. Click the gear icon → "Install Add-on From File..."
4. Select the XPI file

### From Source (Development)
1. Open Firefox and navigate to `about:debugging`
2. Click "This Firefox"
3. Click "Load Temporary Add-on..."
4. Select the `manifest.json` file from this directory

## Setup

1. Install the native messaging host:
   ```bash
   tabctl install
   ```

2. Restart Firefox to detect the native messaging host

3. Verify the extension is working:
   ```bash
   tabctl list --browser Firefox
   ```

## Features

- List all open tabs
- Activate/switch to specific tabs
- Close tabs by ID
- Query tabs by title or URL
- Cross-desktop window switching

## Permissions

- `nativeMessaging` - Required to communicate with the tabctl CLI
- `tabs` - Required to manage browser tabs

## Troubleshooting

### Extension not connecting
1. Check that the extension is enabled in `about:addons`
2. Ensure the native messaging host is installed: `ls ~/.mozilla/native-messaging-hosts/tabctl_mediator.json`
3. Check browser console for errors: Ctrl+Shift+K

### Commands not working
1. Verify mediator is running: `ps aux | grep tabctl-mediator`
2. Check D-Bus registration: `dbus-send --session --print-reply --dest=org.freedesktop.DBus /org/freedesktop/DBus org.freedesktop.DBus.ListNames | grep TabCtl`
3. Enable debug mode: `TABCTL_DEBUG=1 tabctl list --browser Firefox`

## Version History

- 1.1.0 - D-Bus integration, removed console logging
- 1.0.0 - Initial release with Unix socket support

## License

Same as the main TabCtl project - see repository root for details.