package mediator

import (
	"fmt"
	"log"
	"os"

	"github.com/tabctl/tabctl/internal/config"
	"golang.org/x/sys/unix"
)

// OperationMode represents the mediator operation mode
type OperationMode string

const (
	// StdioMode is used when launched by browser via native messaging
	StdioMode OperationMode = "stdio"
	// HTTPMode is used when launched standalone as HTTP server
	HTTPMode OperationMode = "http"
)

// DetectOperationMode detects whether mediator is launched by browser or standalone
func DetectOperationMode() OperationMode {
	// Check if stdin is a terminal
	// When launched by browser, stdin will be a pipe, not a terminal
	if !isTerminal(os.Stdin.Fd()) {
		return StdioMode
	}

	// Check environment variables that browsers might set
	if os.Getenv("BROWSER_NATIVE_MESSAGING") == "1" {
		return StdioMode
	}

	// Check if launched with specific flag
	for _, arg := range os.Args {
		if arg == "--stdio" {
			return StdioMode
		}
		if arg == "--http" {
			return HTTPMode
		}
	}

	// Default to HTTP mode for standalone launch
	return HTTPMode
}

// isTerminal checks if a file descriptor is a terminal
func isTerminal(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	return err == nil
}

// DualModeMediator supports both stdio and HTTP modes
type DualModeMediator struct {
	*Mediator
	mode OperationMode
}

// NewDualModeMediator creates a mediator that can operate in both modes
func NewDualModeMediator(cfg *config.MediatorConfig) (*DualModeMediator, error) {
	mode := DetectOperationMode()
	log.Printf("Mediator starting in %s mode", mode)

	// Adjust configuration based on mode
	if mode == StdioMode {
		// In stdio mode, we communicate via stdin/stdout
		cfg.StdioMode = true
		// Disable HTTP server logging to avoid polluting stdout
		log.SetOutput(os.Stderr)
	}

	mediator, err := New(cfg)
	if err != nil {
		return nil, err
	}

	return &DualModeMediator{
		Mediator: mediator,
		mode:     mode,
	}, nil
}

// Run starts the mediator in the appropriate mode
func (dm *DualModeMediator) Run() error {
	switch dm.mode {
	case StdioMode:
		return dm.runStdioMode()
	case HTTPMode:
		return dm.runHTTPMode()
	default:
		return fmt.Errorf("unknown operation mode: %s", dm.mode)
	}
}

// runStdioMode runs mediator in stdio mode for native messaging
func (dm *DualModeMediator) runStdioMode() error {
	log.Println("Running in stdio mode (browser native messaging)")

	// Create stdio transport for browser communication
	transport := NewStdioTransport()
	dm.transport = transport
	dm.remoteAPI = NewRemoteAPI(transport)

	// Don't start HTTP server in stdio mode
	// Just handle native messages directly
	return dm.handleStdioMessages()
}

// runHTTPMode runs mediator in HTTP mode as standalone server
func (dm *DualModeMediator) runHTTPMode() error {
	log.Printf("Running in HTTP mode on %s:%d", dm.config.Host, dm.config.Port)

	// Use the standard mediator Run method for HTTP mode
	return dm.Mediator.Run()
}

// handleStdioMessages handles messages in stdio mode
func (dm *DualModeMediator) handleStdioMessages() error {
	log.Println("Stdio message handler started")

	for {
		// Receive message from browser
		message, err := dm.transport.Recv()
		if err != nil {
			// EOF is expected when browser closes
			if err.Error() == "EOF" {
				log.Println("Browser closed connection")
				return nil
			}
			log.Printf("Error receiving message: %v", err)
			continue
		}

		// Process message
		response, err := dm.processMessage(message)
		if err != nil {
			log.Printf("Error processing message: %v", err)
			response = map[string]interface{}{
				"error": err.Error(),
			}
		}

		// Send response back to browser
		if err := dm.transport.Send(response); err != nil {
			log.Printf("Error sending response: %v", err)
		}
	}
}

// processMessage processes a message from the browser
func (dm *DualModeMediator) processMessage(message interface{}) (interface{}, error) {
	msg, ok := message.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid message format")
	}

	command, ok := msg["command"].(string)
	if !ok {
		return nil, fmt.Errorf("missing command field")
	}

	// For now, just forward the message to remote API
	// The RemoteAPI will handle the actual command processing
	args, _ := msg["args"].(map[string]interface{})
	cmd := &Command{
		Command: command,
		Args:    args,
	}

	return dm.remoteAPI.sendCommand(cmd)
}

// StdioTransport implements Transport for stdin/stdout communication
type StdioTransport struct {
	*StdTransport
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport() Transport {
	return &StdioTransport{
		StdTransport: &StdTransport{
			input:  os.Stdin,
			output: os.Stdout,
		},
	}
}