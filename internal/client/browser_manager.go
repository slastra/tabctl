package client

import (
	"errors"
	"fmt"
	"strings"

	"github.com/tabctl/tabctl/pkg/api"
	"github.com/tabctl/tabctl/pkg/types"
)

// Test seams: replaced in unit tests to exercise constructor error paths
// without a live D-Bus session.
var (
	discoverMediators = DiscoverMediators
	newDBusClient     = NewDBusClient
)

// BrowserManager manages multiple D-Bus browser clients
type BrowserManager struct {
	clients []api.Client
}

// NewBrowserManager creates a new manager that discovers all browsers on
// D-Bus. It returns an error when discovery fails or when no usable browser
// remains, so callers can distinguish a broken bus from an empty one.
func NewBrowserManager(targetBrowser string) (*BrowserManager, error) {
	mediators, err := discoverMediators()
	if err != nil {
		return nil, err
	}

	clients := make([]api.Client, 0, len(mediators))
	var clientErrs []error

	for _, mediator := range mediators {
		// Filter by target browser if specified
		if targetBrowser != "" && !strings.EqualFold(mediator.Browser, targetBrowser) {
			continue
		}

		client, err := newDBusClient(mediator.Browser)
		if err != nil {
			clientErrs = append(clientErrs, fmt.Errorf("%s: %w", mediator.Browser, err))
			continue
		}

		clients = append(clients, client)
	}

	if len(clients) == 0 {
		if err := errors.Join(clientErrs...); err != nil {
			return nil, fmt.Errorf("failed to connect to discovered browsers: %w", err)
		}
		if targetBrowser != "" && len(mediators) > 0 {
			available := make([]string, len(mediators))
			for i, m := range mediators {
				available[i] = m.Browser
			}
			return nil, fmt.Errorf("no mediator found for browser %q (available: %s)",
				targetBrowser, strings.Join(available, ", "))
		}
		return nil, fmt.Errorf("no browsers found on D-Bus (is the tabctl extension installed and the browser running?)")
	}

	return &BrowserManager{clients: clients}, nil
}

// GetClients returns all available clients
func (bm *BrowserManager) GetClients() []api.Client {
	return bm.clients
}

// ListAllTabs lists tabs from all browsers
func (bm *BrowserManager) ListAllTabs() ([]types.Tab, error) {
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

// CloseTabs closes tabs by ID. It returns the number of tabs actually
// dispatched to a browser; IDs whose prefix matches no connected browser
// are reported as an error instead of being silently dropped.
func (bm *BrowserManager) CloseTabs(tabIDs []string) (int, error) {
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

	closed := 0
	var errs []error

	for _, client := range bm.clients {
		prefix := client.GetPrefix()
		tabs, ok := clientTabs[prefix]
		if !ok || len(tabs) == 0 {
			continue
		}
		delete(clientTabs, prefix)

		if err := client.CloseTabs(tabs); err != nil {
			errs = append(errs, err)
			continue
		}
		closed += len(tabs)
	}

	// Anything left in the map matched no connected browser
	if len(clientTabs) > 0 {
		var unroutable []string
		for _, tabs := range clientTabs {
			unroutable = append(unroutable, tabs...)
		}
		errs = append(errs, fmt.Errorf("no connected browser for tab ID(s): %s",
			strings.Join(unroutable, ", ")))
	}

	return closed, errors.Join(errs...)
}

// ActivateTab activates a specific tab
func (bm *BrowserManager) ActivateTab(tabID string, focused bool) error {
	// Find the right client based on tab prefix
	for _, client := range bm.clients {
		if strings.HasPrefix(tabID, client.GetPrefix()) {
			return client.ActivateTab(tabID, focused)
		}
	}

	return fmt.Errorf("no client found for tab %s", tabID)
}

// OpenURLs opens the given URLs in the connected browser and returns the
// new tab IDs. With more than one browser connected the target is
// ambiguous, so the caller must narrow it with --browser first.
func (bm *BrowserManager) OpenURLs(urls []string) ([]string, error) {
	if len(bm.clients) > 1 {
		browsers := make([]string, len(bm.clients))
		for i, c := range bm.clients {
			browsers[i] = strings.TrimSuffix(c.GetPrefix(), ".")
		}
		return nil, fmt.Errorf("multiple browsers connected (%s); use --browser to choose",
			strings.Join(browsers, ", "))
	}

	client := bm.clients[0]
	ids := make([]string, 0, len(urls))
	for _, url := range urls {
		id, err := client.OpenTab(url)
		if err != nil {
			return ids, fmt.Errorf("failed to open %s: %w", url, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// Close closes all clients
func (bm *BrowserManager) Close() error {
	for _, client := range bm.clients {
		client.Close()
	}
	return nil
}
