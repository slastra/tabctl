package client

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/tabctl/tabctl/internal/utils"
	"github.com/tabctl/tabctl/pkg/api"
	"github.com/tabctl/tabctl/pkg/types"
)

// ProcessClient implements the api.Client interface by connecting to Unix sockets
type ProcessClient struct {
	prefix     string
	browser    string
	socketPath string
	cache      *ResponseCache
}

// NewProcessClient creates a new client that connects to Unix socket
func NewProcessClient(prefix, host string, port int) api.Client {
	// Determine browser and prefix based on port
	// Port 4625 = Firefox, 4626 = Chrome, 4627 = Brave/Chrome
	var browser string
	var correctPrefix string

	switch port {
	case 4625:
		browser = "firefox"
		correctPrefix = "f."
	case 4626, 4627:
		browser = "chrome"
		correctPrefix = "c."
	default:
		browser = "unknown"
		correctPrefix = prefix // Use provided prefix
	}

	// Use XDG_RUNTIME_DIR if available, otherwise /tmp
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}

	// Create socket path based on port
	socketPath := filepath.Join(runtimeDir, fmt.Sprintf("tabctl-%d.sock", port))

	return &ProcessClient{
		prefix:     correctPrefix,
		browser:    browser,
		socketPath: socketPath,
		cache:      NewResponseCache(5 * time.Minute),
	}
}

// GetPrefix returns the client prefix
func (c *ProcessClient) GetPrefix() string {
	return c.prefix
}

// GetHost returns localhost (process communication is always local)
func (c *ProcessClient) GetHost() string {
	return "localhost"
}

// GetPort returns 0 (no network port used)
func (c *ProcessClient) GetPort() int {
	return 0
}

// GetBrowser returns the browser type
func (c *ProcessClient) GetBrowser() string {
	return c.browser
}

// executeCommand connects to Unix socket and sends a command
func (c *ProcessClient) executeCommand(command string, args map[string]interface{}) (interface{}, error) {
	// Create the command to send
	cmd := map[string]interface{}{
		"name": command,
	}
	if args != nil {
		cmd["args"] = args
	}

	// Connect to Unix socket
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Unix socket %s: %w", c.socketPath, err)
	}
	defer conn.Close()

	// Send command
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	decoder := json.NewDecoder(conn)
	var response interface{}
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}


	// Check if response contains an error
	if respMap, ok := response.(map[string]interface{}); ok {
		if errMsg, hasError := respMap["error"]; hasError {
			return nil, fmt.Errorf("mediator error: %v", errMsg)
		}
	}

	return response, nil
}

// ListTabs returns a list of all tabs
func (c *ProcessClient) ListTabs() ([]types.Tab, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("list_tabs:%s", c.prefix)
	if cached := c.cache.Get(cacheKey); cached != nil {
		return cached.([]types.Tab), nil
	}

	result, err := c.executeCommand("list_tabs", nil)
	if err != nil {
		return nil, err
	}

	// Parse the result as array of strings (TSV lines)
	lines, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: %T", result)
	}

	tabs := make([]types.Tab, 0, len(lines))
	for _, line := range lines {
		lineStr, ok := line.(string)
		if !ok {
			continue
		}

		tab, err := utils.ParseTabLine(lineStr)
		if err != nil {
			continue // Skip malformed lines
		}

		// Don't filter by prefix - tabs already have correct prefixes from browser
		tabs = append(tabs, tab)
	}

	// Cache result
	c.cache.Set(cacheKey, tabs, 10*time.Second)

	return tabs, nil
}

// QueryTabs searches for tabs matching the query
func (c *ProcessClient) QueryTabs(query types.TabQuery) ([]types.Tab, error) {
	// Convert query to JSON and base64 encode it
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}
	queryStr := base64.StdEncoding.EncodeToString(queryJSON)

	result, err := c.executeCommand("query_tabs", map[string]interface{}{
		"query_info": queryStr,
	})
	if err != nil {
		return nil, err
	}

	// Parse similar to ListTabs
	lines, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format: %T", result)
	}

	tabs := make([]types.Tab, 0, len(lines))
	for _, line := range lines {
		lineStr, ok := line.(string)
		if !ok {
			continue
		}

		tab, err := utils.ParseTabLine(lineStr)
		if err != nil {
			continue // Skip malformed lines
		}

		tabs = append(tabs, tab)
	}

	return tabs, nil
}

// CloseTabs closes the specified tabs
func (c *ProcessClient) CloseTabs(tabIDs []string) error {
	_, err := c.executeCommand("close_tabs", map[string]interface{}{
		"tab_ids": tabIDs,
	})
	return err
}

// ActivateTab activates the specified tab
func (c *ProcessClient) ActivateTab(tabID string, focused bool) error {
	_, err := c.executeCommand("activate_tab", map[string]interface{}{
		"tab_id": tabID,
		"focused": focused,
	})
	return err
}

// OpenURLs opens URLs in new tabs
func (c *ProcessClient) OpenURLs(urls []string, windowID string) ([]string, error) {
	args := map[string]interface{}{
		"urls": urls,
	}
	if windowID != "" {
		args["window_id"] = windowID
	}

	result, err := c.executeCommand("open_urls", args)
	if err != nil {
		return nil, err
	}

	// Parse result as array of tab IDs
	if tabIDs, ok := result.([]interface{}); ok {
		var ids []string
		for _, id := range tabIDs {
			if idStr, ok := id.(string); ok {
				ids = append(ids, idStr)
			}
		}
		return ids, nil
	}

	return nil, nil
}

// GetActiveTabs returns currently active tabs
func (c *ProcessClient) GetActiveTabs() ([]string, error) {
	result, err := c.executeCommand("get_active_tabs", nil)
	if err != nil {
		return nil, err
	}

	// Parse result as array of tab IDs
	if tabIDs, ok := result.([]interface{}); ok {
		var ids []string
		for _, id := range tabIDs {
			if idStr, ok := id.(string); ok {
				ids = append(ids, idStr)
			}
		}
		return ids, nil
	}

	return nil, nil
}

// GetActiveTab returns the currently active tab
func (c *ProcessClient) GetActiveTab() (string, error) {
	tabs, err := c.GetActiveTabs()
	if err != nil {
		return "", err
	}
	if len(tabs) > 0 {
		return tabs[0], nil
	}
	return "", fmt.Errorf("no active tab found")
}

// Stub implementations for interface compliance
func (c *ProcessClient) MoveTabs() error {
	return fmt.Errorf("not implemented")
}

func (c *ProcessClient) UpdateTabs(updates []types.TabUpdate) error {
	return fmt.Errorf("not implemented")
}

func (c *ProcessClient) NavigateURLs(pairs []types.TabURLPair) error {
	return fmt.Errorf("not implemented")
}

func (c *ProcessClient) GetText(tabIDs []string, options types.TextOptions) ([]types.TabContent, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *ProcessClient) GetHTML(tabIDs []string, options types.TextOptions) ([]types.TabContent, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *ProcessClient) GetWords(tabIDs []string, options types.WordsOptions) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *ProcessClient) GetWindows() ([]types.Window, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *ProcessClient) GetScreenshot() (*types.Screenshot, error) {
	return nil, fmt.Errorf("not implemented")
}

// IsAvailable checks if the Unix socket exists and is connectable
func (c *ProcessClient) IsAvailable() bool {
	// Try to connect to the socket
	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Close closes the client connection (nothing to close for process client)
func (c *ProcessClient) Close() error {
	return nil
}