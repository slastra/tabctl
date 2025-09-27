package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var navigateCmd = &cobra.Command{
	Use:   "navigate <tab_id> <url>",
	Short: "Navigate to URLs",
	Long: `Navigate to URLs. There are two ways to specify tab ids and URLs:
1. stdin: lines with pairs of "tab_id<tab>url"
2. arguments: tabctl navigate <tab_id> "<url>", e.g. tabctl navigate b.20.1 "https://google.com"
stdin has the priority.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNavigateURLs(args[0], args[1])
	},
}

func runNavigateURLs(tabID, url string) error {
	// TODO: Implement navigate URLs functionality
	fmt.Printf("Navigate tab %s to %s - not implemented yet\n", tabID, url)
	return nil
}