package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	installTests bool
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Configure browser settings to use tabctl mediator",
	Long:  `Configure browser settings to use tabctl mediator (native messaging app)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstallMediator()
	},
}

func init() {
	installCmd.Flags().BoolVar(&installTests, "tests", false, "install testing version of manifest for chromium")
}

func runInstallMediator() error {
	fmt.Println("Installing tabctl mediator...")

	// Get the path to tabctl-mediator binary
	mediatorPath, err := findMediatorPath()
	if err != nil {
		return fmt.Errorf("failed to find mediator: %v", err)
	}

	// Install for each browser
	browsers := []string{"firefox", "chrome", "chromium", "brave"}
	for _, browser := range browsers {
		if err := installForBrowser(browser, mediatorPath, installTests); err != nil {
			fmt.Printf("Warning: Failed to install for %s: %v\n", browser, err)
		} else {
			fmt.Printf("✓ Installed for %s\n", browser)
		}
	}

	fmt.Println("\nInstallation complete!")
	fmt.Println("Firefox extension: https://addons.mozilla.org/firefox/addon/tabctl/")
	fmt.Println("Chrome extension: https://chrome.google.com/webstore/detail/tabctl/...")

	return nil
}