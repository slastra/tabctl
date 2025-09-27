# TabCtl

Control your browser's tabs from the command line.

TabCtl is a Go port of [BroTab](https://github.com/balta2ar/brotab) that provides a fast, single-binary command-line interface for managing browser tabs across Firefox and Chrome-based browsers.

## Features

- **Tab Management**: List, close, activate, and move tabs
- **Cross-Browser**: Works with Firefox, Chrome, Chromium, and Brave
- **Search**: Full-text search through tab content using SQLite FTS5
- **Integration**: Works with fzf, rofi, albert, and other command-line tools
- **Single Binary**: No dependencies, just download and run
- **Fast**: Written in Go for optimal performance

## Installation

### Download Binary

Download the latest release for your platform from the [releases page](https://github.com/tabctl/tabctl/releases).

### Build from Source

```bash
git clone https://github.com/tabctl/tabctl.git
cd tabctl
make build
```

### Install Dependencies

```bash
make deps
```

## Quick Start

1. **Install native messaging components**:
   ```bash
   tabctl install
   ```

2. **Install browser extensions**:
   - Firefox: Install from [Firefox Add-ons](https://addons.mozilla.org/firefox/addon/tabctl/)
   - Chrome: Install from [Chrome Web Store](https://chrome.google.com/webstore/detail/tabctl/...)

3. **List your tabs**:
   ```bash
   tabctl list
   ```

4. **Close tabs**:
   ```bash
   tabctl list | grep "example.com" | cut -f1 | tabctl close
   ```

## Commands

### Core Commands

- `tabctl list` - List all open tabs
- `tabctl close <tab_ids>` - Close specified tabs
- `tabctl activate <tab_id>` - Activate a tab
- `tabctl active` - Show active tabs
- `tabctl move` - Interactive tab editor for reordering

### Content Commands

- `tabctl text [tab_ids]` - Get text content from tabs
- `tabctl html [tab_ids]` - Get HTML content from tabs
- `tabctl words [tab_ids]` - Extract words for autocomplete
- `tabctl search <query>` - Search through indexed tab content
- `tabctl index [tab_ids]` - Index tab content for searching

### Management Commands

- `tabctl open <window_id>` - Open URLs from stdin
- `tabctl navigate <tab_id> <url>` - Navigate tab to URL
- `tabctl update <tab_id>` - Update tab properties
- `tabctl query` - Filter tabs using chrome.tabs API
- `tabctl screenshot` - Capture tab screenshots

### Utility Commands

- `tabctl windows` - Show available windows
- `tabctl clients` - Show browser clients
- `tabctl dup` - Show duplicate tab removal commands
- `tabctl install` - Install native messaging components

## Examples

### List and Filter Tabs

```bash
# List all tabs
tabctl list

# List tabs with URLs containing "github"
tabctl list | grep github

# Show only tab IDs and titles
tabctl list | cut -f1,2
```

### Close Tabs

```bash
# Close specific tabs
tabctl close a.1.123 b.2.456

# Close tabs matching pattern
tabctl list | grep "facebook.com" | cut -f1 | tabctl close

# Close duplicate tabs by URL
tabctl list | sort -k3 | awk -F$'\t' '{ if (a[$3]++ > 0) print }' | cut -f1 | tabctl close
```

### Search Tab Content

```bash
# Index current tabs
tabctl index

# Search for content
tabctl search "golang tutorial"

# Search with custom database
tabctl search --sqlite /tmp/tabs.db "react hooks"
```

### Integration with fzf

```bash
# Activate tab with fzf
tabctl list | fzf | cut -f1 | xargs tabctl activate

# Close tabs with fzf
tabctl list | fzf -m | cut -f1 | tabctl close
```

## Configuration

### Environment Variables

- `TABCTL_TARGET` - Default target hosts (e.g., "localhost:4625,localhost:4626")
- `TABCTL_SQLITE` - Default SQLite database path
- `EDITOR` - Editor for interactive tab management

### Native Messaging

TabCtl uses native messaging to communicate with browser extensions. The `tabctl install` command sets up the required configuration files:

- Firefox: `~/.mozilla/native-messaging-hosts/tabctl_mediator.json`
- Chrome: `~/.config/google-chrome/NativeMessagingHosts/tabctl_mediator.json`
- Chromium: `~/.config/chromium/NativeMessagingHosts/tabctl_mediator.json`
- Brave: `~/.config/BraveSoftware/Brave-Browser/NativeMessagingHosts/tabctl_mediator.json`

## Migration from BroTab

TabCtl maintains full compatibility with BroTab's command structure and browser extensions. To migrate:

1. Install tabctl binary
2. Run `tabctl install` (replaces `bt install`)
3. Update browser extensions to TabCtl versions
4. Replace `bt` commands with `tabctl`

All existing scripts and workflows should work without modification.

## Development

### Build

```bash
make build          # Build for current platform
make build-all      # Build for all platforms
make release        # Create full release with packages
```

### Test

```bash
make test           # Run tests
make lint           # Run linter
make fmt            # Format code
```

### Extensions

```bash
make extensions     # Package browser extensions
```

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   tabctl CLI    │───▶│  HTTP Mediator   │───▶│ Browser Extension│
│   (Go binary)   │    │  (Go HTTP srv)   │    │   (JavaScript)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                        │
                                ▼                        ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ SQLite FTS5 DB   │    │  Browser APIs   │
                       │   (Tab Search)    │    │ (Tabs/Windows)  │
                       └──────────────────┘    └─────────────────┘
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Credits

TabCtl is a Go port of [BroTab](https://github.com/balta2ar/brotab) by Yuri Bochkarev.