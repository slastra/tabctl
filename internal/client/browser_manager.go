package client

import (
	"errors"
	"fmt"
	"strconv"
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

// BrowserManager manages multiple D-Bus browser clients.
type BrowserManager struct {
	clients []api.Client
}

// NewBrowserManager discovers browsers on D-Bus and returns an error when
// discovery fails or no usable browser remains, so callers can distinguish
// a broken bus from an empty one.
func NewBrowserManager(targetBrowser string) (*BrowserManager, error) {
	mediators, err := discoverMediators()
	if err != nil {
		return nil, err
	}

	clients := make([]api.Client, 0, len(mediators))
	var clientErrs []error

	for _, mediator := range mediators {
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

// GetClients returns all available clients.
func (bm *BrowserManager) GetClients() []api.Client { return bm.clients }

// numericTabID extracts the browser-assigned numeric tab ID (the last
// dot-separated segment) from a user-facing "<browser>.<window>.<tab>" token.
func numericTabID(userID string) (int, error) {
	parts := strings.Split(userID, ".")
	return strconv.Atoi(parts[len(parts)-1])
}

// routePrefix returns the "<browser>." routing prefix of a user-facing ID.
func routePrefix(userID string) string {
	parts := strings.SplitN(userID, ".", 2)
	return parts[0] + "."
}

// ListAllTabs lists tabs from all browsers.
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

// CloseTabs closes tabs by user-facing ID, returning the number actually
// dispatched. IDs whose prefix matches no connected browser (or that don't
// parse) are reported as an error rather than silently dropped.
func (bm *BrowserManager) CloseTabs(userIDs []string) (int, error) {
	// Group numeric tab IDs by routing prefix.
	byPrefix := make(map[string][]int)
	var errs []error
	for _, uid := range userIDs {
		n, err := numericTabID(uid)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid tab ID %q", uid))
			continue
		}
		byPrefix[routePrefix(uid)] = append(byPrefix[routePrefix(uid)], n)
	}

	closed := 0
	for _, client := range bm.clients {
		ids, ok := byPrefix[client.GetPrefix()]
		if !ok || len(ids) == 0 {
			continue
		}
		delete(byPrefix, client.GetPrefix())
		if err := client.CloseTabs(ids); err != nil {
			errs = append(errs, err)
			continue
		}
		closed += len(ids)
	}

	// Any prefixes left matched no connected browser.
	for prefix := range byPrefix {
		errs = append(errs, fmt.Errorf("no connected browser for tab ID prefix %q", prefix))
	}

	return closed, errors.Join(errs...)
}

// ActivateTab activates a specific tab by user-facing ID.
func (bm *BrowserManager) ActivateTab(userID string, focused bool) error {
	for _, client := range bm.clients {
		if strings.HasPrefix(userID, client.GetPrefix()) {
			n, err := numericTabID(userID)
			if err != nil {
				return fmt.Errorf("invalid tab ID %q", userID)
			}
			return client.ActivateTab(n, focused)
		}
	}
	return fmt.Errorf("no connected browser for tab %s", userID)
}

// OpenURLs opens the given URLs in the connected browser and returns the new
// user-facing tab IDs. With more than one browser connected, the caller must
// narrow the target with --browser.
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
		tab, err := client.OpenTab(url)
		if err != nil {
			return ids, fmt.Errorf("failed to open %s: %w", url, err)
		}
		ids = append(ids, tab.ID)
	}
	return ids, nil
}

// Statuses returns version/compatibility info for every connected browser.
func (bm *BrowserManager) Statuses() []api.Info {
	infos := make([]api.Info, 0, len(bm.clients))
	for _, client := range bm.clients {
		info, err := client.Info()
		if err != nil {
			info = api.Info{Browser: strings.TrimSuffix(client.GetPrefix(), ".")}
		}
		infos = append(infos, info)
	}
	return infos
}

// Close closes all clients.
func (bm *BrowserManager) Close() error {
	for _, client := range bm.clients {
		client.Close()
	}
	return nil
}
