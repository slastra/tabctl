package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/tabctl/tabctl/pkg/types"
)

// FormatTabList writes a list of tabs to stdout using the global output
// format and delimiter flags.
func FormatTabList(tabs []types.Tab) error {
	return writeTabList(os.Stdout, tabs, outputFormat, delimiter)
}

// writeTabList renders tabs to w. Empty lists print a format-appropriate
// placeholder. Factored out for testability.
func writeTabList(w io.Writer, tabs []types.Tab, format, delim string) error {
	if len(tabs) == 0 {
		if format == "json" {
			fmt.Fprintln(w, "[]")
		} else {
			fmt.Fprintln(w, "No tabs found")
		}
		return nil
	}

	switch format {
	case "json":
		return writeJSON(w, tabs)
	case "simple":
		return writeSimple(w, tabs)
	default: // tsv
		return writeTSV(w, tabs, delim)
	}
}

// writeTSV outputs "ID<delim>Title<delim>URL" per tab.
func writeTSV(w io.Writer, tabs []types.Tab, delim string) error {
	for _, tab := range tabs {
		if _, err := fmt.Fprintf(w, "%s%s%s%s%s\n", tab.ID, delim, tab.Title, delim, tab.URL); err != nil {
			return err
		}
	}
	return nil
}

// writeJSON outputs the tabs as an indented JSON array.
func writeJSON(w io.Writer, tabs []types.Tab) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(tabs)
}

// writeSimple outputs tab titles only. Display-only: the output cannot be
// mapped back to tab IDs — use tsv or json for anything round-trippable.
func writeSimple(w io.Writer, tabs []types.Tab) error {
	for _, tab := range tabs {
		if _, err := fmt.Fprintf(w, "%s\n", tab.Title); err != nil {
			return err
		}
	}
	return nil
}
