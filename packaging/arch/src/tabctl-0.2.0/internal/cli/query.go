package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
	"github.com/tabctl/tabctl/pkg/types"
)

var (
	queryActive           bool
	queryPinned           bool
	queryAudible          bool
	queryMuted            bool
	queryHighlighted      bool
	queryDiscarded        bool
	queryAutoDiscardable  bool
	queryCurrentWindow    bool
	queryLastFocusedWindow bool
	queryWindowFocused    bool
	queryStatus           string
	queryTitle            string
	queryURL              []string
	queryWindowID         int
	queryWindowType       string
	queryIndex            int
	queryInfo             string
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Filter tabs using chrome.tabs api",
	Long:  `Filter tabs using chrome.tabs api.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQueryTabs()
	},
}

func init() {
	queryCmd.Flags().BoolVar(&queryActive, "active", false, "tabs are active in their windows")
	queryCmd.Flags().BoolVar(&queryPinned, "pinned", false, "tabs are pinned")
	queryCmd.Flags().BoolVar(&queryAudible, "audible", false, "tabs are audible")
	queryCmd.Flags().BoolVar(&queryMuted, "muted", false, "tabs are muted")
	queryCmd.Flags().BoolVar(&queryHighlighted, "highlighted", false, "tabs are highlighted")
	queryCmd.Flags().BoolVar(&queryDiscarded, "discarded", false, "tabs are discarded")
	queryCmd.Flags().BoolVar(&queryAutoDiscardable, "autoDiscardable", false, "tabs can be discarded automatically")
	queryCmd.Flags().BoolVar(&queryCurrentWindow, "currentWindow", false, "tabs are in the current window")
	queryCmd.Flags().BoolVar(&queryLastFocusedWindow, "lastFocusedWindow", false, "tabs are in the last focused window")
	queryCmd.Flags().BoolVar(&queryWindowFocused, "windowFocused", false, "tabs are in the focused window")

	queryCmd.Flags().StringVar(&queryStatus, "status", "", "whether the tabs have completed loading (loading or complete)")
	queryCmd.Flags().StringVar(&queryTitle, "title", "", "match page titles against a pattern")
	queryCmd.Flags().StringSliceVar(&queryURL, "url", nil, "match tabs against URL patterns")
	queryCmd.Flags().IntVar(&queryWindowID, "windowId", 0, "the ID of the parent window")
	queryCmd.Flags().StringVar(&queryWindowType, "windowType", "", "the type of window (normal, popup, panel, app, devtools)")
	queryCmd.Flags().IntVar(&queryIndex, "index", 0, "the position of the tabs within their windows")
	queryCmd.Flags().StringVar(&queryInfo, "info", "", "the queryInfo parameter as JSON")
}

func runQueryTabs() error {
	// Create parallel client to query all browsers
	pc := client.NewParallelClient(globalHost)

	// Build query from flags
	query := types.TabQuery{}

	// Set boolean flags based on their values
	// For simplicity, just use the values directly since they're booleans
	if queryActive {
		query.Active = &queryActive
	}
	if queryPinned {
		query.Pinned = &queryPinned
	}
	if queryAudible {
		query.Audible = &queryAudible
	}
	if queryMuted {
		query.Muted = &queryMuted
	}
	if queryHighlighted {
		query.Highlighted = &queryHighlighted
	}
	if queryDiscarded {
		query.Discarded = &queryDiscarded
	}
	if queryAutoDiscardable {
		query.AutoDiscardable = &queryAutoDiscardable
	}
	if queryCurrentWindow {
		query.CurrentWindow = &queryCurrentWindow
	}
	if queryLastFocusedWindow {
		query.LastFocusedWindow = &queryLastFocusedWindow
	}

	// Set other fields
	if queryStatus != "" {
		query.Status = queryStatus
	}
	if queryTitle != "" {
		query.Title = queryTitle
	}
	if len(queryURL) > 0 {
		query.URL = queryURL
	}
	if queryWindowID != 0 {
		query.WindowID = &queryWindowID
	}
	if queryWindowType != "" {
		query.WindowType = queryWindowType
	}
	if queryIndex != 0 {
		query.Index = &queryIndex
	}

	// Query tabs
	tabs, err := pc.QueryAllTabs(query)
	if err != nil {
		return fmt.Errorf("failed to query tabs: %w", err)
	}

	// Format output
	return FormatTabList(tabs)
}