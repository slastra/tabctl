package mediator

import (
	"github.com/tabctl/tabctl/internal/config"
)

// Mediator coordinates communication between the browser extension and CLI.
// It runs as a native messaging host, receiving commands via Unix socket
// from the CLI and forwarding them to the browser extension via stdio.
type Mediator struct {
	config     *config.MediatorConfig
	unixServer *UnixServer
	remoteAPI  *RemoteAPI
}

// NewMediator creates a new mediator instance.
// It sets up the stdio transport for browser communication
// and a Unix socket server for CLI connections.
func NewMediator(cfg *config.MediatorConfig) (*Mediator, error) {
	// Create stdio transport for native messaging
	transport := NewDefaultTransport()

	// Create remote API handler
	remoteAPI := NewRemoteAPI(transport)

	// Create Unix socket server for CLI connections
	unixServer, err := NewUnixServer(cfg, remoteAPI)
	if err != nil {
		return nil, err
	}

	return &Mediator{
		config:     cfg,
		unixServer: unixServer,
		remoteAPI:  remoteAPI,
	}, nil
}

// Run starts the mediator's Unix socket server.
// The mediator will automatically exit when the browser closes stdin
// (detected in RemoteAPI.sendCommand).
func (m *Mediator) Run() error {
	return m.unixServer.Start()
}

// Shutdown gracefully shuts down the mediator and cleans up resources
func (m *Mediator) Shutdown() error {
	if m.unixServer != nil {
		return m.unixServer.Shutdown()
	}
	return nil
}

