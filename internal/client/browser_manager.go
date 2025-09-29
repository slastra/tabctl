package client

import (
	"fmt"
	"strings"

	"github.com/tabctl/tabctl/pkg/api"
	"github.com/tabctl/tabctl/pkg/types"
)

// BrowserManager manages multiple D-Bus browser clients
type BrowserManager struct {
	clients []api.Client
}

// NewBrowserManager creates a new manager that discovers all browsers on D-Bus
func NewBrowserManager(targetBrowser string) *BrowserManager {
	mediators := DiscoverMediators()
	clients := make([]api.Client, 0, len(mediators))

	for _, mediator := range mediators {
		// Filter by target browser if specified
		if targetBrowser != "" && !strings.EqualFold(mediator.Browser, targetBrowser) {
			continue
		}

		client, err := NewDBusClient(mediator.Browser)
		if err != nil {
			// Skip failed clients
			continue
		}

		clients = append(clients, client)
	}

	return &BrowserManager{
		clients: clients,
	}
}

// GetClients returns all available clients
func (bm *BrowserManager) GetClients() []api.Client {
	return bm.clients
}

// ListAllTabs lists tabs from all browsers
func (bm *BrowserManager) ListAllTabs() ([]types.Tab, error) {
	if len(bm.clients) == 0 {
		return nil, fmt.Errorf("no browsers found on D-Bus")
	}

	var allTabs []types.Tab
	var lastErr error

	for _, client := range bm.clients {
		tabs, err := client.ListTabs()
		if err != nil {
			lastErr = err
			continue
		}
		allTabs = append(allTabs, tabs...)
	}

	if len(allTabs) == 0 && lastErr != nil {
		return nil, fmt.Errorf("failed to list tabs: %w", lastErr)
	}

	return allTabs, nil
}

// CloseTabs closes tabs by ID
func (bm *BrowserManager) CloseTabs(tabIDs []string) error {
	if len(bm.clients) == 0 {
		return fmt.Errorf("no browsers found on D-Bus")
	}

	// Group tab IDs by prefix to route to correct browser
	clientTabs := make(map[string][]string)
	for _, tabID := range tabIDs {
		// Extract prefix (e.g., "c." or "f.")
		parts := strings.SplitN(tabID, ".", 2)
		if len(parts) > 0 {
			prefix := parts[0] + "."
			clientTabs[prefix] = append(clientTabs[prefix], tabID)
		}
	}

	var lastErr error
	for _, client := range bm.clients {
		prefix := client.GetPrefix()
		tabs, ok := clientTabs[prefix]
		if !ok || len(tabs) == 0 {
			continue
		}

		if err := client.CloseTabs(tabs); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// ActivateTab activates a specific tab
func (bm *BrowserManager) ActivateTab(tabID string) error {
	if len(bm.clients) == 0 {
		return fmt.Errorf("no browsers found on D-Bus")
	}

	// Find the right client based on tab prefix
	for _, client := range bm.clients {
		if strings.HasPrefix(tabID, client.GetPrefix()) {
			return client.ActivateTab(tabID, true)
		}
	}

	return fmt.Errorf("no client found for tab %s", tabID)
}

// Close closes all clients
func (bm *BrowserManager) Close() error {
	for _, client := range bm.clients {
		client.Close()
	}
	return nil
}