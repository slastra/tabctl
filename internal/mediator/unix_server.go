package mediator

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/utils"
)

// UnixServer handles CLI connections via Unix domain socket.
// It receives commands from the CLI tool and forwards them
// to the browser extension through the RemoteAPI.
type UnixServer struct {
	config     *config.MediatorConfig
	remoteAPI  *RemoteAPI  // Handles forwarding to browser
	socketPath string       // Path to Unix domain socket
	listener   net.Listener // Active socket listener
}

// NewUnixServer creates a new Unix socket server.
// The socket is created in XDG_RUNTIME_DIR (or /tmp as fallback)
// with a name based on the configured port for multi-browser support.
func NewUnixServer(cfg *config.MediatorConfig, remoteAPI *RemoteAPI) (*UnixServer, error) {
	// Use XDG_RUNTIME_DIR for better security (user-specific temp dir)
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}

	// Socket name includes port for multiple browser support
	socketPath := filepath.Join(runtimeDir, fmt.Sprintf("tabctl-%d.sock", cfg.Port))

	// Check if socket already exists and is active
	if _, err := os.Stat(socketPath); err == nil {
		// Socket file exists, check if it's active
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			// Another mediator is already running on this socket
			conn.Close()
			return nil, fmt.Errorf("another mediator is already running on %s", socketPath)
		}
		// Socket exists but no one is listening, remove stale socket
		log.Printf("Removing stale socket: %s", socketPath)
		os.Remove(socketPath)
	}

	return &UnixServer{
		config:     cfg,
		remoteAPI:  remoteAPI,
		socketPath: socketPath,
	}, nil
}

// Start starts the Unix socket server and begins accepting connections.
// This method blocks until the server is shut down.
func (us *UnixServer) Start() error {
	// Create Unix domain socket
	listener, err := net.Listen("unix", us.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create unix socket at %s: %w", us.socketPath, err)
	}
	us.listener = listener

	// Restrict socket to current user only for security
	if err := os.Chmod(us.socketPath, 0600); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Accept and handle connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Log error but continue accepting other connections
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each connection concurrently
		go us.handleConnection(conn)
	}
}

// handleConnection processes a single CLI connection.
// It reads one command, processes it, and sends back the response.
func (us *UnixServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// New connection accepted

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	// Read command from CLI
	var command map[string]interface{}
	if err := decoder.Decode(&command); err != nil {
		// Failed to decode command
		encoder.Encode(map[string]interface{}{
			"error": fmt.Sprintf("Failed to decode command: %v", err),
		})
		return
	}

	// Command received

	// Process command and forward to browser
	response, err := us.processCommand(command)
	if err != nil {
		// Command failed
		encoder.Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Command succeeded

	// Send response back to CLI
	if err := encoder.Encode(response); err != nil {
		// Failed to send response
	}
}

// processCommand routes commands to the appropriate RemoteAPI method.
// It handles type conversions and argument extraction for each command.
func (us *UnixServer) processCommand(command map[string]interface{}) (interface{}, error) {
	cmdName, ok := command["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid command name")
	}

	switch cmdName {
	case "list_tabs":
		return us.remoteAPI.ListTabs()
	case "query_tabs":
		args, _ := command["args"].(map[string]interface{})
		queryInfo, _ := args["query_info"].(string)
		return us.remoteAPI.QueryTabs(queryInfo)
	case "close_tabs":
		args, _ := command["args"].(map[string]interface{})
		tabIDsRaw, _ := args["tab_ids"].([]interface{})
		var tabIDs []string
		for _, id := range tabIDsRaw {
			if idStr, ok := id.(string); ok {
				tabIDs = append(tabIDs, idStr)
			}
		}
		// Convert to comma-separated string for RemoteAPI
		tabIDsStr := strings.Join(tabIDs, ",")
		return us.remoteAPI.CloseTabs(tabIDsStr)
	case "activate_tab":
		args, _ := command["args"].(map[string]interface{})
		focused, _ := args["focused"].(bool)

		// Support both "tab_id" and "tabId" for compatibility
		var tabIDStr string
		if tabID, ok := args["tab_id"].(string); ok {
			tabIDStr = tabID
		} else if tabID, ok := args["tabId"].(string); ok {
			tabIDStr = tabID
		} else {
			return nil, fmt.Errorf("missing tab_id in activate_tab command")
		}

		// Extract numeric tab ID from full format (e.g., "f.1234.5678")
		_, _, tabIDNumStr, err := utils.ParseTabID(tabIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid tab ID format: %w", err)
		}

		tabIDInt, err := strconv.Atoi(tabIDNumStr)
		if err != nil {
			return nil, fmt.Errorf("invalid tab ID number: %w", err)
		}

		return nil, us.remoteAPI.ActivateTab(tabIDInt, focused)
	case "open_urls":
		args, _ := command["args"].(map[string]interface{})
		urlsRaw, _ := args["urls"].([]interface{})
		var urls []string
		for _, url := range urlsRaw {
			if urlStr, ok := url.(string); ok {
				urls = append(urls, urlStr)
			}
		}
		var windowID *int // Optional window ID
		return us.remoteAPI.OpenURLs(urls, windowID)
	case "get_active_tabs":
		return us.remoteAPI.GetActiveTabs()
	default:
		return nil, fmt.Errorf("unsupported command: %s", cmdName)
	}
}

// Shutdown gracefully shuts down the Unix socket server and cleans up resources
func (us *UnixServer) Shutdown() error {
	if us.listener != nil {
		us.listener.Close()
	}

	// Remove socket file to prevent stale sockets
	os.Remove(us.socketPath)

	return nil
}

// GetSocketPath returns the path to the Unix domain socket
func (us *UnixServer) GetSocketPath() string {
	return us.socketPath
}