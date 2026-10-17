package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available tabs",
	Long:  `List available tabs from all browsers connected via D-Bus.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runListTabs()
	},
}

func runListTabs() error {
	// Create browser manager to query browsers
	bm, err := client.NewBrowserManager(targetBrowser)
	if err != nil {
		return err
	}
	defer bm.Close()

	// List all tabs (ListAllTabs already contextualizes its error)
	tabs, err := bm.ListAllTabs()
	var partial *client.PartialError
	if errors.As(err, &partial) {
		// Some browsers answered. Print what they returned and exit 0 as
		// before; the warning goes to stderr so stdout stays parseable.
		fmt.Fprintf(os.Stderr, "warning: some browsers did not respond: %v\n", partial)
	} else if err != nil {
		return err
	}

	// Use the format helper
	return FormatTabList(tabs)
}
