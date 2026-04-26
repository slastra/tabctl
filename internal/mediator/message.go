package mediator

// Command represents a command sent to the browser extension over native
// messaging.
type Command struct {
	Command string                 `json:"name"`
	Args    map[string]interface{} `json:"args,omitempty"`
}

// Command names recognized by the browser extension.
const (
	CmdListTabs    = "list_tabs"
	CmdCloseTabs   = "close_tabs"
	CmdActivateTab = "activate_tab"
	CmdOpenURLs    = "open_urls"
)

func NewCommand(name string, args map[string]interface{}) *Command {
	return &Command{Command: name, Args: args}
}