package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/tabctl/tabctl/pkg/types"
)

// FormatOutput formats tabs based on the global output format flag
func FormatOutput(tabs []types.Tab) error {
	switch outputFormat {
	case "json":
		return formatJSON(tabs)
	case "simple":
		return formatSimple(tabs)
	case "tsv":
		fallthrough
	default:
		return formatTSV(tabs)
	}
}

// formatTSV outputs tabs in TSV format
func formatTSV(tabs []types.Tab) error {
	for _, tab := range tabs {
		fmt.Printf("%s%s%s%s%s\n", tab.ID, delimiter, tab.Title, delimiter, tab.URL)
	}
	return nil
}

// formatJSON outputs tabs as JSON
func formatJSON(tabs []types.Tab) error {
	// Convert to a format with string IDs for compatibility
	type JSONTab struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		URL      string `json:"url"`
		WindowID int    `json:"windowId"`
		Index    int    `json:"index"`
		Active   bool   `json:"active"`
		Pinned   bool   `json:"pinned"`
	}

	jsonTabs := make([]JSONTab, len(tabs))
	for i, tab := range tabs {
		jsonTabs[i] = JSONTab{
			ID:       tab.ID,
			Title:    tab.Title,
			URL:      tab.URL,
			WindowID: tab.WindowID,
			Index:    tab.Index,
			Active:   tab.Active,
			Pinned:   tab.Pinned,
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(jsonTabs)
}

// formatSimple outputs just tab IDs and titles (good for rofi)
func formatSimple(tabs []types.Tab) error {
	for _, tab := range tabs {
		// Simple format: just title for display, ID can be parsed by rofi
		fmt.Printf("%s\n", tab.Title)
	}
	return nil
}

// FormatTabList formats a list of tabs with proper IDs (used by multiple commands)
func FormatTabList(tabs []types.Tab) error {
	if len(tabs) == 0 {
		if outputFormat != "json" {
			fmt.Println("No tabs found")
		} else {
			fmt.Println("[]")
		}
		return nil
	}

	return FormatOutput(tabs)
}

// FormatWindowList formats a list of windows
func FormatWindowList(windows []types.Window) error {
	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(windows)
	case "simple":
		for _, window := range windows {
			fmt.Printf("Window %d (%d tabs)\n", window.ID, window.TabCount)
		}
	default: // tsv
		for _, window := range windows {
			fmt.Printf("%d%s%d\n", window.ID, delimiter, window.TabCount)
		}
	}
	return nil
}

// FormatSingleValue formats a single value (like active tab ID)
func FormatSingleValue(value string) error {
	switch outputFormat {
	case "json":
		result := map[string]string{"value": value}
		encoder := json.NewEncoder(os.Stdout)
		return encoder.Encode(result)
	default:
		fmt.Println(value)
	}
	return nil
}

// FormatStringList formats a list of strings
func FormatStringList(items []string) error {
	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(items)
	case "simple":
		for _, item := range items {
			fmt.Println(item)
		}
	default: // tsv
		fmt.Println(strings.Join(items, delimiter))
	}
	return nil
}