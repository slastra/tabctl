package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	searchSQLite string
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across your indexed tabs",
	Long:  `Search across your indexed tabs using sqlite fts5 plugin.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSearchTabs(args[0])
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchSQLite, "sqlite", "/tmp/tabs.sqlite", "sqlite DB filename")
}

func runSearchTabs(query string) error {
	// TODO: Implement search tabs functionality
	fmt.Printf("Search tabs for '%s' in %s - not implemented yet\n", query, searchSQLite)
	return nil
}