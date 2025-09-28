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

// UnixServer represents a Unix socket server for the mediator
type UnixServer struct {
	config     *config.MediatorConfig
	remoteAPI  *RemoteAPI
	socketPath string
	listener   net.Listener
}

// NewUnixServer creates a new Unix socket server
func NewUnixServer(cfg *config.MediatorConfig, remoteAPI *RemoteAPI) (*UnixServer, error) {
	// Use XDG_RUNTIME_DIR if available, otherwise /tmp
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}

	// Create socket path based on port for compatibility
	socketPath := filepath.Join(runtimeDir, fmt.Sprintf("tabctl-%d.sock", cfg.Port))

	// Remove existing socket if it exists
	os.Remove(socketPath)

	return &UnixServer{
		config:     cfg,
		remoteAPI:  remoteAPI,
		socketPath: socketPath,
	}, nil
}

// Start starts the Unix socket server
func (us *UnixServer) Start() error {
	// Create Unix socket listener
	listener, err := net.Listen("unix", us.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create unix socket: %w", err)
	}
	us.listener = listener

	// Set socket permissions to be user-only
	if err := os.Chmod(us.socketPath, 0600); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	log.Printf("Starting Unix socket server on %s", us.socketPath)

	// Accept connections and handle them
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		go us.handleConnection(conn)
	}
}

// handleConnection handles a single Unix socket connection
func (us *UnixServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	log.Printf("New Unix socket connection")

	// Read JSON command from connection
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	var command map[string]interface{}
	if err := decoder.Decode(&command); err != nil {
		log.Printf("Failed to decode command: %v", err)
		encoder.Encode(map[string]interface{}{
			"error": fmt.Sprintf("Failed to decode command: %v", err),
		})
		return
	}

	log.Printf("Received Unix socket command: %v", command)

	// Process the command and get response
	response, err := us.processCommand(command)
	if err != nil {
		log.Printf("Failed to process command: %v", err)
		encoder.Encode(map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Send response back
	if err := encoder.Encode(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

// processCommand processes a command and returns the response
func (us *UnixServer) processCommand(command map[string]interface{}) (interface{}, error) {
	cmdName, ok := command["name"].(string)
	if !ok {
		return nil, fmt.Errorf("missing command name")
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
		tabIDsStr := strings.Join(tabIDs, ",") // Convert to comma-separated string expected by CloseTabs
		return us.remoteAPI.CloseTabs(tabIDsStr)
	case "activate_tab":
		args, _ := command["args"].(map[string]interface{})
		focused, _ := args["focused"].(bool)

		// Get tab ID from args - try both "tab_id" and "tabId" for compatibility
		var tabIDStr string
		if tabID, ok := args["tab_id"].(string); ok {
			tabIDStr = tabID
		} else if tabID, ok := args["tabId"].(string); ok {
			tabIDStr = tabID
		} else {
			return nil, fmt.Errorf("missing tab_id in activate_tab command")
		}

		// Parse the full tab ID (e.g., "a.1874581886.1874581981") to extract just the tab number
		_, _, tabIDNumStr, err := utils.ParseTabID(tabIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid tab ID format: %w", err)
		}

		// Convert to integer
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
		windowID := (*int)(nil) // No window ID for now
		return us.remoteAPI.OpenURLs(urls, windowID)
	case "get_active_tabs":
		return us.remoteAPI.GetActiveTabs()
	default:
		return nil, fmt.Errorf("unsupported command: %s", cmdName)
	}
}

// Shutdown gracefully shuts down the Unix socket server
func (us *UnixServer) Shutdown() error {
	log.Printf("Shutting down Unix socket server")

	if us.listener != nil {
		us.listener.Close()
	}

	// Clean up socket file
	os.Remove(us.socketPath)

	return nil
}

// GetSocketPath returns the Unix socket path
func (us *UnixServer) GetSocketPath() string {
	return us.socketPath
}