package types

// Tab represents a browser tab.
type Tab struct {
	ID       string `json:"id"` // "<browser>.<window>.<tab>", e.g. "helium.1874583011.1874583012"
	Title    string `json:"title"`
	URL      string `json:"url"`
	WindowID int    `json:"windowId"`
	Index    int    `json:"index"`
	Active   bool   `json:"active"`
	Pinned   bool   `json:"pinned"`
	// FavIconURL is the icon the browser already resolved for this tab, so
	// consumers need not guess one from the domain. Empty when the tab has
	// no icon or the extension predates the field. Usually http(s), but a
	// page declaring a data: URI favicon reports it inline.
	FavIconURL string `json:"favIconUrl"`
}
