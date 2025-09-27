#!/bin/bash
# rofi-tabctl.sh - Rofi integration for tabctl with desktop switching
# Usage: rofi -modi tabs:rofi-tabctl.sh -show tabs

# Configuration
TABCTL="${TABCTL:-tabctl}"
ROFI_THEME="${ROFI_THEME:-~/.config/rofi/browser-tabs.rasi}"
SEP="␞"  # Field separator for rofi

# Function to get window manager tool
get_wm_tool() {
    if command -v wmctrl &> /dev/null; then
        echo "wmctrl"
    elif command -v xdotool &> /dev/null; then
        echo "xdotool"
    else
        echo "none"
    fi
}

# Function to activate window using available tools
activate_window() {
    local window_name="$1"
    local wm_tool=$(get_wm_tool)

    case "$wm_tool" in
        wmctrl)
            # Use wmctrl to find and activate window
            local window_id=$(wmctrl -l | grep -F "$window_name" | head -1 | cut -d" " -f1)
            if [ -n "$window_id" ]; then
                wmctrl -i -a "$window_id"
                return 0
            fi
            ;;
        xdotool)
            # Use xdotool as fallback
            local window_id=$(xdotool search --name "$window_name" | head -1)
            if [ -n "$window_id" ]; then
                xdotool windowactivate "$window_id"
                return 0
            fi
            ;;
    esac
    return 1
}

# Function to get browser name from port/prefix
get_browser_name() {
    local prefix="$1"
    case "$prefix" in
        a*) echo "Firefox" ;;
        b*) echo "Chrome" ;;
        c*) echo "Chromium" ;;
        d*) echo "Brave" ;;
        *) echo "Browser" ;;
    esac
}

if [ -z "$1" ]; then
    # First run - list all tabs for rofi
    # Add "New Window" option at the top
    echo -e "New Window${SEP}new"

    # Get tabs and format for rofi display
    $TABCTL list | while IFS=$'\t' read -r id title url; do
        # Truncate title and URL for display
        if [ ${#title} -gt 60 ]; then
            title="${title:0:57}..."
        fi
        if [ ${#url} -gt 40 ]; then
            domain=$(echo "$url" | sed -E 's|https?://([^/]+).*|\1|')
            if [ ${#domain} -gt 40 ]; then
                domain="${domain:0:37}..."
            fi
            url="$domain"
        fi

        # Format: "Title: URL ␞ tab_id"
        echo -e "${title}: ${url}${SEP}${id}"
    done
else
    # User selected an entry
    selected="$1"

    # Extract tab ID after separator
    tab_id=$(echo "$selected" | awk -F "$SEP" '{print $2}')

    if [ "$tab_id" = "new" ]; then
        # Open new browser window
        browser=$(get_browser_name "a")
        case "$browser" in
            Firefox) firefox --new-window & ;;
            Chrome) google-chrome --new-window & ;;
            Chromium) chromium --new-window & ;;
            Brave) brave --new-window & ;;
            *) xdg-open "http://" & ;;
        esac
        exit 0
    fi

    if [ -n "$tab_id" ]; then
        # Activate the selected tab
        $TABCTL activate --focused "$tab_id"

        # Extract window ID from tab ID (format: prefix.window_id.tab_id)
        bt_window_id=$(echo "$tab_id" | cut -d'.' -f2)

        # Get the active tab in that window to find window title
        # Note: This requires the 'active' command which might not work with Python mediator
        # As fallback, we'll use the selected tab's title

        # Extract the title from the original selection
        window_title=$(echo "$selected" | cut -d':' -f1 | sed 's/'"$SEP"'.*//')

        # Try to activate the window
        if ! activate_window "$window_title"; then
            # Fallback: try with browser name
            prefix=$(echo "$tab_id" | cut -d'.' -f1)
            browser=$(get_browser_name "$prefix")
            activate_window "$browser"
        fi
    fi
fi