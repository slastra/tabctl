#!/bin/bash
# Rofi browser tab switcher with favicons using twenty-icons.com

ICON_CACHE="$HOME/.cache/rofi-tabs/icons"
ICON_SIZE=32
FINAL_SIZE=64
LOG_FILE="$HOME/.cache/rofi-tabs/debug.log"
mkdir -p "$ICON_CACHE"

# Enable debug logging
exec 2>>"$LOG_FILE"
echo "=== Script started at $(date) ===" >>"$LOG_FILE"

# Function to extract domain from URL
get_domain() {
  echo "$1" | sed -E 's|^https?://||' | sed -E 's|^www\.||' | cut -d'/' -f1
}

# Function to lighten dark icons while preserving transparency
lighten_dark_icon() {
  local icon_path="$1"
  local lightened_path="${icon_path%.png}-light.png"

  # If lightened version exists, return it
  [ -f "$lightened_path" ] && echo "$lightened_path" && return

  # Check if icon is too dark (average brightness < 128)
  local brightness=$(convert "$icon_path" -alpha off -colorspace gray -format "%[fx:int(mean*255)]" info: 2>/dev/null)

  if [ -n "$brightness" ] && [ "$brightness" -lt 128 ]; then
    # Icon is dark, put it on a white circle background with padding
    # Resize to final size (32px) with quality downsampling
    local padding=8
    local margin=4
    local circle_radius=$((FINAL_SIZE / 2 - margin))
    local icon_size=$((FINAL_SIZE - padding * 2 - margin * 2))
    convert -size ${FINAL_SIZE}x${FINAL_SIZE} xc:none \
      -fill white -draw "circle $((FINAL_SIZE / 2)),$((FINAL_SIZE / 2)) $((FINAL_SIZE / 2)),$margin" \
      \( "$icon_path" -resize ${icon_size}x${icon_size} \) \
      -gravity center -composite \
      "$lightened_path" 2>>"$LOG_FILE"

    [ -f "$lightened_path" ] && echo "$lightened_path" || echo "$icon_path"
  else
    # Icon is light enough, resize to final size
    convert "$icon_path" -resize ${FINAL_SIZE}x${FINAL_SIZE} "$lightened_path" 2>>"$LOG_FILE"
    echo "$lightened_path"
  fi
}

# Function to get cached favicon only (no downloads)
get_cached_favicon() {
  local url="$1"
  local domain=$(get_domain "$url")
  local icon_path="$ICON_CACHE/$domain.png"

  # Check if we have a lightened version cached
  local lightened_path="${icon_path%.png}-light.png"
  [ -f "$lightened_path" ] && echo "$lightened_path" && return

  # Return cached icon if exists (and lighten if needed)
  if [ -f "$icon_path" ]; then
    lighten_dark_icon "$icon_path"
    return
  fi

  # No cached icon available
  echo ""
}

# Function to download favicon in background
download_favicon() {
  local url="$1"
  local domain=$(get_domain "$url")
  local icon_path="$ICON_CACHE/$domain.png"
  local fallback_icon="$ICON_CACHE/fallback-firefox.png"

  # Skip if already cached
  [ -f "$icon_path" ] && return

  echo "Downloading favicon for domain: $domain" >>"$LOG_FILE"

  # Download favicon from twenty-icons.com
  local favicon_url="https://twenty-icons.com/$domain/$ICON_SIZE"
  if curl -s -f "$favicon_url" -o "$icon_path" 2>>"$LOG_FILE"; then
    echo "Successfully downloaded icon to: $icon_path" >>"$LOG_FILE"
    lighten_dark_icon "$icon_path" >/dev/null
  else
    echo "Failed to download favicon for $domain, using fallback" >>"$LOG_FILE"
    # Create fallback Firefox icon if it doesn't exist
    if [ ! -f "$fallback_icon" ]; then
      curl -s -f "https://twenty-icons.com/mozilla.org/$ICON_SIZE" -o "$fallback_icon" 2>>"$LOG_FILE"
    fi
    # Copy fallback to domain-specific cache so we don't retry
    if [ -f "$fallback_icon" ]; then
      cp "$fallback_icon" "$icon_path"
      lighten_dark_icon "$icon_path" >/dev/null
    fi
  fi
}

# Get tabs from tabctl as JSON
tabs_json=$(tabctl list --format json)

# Build rofi entries with cached icons only (instant display)
entries="New Window\x00icon\x1ffirefox\n"
declare -A tab_map
missing_urls=()

# Process each tab in a single jq pass
while IFS=$'\t' read -r id title url; do
  icon=$(get_cached_favicon "$url")

  # Track missing icons for background download
  if [ -z "$icon" ]; then
    missing_urls+=("$url")
  fi

  # Build entry with icon metadata
  if [ -n "$icon" ]; then
    entries+="$title\x00icon\x1f$icon\x00info\x1f$id\n"
  else
    entries+="$title\x00info\x1f$id\n"
  fi

  tab_map["$title"]="$id"
done < <(echo "$tabs_json" | jq -r '.[] | [.id, .title, .url] | @tsv')

# Download missing favicons in background for next run
if [ ${#missing_urls[@]} -gt 0 ]; then
  echo "Downloading ${#missing_urls[@]} missing favicons in background" >>"$LOG_FILE"
  (
    for url in "${missing_urls[@]}"; do
      download_favicon "$url" &
    done
    wait
  ) &
fi

# Show rofi menu
echo "Showing rofi menu..." >>"$LOG_FILE"
echo "Entries (first 500 chars): ${entries:0:500}" >>"$LOG_FILE"
selected=$(echo -ne "$entries" | rofi -dmenu -i -p "󱦞 " -show-icons -theme ~/.config/rofi/browser-tabs.rasi)

echo "User selected: $selected" >>"$LOG_FILE"

if [ -n "$selected" ]; then
  if [ "$selected" == "New Window" ]; then
    echo "Launching new Firefox window" >>"$LOG_FILE"
    firefox &
    exit 0
  fi

  # Get tab ID from selection
  tab_id=$(echo -ne "$entries" | grep -F "$selected" | head -1 | sed -n 's/.*\x00info\x1f\([^\x00]*\).*/\1/p')
  echo "Extracted tab_id from entry: $tab_id" >>"$LOG_FILE"

  if [ -z "$tab_id" ]; then
    # Fallback: lookup in map
    tab_id="${tab_map[$selected]}"
    echo "Fallback tab_id from map: $tab_id" >>"$LOG_FILE"
  fi

  if [ -n "$tab_id" ]; then
    # Activate the tab and get its title in one jq call
    tab_title=$(echo "$tabs_json" | jq -r ".[] | select(.id == \"$tab_id\") | .title")

    tabctl activate "$tab_id"

    # Find and focus the window in niri
    window_id=$(niri msg -j windows | jq -r --arg title "$tab_title" '.[] | select(.title | contains($title)) | .id' | head -1)

    [ -n "$window_id" ] && niri msg action focus-window --id "$window_id"
  fi
else
  echo "No selection made (user cancelled)" >>"$LOG_FILE"
fi

echo "=== Script ended at $(date) ===" >>"$LOG_FILE"
