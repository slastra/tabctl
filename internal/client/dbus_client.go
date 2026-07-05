package client

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/dbus"
	"github.com/tabctl/tabctl/pkg/api"
	"github.com/tabctl/tabctl/pkg/types"
)

// DBusClient implements api.Client using D-Bus.
type DBusClient struct {
	client  *dbus.Client
	browser string
	prefix  string
}

func NewDBusClient(browser string) (api.Client, error) {
	dbusClient, err := dbus.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create D-Bus client: %w", err)
	}

	return &DBusClient{
		client:  dbusClient,
		browser: browser,
		prefix:  determinePrefixForBrowser(browser),
	}, nil
}

// determinePrefixForBrowser returns the user-visible tab-ID prefix for this
// browser. The lowercased browser name is used as-is so two browsers in the
// same family (e.g. Brave and Helium) get distinct prefixes and the CLI's
// tab-ID-based routing stays unambiguous.
func determinePrefixForBrowser(browser string) string {
	if browser == "" {
		return "unknown."
	}
	return strings.ToLower(browser) + "."
}

func (c *DBusClient) GetPrefix() string {
	return c.prefix
}

func (c *DBusClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

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
			ID:       c.prefix + info.ID,
			Title:    info.Title,
			URL:      info.URL,
			WindowID: windowIDFromTabID(info.ID),
			Index:    int(info.Index),
			Active:   info.Active,
			Pinned:   info.Pinned,
		}
	}

	return tabs, nil
}

// windowIDFromTabID extracts the window ID embedded in the extension's tab
// ID format "<family>.<window>.<tab>" (e.g. "c.1.123" -> 1).
func windowIDFromTabID(id string) int {
	parts := strings.Split(id, ".")
	if len(parts) != 3 {
		return 0
	}
	winID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}
	return winID
}

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

func (c *DBusClient) ActivateTab(tabID string, focused bool) error {
	ctx, cancel := config.CommandContext()
	defer cancel()

	return c.client.ActivateTab(ctx, c.browser, strings.TrimPrefix(tabID, c.prefix), focused)
}

func (c *DBusClient) OpenTab(url string) (string, error) {
	ctx, cancel := config.CommandContext()
	defer cancel()

	id, err := c.client.OpenTab(ctx, c.browser, url)
	if err != nil {
		return "", fmt.Errorf("failed to open tab via D-Bus: %w", err)
	}
	return c.prefix + id, nil
}
