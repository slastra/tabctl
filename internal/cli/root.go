package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	outputFormat  string = "tsv"  // Output format: tsv, json, simple
	delimiter     string = "\t"   // Field delimiter
	noHeaders     bool            // Suppress headers in output
	targetBrowser string = ""     // Target specific browser (empty = all)
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tabctl",
	Short: "Control your browser's tabs from the command line",
	Long: `tabctl (tab control) is a command-line tool that helps you manage
browser tabs via D-Bus. It can list, close, and activate tabs across
Firefox and Chrome-based browsers.`,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
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
	rootCmd.PersistentFlags().StringVar(&outputFormat, "format", "tsv", "Output format: tsv, json, simple")
	rootCmd.PersistentFlags().StringVar(&delimiter, "delimiter", "\t", "Field delimiter for TSV output")
	rootCmd.PersistentFlags().BoolVar(&noHeaders, "no-headers", false, "Suppress headers in output")
	rootCmd.PersistentFlags().StringVar(&targetBrowser, "browser", "", "Target specific browser (e.g., Firefox, Brave)")

	// Add subcommands
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(activateCmd)
	rootCmd.AddCommand(installCmd)
}