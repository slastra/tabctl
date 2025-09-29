package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tabctl/tabctl/internal/dbus"
	"github.com/tabctl/tabctl/pkg/api"
	"github.com/tabctl/tabctl/pkg/types"
)

// DBusClient implements api.Client interface using D-Bus
type DBusClient struct {
	client  *dbus.Client
	browser string
	prefix  string
}

// NewDBusClient creates a new D-Bus client for a specific browser
func NewDBusClient(browser string) (api.Client, error) {
	dbusClient, err := dbus.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create D-Bus client: %w", err)
	}

	// Determine prefix based on browser
	prefix := determinePrefixForBrowser(browser)

	return &DBusClient{
		client:  dbusClient,
		browser: browser,
		prefix:  prefix,
	}, nil
}

func determinePrefixForBrowser(browser string) string {
	switch strings.ToLower(browser) {
	case "firefox":
		return "f."
	case "chrome", "chromium", "brave":
		return "c."
	default:
		return "u." // unknown
	}
}

// GetPrefix returns the client prefix
func (c *DBusClient) GetPrefix() string {
	return c.prefix
}

// GetHost returns localhost (D-Bus is always local)
func (c *DBusClient) GetHost() string {
	return "localhost"
}

// GetPort returns 0 (no network port used)
func (c *DBusClient) GetPort() int {
	return 0
}

// GetBrowser returns the browser type
func (c *DBusClient) GetBrowser() string {
	return c.browser
}

