package mediator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tabctl/tabctl/internal/dbus"
)

// DBusHandler adapts BrowserAPI to the dbus.BrowserHandler interface
type DBusHandler struct {
	api *BrowserAPI
}

func NewDBusHandler(api *BrowserAPI) *DBusHandler {
	return &DBusHandler{api: api}
}

func (h *DBusHandler) ListTabs() ([]dbus.TabInfo, error) {
	// Get tabs from browser via BrowserAPI (returns TSV strings)
	tabLines, err := h.api.ListTabs()
	if err != nil {
		return nil, err
	}

	// Parse TSV format and convert to D-Bus format
	var dbusTabsInfo []dbus.TabInfo
	for _, line := range tabLines {
		// TSV format: ID\tTitle\tURL\tIndex\tActive\tPinned
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			continue // Skip malformed lines
		}

		index, _ := strconv.Atoi(fields[3])
		active := fields[4] == "true"
		pinned := fields[5] == "true"

		dbusTabsInfo = append(dbusTabsInfo, dbus.TabInfo{
			ID:     fields[0],
			Title:  fields[1],
			URL:    fields[2],
			Index:  int32(index),
			Active: active,
			Pinned: pinned,
		})
	}

	return dbusTabsInfo, nil
}

func (h *DBusHandler) ActivateTab(tabID string) error {
	// Extract numeric tab ID from format like "c.1.123"
	parts := strings.Split(tabID, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid tab ID format: %s", tabID)
	}

	tabIDNum, err := strconv.Atoi(parts[2])
	if err != nil {
		return fmt.Errorf("invalid tab ID: %s", parts[2])
	}

	// ActivateTab takes (tabID int, focused bool)
	return h.api.ActivateTab(tabIDNum, true)
}

func (h *DBusHandler) CloseTab(tabID string) error {
	// CloseTabs expects the full tab ID string (can be comma-separated)
	_, err := h.api.CloseTabs(tabID)
	return err
}

func (h *DBusHandler) OpenTab(url string) (string, error) {
	// Use OpenURLs to open a single URL
	tabIDs, err := h.api.OpenURLs([]string{url}, nil)
	if err != nil {
		return "", err
	}

	if len(tabIDs) > 0 {
		return tabIDs[0], nil
	}

	return "", fmt.Errorf("failed to open tab")
}