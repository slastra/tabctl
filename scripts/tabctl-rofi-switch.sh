#!/bin/bash
# tabctl-rofi-switch.sh - Complete tab switcher with virtual desktop support
# Usage: ./tabctl-rofi-switch.sh

# Configuration
TABCTL="${TABCTL:-tabctl}"
ROFI_THEME="${ROFI_THEME:-}"
SEP="␞"

# Colors for rofi (optional)
ROFI_OPTIONS=""
if [ -n "$ROFI_THEME" ]; then
    ROFI_OPTIONS="-theme $ROFI_THEME"
fi

# Get list of tabs formatted for display
get_tabs() {
    # Add "New Window" option
    echo -e "New Window${SEP}new"

    # Format tabs for display
    $TABCTL list --target "${TABCTL_TARGET:-localhost:4625}" | while IFS=$'\t' read -r id title url; do
        # Skip empty lines
        [ -z "$id" ] && continue

        # Truncate title for display
        display_title="$title"
        if [ ${#display_title} -gt 60 ]; then
            display_title="${display_title:0:57}..."
        fi

        # Extract domain from URL
        domain=$(echo "$url" | sed -E 's|https?://([^/]+).*|\1|')
        if [ ${#domain} -gt 40 ]; then
            domain="${domain:0:37}..."
        fi

        # Output format: "Title: domain ␞ tab_id"
        echo -e "${display_title}: ${domain}${SEP}${id}"
    done
}

# Main selection
selected=$(get_tabs | rofi -dmenu -i -p "󰖟 " \
    -display-columns 1 \
    -display-column-separator "$SEP" \
    $ROFI_OPTIONS)

# Exit if nothing selected
[ -z "$selected" ] && exit 0

# Extract tab ID
tab_id=$(echo "$selected" | awk -F "$SEP" '{print $2}')

if [ "$tab_id" = "new" ]; then
    # Open new browser window - try to detect which browser is running
    if pgrep -x firefox > /dev/null; then
        firefox --new-window &
    elif pgrep -x brave > /dev/null; then
        brave --new-window &
    elif pgrep -x chrome > /dev/null; then
        google-chrome --new-window &
    elif pgrep -x chromium > /dev/null; then
        chromium --new-window &
    else
        # Fallback to default browser
        xdg-open "https://google.com" &
    fi
    exit 0
fi

# Activate the selected tab
echo "Activating tab: $tab_id"
$TABCTL activate --focused "$tab_id"

# Extract window title from selection
window_title=$(echo "$selected" | cut -d':' -f1 | sed 's/'"$SEP"'.*//')
echo "Looking for window: $window_title"

# Try to find and activate the window
# Method 1: Search by exact title match
window_id=$(wmctrl -l | grep -F "$window_title" | head -1 | awk '{print $1}')

if [ -z "$window_id" ]; then
    # Method 2: Search by partial match (first few words)
    short_title=$(echo "$window_title" | cut -d' ' -f1-3)
    window_id=$(wmctrl -l | grep -F "$short_title" | head -1 | awk '{print $1}')
fi

if [ -z "$window_id" ]; then
    # Method 3: Try to find any browser window
    for browser in Firefox Brave Chrome Chromium; do
        window_id=$(wmctrl -l | grep -i "$browser" | head -1 | awk '{print $1}')
        [ -n "$window_id" ] && break
    done
fi

if [ -n "$window_id" ]; then
    echo "Activating window: $window_id"
    # -i flag uses window ID, -a activates (switches desktop and raises)
    wmctrl -i -a "$window_id"
else
    echo "Could not find window to activate"
    # As a last resort, try to activate by class
    wmctrl -x -a "firefox" 2>/dev/null || \
    wmctrl -x -a "brave-browser" 2>/dev/null || \
    wmctrl -x -a "google-chrome" 2>/dev/null || \
    wmctrl -x -a "chromium" 2>/dev/null
fi