// Close closes the D-Bus connection
func (c *DBusClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// ListTabs returns all tabs from the browser
func (c *DBusClient) ListTabs() ([]types.Tab, error) {
	tabInfos, err := c.client.ListTabs(c.browser)
	if err != nil {
		return nil, fmt.Errorf("failed to list tabs via D-Bus: %w", err)
	}

	tabs := make([]types.Tab, len(tabInfos))
	for i, info := range tabInfos {
		tabs[i] = types.Tab{
			ID:     info.ID,
			Title:  info.Title,
			URL:    info.URL,
			Index:  int(info.Index),
			Active: info.Active,
			Pinned: info.Pinned,
		}
	}

	return tabs, nil
}

// CloseTabs closes the specified tabs
func (c *DBusClient) CloseTabs(tabIDs []string) error {
	// D-Bus CloseTab expects comma-separated IDs
	tabIDStr := strings.Join(tabIDs, ",")
	return c.client.CloseTab(c.browser, tabIDStr)
}

// ActivateTab activates the specified tab
func (c *DBusClient) ActivateTab(tabID string, focused bool) error {
	return c.client.ActivateTab(c.browser, tabID)
}

// MoveTabs moves tabs (not implemented)
func (c *DBusClient) MoveTabs() error {
	return errors.New("MoveTabs not implemented for D-Bus client")
}

// UpdateTabs updates tabs with the given properties
func (c *DBusClient) UpdateTabs(updates []types.TabUpdate) error {
	// For now, handle URL updates and properties
	for _, update := range updates {
		if update.URL != "" {
			// Would need a new D-Bus method to update URL
			continue
		}
		// Check properties for active state
		if update.Properties != nil {
			if active, ok := update.Properties["active"].(bool); ok && active {
				if err := c.ActivateTab(update.TabID, true); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// QueryTabs filters tabs based on a query
func (c *DBusClient) QueryTabs(query types.TabQuery) ([]types.Tab, error) {
	// Get all tabs first
	tabs, err := c.ListTabs()
	if err != nil {
		return nil, err
	}

	// Filter based on query
	var filtered []types.Tab
	for _, tab := range tabs {
		if matchesQuery(tab, query) {
			filtered = append(filtered, tab)
		}
	}

	return filtered, nil
}

func matchesQuery(tab types.Tab, query types.TabQuery) bool {
	// Simple query matching implementation
	if query.Active != nil && tab.Active != *query.Active {
		return false
	}
	if query.Pinned != nil && tab.Pinned != *query.Pinned {
		return false
	}
	if query.Title != "" && !strings.Contains(strings.ToLower(tab.Title), strings.ToLower(query.Title)) {
		return false
	}
	if len(query.URL) > 0 {
		// Check if any of the URL patterns match
		matches := false
		for _, urlPattern := range query.URL {
			if strings.Contains(strings.ToLower(tab.URL), strings.ToLower(urlPattern)) {
				matches = true
				break
			}
		}
		if !matches {
			return false
		}
	}
	return true
}

// NavigateURLs navigates tabs to new URLs
func (c *DBusClient) NavigateURLs(pairs []types.TabURLPair) error {
	// This would require a new D-Bus method
	return errors.New("NavigateURLs not implemented for D-Bus client")
}

// GetText gets text content from tabs
func (c *DBusClient) GetText(tabIDs []string, options types.TextOptions) ([]types.TabContent, error) {
	return nil, errors.New("GetText not implemented for D-Bus client")
}

// GetHTML gets HTML content from tabs
func (c *DBusClient) GetHTML(tabIDs []string, options types.TextOptions) ([]types.TabContent, error) {
	return nil, errors.New("GetHTML not implemented for D-Bus client")
}

// GetWords gets words from tabs
func (c *DBusClient) GetWords(tabIDs []string, options types.WordsOptions) ([]string, error) {
	return nil, errors.New("GetWords not implemented for D-Bus client")
}

// GetWindows returns all windows
func (c *DBusClient) GetWindows() ([]types.Window, error) {
	// Get all tabs and group by window
	tabs, err := c.ListTabs()
	if err != nil {
		return nil, err
	}

	// Group tabs by window ID
	windowMap := make(map[string][]types.Tab)
	for _, tab := range tabs {
		// Extract window ID from tab ID (e.g., "c.1.123" -> "1")
		parts := strings.Split(tab.ID, ".")
		if len(parts) >= 2 {
			windowID := parts[1]
			windowMap[windowID] = append(windowMap[windowID], tab)
		}
	}

	// Convert to Window types
	var windows []types.Window
	for windowID, windowTabs := range windowMap {
		winID, _ := strconv.Atoi(windowID)
		windows = append(windows, types.Window{
			ID:       winID,
			TabCount: len(windowTabs),
		})
	}

	return windows, nil
}

// GetActiveTab returns the ID of the active tab
func (c *DBusClient) GetActiveTab() (string, error) {
	tabs, err := c.ListTabs()
	if err != nil {
		return "", err
	}

	for _, tab := range tabs {
		if tab.Active {
			return tab.ID, nil
		}
	}

	return "", errors.New("no active tab found")
}

// GetActiveTabs returns all active tabs (one per window)
func (c *DBusClient) GetActiveTabs() ([]string, error) {
	tabs, err := c.ListTabs()
	if err != nil {
		return nil, err
	}

	var activeTabs []string
	for _, tab := range tabs {
		if tab.Active {
			activeTabs = append(activeTabs, tab.ID)
		}
	}

	return activeTabs, nil
}

// OpenURLs opens new tabs with the given URLs
func (c *DBusClient) OpenURLs(urls []string, windowID string) ([]string, error) {
	var tabIDs []string

	for _, url := range urls {
		tabID, err := c.client.OpenTab(c.browser, url)
		if err != nil {
			return tabIDs, fmt.Errorf("failed to open URL %s: %w", url, err)
		}
		tabIDs = append(tabIDs, tabID)
	}

	return tabIDs, nil
}

// RemoveDuplicates removes duplicate tabs
func (c *DBusClient) RemoveDuplicates() error {
	return errors.New("RemoveDuplicates not implemented for D-Bus client")
}

// GetScreenshot gets a screenshot (not implemented for D-Bus)
func (c *DBusClient) GetScreenshot() (*types.Screenshot, error) {
	return nil, errors.New("GetScreenshot not implemented for D-Bus client")
}

// GetClient returns the underlying D-Bus client (for testing)
func (c *DBusClient) GetClient() *dbus.Client {
	return c.client
}

// DiscoverDBusBrowsers discovers all browsers available on D-Bus
func DiscoverDBusBrowsers() ([]string, error) {
	client, err := dbus.NewClient()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return client.DiscoverBrowsers()
}