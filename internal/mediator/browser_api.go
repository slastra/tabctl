package mediator

import (
	"fmt"
	"os"
	"strings"

	"github.com/tabctl/tabctl/internal/errors"
)

// BrowserAPI handles communication with the browser extension
type BrowserAPI struct {
	transport Transport
	browser   string
}

// NewBrowserAPI creates a new browser API with the specified browser name
func NewBrowserAPI(transport Transport, browser string) *BrowserAPI {
	return &BrowserAPI{
		transport: transport,
		browser:   browser,
	}
}

// sendCommand sends a command to the browser and returns the response.
// If the browser has disconnected (stdin closed), it exits the process.
func (r *BrowserAPI) sendCommand(cmd *Command) (interface{}, error) {
	// Sending command to browser
	if err := r.transport.Send(cmd); err != nil {
		// Check if stdin/stdout is closed (browser disconnected)
		if r.isConnectionClosed(err) {
			// Browser connection closed during send
			os.Exit(0) // Clean exit when browser disconnects
		}
		return nil, err
	}

	response, err := r.transport.Recv()
	if err != nil {
		// Check if stdin is closed (browser disconnected)
		if r.isConnectionClosed(err) {
			// Browser connection closed during recv
			os.Exit(0) // Clean exit when browser disconnects
		}
		return nil, err
	}

	// Received response from browser

	if errMsg, ok := response["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("browser error: %s", errMsg)
	}

	result := response["result"]
	// Result extracted
	return result, nil
}

// isConnectionClosed checks if an error indicates the browser connection is closed
func (r *BrowserAPI) isConnectionClosed(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "connection closed") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "EOF")
}

// ListTabs returns a list of all tabs
func (r *BrowserAPI) ListTabs() ([]string, error) {
	cmd := NewCommand(CmdListTabs, nil)
	result, err := r.sendCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	// Processing tab list

	// Convert result to string array
	if tabs, ok := result.([]interface{}); ok {
		// Converting tabs
		var lines []string
		for _, tab := range tabs {
			// Processing tab
			if tabStr, ok := tab.(string); ok {
				lines = append(lines, tabStr)
			} else {
				// Conversion failed
			}
		}
		// Tab list ready
		return lines, nil
	}

	// Unexpected result format
	return nil, errors.NewTransportError("unexpected response format", nil)
}

