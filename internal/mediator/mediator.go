package mediator

import (
	"errors"
	"io"
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

// Run starts the D-Bus server and blocks until the browser disconnects.
// A clean disconnect (EOF or closed transport) returns nil so callers can
// unwind gracefully instead of exiting mid-goroutine.
func (m *Mediator) Run() error {
	// Start D-Bus server
	if err := m.dbusServer.Start(); err != nil {
		return err
	}

	// Block until the transport reports disconnection (non-polling)
	err, ok := <-m.transport.GetErrorChannel()
	if !ok || err == nil || errors.Is(err, io.EOF) {
		return nil // browser disconnected cleanly
	}
	return err
}

// Shutdown gracefully shuts down the mediator.
func (m *Mediator) Shutdown() error {
	if m.transport != nil {
		m.transport.Close()
	}
	if m.dbusServer != nil {
		return m.dbusServer.Stop()
	}
	return nil
}