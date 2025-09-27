package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	targetHosts   string
	globalHost    string = "localhost" // Default host for mediators
	outputFormat  string = "tsv"       // Output format: tsv, json, simple
	delimiter     string = "\t"        // Field delimiter
	noHeaders     bool                 // Suppress headers in output
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tabctl",
	Short: "Control your browser's tabs from the command line",
	Long: `tabctl (tab control) is a command-line tool that helps you manage
browser tabs. It can help you list, close, reorder, open and activate
your tabs across Firefox and Chrome-based browsers.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Println("No command has been specified")
			cmd.Help()
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&targetHosts, "target", "", "Target hosts IP:Port")
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "tsv", "Output format: tsv, json, simple")
	rootCmd.PersistentFlags().StringVar(&delimiter, "delimiter", "\t", "Field delimiter for TSV output")
	rootCmd.PersistentFlags().BoolVar(&noHeaders, "no-headers", false, "Suppress headers in output")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(activateCmd)
	rootCmd.AddCommand(activeCmd)
	rootCmd.AddCommand(moveCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(navigateCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(indexCmd)
	rootCmd.AddCommand(wordsCmd)
	rootCmd.AddCommand(textCmd)
	rootCmd.AddCommand(htmlCmd)
	rootCmd.AddCommand(dupCmd)
	rootCmd.AddCommand(windowsCmd)
	rootCmd.AddCommand(clientsCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(screenshotCmd)
}

// Helper function to get target hosts
func getTargetHosts() string {
	return targetHosts
}