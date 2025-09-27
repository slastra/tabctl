package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	textTSV            string
	textCleanup        bool
	textDelimiterRegex string
	textReplaceWith    string
)

var textCmd = &cobra.Command{
	Use:   "text [tab_ids...]",
	Short: "Show text from tabs",
	Long:  `Show text from all tabs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGetText(args)
	},
}

func init() {
	textCmd.Flags().StringVar(&textTSV, "tsv", "", "tsv file to save results to")
	textCmd.Flags().BoolVar(&textCleanup, "cleanup", false, "force removal of extra whitespace")
	textCmd.Flags().StringVar(&textDelimiterRegex, "delimiter-regex", `\n|\r|\t`, "Regex that is used to match delimiters in the page text")
	textCmd.Flags().StringVar(&textReplaceWith, "replace-with", " ", "String that is used to replaced matched delimiters")
}

func runGetText(tabIDs []string) error {
	// TODO: Implement get text functionality
	fmt.Printf("Get text from tabs %v - not implemented yet\n", tabIDs)
	return nil
}