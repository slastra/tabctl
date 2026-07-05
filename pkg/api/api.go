package api

import (
	"github.com/tabctl/tabctl/pkg/types"
)

// Client is the interface a single mediator (one browser) exposes to the CLI.
type Client interface {
	ListTabs() ([]types.Tab, error)
	CloseTabs(tabIDs []string) error
	ActivateTab(tabID string, focused bool) error
	OpenTab(url string) (string, error)
	GetPrefix() string
	Close() error
}
