package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var clientsCmd = &cobra.Command{
	Use:   "clients",
	Short: "Display available browser clients",
	Long: `Display available browser clients (mediators), their prefixes and
address (host:port), native app PIDs, and browser names`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShowClients()
	},
}

func runShowClients() error {
	// TODO: Implement show clients functionality
	fmt.Println("Show clients - not implemented yet")
	return nil
}