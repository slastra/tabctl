package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
)

var navigateCmd = &cobra.Command{
	Use:   "navigate <tab_id> <url>",
	Short: "Load a URL in an existing tab",
	Long: `Load a URL in an existing tab, in place. The tab keeps its position,
its pinned state and its history; only its page changes. Tab ID should be in
the following format: "<prefix>.<window_id>.<tab_id>"

Navigation does not switch to the tab. Combine with activate when you want
both:

  tabctl navigate firefox.1.2 https://example.com && tabctl activate --focused firefox.1.2`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNavigateTab(args[0], args[1])
	},
}

func runNavigateTab(tabID, url string) error {
	if url == "" {
		return fmt.Errorf("url must not be empty")
	}

	bm, err := client.NewBrowserManager(targetBrowser)
	if err != nil {
		return err
	}
	defer bm.Close()

	if err := bm.NavigateTab(tabID, url); err != nil {
		return fmt.Errorf("failed to navigate tab: %w", err)
	}

	fmt.Printf("Navigated tab %s to %s\n", tabID, url)
	return nil
}
