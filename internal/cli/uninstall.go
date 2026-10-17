package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/browsers"
	"github.com/tabctl/tabctl/internal/config"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the tabctl native-messaging host from every browser",
	Long: `Remove the tabctl native-messaging manifest from every supported browser
(or just one with --browser). Browsers that have none are skipped.

Unlike install, this does not require the browser to be detected: a manifest
left behind by a browser that has since been removed is still cleaned up.
The browser extension itself is not touched; remove it from the browser.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUninstall()
	},
}

func runUninstall() error {
	selected, err := filterBrowsersByTarget(browsers.All(), targetBrowser)
	if err != nil {
		return err
	}

	removed, err := uninstallFrom(selected)
	for _, path := range removed {
		fmt.Printf("  ✓ removed %s\n", path)
	}
	if len(removed) == 0 && err == nil {
		fmt.Println("No tabctl native-messaging manifests found")
	}
	return err
}

// uninstallFrom deletes the manifest from each browser's manifest directory,
// returning the paths it removed. A missing manifest is not an error; any
// other failure is collected and the rest are still attempted.
func uninstallFrom(list []browsers.Browser) ([]string, error) {
	var removed []string
	var errs []error
	for _, b := range list {
		path := filepath.Join(b.ManifestDir(), config.NativeHostName+".json")
		err := os.Remove(path)
		switch {
		case err == nil:
			removed = append(removed, path)
		case errors.Is(err, fs.ErrNotExist):
		default:
			errs = append(errs, fmt.Errorf("%s: %w", b.Name, err))
		}
	}
	return removed, errors.Join(errs...)
}
