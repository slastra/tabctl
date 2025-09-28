package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tabs",
	Long: `List available tabs. The command will request all available clients
(browser plugins, mediators), and will display browser tabs in the
following format:
"<prefix>.<window_id>.<tab_id><Tab>Page title<Tab>URL"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runListTabs()
	},
}

func runListTabs() error {
	// Create parallel client to query all browsers
	pc := client.NewParallelClient(globalHost)

	// List all tabs
	tabs, err := pc.ListAllTabs()
	if err != nil {
		return fmt.Errorf("failed to list tabs: %w", err)
	}

	// Use the format helper
	return FormatTabList(tabs)
}

// getTabPrefix determines the prefix for a tab
func getTabPrefix(tab interface{}) string {
	// This would normally be determined by which client returned the tab
	// For now, return a default prefix
	return "a"
}