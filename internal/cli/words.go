package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	wordsMatchRegex string
	wordsJoinWith   string
)

var wordsCmd = &cobra.Command{
	Use:   "words [tab_ids...]",
	Short: "Show sorted unique words from tabs",
	Long: `Show sorted unique words from all active tabs of all clients. This is
a helper for webcomplete plugin that helps complete words from the
browser`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGetWords(args)
	},
}

func init() {
	wordsCmd.Flags().StringVar(&wordsMatchRegex, "match-regex", `\w+`, "Regex that is used to match words in the page text")
	wordsCmd.Flags().StringVar(&wordsJoinWith, "join-with", "\n", "String that is used to join matched words")
}

func runGetWords(tabIDs []string) error {
	// TODO: Implement get words functionality
	fmt.Printf("Get words from tabs %v - not implemented yet\n", tabIDs)
	return nil
}