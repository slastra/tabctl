package dbus

import (
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

const (
	ServiceNameBase  = "dev.slastra.TabCtl"
	InterfaceBrowser = "dev.slastra.TabCtl.Browser"

	// MaxInstances bounds how many mediators of the same browser (one per
	// browser profile) can claim a bus name before we give up.
	MaxInstances = 16
)

// TabInfo is a tab as it crosses the D-Bus boundary: raw browser-assigned
// numeric IDs. The CLI composes the user-facing "<browser>.<window>.<tab>"
// token from these fields.
//
// Its field set is frozen: it is the "a(iissibb)" return of ListTabs, and
// D-Bus matches struct signatures by arity, so appending a field here would
// break every existing consumer at the call. New data goes on
// TabInfoWithIcon behind a new method instead.
type TabInfo struct {
	WindowID int32
	TabID    int32
	Title    string
	URL      string
	Index    int32
	Active   bool
	Pinned   bool
}

// TabInfoWithIcon is TabInfo plus the browser-resolved favicon URL, returned
// by ListTabsWithIcons as "a(iissibbs)". Consumers opt in by calling the new
// method; ListTabs keeps its original signature.
type TabInfoWithIcon struct {
	WindowID   int32
	TabID      int32
	Title      string
	URL        string
	Index      int32
	Active     bool
	Pinned     bool
	FavIconURL string
}

// TabInfo projects away the favicon, for the legacy ListTabs signature.
func (t TabInfoWithIcon) TabInfo() TabInfo {
	return TabInfo{
		WindowID: t.WindowID,
		TabID:    t.TabID,
		Title:    t.Title,
		URL:      t.URL,
		Index:    t.Index,
		Active:   t.Active,
		Pinned:   t.Pinned,
	}
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
	ListTabsWithIcons() ([]TabInfoWithIcon, *dbus.Error)
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

// InstanceName returns the bus-name element for the nth mediator of a
// browser. Each browser profile runs its own mediator, so a single browser
// can need several names. The first instance keeps the bare browser name,
// leaving existing tab IDs and --browser values untouched; later ones take a
// numeric suffix: Chrome, Chrome2, Chrome3.
//
// The suffix is a digit rather than a separator on purpose: it must be legal
// in a D-Bus object path element (letters, digits, underscore only) and must
// not contain the "." that delimits a user-facing tab ID.
func InstanceName(browser string, n int) string {
	if n <= 1 {
		return browser
	}
	return browser + strconv.Itoa(n)
}

// BaseBrowser strips the instance suffix InstanceName adds, so "Chrome2"
// resolves back to the browser "Chrome". Names that carry no suffix, or that
// are entirely digits, come back unchanged.
func BaseBrowser(instance string) string {
	base := strings.TrimRight(instance, "0123456789")
	if base == "" {
		return instance
	}
	return base
}
