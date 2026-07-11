package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Register the tabctl native-messaging host for detected browsers",
	Long: `Register the tabctl native-messaging host for every detected browser
(or just one with --browser), then print which extension to install.

Runs non-interactively, so it is safe to call from setup scripts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstallMediator()
	},
}

// installResult is the outcome of installing for one browser.
type installResult struct {
	browser BrowserInfo
	err     error
}

func runInstallMediator() error {
	detected := detectInstalledBrowsers()
	if len(detected) == 0 {
		supported := getSupportedBrowsers()
		names := make([]string, len(supported))
		for i, b := range supported {
			names[i] = b.Name
		}
		return fmt.Errorf("no supported browsers detected (supported: %s)", strings.Join(names, ", "))
	}

	// Narrow to a single browser when --browser is given.
	selected, err := filterBrowsersByTarget(detected, targetBrowser)
	if err != nil {
		return err
	}

	mediatorPath, err := findMediatorPath()
	if err != nil {
		return fmt.Errorf("failed to find mediator: %w", err)
	}

	results := make([]installResult, 0, len(selected))
	for _, browser := range selected {
		results = append(results, installResult{browser: browser, err: installForBrowserInfo(browser, mediatorPath)})
	}

	printInstallResults(results)

	for _, r := range results {
		if r.err != nil {
			return fmt.Errorf("one or more browsers failed to configure")
		}
	}
	return nil
}

// filterBrowsersByTarget returns the detected browsers matching target (by
// full or first-word name, case-insensitive). An empty target selects all.
func filterBrowsersByTarget(detected []BrowserInfo, target string) ([]BrowserInfo, error) {
	if target == "" {
		return detected, nil
	}
	var selected []BrowserInfo
	for _, b := range detected {
		firstWord := strings.SplitN(b.Name, " ", 2)[0]
		if strings.EqualFold(b.Name, target) || strings.EqualFold(firstWord, target) {
			selected = append(selected, b)
		}
	}
	if len(selected) == 0 {
		names := make([]string, len(detected))
		for i, b := range detected {
			names[i] = b.Name
		}
		return nil, fmt.Errorf("no detected browser matches %q (detected: %s)", target, strings.Join(names, ", "))
	}
	return selected, nil
}

// printInstallResults writes a persistent plain-text summary and the store
// links for the extension types that were configured.
func printInstallResults(results []installResult) {
	fmt.Println("Native-messaging host:")
	hasFirefox, hasChrome := false, false
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("  ✗ %s: %v\n", r.browser.Name, r.err)
			continue
		}
		fmt.Printf("  ✓ %s\n", r.browser.Name)
		switch r.browser.Type {
		case "firefox":
			hasFirefox = true
		case "chromium":
			hasChrome = true
		}
	}

	if !hasFirefox && !hasChrome {
		return
	}

	fmt.Println("\nInstall the browser extension:")
	if hasFirefox {
		fmt.Println("  • Firefox / Zen:")
		fmt.Println("      https://addons.mozilla.org/en-US/firefox/addon/tabctl1/")
	}
	if hasChrome {
		fmt.Println("  • Chrome / Chromium / Brave / Helium:")
		fmt.Println("      https://chromewebstore.google.com/detail/tabctl/baomblllgemcgbignhpbipgiofmjdhpn")
	}
	fmt.Println("\nThen restart the browser.")
}
