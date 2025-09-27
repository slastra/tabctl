package api

import (
	"github.com/tabctl/tabctl/pkg/types"
)

// TabAPI defines the interface for tab operations
type TabAPI interface {
	// Core tab operations
	ListTabs() ([]types.Tab, error)
	CloseTabs(tabIDs []string) error
	ActivateTab(tabID string, focused bool) error
	MoveTabs() error

	// Tab state operations
	UpdateTabs(updates []types.TabUpdate) error
	QueryTabs(query types.TabQuery) ([]types.Tab, error)
	NavigateURLs(pairs []types.TabURLPair) error

	// Content operations
	GetText(tabIDs []string, options types.TextOptions) ([]types.TabContent, error)
	GetHTML(tabIDs []string, options types.TextOptions) ([]types.TabContent, error)
	GetWords(tabIDs []string, options types.WordsOptions) ([]string, error)

	// Window operations
	GetWindows() ([]types.Window, error)
	GetActiveTab() (string, error)
	GetActiveTabs() ([]string, error)

	// URL operations
	OpenURLs(urls []string, windowID string) ([]string, error)

	// Screenshot operations
	GetScreenshot() (*types.Screenshot, error)
}

// SearchAPI defines the interface for search operations
type SearchAPI interface {
	IndexTabs(tabs []types.TabContent) error
	Search(query string) ([]types.SearchResult, error)
	UpdateIndex(tabID string, content types.TabContent) error
}

// ClientAPI defines the interface for client management
type ClientAPI interface {
	GetClients() ([]types.Client, error)
	CreateClient(host string, port int) (Client, error)
}

// Client represents a single mediator client
type Client interface {
	TabAPI
	GetPrefix() string
	GetHost() string
	GetPort() int
	GetBrowser() string
	Close() error
}

// MultiClient represents multiple mediator clients
type MultiClient interface {
	TabAPI
	GetClients() []Client
	AddClient(client Client)
	RemoveClient(prefix string)
}