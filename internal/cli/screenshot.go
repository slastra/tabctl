package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Return base64 screenshot in json object",
	Long: `Return base64 screenshot in json object with keys: 'data' (base64 png), 
'tab' (tab id of visible tab), 'window' (window id of visible tab), 
'api' (prefix of client api)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScreenshot()
	},
}

func runScreenshot() error {
	// TODO: Implement screenshot functionality
	fmt.Println("Screenshot - not implemented yet")
	return nil
}