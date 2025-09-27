package mediator

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/tabctl/tabctl/internal/config"
)

// Mediator coordinates between HTTP server and browser extension
type Mediator struct {
	config    *config.MediatorConfig
	server    *Server
	transport Transport
	remoteAPI *RemoteAPI
	wg        sync.WaitGroup
	shutdown  chan struct{}
}

// New creates a new mediator
func New(cfg *config.MediatorConfig) (*Mediator, error) {
	// Create transport
	transport := NewDefaultTransport()
	if cfg.Timeout > 0 {
		transport = NewTimeoutTransport(transport, cfg.Timeout)
	}

	// Create remote API
	remoteAPI := NewRemoteAPI(transport)

	// Create HTTP server
	server := NewServer(cfg, remoteAPI)

	return &Mediator{
		config:    cfg,
		server:    server,
		transport: transport,
		remoteAPI: remoteAPI,
		shutdown:  make(chan struct{}),
	}, nil
}

// Run starts the mediator
func (m *Mediator) Run() error {
	log.Printf("Mediator starting on %s:%d", m.config.Host, m.config.Port)

	// Start HTTP server
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		if err := m.server.Start(); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start native messaging handler
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.handleNativeMessages()
	}()

	// Wait for shutdown
	<-m.shutdown

	return nil
}

// Shutdown gracefully shuts down the mediator
func (m *Mediator) Shutdown() error {
	log.Println("Mediator shutting down")

	// Signal shutdown
	close(m.shutdown)

	// Shutdown HTTP server
	if err := m.server.Shutdown(); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	// Close transport
	if err := m.transport.Close(); err != nil {
		log.Printf("Error closing transport: %v", err)
	}

	// Wait for all goroutines to finish
	m.wg.Wait()

	log.Println("Mediator shutdown complete")
	return nil
}

// handleNativeMessages handles incoming messages from the browser extension
func (m *Mediator) handleNativeMessages() {
	log.Println("Native messaging handler started")

	for {
		select {
		case <-m.shutdown:
			log.Println("Native messaging handler stopping")
			return
		default:
			// Check for incoming messages (non-blocking)
			// This is a simplified version - in production, you'd want
			// to handle browser-initiated messages properly

			// For now, just sleep to avoid busy loop
			// Real implementation would block on Recv() or use select with channels
			continue
		}
	}
}

// ShutdownWithContext performs a graceful shutdown with timeout
func (m *Mediator) ShutdownWithContext(ctx context.Context) error {
	done := make(chan error, 1)

	go func() {
		done <- m.Shutdown()
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout exceeded")
	}
}