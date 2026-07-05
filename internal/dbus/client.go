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
	// dbus.SessionBus() returns a shared process-level connection.
	// Closing it would break all other callers. The connection is
	// cleaned up automatically when the process exits.
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

// filterBrowserNames extracts browser names from the bus name list,
// skipping the base Manager name and anything outside our namespace.
func filterBrowserNames(names []string) []string {
	var browsers []string
	prefix := ServiceNameBase + "."
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			browser := strings.TrimPrefix(name, prefix)
			if browser != "" && browser != "Manager" {
				browsers = append(browsers, browser)
			}
		}
	}
	return browsers
}

func (c *Client) ListTabs(ctx context.Context, browser string) ([]TabInfo, error) {
	serviceName := ServiceName(browser)
	objectPath := ObjectPath(browser)

	obj := c.conn.Object(serviceName, objectPath)

	var tabs []TabInfo
	err := obj.CallWithContext(ctx, InterfaceBrowser+".ListTabs", 0).Store(&tabs)
	if err != nil {
		return nil, wrapTimeoutError(err, "ListTabs")
	}

	return tabs, nil
}

func (c *Client) ActivateTab(ctx context.Context, browser, tabID string, focused bool) error {
	serviceName := ServiceName(browser)
	objectPath := ObjectPath(browser)

	obj := c.conn.Object(serviceName, objectPath)

	var success bool
	err := obj.CallWithContext(ctx, InterfaceBrowser+".ActivateTab", 0, tabID, focused).Store(&success)
	if err != nil {
		return wrapTimeoutError(err, "ActivateTab")
	}
	if !success {
		return fmt.Errorf("tab activation failed")
	}

	return nil
}

func (c *Client) CloseTab(ctx context.Context, browser, tabID string) error {
	serviceName := ServiceName(browser)
	objectPath := ObjectPath(browser)

	obj := c.conn.Object(serviceName, objectPath)

	var success bool
	err := obj.CallWithContext(ctx, InterfaceBrowser+".CloseTab", 0, tabID).Store(&success)
	if err != nil {
		return wrapTimeoutError(err, "CloseTab")
	}
	if !success {
		return fmt.Errorf("tab close failed")
	}

	return nil
}

func (c *Client) OpenTab(ctx context.Context, browser, url string) (string, error) {
	serviceName := ServiceName(browser)
	objectPath := ObjectPath(browser)

	obj := c.conn.Object(serviceName, objectPath)

	var tabID string
	err := obj.CallWithContext(ctx, InterfaceBrowser+".OpenTab", 0, url).Store(&tabID)
	if err != nil {
		return "", wrapTimeoutError(err, "OpenTab")
	}

	return tabID, nil
}
