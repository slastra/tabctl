package mediator

import (
	"os"
	"time"

	"github.com/tabctl/tabctl/internal/dbus"
)

// Mediator coordinates communication between the browser extension and CLI via D-Bus.
type Mediator struct {
	browser    string
	browserAPI *BrowserAPI
	dbusServer *dbus.Server
}

// NewMediator creates a new D-Bus-only mediator instance.
func NewMediator(browser string) (*Mediator, error) {
	// Create stdio transport for native messaging
	transport := NewDefaultTransport()

	// Create browser API handler
	browserAPI := NewBrowserAPI(transport, browser)

	// Create D-Bus handler adapter
	dbusHandler := NewDBusHandler(browserAPI)

	// Create D-Bus server
	dbusServer, err := dbus.NewServer(browser, dbusHandler)
	if err != nil {
		return nil, err
	}

	return &Mediator{
		browser:    browser,
		browserAPI: browserAPI,
		dbusServer: dbusServer,
	}, nil
}

// Run starts the D-Bus server and waits for browser disconnection.
func (m *Mediator) Run() error {
	// Start D-Bus server
	if err := m.dbusServer.Start(); err != nil {
		return err
	}

	// Mediator is running

	// Keep running until interrupted
	// The mediator will exit when browser closes stdin
	for {
		time.Sleep(1 * time.Second)
		// Check if stdin is still open
		if _, err := os.Stdin.Stat(); err != nil {
			// Browser disconnected, shutting down
			break
		}
	}

	return nil
}

// Shutdown gracefully shuts down the mediator.
func (m *Mediator) Shutdown() error {
	if m.dbusServer != nil {
		return m.dbusServer.Stop()
	}
	return nil
}