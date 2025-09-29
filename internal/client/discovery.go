package client

import (
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

	// Discover browsers on D-Bus
	browsers, err := client.DiscoverBrowsers()
	if err != nil {
		return mediators
	}

	// Create MediatorInfo for each browser
	for _, browser := range browsers {
		prefix := determineBrowserPrefix(browser)

		mediator := MediatorInfo{
			Browser: browser,
			Prefix:  prefix,
		}

		mediators = append(mediators, mediator)
	}

	return mediators
}

func determineBrowserPrefix(browser string) string {
	switch browser {
	case "Firefox":
		return "f."
	case "Chrome", "Chromium", "Brave":
		return "c."
	default:
		return "u."
	}
}