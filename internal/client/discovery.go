package client

import (
	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/dbus"
)

// MediatorInfo represents information about a discovered mediator
type MediatorInfo struct {
	Browser string
	Prefix  string
}

// DiscoverMediators discovers all available D-Bus mediators
func DiscoverMediators() []MediatorInfo {
	var mediators []MediatorInfo

	// Create D-Bus client
	client, err := dbus.NewClient()
	if err != nil {
		return mediators
	}
	defer client.Close()

	ctx, cancel := config.CommandContext()
	defer cancel()

	// Discover browsers on D-Bus
	browsers, err := client.DiscoverBrowsers(ctx)
	if err != nil {
		return mediators
	}

	// Create MediatorInfo for each browser
	for _, browser := range browsers {
		mediator := MediatorInfo{
			Browser: browser,
			Prefix:  determinePrefixForBrowser(browser),
		}

		mediators = append(mediators, mediator)
	}

	return mediators
}