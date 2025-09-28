package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
	"github.com/tabctl/tabctl/internal/utils"
	"github.com/tabctl/tabctl/pkg/api"
)

var windowIDCmd = &cobra.Command{
	Use:   "window-id <tab_id>",
	Short: "Get the system window ID for a browser tab",
	Long: `Get the system window ID (X11) for a browser tab by activating it
and matching its title with wmctrl output.`,
	Args: cobra.ExactArgs(1),
	RunE: runWindowID,
}

func init() {
	rootCmd.AddCommand(windowIDCmd)
}

func runWindowID(cmd *cobra.Command, args []string) error {
	tabID := args[0]

	// Parse tab ID to get components
	prefix, _, _, err := utils.ParseTabID(tabID)
	if err != nil {
		return fmt.Errorf("invalid tab ID: %w", err)
	}

	// Create parallel client to query all browsers
	pc := client.NewParallelClient(globalHost)

	// Find the correct client for this tab
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

	// First, activate the tab to ensure its window is focused
	if err := targetClient.ActivateTab(tabID, true); err != nil {
		return fmt.Errorf("failed to activate tab: %w", err)
	}

	// Get the tab's title
	tabs, err := targetClient.ListTabs()
	if err != nil {
		return fmt.Errorf("failed to list tabs: %w", err)
	}

	var tabTitle string
	for _, tab := range tabs {
		// Tab.ID already contains the full ID (e.g., "f.1.1")
		if tab.ID == tabID {
			tabTitle = tab.Title
			break
		}
	}

	if tabTitle == "" {
		return fmt.Errorf("could not find title for tab %s", tabID)
	}

	// Use wmctrl to find the window by title
	wmctrlCmd := exec.Command("wmctrl", "-l")
	output, err := wmctrlCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run wmctrl: %w", err)
	}

	// Search for the window with matching title
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, tabTitle) {
			// Extract window ID (first field)
			fields := strings.Fields(line)
			if len(fields) > 0 {
				fmt.Println(fields[0])
				return nil
			}
		}
	}

	// If exact match fails, try partial match (first few words)
	titleWords := strings.Fields(tabTitle)
	if len(titleWords) > 3 {
		shortTitle := strings.Join(titleWords[:3], " ")
		for _, line := range lines {
			if strings.Contains(line, shortTitle) {
				fields := strings.Fields(line)
				if len(fields) > 0 {
					fmt.Println(fields[0])
					return nil
				}
			}
		}
	}

	return fmt.Errorf("could not find window for tab %s (title: %s)", tabID, tabTitle)
}