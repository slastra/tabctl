package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
	"github.com/tabctl/tabctl/internal/utils"
	"github.com/tabctl/tabctl/pkg/api"
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
	// Parse tab ID to get prefix
	prefix, _, _, err := utils.ParseTabID(tabID)
	if err != nil {
		return fmt.Errorf("invalid tab ID: %w", err)
	}

	// Find the appropriate client
	pc := client.NewParallelClient(globalHost)
	clients := pc.GetClients()

	var targetClient api.Client
	for _, c := range clients {
		if c.GetPrefix() == prefix+"." {
			targetClient = c
			break
		}
	}

	if targetClient == nil {
		return fmt.Errorf("no mediator found for prefix %s", prefix)
	}

	// Activate the tab
	if err := targetClient.ActivateTab(tabID, focused); err != nil {
		return fmt.Errorf("failed to activate tab: %w", err)
	}

	fmt.Printf("Activated tab %s\n", tabID)
	return nil
}