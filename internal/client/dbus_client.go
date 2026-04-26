package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tabctl/tabctl/internal/config"
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

// determinePrefixForBrowser returns the user-visible prefix for tab IDs from
// this browser. The lowercased browser name is used as-is so two browsers in
// the same family (e.g. Brave and Helium) get distinct prefixes and the
// CLI's tab-ID-based routing stays unambiguous.
func determinePrefixForBrowser(browser string) string {
	if browser == "" {
		return "unknown."
	}
	return strings.ToLower(browser) + "."
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
	ctx, cancel := config.CommandContext()
	defer cancel()

	tabInfos, err := c.client.ListTabs(ctx, c.browser)
	if err != nil {
		return nil, fmt.Errorf("failed to list tabs via D-Bus: %w", err)
	}

	tabs := make([]types.Tab, len(tabInfos))
	for i, info := range tabInfos {
		tabs[i] = types.Tab{
			ID:     c.prefix + info.ID,
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
	ctx, cancel := config.CommandContext()
	defer cancel()

	// Strip the browser-name prefix before forwarding: the mediator and the
	// JS extension speak the original "c.win.tab" / "f.win.tab" format.
	stripped := make([]string, len(tabIDs))
	for i, id := range tabIDs {
		stripped[i] = strings.TrimPrefix(id, c.prefix)
	}
	return c.client.CloseTab(ctx, c.browser, strings.Join(stripped, ","))
}

// ActivateTab activates the specified tab
func (c *DBusClient) ActivateTab(tabID string, focused bool) error {
	ctx, cancel := config.CommandContext()
	defer cancel()

	return c.client.ActivateTab(ctx, c.browser, strings.TrimPrefix(tabID, c.prefix), focused)
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

	// Group tabs by window ID. Tab IDs look like "<browser>.<family>.<win>.<tab>"
	// (e.g. "helium.c.1.123"); the window component is the second-to-last segment.
	windowMap := make(map[string][]types.Tab)
	for _, tab := range tabs {
		parts := strings.Split(tab.ID, ".")
		if len(parts) < 2 {
			continue
		}
		windowID := parts[len(parts)-2]
		windowMap[windowID] = append(windowMap[windowID], tab)
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
		ctx, cancel := config.CommandContext()
		tabID, err := c.client.OpenTab(ctx, c.browser, url)
		cancel()
		if err != nil {
			return tabIDs, fmt.Errorf("failed to open URL %s: %w", url, err)
		}
		tabIDs = append(tabIDs, c.prefix+tabID)
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