// QueryTabs queries tabs with the given criteria
func (r *BrowserAPI) QueryTabs(queryInfo string) ([]string, error) {
	cmd := NewCommand(CmdQueryTabs, map[string]interface{}{
		"query_info": queryInfo,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	// Convert result to string array
	if tabs, ok := result.([]interface{}); ok {
		var lines []string
		for _, tab := range tabs {
			if tabStr, ok := tab.(string); ok {
				lines = append(lines, tabStr)
			}
		}
		return lines, nil
	}

	return nil, errors.NewTransportError("unexpected response format", nil)
}

// MoveTabs moves tabs according to the given triplets
func (r *BrowserAPI) MoveTabs(moveTriplets string) (string, error) {
	cmd := NewCommand(CmdMoveTabs, map[string]interface{}{
		"move_triplets": moveTriplets,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	if resultStr, ok := result.(string); ok {
		return resultStr, nil
	}

	return "OK", nil
}

// OpenURLs opens the given URLs
func (r *BrowserAPI) OpenURLs(urls []string, windowID *int) ([]string, error) {
	args := map[string]interface{}{
		"urls": urls,
	}
	if windowID != nil {
		args["window_id"] = *windowID
	}

	cmd := NewCommand(CmdOpenURLs, args)
	result, err := r.sendCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	// Convert result to string array
	if ids, ok := result.([]interface{}); ok {
		var lines []string
		for _, id := range ids {
			if idStr, ok := id.(string); ok {
				lines = append(lines, idStr)
			}
		}
		return lines, nil
	}

	return nil, errors.NewTransportError("unexpected response format", nil)
}

// UpdateTabs updates tabs with the given properties
func (r *BrowserAPI) UpdateTabs(updates []map[string]interface{}) ([]string, error) {
	cmd := NewCommand(CmdUpdateTabs, map[string]interface{}{
		"updates": updates,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	// Convert result to string array
	if results, ok := result.([]interface{}); ok {
		var lines []string
		for _, res := range results {
			if resStr, ok := res.(string); ok {
				lines = append(lines, resStr)
			}
		}
		return lines, nil
	}

	return nil, errors.NewTransportError("unexpected response format", nil)
}

// CloseTabs closes the specified tabs
func (r *BrowserAPI) CloseTabs(tabIDs string) (string, error) {
	ids := strings.Split(tabIDs, ",")
	cmd := NewCommand(CmdCloseTabs, map[string]interface{}{
		"tab_ids": ids,
	})

	_, err := r.sendCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	return "OK", nil
}

// NewTab opens a new tab with a search query
func (r *BrowserAPI) NewTab(query string) (string, error) {
	cmd := NewCommand(CmdNewTab, map[string]interface{}{
		"url": query,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	if tabID, ok := result.(string); ok {
		return tabID, nil
	}

	return "", errors.NewTransportError("unexpected response format", nil)
}

// ActivateTab activates the specified tab
func (r *BrowserAPI) ActivateTab(tabID int, focused bool) error {
	cmd := NewCommand(CmdActivateTab, map[string]interface{}{
		"tab_id":  tabID,
		"focused": focused,
	})

	_, err := r.sendCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	return nil
}

// GetActiveTabs returns the active tabs
func (r *BrowserAPI) GetActiveTabs() (string, error) {
	cmd := NewCommand(CmdGetActiveTabs, nil)
	result, err := r.sendCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	if tabs, ok := result.(string); ok {
		return tabs, nil
	}

	return "", errors.NewTransportError("unexpected response format", nil)
}

// GetScreenshot captures a screenshot
func (r *BrowserAPI) GetScreenshot() (string, error) {
	cmd := NewCommand(CmdGetScreenshot, nil)
	result, err := r.sendCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	if screenshot, ok := result.(string); ok {
		return screenshot, nil
	}

	return "", errors.NewTransportError("unexpected response format", nil)
}

// GetWords extracts words from tabs
func (r *BrowserAPI) GetWords(tabID *int, matchRegex, joinWith string) ([]string, error) {
	args := map[string]interface{}{
		"matchRegex": matchRegex,
		"joinWith":   joinWith,
	}
	if tabID != nil {
		args["tabId"] = *tabID
	}

	cmd := NewCommand(CmdGetWords, args)
	result, err := r.sendCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	// Convert result to string array
	if words, ok := result.([]interface{}); ok {
		var lines []string
		for _, word := range words {
			if wordStr, ok := word.(string); ok {
				lines = append(lines, wordStr)
			}
		}
		return lines, nil
	}

	return nil, errors.NewTransportError("unexpected response format", nil)
}

// GetText extracts text content from tabs.
// The delimiter regex splits the text and replaceWith joins it back.
func (r *BrowserAPI) GetText(delimiterRegex, replaceWith string) ([]string, error) {
	cmd := NewCommand(CmdGetText, map[string]interface{}{
		"delimiterRegex": delimiterRegex,
		"replaceWith":    replaceWith,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text: %w", err)
	}

	return r.parseStringArray(result, "get text")
}

// GetHTML extracts HTML from tabs
func (r *BrowserAPI) GetHTML(delimiterRegex, replaceWith string) ([]string, error) {
	cmd := NewCommand(CmdGetHTML, map[string]interface{}{
		"delimiterRegex": delimiterRegex,
		"replaceWith":    replaceWith,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	// Convert result to string array
	if lines, ok := result.([]interface{}); ok {
		var htmlLines []string
		for _, line := range lines {
			if lineStr, ok := line.(string); ok {
				htmlLines = append(htmlLines, lineStr)
			}
		}
		return htmlLines, nil
	}

	return nil, errors.NewTransportError("unexpected response format", nil)
}

// GetPID returns the mediator process ID
func (r *BrowserAPI) GetPID() int {
	return os.Getpid()
}

// GetBrowser returns the detected browser name.
// It queries the extension if not already known.
func (r *BrowserAPI) GetBrowser() string {
	if r.browser == "" {
		// Query browser from extension
		cmd := NewCommand(CmdGetBrowser, nil)
		result, err := r.sendCommand(cmd)
		if err == nil {
			if browser, ok := result.(string); ok {
				r.browser = browser
			}
		}
	}

	if r.browser == "" {
		return "unknown"
	}
	return r.browser
}

// parseStringArray converts an interface{} result to []string.
// Used to reduce code duplication in response parsing.
func (r *BrowserAPI) parseStringArray(result interface{}, operation string) ([]string, error) {
	if items, ok := result.([]interface{}); ok {
		var lines []string
		for _, item := range items {
			if str, ok := item.(string); ok {
				lines = append(lines, str)
			}
		}
		return lines, nil
	}

	return nil, errors.NewTransportError(fmt.Sprintf("unexpected response format for %s", operation), nil)
}