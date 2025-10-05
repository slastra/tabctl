#!/bin/bash
# This script is used to activate a tab/window/workspace using rofi with tabctl

sep="␞"
# Get list of tabs: "Title: URL [separator] tab_id"
tabs=$(tabctl list | awk -v sep="$sep" -F "\t" '{if ($2) print $2 ": " $3 sep $1}')
tabs=$(echo -e "New Window\n$tabs")

# Show rofi menu and get selection
selected=$(echo "$tabs" \
    | rofi -dmenu -i -p "󱦞" -display-columns 1 -display-column-separator "$sep" -theme ~/.config/rofi/browser-tabs.rasi \
    | head -1)

if [ "$selected" ]; then
    if [ "$selected" == "New Window" ]; then
        # Open new browser window (try common browsers)
        if command -v brave &> /dev/null; then
            brave
        elif command -v firefox &> /dev/null; then
            firefox
        elif command -v chromium &> /dev/null; then
            chromium
        elif command -v google-chrome &> /dev/null; then
            google-chrome
        fi
        exit 0
    fi

    # Extract tab ID from selection
    tab_id=$(echo "$selected" | awk -F "$sep" '{print $2}')

    # Activate the tab
    tabctl activate "$tab_id"

    # Give the browser a moment to switch tabs
    sleep 0.2

    # Get the active tab's title after activation
    active_tab_info=$(tabctl list | grep -E "^${tab_id}\s" | head -1)

    if [ "$active_tab_info" ]; then
        # Extract the title from the active tab info
        tab_title=$(echo "$active_tab_info" | cut -f2)

        # Find window by partial title match (browsers often append " - Browser Name")
        # First try exact match, then partial match
        window_id=$(wmctrl -l | grep -F "$tab_title" | head -1 | cut -d" " -f1)

        if [ -z "$window_id" ]; then
            # Try partial match (first part of title)
            partial_title=$(echo "$tab_title" | cut -d' ' -f1-5)
            window_id=$(wmctrl -l | grep -F "$partial_title" | head -1 | cut -d" " -f1)
        fi

        if [ "$window_id" ]; then
            # Focus the window
            wmctrl -i -a "$window_id"
        else
            echo "Could not find window for tab: $tab_title" >&2
        fi
    else
        echo "Could not get info for activated tab: $tab_id" >&2
    fi
fi
