package mediator

import (
	"errors"
	"io"
	"os"

	"github.com/tabctl/tabctl/internal/dbus"
)

// Mediator coordinates communication between the browser extension and CLI via D-Bus.
type Mediator struct {
	browserAPI *BrowserAPI
	dbusServer *dbus.Server
	transport  *StdTransport
}

// NewMediator creates a new mediator with automatic disconnection detection.
func NewMediator(browser string) (*Mediator, error) {
	transport := NewStdTransport(os.Stdin, os.Stdout)
	browserAPI := NewBrowserAPI(transport)
	dbusHandler := NewDBusHandler(browserAPI)

	dbusServer, err := dbus.NewServer(browser, dbusHandler)
	if err != nil {
		return nil, err
	}
	// Extension tabs_changed notifications fan out as D-Bus TabsUpdated
	// signals, so bar/strip consumers get push updates instead of polling.
	browserAPI.SetTabsChangedHandler(dbusServer.EmitTabsUpdated)

	return &Mediator{
		browserAPI: browserAPI,
		dbusServer: dbusServer,
		transport:  transport,
	}, nil
}

// Run starts the D-Bus server and blocks until the browser disconnects.
// A clean disconnect (EOF or closed transport) returns nil.
func (m *Mediator) Run() error {
	if err := m.dbusServer.Start(); err != nil {
		return err
	}

	err, ok := <-m.transport.Errors()
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
