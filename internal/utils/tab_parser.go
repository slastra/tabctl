package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tabctl/tabctl/pkg/types"
)

// ParseTabID parses a tab ID string into its components
func ParseTabID(tabID string) (prefix, windowID, tabIDNum string, err error) {
	parts := strings.Split(tabID, ".")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid tab ID format: %s", tabID)
	}
	return parts[0], parts[1], parts[2], nil
}

// ParseTabLine parses a tab line from list output
func ParseTabLine(line string) (types.Tab, error) {
	parts := strings.Split(line, "\t")
	if len(parts) < 3 {
		return types.Tab{}, fmt.Errorf("invalid tab line format: %s", line)
	}

	// Parse tab ID to extract the window ID
	_, windowIDStr, _, err := ParseTabID(parts[0])
	if err != nil {
		return types.Tab{}, err
	}

	windowID, err := strconv.Atoi(windowIDStr)
	if err != nil {
		return types.Tab{}, fmt.Errorf("invalid window ID number: %s", windowIDStr)
	}

	tab := types.Tab{
		ID:       parts[0], // Use the full tab ID string (e.g., "a.1.1")
		Title:    parts[1],
		URL:      parts[2],
		WindowID: windowID,
	}

	// Parse additional fields if available (index, active, pinned)
	if len(parts) >= 4 {
		if index, err := strconv.Atoi(parts[3]); err == nil {
			tab.Index = index
		}
	}
	if len(parts) >= 5 {
		if active, err := strconv.ParseBool(parts[4]); err == nil {
			tab.Active = active
		}
	}
	if len(parts) >= 6 {
		if pinned, err := strconv.ParseBool(parts[5]); err == nil {
			tab.Pinned = pinned
		}
	}

	return tab, nil
}

// FormatTabLine formats a tab into the standard output format
func FormatTabLine(tab types.Tab) string {
	return fmt.Sprintf("%s\t%s\t%s", tab.ID, tab.Title, tab.URL)
}

// SplitTabIDs splits a string containing multiple tab IDs
func SplitTabIDs(input string) []string {
	if input == "" {
		return []string{}
	}

	// Handle both space and newline separated tab IDs
	input = strings.ReplaceAll(input, "\n", " ")
	input = strings.ReplaceAll(input, "\t", " ")

	var tabIDs []string
	for _, id := range strings.Fields(input) {
		if id != "" {
			tabIDs = append(tabIDs, id)
		}
	}

	return tabIDs
}

// ParsePrefixAndWindowID parses prefix and window ID from a string
func ParsePrefixAndWindowID(prefixWindowID string) (prefix, windowID string, err error) {
	if strings.Contains(prefixWindowID, ".") {
		parts := strings.Split(prefixWindowID, ".")
		if len(parts) >= 2 {
			prefix = parts[0] + "."
			if parts[1] != "" {
				windowID = parts[1]
			}
			return prefix, windowID, nil
		}
	}

	// Just a prefix
	if !strings.HasSuffix(prefixWindowID, ".") {
		prefix = prefixWindowID + "."
	} else {
		prefix = prefixWindowID
	}

	return prefix, "", nil
}

// ValidateTabID validates a tab ID format
func ValidateTabID(tabID string) error {
	_, _, _, err := ParseTabID(tabID)
	return err
}

// GetTabPrefix extracts the prefix from a tab ID
func GetTabPrefix(tabID string) string {
	if len(tabID) >= 2 && tabID[1] == '.' {
		return tabID[:2]
	}
	return ""
}

// GetWindowID extracts the window ID from a tab ID
func GetWindowID(tabID string) string {
	parts := strings.Split(tabID, ".")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// MakeTabID creates a tab ID from components
func MakeTabID(prefix, windowID, tabIDNum string) string {
	return fmt.Sprintf("%s.%s.%s", prefix, windowID, tabIDNum)
}

// GroupTabsByPrefix groups tab IDs by their prefix
func GroupTabsByPrefix(tabIDs []string) map[string][]string {
	groups := make(map[string][]string)
	for _, tabID := range tabIDs {
		prefix := GetTabPrefix(tabID)
		if prefix != "" {
			groups[prefix] = append(groups[prefix], tabID)
		}
	}
	return groups
}

// FilterTabsByURL filters tabs by URL pattern
func FilterTabsByURL(tabs []types.Tab, pattern string) []types.Tab {
	var filtered []types.Tab
	for _, tab := range tabs {
		if strings.Contains(tab.URL, pattern) {
			filtered = append(filtered, tab)
		}
	}
	return filtered
}

// FilterTabsByTitle filters tabs by title pattern
func FilterTabsByTitle(tabs []types.Tab, pattern string) []types.Tab {
	var filtered []types.Tab
	for _, tab := range tabs {
		if strings.Contains(tab.Title, pattern) {
			filtered = append(filtered, tab)
		}
	}
	return filtered
}