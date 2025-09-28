package mediator

import (
	"log"
	"os"

	"github.com/tabctl/tabctl/internal/config"
)

// Mediator represents a simplified browser tab mediator
type Mediator struct {
	config     *config.MediatorConfig
	unixServer *UnixServer
	remoteAPI  *RemoteAPI
}

// NewMediator creates a new simplified mediator
func NewMediator(cfg *config.MediatorConfig) (*Mediator, error) {
	log.Printf("Creating mediator in CLI mode")

	// Create stdio transport for browser communication
	transport := NewStdioTransport()

	// Create remote API
	remoteAPI := NewRemoteAPI(transport)

	// Create Unix socket server
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

// Run starts the mediator
func (m *Mediator) Run() error {
	log.Println("Starting mediator")

	// Start Unix socket server
	return m.unixServer.Start()
}

// Shutdown shuts down the mediator
func (m *Mediator) Shutdown() error {
	log.Println("Shutting down mediator")

	if m.unixServer != nil {
		return m.unixServer.Shutdown()
	}
	return nil
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() Transport {
	return NewStdTransport(os.Stdin, os.Stdout)
}