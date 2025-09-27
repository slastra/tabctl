package client

import (
	"fmt"

	"github.com/tabctl/tabctl/pkg/api"
	"github.com/tabctl/tabctl/pkg/types"
)

// Client implements the api.Client interface using HTTP
type Client struct {
	*HTTPClient
	prefix  string
	browser string
}

// NewClient creates a new client
func NewClient(prefix, host string, port int) api.Client {
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = FindAvailablePort(host)
		if port == 0 {
			port = 4625 // Fallback
		}
	}

	// Determine browser from port
	browser := "unknown"
	switch port {
	case 4625:
		browser = "firefox"
	case 4626:
		browser = "chrome"
	case 4627:
		browser = "chromium"
	}

	return &Client{
		HTTPClient: NewHTTPClient(host, port),
		prefix:     prefix,
		browser:    browser,
	}
}

// GetPrefix returns the client prefix
func (c *Client) GetPrefix() string {
	return c.prefix
}

// GetHost returns the client host
func (c *Client) GetHost() string {
	return c.defaultHost
}

// GetPort returns the client port
func (c *Client) GetPort() int {
	return c.defaultPort
}

// GetBrowser returns the browser name
func (c *Client) GetBrowser() string {
	return c.browser
}

// MoveTabs implements the TabAPI interface
func (c *Client) MoveTabs() error {
	// This is handled by the move command with editor integration
	return fmt.Errorf("MoveTabs requires editor integration, use move command")
}

// NavigateURLs navigates tabs to new URLs
func (c *Client) NavigateURLs(pairs []types.TabURLPair) error {
	// Convert to tab updates
	var updates []types.TabUpdate
	for _, pair := range pairs {
		updates = append(updates, types.TabUpdate{
			TabID: pair.TabID,
			URL:   pair.URL,
		})
	}
	return c.UpdateTabs(updates)
}

// GetText with options
func (c *Client) GetText(tabIDs []string, options types.TextOptions) ([]types.TabContent, error) {
	// For now, ignore options and use basic implementation
	return c.HTTPClient.GetText(tabIDs)
}

// GetHTML with options
func (c *Client) GetHTML(tabIDs []string, options types.TextOptions) ([]types.TabContent, error) {
	// For now, ignore options and use basic implementation
	return c.HTTPClient.GetHTML(tabIDs)
}

// GetWords with options
func (c *Client) GetWords(tabIDs []string, options types.WordsOptions) ([]string, error) {
	// For now, ignore tab IDs and get all words
	return c.HTTPClient.GetWords(nil)
}

// GetWindows gets all browser windows
func (c *Client) GetWindows() ([]types.Window, error) {
	// Get all tabs and organize by window
	tabs, err := c.ListTabs()
	if err != nil {
		return nil, err
	}

	windowMap := make(map[int]*types.Window)
	for _, tab := range tabs {
		if _, ok := windowMap[tab.WindowID]; !ok {
			windowMap[tab.WindowID] = &types.Window{
				ID:   tab.WindowID,
				Tabs: []types.Tab{},
			}
		}
		windowMap[tab.WindowID].Tabs = append(windowMap[tab.WindowID].Tabs, tab)
	}

	var windows []types.Window
	for _, window := range windowMap {
		windows = append(windows, *window)
	}

	return windows, nil
}

// GetActiveTab gets the currently active tab
func (c *Client) GetActiveTab() (string, error) {
	active := true
	query := types.TabQuery{
		Active: &active,
	}

	tabs, err := c.QueryTabs(query)
	if err != nil {
		return "", err
	}

	if len(tabs) == 0 {
		return "", fmt.Errorf("no active tab found")
	}

	return fmt.Sprintf("%s.%d.%d", c.prefix, tabs[0].WindowID, tabs[0].ID), nil
}

// GetActiveTabs gets all active tabs (one per window)
func (c *Client) GetActiveTabs() ([]string, error) {
	active := true
	query := types.TabQuery{
		Active: &active,
	}

	tabs, err := c.QueryTabs(query)
	if err != nil {
		return nil, err
	}

	var tabIDs []string
	for _, tab := range tabs {
		tabIDs = append(tabIDs, fmt.Sprintf("%s.%d.%d", c.prefix, tab.WindowID, tab.ID))
	}

	return tabIDs, nil
}

// GetScreenshot gets a screenshot of the active tab
func (c *Client) GetScreenshot() (*types.Screenshot, error) {
	activeTab, err := c.GetActiveTab()
	if err != nil {
		return nil, err
	}

	data, err := c.HTTPClient.GetScreenshot(activeTab)
	if err != nil {
		return nil, err
	}

	return &types.Screenshot{
		TabID: activeTab,
		Data:  data,
	}, nil
}