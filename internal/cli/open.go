package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tabctl/tabctl/internal/client"
)

var openCmd = &cobra.Command{
	Use:   "open <prefix.window_id>",
	Short: "Open URLs from stdin",
	Long: `Open URLs from the stdin (one URL per line). One positional argument is
required: <prefix>.<window_id> OR <client>. If window_id is not
specified, URL will be opened in the active window of the specified
client. If window_id is 0, URLs will be opened in new window.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runOpenURLs(args[0])
	},
}

func runOpenURLs(prefixWindowID string) error {
	// Parse the prefix and window ID
	parts := strings.Split(prefixWindowID, ".")
	if len(parts) < 1 || len(parts) > 2 {
		return fmt.Errorf("invalid format, expected: prefix or prefix.window_id")
	}

	prefix := parts[0]
	windowID := ""
	if len(parts) == 2 {
		windowID = parts[1]
	}

	// Read URLs from stdin
	var urls []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			urls = append(urls, url)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading from stdin: %w", err)
	}

	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided")
	}

	// Find the appropriate client
	pc := client.NewParallelClient(globalHost)
	clients := pc.GetClients()

	var targetClient *client.Client
	for _, c := range clients {
		if c.GetPrefix() == prefix+"." {
			if tc, ok := c.(*client.Client); ok {
				targetClient = tc
				break
			}
		}
	}

	if targetClient == nil {
		// If no client found with prefix, try using the first available
		if len(clients) > 0 {
			if tc, ok := clients[0].(*client.Client); ok {
				targetClient = tc
			}
		}
	}

	if targetClient == nil {
		return fmt.Errorf("no mediator found for prefix %s", prefix)
	}

	// Open URLs
	tabIDs, err := targetClient.OpenURLs(urls, windowID)
	if err != nil {
		return fmt.Errorf("failed to open URLs: %w", err)
	}

	// Output the new tab IDs
	if outputFormat == "json" {
		return FormatStringList(tabIDs)
	}

	for _, tabID := range tabIDs {
		fmt.Println(tabID)
	}

	return nil
}