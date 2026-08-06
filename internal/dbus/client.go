package dbus

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/tabctl/tabctl/internal/config"
	tabErrors "github.com/tabctl/tabctl/internal/errors"
)

type Client struct {
	conn *dbus.Conn
}

func NewClient() (*Client, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session bus: %w", err)
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() error {
	// dbus.SessionBus() returns a shared process-level connection; closing
	// it would break other callers. It is cleaned up on process exit.
	return nil
}

// wrapTimeoutError wraps context.DeadlineExceeded as a TimeoutError.
func wrapTimeoutError(err error, operation string) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return tabErrors.NewTimeoutError(operation, config.TransportTimeout.String())
	}
	return err
}

func (c *Client) DiscoverBrowsers(ctx context.Context) ([]string, error) {
	var names []string
	obj := c.conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")
	err := obj.CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		return nil, wrapTimeoutError(err, "DiscoverBrowsers")
	}
	return filterBrowserNames(names), nil
}

// filterBrowserNames extracts browser names from the bus name list.
func filterBrowserNames(names []string) []string {
	var browsers []string
	prefix := ServiceNameBase + "."
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			browser := strings.TrimPrefix(name, prefix)
			if browser != "" {
				browsers = append(browsers, browser)
			}
		}
	}
	return browsers
}

func (c *Client) obj(browser string) dbus.BusObject {
	return c.conn.Object(ServiceName(browser), ObjectPath(browser))
}

func (c *Client) ListTabs(ctx context.Context, browser string) ([]TabInfo, error) {
	var tabs []TabInfo
	err := c.obj(browser).CallWithContext(ctx, InterfaceBrowser+".ListTabs", 0).Store(&tabs)
	if err != nil {
		return nil, wrapTimeoutError(err, "ListTabs")
	}
	return tabs, nil
}

// ListTabsWithIcons calls the richer method. The CLI and the mediator ship in
// one package and are always the same build, so there is no need to fall back
// to ListTabs when the method is missing.
func (c *Client) ListTabsWithIcons(ctx context.Context, browser string) ([]TabInfoWithIcon, error) {
	var tabs []TabInfoWithIcon
	err := c.obj(browser).CallWithContext(ctx, InterfaceBrowser+".ListTabsWithIcons", 0).Store(&tabs)
	if err != nil {
		return nil, wrapTimeoutError(err, "ListTabsWithIcons")
	}
	return tabs, nil
}

func (c *Client) ActivateTab(ctx context.Context, browser string, tabID int32, focused bool) error {
	call := c.obj(browser).CallWithContext(ctx, InterfaceBrowser+".ActivateTab", 0, tabID, focused)
	if call.Err != nil {
		return wrapTimeoutError(call.Err, "ActivateTab")
	}
	return nil
}

func (c *Client) CloseTabs(ctx context.Context, browser string, tabIDs []int32) error {
	call := c.obj(browser).CallWithContext(ctx, InterfaceBrowser+".CloseTabs", 0, tabIDs)
	if call.Err != nil {
		return wrapTimeoutError(call.Err, "CloseTabs")
	}
	return nil
}

func (c *Client) OpenTab(ctx context.Context, browser, url string) (windowID, tabID int32, err error) {
	err = c.obj(browser).CallWithContext(ctx, InterfaceBrowser+".OpenTab", 0, url).Store(&windowID, &tabID)
	if err != nil {
		return 0, 0, wrapTimeoutError(err, "OpenTab")
	}
	return windowID, tabID, nil
}

func (c *Client) GetInfo(ctx context.Context, browser string) (Info, error) {
	var i Info
	err := c.obj(browser).CallWithContext(ctx, InterfaceBrowser+".GetInfo", 0).Store(
		&i.MediatorVersion, &i.ExtensionVersion, &i.MediatorProtocol, &i.ExtensionProtocol, &i.Compatible)
	if err != nil {
		return Info{}, wrapTimeoutError(err, "GetInfo")
	}
	return i, nil
}
