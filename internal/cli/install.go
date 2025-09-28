package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/utils"
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
	// No flags needed - interactive mode is default
}

func runInstallMediator() error {
	// Check if we're in a terminal
	if !utils.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("interactive mode requires a terminal environment")
	}

	// Detect available browsers for selection (silent)
	detected := detectInstalledBrowsers()

	if len(detected) == 0 {
		fmt.Println("No supported browsers detected.")
		fmt.Println("Supported browsers: Firefox, Zen Browser, Chrome, Chromium, Brave")
		return fmt.Errorf("no supported browsers found")
	}

	// Show interactive selection
	browsers, err := selectBrowsersInteractive(detected)
	if err != nil {
		return fmt.Errorf("browser selection failed: %w", err)
	}

	if len(browsers) == 0 {
		fmt.Println("No browsers selected for installation.")
		return nil
	}

	// Get the path to tabctl-mediator binary
	mediatorPath, err := findMediatorPath()
	if err != nil {
		return fmt.Errorf("failed to find mediator: %v", err)
	}

	// Install for each selected browser (collect results)
	var results []InstallResult
	for _, browser := range browsers {
		result := InstallResult{
			Browser: browser,
			Success: false,
		}

		if err := installForBrowserInfo(browser, mediatorPath); err != nil {
			result.Error = err
		} else {
			result.Success = true
		}

		results = append(results, result)
	}

	// Show results using bubbletea
	return showInstallationResults(results)
}

// hasFirefoxBased checks if any detected browsers are Firefox-based
func hasFirefoxBased(browsers []BrowserInfo) bool {
	for _, browser := range browsers {
		if browser.Type == "firefox" {
			return true
		}
	}
	return false
}

// hasChromiumBased checks if any detected browsers are Chromium-based
func hasChromiumBased(browsers []BrowserInfo) bool {
	for _, browser := range browsers {
		if browser.Type == "chromium" {
			return true
		}
	}
	return false
}