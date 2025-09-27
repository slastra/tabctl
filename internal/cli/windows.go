package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
	"github.com/tabctl/tabctl/pkg/types"
)

var windowsCmd = &cobra.Command{
	Use:   "windows",
	Short: "Display available windows",
	Long: `Display available prefixes and window IDs, along with the number of
tabs in every window`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShowWindows()
	},
}

func runShowWindows() error {
	// Create parallel client to query all browsers
	pc := client.NewParallelClient(globalHost)

	// Get all tabs to organize by window
	tabs, err := pc.ListAllTabs()
	if err != nil {
		return fmt.Errorf("failed to list tabs: %w", err)
	}

	// Organize tabs by window
	windowMap := make(map[string]*types.Window)
	for _, tab := range tabs {
		// Create window key with prefix
		prefix := getTabPrefix(tab)
		windowKey := fmt.Sprintf("%s.%d", prefix, tab.WindowID)

		if _, exists := windowMap[windowKey]; !exists {
			windowMap[windowKey] = &types.Window{
				ID:       tab.WindowID,
				Tabs:     []types.Tab{},
				TabCount: 0,
			}
		}
		windowMap[windowKey].Tabs = append(windowMap[windowKey].Tabs, tab)
		windowMap[windowKey].TabCount++
	}

	// Convert map to slice and format output
	switch outputFormat {
	case "json":
		// For JSON, include the full window details
		var windows []types.Window
		for _, window := range windowMap {
			windows = append(windows, *window)
		}
		return FormatWindowList(windows)

	case "simple":
		// Simple format: just window ID and tab count
		for windowKey, window := range windowMap {
			fmt.Printf("%s (%d tabs)\n", windowKey, window.TabCount)
		}

	default: // tsv
		// TSV format: window_key tab_count
		for windowKey, window := range windowMap {
			fmt.Printf("%s%s%d\n", windowKey, delimiter, window.TabCount)
		}
	}

	return nil
}