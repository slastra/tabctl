package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	htmlTSV            string
	htmlCleanup        bool
	htmlDelimiterRegex string
	htmlReplaceWith    string
)

var htmlCmd = &cobra.Command{
	Use:   "html [tab_ids...]",
	Short: "Show html from tabs",
	Long:  `Show html from all tabs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGetHTML(args)
	},
}

func init() {
	htmlCmd.Flags().StringVar(&htmlTSV, "tsv", "", "tsv file to save results to")
	htmlCmd.Flags().BoolVar(&htmlCleanup, "cleanup", false, "force removal of extra whitespace")
	htmlCmd.Flags().StringVar(&htmlDelimiterRegex, "delimiter-regex", `\n|\r|\t`, "Regex that is used to match delimiters in the page text")
	htmlCmd.Flags().StringVar(&htmlReplaceWith, "replace-with", " ", "String that is used to replaced matched delimiters")
}

func runGetHTML(tabIDs []string) error {
	// TODO: Implement get HTML functionality
	fmt.Printf("Get HTML from tabs %v - not implemented yet\n", tabIDs)
	return nil
}