package api

import (
	"github.com/tabctl/tabctl/pkg/types"
)

// Client is the interface a single mediator (one browser) exposes to the CLI.
type Client interface {
	ListTabs() ([]types.Tab, error)
	CloseTabs(tabIDs []string) error
	ActivateTab(tabID string, focused bool) error
	GetPrefix() string
	Close() error
}
