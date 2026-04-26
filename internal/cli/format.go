package cli

import (
	"encoding/json"
	"fmt"
	"os"

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

