package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
	"github.com/tabctl/tabctl/pkg/types"
)

var activeCmd = &cobra.Command{
	Use:   "active",
	Short: "Display active tabs",
	Long: `Display active tab for each client/window in the following format:
"<prefix>.<window_id>.<tab_id>"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShowActiveTabs()
	},
}

func runShowActiveTabs() error {
	// Create parallel client
	pc := client.NewParallelClient(globalHost)

	// Query for active tabs
	active := true
	query := types.TabQuery{
		Active: &active,
	}

	// Get active tabs from all browsers
	tabs, err := pc.QueryAllTabs(query)
	if err != nil {
		// Fallback: if query doesn't work, get all tabs and filter
		allTabs, err := pc.ListAllTabs()
		if err != nil {
			return fmt.Errorf("failed to get tabs: %w", err)
		}

		// Filter for active tabs manually
		tabs = []types.Tab{}
		for _, tab := range allTabs {
			if tab.Active {
				tabs = append(tabs, tab)
			}
		}
	}

	// Format active tabs
	return FormatTabList(tabs)
}