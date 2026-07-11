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
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(tabs)
}

// formatSimple outputs tab titles only. Display-only: the output cannot be
// mapped back to tab IDs — use tsv or json for anything round-trippable.
func formatSimple(tabs []types.Tab) error {
	for _, tab := range tabs {
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

