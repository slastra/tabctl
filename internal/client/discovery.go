package client

import (
	"fmt"

	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/dbus"
)

// MediatorInfo represents information about a discovered mediator
type MediatorInfo struct {
	Browser string
}

// DiscoverMediators discovers all available D-Bus mediators. An empty
// result with a nil error means the bus is healthy but no mediator is
// registered; errors indicate the bus itself could not be queried.
func DiscoverMediators() ([]MediatorInfo, error) {
	// Create D-Bus client
	client, err := dbus.NewClient()
	if err != nil {
		return nil, fmt.Errorf("cannot connect to D-Bus session bus: %w", err)
	}
	defer client.Close()

	ctx, cancel := config.CommandContext()
	defer cancel()

	// Discover browsers on D-Bus
	browsers, err := client.DiscoverBrowsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("D-Bus discovery failed: %w", err)
	}

	// Create MediatorInfo for each browser
	mediators := make([]MediatorInfo, 0, len(browsers))
	for _, browser := range browsers {
		mediators = append(mediators, MediatorInfo{Browser: browser})
	}

	return mediators, nil
}
