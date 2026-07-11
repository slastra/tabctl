package api

import (
	"github.com/tabctl/tabctl/pkg/types"
)

// Client is the interface a single mediator (one browser) exposes to the CLI.
type Client interface {
	ListTabs() ([]types.Tab, error)
	CloseTabs(tabIDs []int) error
	ActivateTab(tabID int, focused bool) error
	OpenTab(url string) (types.Tab, error)
	GetPrefix() string
	Info() (Info, error)
	Close() error
}

// Info reports one browser's mediator/extension versions and protocol
// compatibility, for `tabctl status`.
type Info struct {
	Browser           string
	MediatorVersion   string
	ExtensionVersion  string
	MediatorProtocol  int
	ExtensionProtocol int
	Compatible        bool
}
