package client

import (
	"fmt"
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
// same family (e.g. Brave and Helium) get distinct prefixes.
func determinePrefixForBrowser(browser string) string {
	return strings.ToLower(browser) + "."
}

func (c *DBusClient) GetPrefix() string { return c.prefix }

func (c *DBusClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// composeID builds the user-facing "<browser>.<window>.<tab>" token.
func (c *DBusClient) composeID(windowID, tabID int32) string {
	return fmt.Sprintf("%s%d.%d", c.prefix, windowID, tabID)
}

func (c *DBusClient) ListTabs() ([]types.Tab, error) {
	ctx, cancel := config.CommandContext()
	defer cancel()

	infos, err := c.client.ListTabsWithIcons(ctx, c.browser)
	if err != nil {
		return nil, err
	}

	tabs := make([]types.Tab, len(infos))
	for i, info := range infos {
		tabs[i] = types.Tab{
			ID:         c.composeID(info.WindowID, info.TabID),
			Title:      info.Title,
			URL:        info.URL,
			WindowID:   int(info.WindowID),
			Index:      int(info.Index),
			Active:     info.Active,
			Pinned:     info.Pinned,
			FavIconURL: info.FavIconURL,
		}
	}
	return tabs, nil
}

func (c *DBusClient) CloseTabs(tabIDs []int) error {
	ctx, cancel := config.CommandContext()
	defer cancel()

	ids := make([]int32, len(tabIDs))
	for i, id := range tabIDs {
		ids[i] = int32(id)
	}
	return c.client.CloseTabs(ctx, c.browser, ids)
}

func (c *DBusClient) ActivateTab(tabID int, focused bool) error {
	ctx, cancel := config.CommandContext()
	defer cancel()

	return c.client.ActivateTab(ctx, c.browser, int32(tabID), focused)
}

func (c *DBusClient) OpenTab(url string) (types.Tab, error) {
	ctx, cancel := config.CommandContext()
	defer cancel()

	windowID, tabID, err := c.client.OpenTab(ctx, c.browser, url)
	if err != nil {
		return types.Tab{}, fmt.Errorf("failed to open tab via D-Bus: %w", err)
	}
	return types.Tab{
		ID:       c.composeID(windowID, tabID),
		URL:      url,
		WindowID: int(windowID),
	}, nil
}

func (c *DBusClient) Info() (api.Info, error) {
	ctx, cancel := config.CommandContext()
	defer cancel()

	i, err := c.client.GetInfo(ctx, c.browser)
	if err != nil {
		return api.Info{}, err
	}
	return api.Info{
		Browser:           c.browser,
		MediatorVersion:   i.MediatorVersion,
		ExtensionVersion:  i.ExtensionVersion,
		MediatorProtocol:  int(i.MediatorProtocol),
		ExtensionProtocol: int(i.ExtensionProtocol),
		Compatible:        i.Compatible,
	}, nil
}
