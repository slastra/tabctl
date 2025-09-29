package dbus

import "github.com/godbus/dbus/v5"

const (
	ServiceNameBase = "dev.slastra.TabCtl"
	InterfaceBrowser = "dev.slastra.TabCtl.Browser"
	InterfaceManager = "dev.slastra.TabCtl.Manager"
)

type TabInfo struct {
	ID     string
	Title  string
	URL    string
	Index  int32
	Active bool
	Pinned bool
}

type BrowserServer interface {
	ListTabs() ([]TabInfo, *dbus.Error)
	ActivateTab(tabID string) (bool, *dbus.Error)
	CloseTab(tabID string) (bool, *dbus.Error)
	OpenTab(url string) (string, *dbus.Error)
}

type ManagerServer interface {
	ListBrowsers() ([]string, *dbus.Error)
}

func ServiceName(browser string) string {
	if browser == "" {
		return ServiceNameBase
	}
	return ServiceNameBase + "." + browser
}

func ObjectPath(browser string) dbus.ObjectPath {
	if browser == "" {
		return "/dev/slastra/TabCtl/Manager"
	}
	return dbus.ObjectPath("/dev/slastra/TabCtl/Browser/" + browser)
}