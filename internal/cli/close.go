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
	bm, err := client.NewBrowserManager(targetBrowser)
	if err != nil {
		return err
	}
	defer bm.Close()

	// Close tabs; report the count actually dispatched, not the input count
	closed, err := bm.CloseTabs(tabIDs)
	if closed > 0 {
		fmt.Printf("Closed %d tab(s)\n", closed)
	}
	if err != nil {
		return fmt.Errorf("failed to close tabs: %w", err)
	}
	return nil
}