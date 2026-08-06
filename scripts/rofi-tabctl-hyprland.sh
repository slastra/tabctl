#!/bin/bash
# Rofi browser tab switcher with favicons (twenty-icons.com) for Hyprland.
# Lists tabs across all connected browsers, activates the chosen tab via
# tabctl, then focuses the owning Hyprland window (scoped by window class).

ICON_CACHE="$HOME/.cache/rofi-tabs/icons"
ICON_SIZE=32
FINAL_SIZE=64
LOG_FILE="$HOME/.cache/rofi-tabs/debug.log"
mkdir -p "$ICON_CACHE"

# A second profile of the same browser registers as chrome2, chrome3, ...
# Strip that suffix so every profile resolves to the same browser.
browser_base() {
  echo "${1%%[0-9]*}"
}

# Map a tabctl browser prefix → executable, display name, and icon-theme name.
# Add new browsers here as needed. Unknown prefixes fall through to a best guess.
browser_exec() {
  local b; b=$(browser_base "$1")
  case "$b" in
    firefox)  echo "firefox" ;;
    helium)   echo "helium-browser" ;;
    brave)    echo "brave" ;;
    chrome)   echo "google-chrome-stable" ;;
    chromium) echo "chromium" ;;
    zen)      echo "zen-browser" ;;
    *)        echo "$b" ;;
  esac
}
browser_name() {
  local b; b=$(browser_base "$1")
  case "$b" in
    firefox)  echo "Firefox" ;;
    helium)   echo "Helium" ;;
    brave)    echo "Brave" ;;
    chrome)   echo "Chrome" ;;
    chromium) echo "Chromium" ;;
    zen)      echo "Zen" ;;
    *)        echo "$b" ;;
  esac
}
browser_icon() {
  local b; b=$(browser_base "$1")
  case "$b" in
    firefox)  echo "firefox" ;;
    helium)   echo "helium-browser" ;;
    brave)    echo "brave-browser" ;;
    chrome)   echo "google-chrome" ;;
    chromium) echo "chromium" ;;
    zen)      echo "zen-browser" ;;
    *)        echo "$b" ;;
  esac
}

exec 2>>"$LOG_FILE"
echo "=== Script started at $(date) ===" >>"$LOG_FILE"

get_domain() {
  echo "$1" | sed -E 's|^https?://||' | sed -E 's|^www\.||' | cut -d'/' -f1
}

# Composite a favicon onto a white rounded square so dark themes don't swallow it.
add_rounded_square_background() {
  local icon_path="$1"
  local processed_path="${icon_path%.png}-processed.png"

  [ -s "$processed_path" ] && echo "$processed_path" && return

  local padding=8
  local margin=4
  local corner_radius=8
  local icon_size=$((FINAL_SIZE - padding * 2 - margin * 2))

  magick -size ${FINAL_SIZE}x${FINAL_SIZE} xc:none \
    -fill white -draw "roundrectangle $margin,$margin $((FINAL_SIZE - margin)),$((FINAL_SIZE - margin)) $corner_radius,$corner_radius" \
    \( "$icon_path" -resize ${icon_size}x${icon_size} \) \
    -gravity center -composite \
    "$processed_path" 2>>"$LOG_FILE"

  [ -s "$processed_path" ] && echo "$processed_path" || echo "$icon_path"
}

get_cached_favicon() {
  local url="$1"
  local domain
  domain=$(get_domain "$url")
  local icon_path="$ICON_CACHE/$domain.png"
  local processed_path="${icon_path%.png}-processed.png"

  [ -s "$processed_path" ] && echo "$processed_path" && return
  if [ -s "$icon_path" ]; then
    add_rounded_square_background "$icon_path"
    return
  fi
  echo ""
}

download_favicon() {
  local url="$1"
  local domain
  domain=$(get_domain "$url")
  local icon_path="$ICON_CACHE/$domain.png"

  [ -s "$icon_path" ] && return

  echo "Downloading favicon for domain: $domain" >>"$LOG_FILE"

  local favicon_url="https://twenty-icons.com/$domain/$ICON_SIZE"
  # --remove-on-error so a failed fetch doesn't leave a 0-byte file that
  # poisons the cache and blocks future retries.
  if curl -s -f --remove-on-error "$favicon_url" -o "$icon_path" 2>>"$LOG_FILE"; then
    add_rounded_square_background "$icon_path" >/dev/null
  else
    echo "No favicon for $domain (entry shown without an icon)" >>"$LOG_FILE"
  fi
}

tabs_json=$(tabctl list --format json)

# Build a "New Window" entry per browser that currently has tabs open.
# new_window_cmds[i] is the executable for the i-th top entry.
new_window_cmds=()
entries=""
declare -A seen_browsers
while read -r prefix; do
  [ -z "$prefix" ] && continue
  # Two profiles of one browser (chrome, chrome2) get one "New Window" entry.
  base=$(browser_base "$prefix")
  [ "${seen_browsers[$base]+x}" ] && continue
  seen_browsers[$base]=1

  exec_cmd=$(browser_exec "$prefix")
  command -v "$exec_cmd" >/dev/null 2>&1 || continue

  label="New $(browser_name "$prefix") Window"
  icon_name=$(browser_icon "$prefix")
  entries+="$label\x00icon\x1f$icon_name\n"
  new_window_cmds+=("$exec_cmd")
