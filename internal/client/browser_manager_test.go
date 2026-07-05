package client

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tabctl/tabctl/pkg/api"
	"github.com/tabctl/tabctl/pkg/types"
)

// fakeClient implements api.Client for routing tests
type fakeClient struct {
	prefix    string
	closeErr  error
	closed    [][]string
	activated []string
	opened    []string
}

func (f *fakeClient) ListTabs() ([]types.Tab, error) { return nil, nil }

func (f *fakeClient) CloseTabs(tabIDs []string) error {
	if f.closeErr != nil {
		return f.closeErr
	}
	f.closed = append(f.closed, tabIDs)
	return nil
}

func (f *fakeClient) ActivateTab(tabID string, focused bool) error {
	f.activated = append(f.activated, tabID)
	return nil
}

func (f *fakeClient) OpenTab(url string) (string, error) {
	f.opened = append(f.opened, url)
	return f.prefix + "1.100", nil
}

func (f *fakeClient) GetPrefix() string { return f.prefix }
func (f *fakeClient) Close() error      { return nil }

func TestCloseTabsRouting(t *testing.T) {
	firefox := &fakeClient{prefix: "firefox."}
	chrome := &fakeClient{prefix: "chrome."}
	bm := &BrowserManager{clients: []api.Client{firefox, chrome}}

	closed, err := bm.CloseTabs([]string{"firefox.1.1", "firefox.1.2", "chrome.2.3"})
	if err != nil {
		t.Fatalf("CloseTabs failed: %v", err)
	}
	if closed != 3 {
		t.Errorf("closed count: got %d, want 3", closed)
	}
	if len(firefox.closed) != 1 || len(firefox.closed[0]) != 2 {
		t.Errorf("firefox received %v, want 2 tabs in one call", firefox.closed)
	}
	if len(chrome.closed) != 1 || chrome.closed[0][0] != "chrome.2.3" {
		t.Errorf("chrome received %v, want [chrome.2.3]", chrome.closed)
	}
}

func TestCloseTabsUnroutable(t *testing.T) {
	firefox := &fakeClient{prefix: "firefox."}
	bm := &BrowserManager{clients: []api.Client{firefox}}

	closed, err := bm.CloseTabs([]string{"firefox.1.1", "zen.1.9"})
	if err == nil {
		t.Fatal("expected error for unroutable tab ID, got nil")
	}
	if !strings.Contains(err.Error(), "zen.1.9") {
		t.Errorf("error should name the unroutable ID, got: %v", err)
	}
	if closed != 1 {
		t.Errorf("closed count: got %d, want 1 (only the routable tab)", closed)
	}
}

func TestCloseTabsClientError(t *testing.T) {
	firefox := &fakeClient{prefix: "firefox.", closeErr: fmt.Errorf("browser gone")}
	chrome := &fakeClient{prefix: "chrome."}
	bm := &BrowserManager{clients: []api.Client{firefox, chrome}}

	closed, err := bm.CloseTabs([]string{"firefox.1.1", "chrome.2.3"})
	if err == nil {
		t.Fatal("expected error from failing client, got nil")
	}
	if closed != 1 {
		t.Errorf("closed count: got %d, want 1 (failing client's tabs excluded)", closed)
	}
}

func TestActivateTabRouting(t *testing.T) {
	firefox := &fakeClient{prefix: "firefox."}
	chrome := &fakeClient{prefix: "chrome."}
	bm := &BrowserManager{clients: []api.Client{firefox, chrome}}

	if err := bm.ActivateTab("chrome.1.5", true); err != nil {
		t.Fatalf("ActivateTab failed: %v", err)
	}
	if len(chrome.activated) != 1 || chrome.activated[0] != "chrome.1.5" {
		t.Errorf("chrome activated %v, want [chrome.1.5]", chrome.activated)
	}
	if len(firefox.activated) != 0 {
		t.Errorf("firefox should not have been called, got %v", firefox.activated)
	}

	if err := bm.ActivateTab("zen.1.1", false); err == nil {
		t.Error("expected error for unroutable tab, got nil")
	}
}

func TestOpenURLs(t *testing.T) {
	firefox := &fakeClient{prefix: "firefox."}
	bm := &BrowserManager{clients: []api.Client{firefox}}

	ids, err := bm.OpenURLs([]string{"https://a.example", "https://b.example"})
	if err != nil {
		t.Fatalf("OpenURLs failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("got %d ids, want 2", len(ids))
	}
	if len(firefox.opened) != 2 {
		t.Errorf("client opened %v, want 2 urls", firefox.opened)
	}
}

func TestOpenURLsMultipleBrowsers(t *testing.T) {
	bm := &BrowserManager{clients: []api.Client{
		&fakeClient{prefix: "firefox."},
		&fakeClient{prefix: "chrome."},
	}}

	_, err := bm.OpenURLs([]string{"https://a.example"})
	if err == nil {
		t.Fatal("expected error with multiple browsers, got nil")
	}
	if !strings.Contains(err.Error(), "--browser") {
		t.Errorf("error should suggest --browser, got: %v", err)
	}
}

func TestNewBrowserManagerErrors(t *testing.T) {
	origDiscover, origNewClient := discoverMediators, newDBusClient
	defer func() { discoverMediators, newDBusClient = origDiscover, origNewClient }()

	t.Run("discovery error propagates", func(t *testing.T) {
		discoverMediators = func() ([]MediatorInfo, error) {
			return nil, fmt.Errorf("cannot connect to D-Bus session bus: boom")
		}
		_, err := NewBrowserManager("")
		if err == nil || !strings.Contains(err.Error(), "session bus") {
			t.Errorf("want session bus error, got: %v", err)
		}
	})

	t.Run("empty bus", func(t *testing.T) {
		discoverMediators = func() ([]MediatorInfo, error) { return nil, nil }
		_, err := NewBrowserManager("")
		if err == nil || !strings.Contains(err.Error(), "no browsers found on D-Bus") {
			t.Errorf("want no-browsers error, got: %v", err)
		}
	})

	t.Run("target browser not registered", func(t *testing.T) {
		discoverMediators = func() ([]MediatorInfo, error) {
			return []MediatorInfo{{Browser: "Firefox", Prefix: "firefox."}}, nil
		}
		_, err := NewBrowserManager("Chrome")
		if err == nil || !strings.Contains(err.Error(), "Firefox") {
			t.Errorf("error should list available browsers, got: %v", err)
		}
	})

	t.Run("client connect failure surfaces", func(t *testing.T) {
		discoverMediators = func() ([]MediatorInfo, error) {
			return []MediatorInfo{{Browser: "Firefox", Prefix: "firefox."}}, nil
		}
		newDBusClient = func(browser string) (api.Client, error) {
			return nil, fmt.Errorf("connect refused")
		}
		_, err := NewBrowserManager("")
		if err == nil || !strings.Contains(err.Error(), "connect refused") {
			t.Errorf("client error should propagate, got: %v", err)
		}
	})
}
