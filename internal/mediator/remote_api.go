package mediator

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/errors"
)

// RemoteAPI handles communication with the browser extension
type RemoteAPI struct {
	transport Transport
	browser   string
}

// NewRemoteAPI creates a new remote API
func NewRemoteAPI(transport Transport) *RemoteAPI {
	return &RemoteAPI{
		transport: transport,
		browser:   config.GetBrowserName(),
	}
}

// sendCommand sends a command to the browser and returns the response
func (r *RemoteAPI) sendCommand(cmd *Command) (interface{}, error) {
	if err := r.transport.Send(cmd); err != nil {
		return nil, err
	}

	response, err := r.transport.Recv()
	if err != nil {
		return nil, err
	}

	if errMsg, ok := response["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("browser error: %s", errMsg)
	}

	return response["result"], nil
}

// ListTabs returns a list of all tabs
func (r *RemoteAPI) ListTabs() ([]string, error) {
	cmd := NewCommand(CmdListTabs, nil)
	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("ListTabs error: %v", err)
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

// QueryTabs queries tabs with the given criteria
func (r *RemoteAPI) QueryTabs(queryInfo string) ([]string, error) {
	cmd := NewCommand(CmdQueryTabs, map[string]interface{}{
		"query_info": queryInfo,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("QueryTabs error: %v", err)
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
func (r *RemoteAPI) MoveTabs(moveTriplets string) (string, error) {
	cmd := NewCommand(CmdMoveTabs, map[string]interface{}{
		"move_triplets": moveTriplets,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("MoveTabs error: %v", err)
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	if resultStr, ok := result.(string); ok {
		return resultStr, nil
	}

	return "OK", nil
}

// OpenURLs opens the given URLs
func (r *RemoteAPI) OpenURLs(urls []string, windowID *int) ([]string, error) {
	args := map[string]interface{}{
		"urls": urls,
	}
	if windowID != nil {
		args["window_id"] = *windowID
	}

	cmd := NewCommand(CmdOpenURLs, args)
	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("OpenURLs error: %v", err)
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
func (r *RemoteAPI) UpdateTabs(updates []map[string]interface{}) ([]string, error) {
	cmd := NewCommand(CmdUpdateTabs, map[string]interface{}{
		"updates": updates,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("UpdateTabs error: %v", err)
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
func (r *RemoteAPI) CloseTabs(tabIDs string) (string, error) {
	ids := strings.Split(tabIDs, ",")
	cmd := NewCommand(CmdCloseTabs, map[string]interface{}{
		"tab_ids": ids,
	})

	_, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("CloseTabs error: %v", err)
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	return "OK", nil
}

// NewTab opens a new tab with a search query
func (r *RemoteAPI) NewTab(query string) (string, error) {
	cmd := NewCommand(CmdNewTab, map[string]interface{}{
		"url": query,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("NewTab error: %v", err)
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	if tabID, ok := result.(string); ok {
		return tabID, nil
	}

	return "", errors.NewTransportError("unexpected response format", nil)
}

// ActivateTab activates the specified tab
func (r *RemoteAPI) ActivateTab(tabID int, focused bool) error {
	cmd := NewCommand(CmdActivateTab, map[string]interface{}{
		"tab_id":  tabID,
		"focused": focused,
	})

	_, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("ActivateTab error: %v", err)
		return fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	return nil
}

// GetActiveTabs returns the active tabs
func (r *RemoteAPI) GetActiveTabs() (string, error) {
	cmd := NewCommand(CmdGetActiveTabs, nil)
	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("GetActiveTabs error: %v", err)
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	if tabs, ok := result.(string); ok {
		return tabs, nil
	}

	return "", errors.NewTransportError("unexpected response format", nil)
}

// GetScreenshot captures a screenshot
func (r *RemoteAPI) GetScreenshot() (string, error) {
	cmd := NewCommand(CmdGetScreenshot, nil)
	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("GetScreenshot error: %v", err)
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	if screenshot, ok := result.(string); ok {
		return screenshot, nil
	}

	return "", errors.NewTransportError("unexpected response format", nil)
}

// GetWords extracts words from tabs
func (r *RemoteAPI) GetWords(tabID *int, matchRegex, joinWith string) ([]string, error) {
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
		log.Printf("GetWords error: %v", err)
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

// GetText extracts text from tabs
func (r *RemoteAPI) GetText(delimiterRegex, replaceWith string) ([]string, error) {
	cmd := NewCommand(CmdGetText, map[string]interface{}{
		"delimiterRegex": delimiterRegex,
		"replaceWith":    replaceWith,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("GetText error: %v", err)
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	// Convert result to string array
	if lines, ok := result.([]interface{}); ok {
		var textLines []string
		for _, line := range lines {
			if lineStr, ok := line.(string); ok {
				textLines = append(textLines, lineStr)
			}
		}
		return textLines, nil
	}

	return nil, errors.NewTransportError("unexpected response format", nil)
}

// GetHTML extracts HTML from tabs
func (r *RemoteAPI) GetHTML(delimiterRegex, replaceWith string) ([]string, error) {
	cmd := NewCommand(CmdGetHTML, map[string]interface{}{
		"delimiterRegex": delimiterRegex,
		"replaceWith":    replaceWith,
	})

	result, err := r.sendCommand(cmd)
	if err != nil {
		log.Printf("GetHTML error: %v", err)
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
func (r *RemoteAPI) GetPID() int {
	return os.Getpid()
}

// GetBrowser returns the browser name
func (r *RemoteAPI) GetBrowser() string {
	if r.browser == "" {
		// Try to get from extension
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