package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
	"github.com/tabctl/tabctl/internal/utils"
)

var openCmd = &cobra.Command{
	Use:   "open [urls...]",
	Short: "Open URLs in new tabs",
	Long: `Open URLs in new tabs and print the new tab IDs, one per line.
If no URLs are provided, reads them from stdin. With more than one
browser connected, use --browser to choose the target.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpenURLs(args)
	},
}

func runOpenURLs(urls []string) error {
	// Read from stdin if no args provided
	if len(urls) == 0 {
		lines, err := utils.ReadStdinLines()
		if err != nil {
			return fmt.Errorf("failed to read from stdin: %w", err)
		}
		urls = lines
	}

	if len(urls) == 0 {
		fmt.Println("No URLs to open")
		return nil
	}

	// Create browser manager
	bm, err := client.NewBrowserManager(targetBrowser)
	if err != nil {
		return err
	}
	defer bm.Close()

	// Open URLs; print IDs for tabs that were opened even on partial failure
	ids, err := bm.OpenURLs(urls)
	for _, id := range ids {
		fmt.Println(id)
	}
	if err != nil {
		return fmt.Errorf("failed to open tabs: %w", err)
	}
	return nil
}
