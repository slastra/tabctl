package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
)

var (
	activateFocused bool
)

var activateCmd = &cobra.Command{
	Use:   "activate <tab_id>",
	Short: "Activate given tab ID",
	Long: `Activate given tab ID. Tab ID should be in the following format:
"<prefix>.<window_id>.<tab_id>"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runActivateTab(args[0], activateFocused)
	},
}

func init() {
	activateCmd.Flags().BoolVar(&activateFocused, "focused", false, "make browser focused after tab activation")
}

func runActivateTab(tabID string, focused bool) error {
	// Create browser manager
	bm := client.NewBrowserManager(targetBrowser)
	defer bm.Close()

	// Activate the tab
	if err := bm.ActivateTab(tabID); err != nil {
		return fmt.Errorf("failed to activate tab: %w", err)
	}

	fmt.Printf("Activated tab %s\n", tabID)
	return nil
}