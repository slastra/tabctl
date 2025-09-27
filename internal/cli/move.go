package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Move tabs around",
	Long: `Move tabs around. This command lists available tabs and runs
the editor. In the editor you can:
1) reorder tabs -- tabs will be moved in the browser
2) delete tabs -- tabs will be closed
3) change window ID of the tabs -- tabs will be moved to specified windows`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMoveTabs()
	},
}

func runMoveTabs() error {
	// TODO: Implement move tabs functionality
	fmt.Println("Move tabs (interactive editor) - not implemented yet")
	return nil
}