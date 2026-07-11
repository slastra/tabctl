package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
	"github.com/tabctl/tabctl/internal/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mediator/extension versions and protocol compatibility",
	Long: `Show, for each connected browser, the mediator and extension
versions and whether their native-messaging protocol versions match.
A mismatch means the extension and the tabctl package are out of sync —
update whichever is behind.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

func runStatus() error {
	fmt.Printf("tabctl:   %s (protocol v%d)\n\n", config.Version, config.ProtocolVersion)

	bm, err := client.NewBrowserManager(targetBrowser)
	if err != nil {
		fmt.Printf("No browsers connected: %v\n", err)
		return nil
	}
	defer bm.Close()

	anyMismatch := false
	for _, info := range bm.Statuses() {
		fmt.Printf("%s\n", info.Browser)
		if info.ExtensionVersion == "" {
			fmt.Printf("  extension: not yet handshaked\n")
			continue
		}
		fmt.Printf("  mediator:  %s (protocol v%d)\n", info.MediatorVersion, info.MediatorProtocol)
		fmt.Printf("  extension: %s (protocol v%d)\n", info.ExtensionVersion, info.ExtensionProtocol)
		if info.Compatible {
			fmt.Printf("  status:    OK\n")
		} else {
			anyMismatch = true
			fmt.Printf("  status:    PROTOCOL MISMATCH — update the extension and the tabctl package to matching versions\n")
		}
	}

	if anyMismatch {
		return fmt.Errorf("one or more browsers have a protocol mismatch")
	}
	return nil
}
