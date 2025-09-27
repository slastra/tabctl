package mediator

// Message types for browser extension communication

// Command represents a command to send to the browser
type Command struct {
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args,omitempty"`
}

// Response represents a response from the browser
type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// TabInfo represents information about a tab
type TabInfo struct {
	ID       int    `json:"id"`
	WindowID int    `json:"windowId"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Active   bool   `json:"active"`
	Pinned   bool   `json:"pinned"`
	Audible  bool   `json:"audible"`
	Muted    bool   `json:"mutedInfo,omitempty"`
	Index    int    `json:"index"`
	Status   string `json:"status"`
}

// Common command names
const (
	CmdListTabs      = "list_tabs"
	CmdQueryTabs     = "query_tabs"
	CmdCloseTabs     = "close_tabs"
	CmdActivateTab   = "activate_tab"
	CmdMoveTabs      = "move_tabs"
	CmdOpenURLs      = "open_urls"
	CmdUpdateTabs    = "update_tabs"
	CmdNewTab        = "new_tab"
	CmdGetActiveTabs = "get_active_tabs"
	CmdGetScreenshot = "get_screenshot"
	CmdGetWords      = "get_words"
	CmdGetText       = "get_text"
	CmdGetHTML       = "get_html"
	CmdGetBrowser    = "get_browser"
)

// NewCommand creates a new command
func NewCommand(name string, args map[string]interface{}) *Command {
	return &Command{
		Command: name,
		Args:    args,
	}
}

// ValidateCommand validates a command
func ValidateCommand(cmd *Command) error {
	if cmd.Command == "" {
		return &ValidationError{
			Field:   "command",
			Value:   cmd.Command,
			Message: "command name is required",
		}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}