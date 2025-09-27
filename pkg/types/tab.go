package types

// Tab represents a browser tab
type Tab struct {
	ID       int    `json:"id"`       // Browser's internal tab ID
	Title    string `json:"title"`
	URL      string `json:"url"`
	WindowID int    `json:"windowId"`
	Index    int    `json:"index"`
	Active   bool   `json:"active"`
	Pinned   bool   `json:"pinned"`
	Audible  bool   `json:"audible"`
	Muted    bool   `json:"muted"`
}

// TabContent represents tab text/html content
type TabContent struct {
	TabID   string `json:"tab_id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

// TabUpdate represents a tab update request
type TabUpdate struct {
	TabID      string                 `json:"tab_id"`
	URL        string                 `json:"url,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// TabQuery represents query parameters for filtering tabs
type TabQuery struct {
	Active           *bool    `json:"active,omitempty"`
	Pinned           *bool    `json:"pinned,omitempty"`
	Audible          *bool    `json:"audible,omitempty"`
	Muted            *bool    `json:"muted,omitempty"`
	Highlighted      *bool    `json:"highlighted,omitempty"`
	Discarded        *bool    `json:"discarded,omitempty"`
	AutoDiscardable  *bool    `json:"autoDiscardable,omitempty"`
	CurrentWindow    *bool    `json:"currentWindow,omitempty"`
	LastFocusedWindow *bool   `json:"lastFocusedWindow,omitempty"`
	Status           string   `json:"status,omitempty"`
	Title            string   `json:"title,omitempty"`
	URL              []string `json:"url,omitempty"`
	WindowID         *int     `json:"windowId,omitempty"`
	WindowType       string   `json:"windowType,omitempty"`
	Index            *int     `json:"index,omitempty"`
}

// SearchResult represents a search result from SQLite FTS5
type SearchResult struct {
	TabID   string  `json:"tab_id"`
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Snippet string  `json:"snippet"`
	Rank    float64 `json:"rank"`
}

// Client represents a browser mediator client
type Client struct {
	Prefix  string `json:"prefix"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Browser string `json:"browser"`
	PID     int    `json:"pid"`
}

// Screenshot represents a tab screenshot
type Screenshot struct {
	Data     []byte `json:"data"`      // PNG data (bytes)
	TabID    string `json:"tab"`       // Tab ID of visible tab
	WindowID string `json:"window"`    // Window ID of visible tab
	API      string `json:"api"`       // Prefix of client API
}

// TabURLPair represents a tab ID and URL pair for navigation
type TabURLPair struct {
	TabID string `json:"tab_id"`
	URL   string `json:"url"`
}

// Window represents a browser window
type Window struct {
	ID       int   `json:"id"`
	Tabs     []Tab `json:"tabs"`
	TabCount int   `json:"tab_count"`
}

// TabMove represents a tab move operation
type TabMove struct {
	TabID    string `json:"tab_id"`
	WindowID int    `json:"window_id"`
	Index    int    `json:"index"`
}

// WordsOptions represents options for extracting words from tabs
type WordsOptions struct {
	MatchRegex string `json:"match_regex"`
	JoinWith   string `json:"join_with"`
}

// TextOptions represents options for extracting text from tabs
type TextOptions struct {
	DelimiterRegex string `json:"delimiter_regex"`
	ReplaceWith    string `json:"replace_with"`
	Cleanup        bool   `json:"cleanup"`
}

// Constants for default values
const (
	DefaultGetWordsMatchRegex     = `\w+`
	DefaultGetWordsJoinWith       = "\n"
	DefaultGetTextDelimiterRegex  = `\n|\r|\t`
	DefaultGetTextReplaceWith     = " "
	DefaultGetHTMLDelimiterRegex  = `\n|\r|\t`
	DefaultGetHTMLReplaceWith     = " "
)