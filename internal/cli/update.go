package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	updateTabID         string
	updateURL           string
	updateOpenerTabID   string
	updateActive        bool
	updateAutoDiscardable bool
	updateHighlighted   bool
	updateMuted         bool
	updatePinned        bool
	updateInfo          string
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update tabs state",
	Long: `Update tabs state, e.g. URL. There are two ways to specify updates:
1. stdin, pass JSON of the form:
[{"tab_id": "b.20.130", "properties": {"url": "http://www.google.com"}}]
Where "properties" can be anything defined here:
https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/API/tabs/update
Example:
echo '[{"tab_id":"a.2118.2156", "properties":{"url":"https://google.com"}}]' | tabctl update

2. arguments, e.g.: tabctl update -tabId b.1.862 -url="http://www.google.com" +muted`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdateTabs()
	},
}

func init() {
	updateCmd.Flags().StringVar(&updateTabID, "tabId", "", "tab id to apply updates to")
	updateCmd.Flags().StringVar(&updateURL, "url", "", "a URL to navigate the tab to")
	updateCmd.Flags().StringVar(&updateOpenerTabID, "openerTabId", "", "the ID of the tab that opened this tab")
	updateCmd.Flags().StringVar(&updateInfo, "info", "", "JSON update info")

	// Boolean flags with + and - variants
	updateCmd.Flags().BoolVar(&updateActive, "active", false, "make tab active")
	updateCmd.Flags().BoolVar(&updateAutoDiscardable, "autoDiscardable", false, "whether the tab should be discarded automatically")
	updateCmd.Flags().BoolVar(&updateHighlighted, "highlighted", false, "adds/removes the tab to/from the current selection")
	updateCmd.Flags().BoolVar(&updateMuted, "muted", false, "mute/unmute tab")
	updateCmd.Flags().BoolVar(&updatePinned, "pinned", false, "pin/unpin tab")
}

func runUpdateTabs() error {
	// TODO: Implement update tabs functionality
	if updateInfo != "" {
		fmt.Printf("Update tabs with JSON info: %s - not implemented yet\n", updateInfo)
	} else {
		fmt.Printf("Update tab %s - not implemented yet\n", updateTabID)
	}
	return nil
}