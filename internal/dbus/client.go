package dbus

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
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
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) DiscoverBrowsers() ([]string, error) {
	var names []string
	obj := c.conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")
	err := obj.Call("org.freedesktop.DBus.ListNames", 0).Store(&names)
	if err != nil {
		return nil, fmt.Errorf("failed to list D-Bus names: %w", err)
	}

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

	return browsers, nil
}

func (c *Client) ListTabs(browser string) ([]TabInfo, error) {
	serviceName := ServiceName(browser)
	objectPath := ObjectPath(browser)

	obj := c.conn.Object(serviceName, objectPath)

	var tabs []TabInfo
	err := obj.Call(InterfaceBrowser+".ListTabs", 0).Store(&tabs)
	if err != nil {
		return nil, fmt.Errorf("failed to list tabs: %w", err)
	}

	return tabs, nil
}

func (c *Client) ActivateTab(browser, tabID string) error {
	serviceName := ServiceName(browser)
	objectPath := ObjectPath(browser)

	obj := c.conn.Object(serviceName, objectPath)

	var success bool
	err := obj.Call(InterfaceBrowser+".ActivateTab", 0, tabID).Store(&success)
	if err != nil {
		return fmt.Errorf("failed to activate tab: %w", err)
	}
	if !success {
		return fmt.Errorf("tab activation failed")
	}

	return nil
}

func (c *Client) CloseTab(browser, tabID string) error {
	serviceName := ServiceName(browser)
	objectPath := ObjectPath(browser)

	obj := c.conn.Object(serviceName, objectPath)

	var success bool
	err := obj.Call(InterfaceBrowser+".CloseTab", 0, tabID).Store(&success)
	if err != nil {
		return fmt.Errorf("failed to close tab: %w", err)
	}
	if !success {
		return fmt.Errorf("tab close failed")
	}

	return nil
}

func (c *Client) OpenTab(browser, url string) (string, error) {
	serviceName := ServiceName(browser)
	objectPath := ObjectPath(browser)

	obj := c.conn.Object(serviceName, objectPath)

	var tabID string
	err := obj.Call(InterfaceBrowser+".OpenTab", 0, url).Store(&tabID)
	if err != nil {
		return "", fmt.Errorf("failed to open tab: %w", err)
	}

	return tabID, nil
}