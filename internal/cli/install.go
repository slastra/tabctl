package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/browsers"
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
	browser browsers.Browser
	err     error
}

func runInstallMediator() error {
	detected := browsers.Installed()
	if len(detected) == 0 {
		return fmt.Errorf("no supported browsers detected (supported: %s)", strings.Join(browserNames(browsers.All()), ", "))
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
		results = append(results, installResult{browser: browser, err: installForBrowser(browser, mediatorPath)})
	}

	printInstallResults(results)

	for _, r := range results {
		if r.err != nil {
			return fmt.Errorf("one or more browsers failed to configure")
		}
	}
	return nil
}

// filterBrowsersByTarget returns the detected browsers matching target. An
// empty target selects all. See Browser.Matches for what counts as a match.
func filterBrowsersByTarget(detected []browsers.Browser, target string) ([]browsers.Browser, error) {
	if target == "" {
		return detected, nil
	}
	var selected []browsers.Browser
	for _, b := range detected {
		if b.Matches(target) {
			selected = append(selected, b)
		}
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("no detected browser matches %q (detected: %s)", target, strings.Join(browserNames(detected), ", "))
	}
	return selected, nil
}

func browserNames(list []browsers.Browser) []string {
	names := make([]string, len(list))
	for i, b := range list {
		names[i] = b.Name
	}
	return names
}

// printInstallResults writes a persistent plain-text summary and the store
// links for the extension families that were configured.
func printInstallResults(results []installResult) {
	fmt.Println("Native-messaging host:")
	hasFirefox, hasChrome := false, false
	for _, r := range results {
		if r.err != nil {
			fmt.Printf("  ✗ %s: %v\n", r.browser.Name, r.err)
			continue
		}
		fmt.Printf("  ✓ %s\n", r.browser.Name)
		switch r.browser.Family {
		case browsers.Firefox:
			hasFirefox = true
		case browsers.Chromium:
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
		fmt.Println("  • Chrome / Chromium / Brave / Brave Origin / Helium:")
		fmt.Println("      https://chromewebstore.google.com/detail/tabctl/baomblllgemcgbignhpbipgiofmjdhpn")
	}
	fmt.Println("\nThen restart the browser.")
}
