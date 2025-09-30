package mediator

import (
	"os"

	"github.com/tabctl/tabctl/internal/dbus"
)

// Mediator coordinates communication between the browser extension and CLI via D-Bus.
type Mediator struct {
	browser       string
	browserAPI    *BrowserAPI
	dbusServer    *dbus.Server
	transport     *StdTransport
}

// NewMediator creates a new mediator with automatic disconnection detection.
func NewMediator(browser string) (*Mediator, error) {
	// Create transport with automatic browser disconnection detection
	transport := NewStdTransport(os.Stdin, os.Stdout)

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
		transport:  transport,
	}, nil
}

// Run starts the D-Bus server and keeps running until the browser disconnects.
func (m *Mediator) Run() error {
	// Start D-Bus server
	if err := m.dbusServer.Start(); err != nil {
		return err
	}

	// Monitor for browser disconnection (non-polling, immediate detection)
	go func() {
		<-m.transport.GetErrorChannel()
		// Browser disconnected, exit cleanly
		os.Exit(0)
	}()

	// Keep the main goroutine alive
	select {}
}

// Shutdown gracefully shuts down the mediator.
func (m *Mediator) Shutdown() error {
	if m.dbusServer != nil {
		return m.dbusServer.Stop()
	}
	return nil
}