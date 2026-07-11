package dbus

import "github.com/godbus/dbus/v5"

const (
	ServiceNameBase  = "dev.slastra.TabCtl"
	InterfaceBrowser = "dev.slastra.TabCtl.Browser"
)

// TabInfo is a tab as it crosses the D-Bus boundary: raw browser-assigned
// numeric IDs. The CLI composes the user-facing "<browser>.<window>.<tab>"
// token from these fields.
type TabInfo struct {
	WindowID int32
	TabID    int32
	Title    string
	URL      string
	Index    int32
	Active   bool
	Pinned   bool
}

// Info reports mediator/extension versions and protocol compatibility.
type Info struct {
	MediatorVersion   string
	ExtensionVersion  string
	MediatorProtocol  int32
	ExtensionProtocol int32
	Compatible        bool
}

// BrowserServer is the D-Bus-facing method set (godbus signatures).
type BrowserServer interface {
	ListTabs() ([]TabInfo, *dbus.Error)
	ActivateTab(tabID int32, focused bool) *dbus.Error
	CloseTabs(tabIDs []int32) *dbus.Error
	OpenTab(url string) (int32, int32, *dbus.Error)
	GetInfo() (string, string, int32, int32, bool, *dbus.Error)
}

func ServiceName(browser string) string {
	return ServiceNameBase + "." + browser
}

func ObjectPath(browser string) dbus.ObjectPath {
	return dbus.ObjectPath("/dev/slastra/TabCtl/Browser/" + browser)
}
