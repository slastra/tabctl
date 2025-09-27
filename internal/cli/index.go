package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	indexSQLite        string
	indexTSV           string
	indexDelimiterRegex string
	indexReplaceWith   string
)

var indexCmd = &cobra.Command{
	Use:   "index [tab_ids...]",
	Short: "Index the text from browser's tabs",
	Long:  `Index the text from browser's tabs. Text is put into sqlite fts5 table.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runIndexTabs(args)
	},
}

func init() {
	indexCmd.Flags().StringVar(&indexSQLite, "sqlite", "/tmp/tabs.sqlite", "sqlite DB filename")
	indexCmd.Flags().StringVar(&indexTSV, "tsv", "", "get text from tabs and index the results")
	indexCmd.Flags().StringVar(&indexDelimiterRegex, "delimiter-regex", `\n|\r|\t`, "Regex that is used to match delimiters in the page text")
	indexCmd.Flags().StringVar(&indexReplaceWith, "replace-with", " ", "String that is used to replaced matched delimiters")
}

func runIndexTabs(tabIDs []string) error {
	// TODO: Implement index tabs functionality
	fmt.Printf("Index tabs %v to %s - not implemented yet\n", tabIDs, indexSQLite)
	return nil
}