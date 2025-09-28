package client

import (
	"github.com/tabctl/tabctl/pkg/api"
)

// NewClient creates a new process-based client
func NewClient(prefix, host string, port int) api.Client {
	// Use port for compatibility with discovery, but actual communication is via process
	if port == 0 {
		port = 4625 // Default for now
	}

	// Use the new ProcessClient instead of Unix socket
	return NewProcessClient(prefix, host, port)
}