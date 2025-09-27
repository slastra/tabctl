package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dupCmd = &cobra.Command{
	Use:   "dup",
	Short: "Display reminder on how to close duplicate tabs",
	Long:  `Display reminder on how to show duplicate tabs using command-line tools`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runShowDuplicates()
	},
}

func runShowDuplicates() error {
	fmt.Println("Close duplicates by Title:")
	fmt.Println(`tabctl list | sort -k2 | awk -F$'\t' '{ if (a[$2]++ > 0) print }' | cut -f1 | tabctl close`)
	fmt.Println("")
	fmt.Println("Close duplicates by URL:")
	fmt.Println(`tabctl list | sort -k3 | awk -F$'\t' '{ if (a[$3]++ > 0) print }' | cut -f1 | tabctl close`)
	return nil
}