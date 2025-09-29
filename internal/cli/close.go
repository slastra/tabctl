package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
	"github.com/tabctl/tabctl/internal/utils"
)

var closeCmd = &cobra.Command{
	Use:   "close [tab_ids...]",
	Short: "Close specified tab IDs",
	Long: `Close specified tab IDs. Tab IDs should be in the following format:
"<prefix>.<window_id>.<tab_id>". You can use "list" command to obtain
tab IDs (first column). If no tab IDs are provided, reads from stdin.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCloseTabs(args)
	},
}

func runCloseTabs(tabIDs []string) error {
	// Read from stdin if no args provided
	if len(tabIDs) == 0 {
		lines, err := utils.ReadStdinLines()
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		tabIDs = lines
	}

	if len(tabIDs) == 0 {
		fmt.Println("No tabs to close")
		return nil
	}

	// Create browser manager
	bm := client.NewBrowserManager(targetBrowser)
	defer bm.Close()

	// Close tabs
	if err := bm.CloseTabs(tabIDs); err != nil {
		return fmt.Errorf("failed to close tabs: %w", err)
	}

	fmt.Printf("Closed %d tab(s)\n", len(tabIDs))
	return nil
}