done < <(echo "$tabs_json" | jq -r '.[].id | split(".")[0]' | sort -u)

# Fallback: if no browsers detected (no tabs), offer a generic entry via xdg default.
if [ ${#new_window_cmds[@]} -eq 0 ]; then
  default_app=$(xdg-mime query default x-scheme-handler/https 2>/dev/null)
  if [ -n "$default_app" ] && command -v gtk-launch >/dev/null 2>&1; then
    entries+="New Window\x00icon\x1fweb-browser\n"
    new_window_cmds+=("gtk-launch ${default_app%.desktop}")
  fi
fi

n_browsers=${#new_window_cmds[@]}

# Tab entries follow the New Window entries. tab_ids[i] is the id for index i + n_browsers.
tab_ids=()
missing_urls=()

while IFS=$'\t' read -r id title url; do
  icon=$(get_cached_favicon "$url")
  domain=$(get_domain "$url")

  [ -z "$icon" ] && missing_urls+=("$url")

  display_text="$title - $domain"
  if [ -n "$icon" ]; then
    entries+="$display_text\x00icon\x1f$icon\n"
  else
    entries+="$display_text\n"
  fi

  tab_ids+=("$id")
done < <(echo "$tabs_json" | jq -r '.[] | [.id, .title, .url] | @tsv')

if [ ${#missing_urls[@]} -gt 0 ]; then
  echo "Downloading ${#missing_urls[@]} missing favicons in background" >>"$LOG_FILE"
  (
    for url in "${missing_urls[@]}"; do
      download_favicon "$url" &
    done
    wait
  ) &
fi

# Enter switches to the tab; Ctrl+w closes it. rofi returns the selected index
# on stdout (-format i) and signals the key via its exit code: 0 = Enter,
# 10 = custom-1 (Ctrl+w), 1 = cancelled (Escape). Ctrl+w defaults to
# kb-clear-line, so free that binding to give it to close.
selected_index=$(echo -ne "$entries" | rofi -dmenu -i -p "󱦞 " -show-icons -format i \
  -kb-clear-line "" -kb-custom-1 "Control+w" -mesg "Enter: switch • Ctrl+w: close")
rofi_exit=$?

echo "User selected index: $selected_index (rofi exit $rofi_exit)" >>"$LOG_FILE"

if [ "$rofi_exit" -eq 1 ] || [ -z "$selected_index" ] || [ "$selected_index" = "-1" ]; then
  echo "No selection made (user cancelled)" >>"$LOG_FILE"
  echo "=== Script ended at $(date) ===" >>"$LOG_FILE"
  exit 0
fi

# First n_browsers entries are "New <Browser> Window" launchers (Ctrl+q is a
# no-op on those — there's nothing to close, so just launch).
if [ "$selected_index" -lt "$n_browsers" ]; then
  cmd="${new_window_cmds[$selected_index]}"
  echo "Launching: $cmd" >>"$LOG_FILE"
  # Word-split intentional so "gtk-launch foo" works as well as "firefox".
  $cmd &
  exit 0
fi

tab_id="${tab_ids[$((selected_index - n_browsers))]}"
[ -z "$tab_id" ] && exit 0

# Ctrl+w (custom-1): close the tab, then reopen the menu so several can be
# closed in a row. v2's `close` is synchronous, so the tab is gone before we
# rebuild the list.
if [ "$rofi_exit" -eq 10 ]; then
  echo "Closing tab: $tab_id" >>"$LOG_FILE"
  tabctl close "$tab_id"
  exec "$0" "$@"
fi

tab_title=$(echo "$tabs_json" | jq -r --arg id "$tab_id" '.[] | select(.id == $id) | .title')
tabctl activate "$tab_id"

# tabctl ids are prefixed with the browser (e.g. firefox.1.1, helium.x.y).
# Use that to scope window matching to the right browser class. Profiles of
# one browser share a window class, so match on the base (chrome2 -> chrome).
browser_prefix=$(browser_base "${tab_id%%.*}")

# Prefer an exact title match within the right browser; fall back to substring.
# If nothing matches by class (rare — e.g. window class doesn't share the prefix),
# fall back to title-only match across all windows.
window_address=$(hyprctl clients -j | jq -r --arg title "$tab_title" --arg prefix "$browser_prefix" '
  [.[] | select((.class | ascii_downcase) | contains($prefix))] as $scoped
  | (($scoped | map(select(.title == $title))) + ($scoped | map(select(.title | contains($title)))))[0].address // empty
')

if [ -z "$window_address" ]; then
  window_address=$(hyprctl clients -j | jq -r --arg title "$tab_title" '
    [.[] | select(.title | contains($title))][0].address // empty
  ')
fi

if [ -n "$window_address" ]; then
  echo "Focusing window with address: $window_address" >>"$LOG_FILE"
  hyprctl dispatch focuswindow "address:$window_address"
else
  echo "No window found matching title: $tab_title (browser: $browser_prefix)" >>"$LOG_FILE"
fi

echo "=== Script ended at $(date) ===" >>"$LOG_FILE"
