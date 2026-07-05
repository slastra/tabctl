package types

// Tab represents a browser tab.
type Tab struct {
	ID       string `json:"id"` // "browser.family.window.tab", e.g. "helium.c.1.123"
	Title    string `json:"title"`
	URL      string `json:"url"`
	WindowID int    `json:"windowId"`
	Index    int    `json:"index"`
	Active   bool   `json:"active"`
	Pinned   bool   `json:"pinned"`
}
