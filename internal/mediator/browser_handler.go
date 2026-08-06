package mediator

import (
	"fmt"

	"github.com/tabctl/tabctl/internal/dbus"
)

// DBusHandler adapts BrowserAPI to the dbus.BrowserHandler interface.
type DBusHandler struct {
	api *BrowserAPI
}

func NewDBusHandler(api *BrowserAPI) *DBusHandler {
	return &DBusHandler{api: api}
}

func (h *DBusHandler) ListTabs() ([]dbus.TabInfoWithIcon, error) {
	tabs, err := h.api.ListTabs()
	if err != nil {
		return nil, err
	}

	infos := make([]dbus.TabInfoWithIcon, len(tabs))
	for i, t := range tabs {
		infos[i] = dbus.TabInfoWithIcon{
			WindowID:   int32(t.WindowID),
			TabID:      int32(t.TabID),
			Title:      t.Title,
			URL:        t.URL,
			Index:      int32(t.Index),
			Active:     t.Active,
			Pinned:     t.Pinned,
			FavIconURL: t.FavIconURL,
		}
	}
	return infos, nil
}

func (h *DBusHandler) ActivateTab(tabID int32, focused bool) error {
	return h.api.ActivateTab(int(tabID), focused)
}

func (h *DBusHandler) CloseTabs(tabIDs []int32) error {
	ids := make([]int, len(tabIDs))
	for i, id := range tabIDs {
		ids[i] = int(id)
	}
	return h.api.CloseTabs(ids)
}

func (h *DBusHandler) OpenTab(url string) (int32, int32, error) {
	tabs, err := h.api.OpenURLs([]string{url})
	if err != nil {
		return 0, 0, err
	}
	if len(tabs) == 0 {
		return 0, 0, fmt.Errorf("failed to open tab")
	}
	return int32(tabs[0].WindowID), int32(tabs[0].TabID), nil
}

func (h *DBusHandler) GetInfo() dbus.Info {
	i := h.api.Info()
	return dbus.Info{
		MediatorVersion:   i.MediatorVersion,
		ExtensionVersion:  i.ExtensionVersion,
		MediatorProtocol:  int32(i.MediatorProtocol),
		ExtensionProtocol: int32(i.ExtensionProtocol),
		Compatible:        i.Compatible,
	}
}
