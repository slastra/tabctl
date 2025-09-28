#!/bin/bash
# rofi-hyprctl.sh - Tab switcher for Hyprland using hyprctl
# Usage: ./rofi-hyprctl.sh

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

# Small delay to let the window title update
sleep 0.1

# Find and focus the browser window in Hyprland
# Method 1: Get the active window (most reliable after activation with --focused)
active_window=$(hyprctl activewindow -j | jq -r '.address')

if [ -n "$active_window" ] && [ "$active_window" != "null" ]; then
    echo "Active window found: $active_window"
    # The window is already focused since we used --focused
else
    echo "No active window found, searching for browser..."

    # Method 2: Find browser window by class
    # Get all windows and find first browser match
    browser_window=""

    for class in firefox brave chromium google-chrome; do
        browser_window=$(hyprctl clients -j | jq -r ".[] | select(.class | ascii_downcase | contains(\"$class\")) | .address" | head -1)
        if [ -n "$browser_window" ] && [ "$browser_window" != "null" ]; then
            echo "Found browser window: $browser_window (class: $class)"
            break
        fi
    done

    if [ -n "$browser_window" ] && [ "$browser_window" != "null" ]; then
        # Focus the browser window
        hyprctl dispatch focuswindow "address:$browser_window"

        # Get the workspace of the browser window
        workspace=$(hyprctl clients -j | jq -r ".[] | select(.address == \"$browser_window\") | .workspace.id")

        if [ -n "$workspace" ] && [ "$workspace" != "null" ]; then
            echo "Switching to workspace: $workspace"
            hyprctl dispatch workspace "$workspace"
        fi
    else
        echo "Could not find browser window"

        # As a last resort, try to focus by class name
        for class in firefox brave chromium google-chrome; do
            if hyprctl dispatch focuswindow "class:^$class" 2>/dev/null; then
                echo "Focused window by class: $class"
                break
            fi
        done
    fi
fi