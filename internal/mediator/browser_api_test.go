package mediator

import (
	"fmt"
	"testing"
)

// mockTransport implements Transport for testing
type mockTransport struct {
	sendErr    error
	recvMsg    map[string]interface{}
	recvErr    error
	lastSent   interface{}
}

func (m *mockTransport) Send(message interface{}) error {
	m.lastSent = message
	return m.sendErr
}

func (m *mockTransport) Recv() (map[string]interface{}, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	return m.recvMsg, nil
}

func (m *mockTransport) Close() error {
	return nil
}

func TestListTabs(t *testing.T) {
	mock := &mockTransport{
		recvMsg: map[string]interface{}{
			"result": []interface{}{
				"c.1.1\tGoogle\thttps://google.com",
				"c.1.2\tGitHub\thttps://github.com",
			},
		},
	}

	api := NewBrowserAPI(mock, "chrome")
	tabs, err := api.ListTabs()
	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}

	if len(tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(tabs))
	}
	if tabs[0] != "c.1.1\tGoogle\thttps://google.com" {
		t.Errorf("tab[0]: got %q", tabs[0])
	}
}

func TestListTabsError(t *testing.T) {
	mock := &mockTransport{
		recvMsg: map[string]interface{}{
			"error": "extension not ready",
		},
	}

	api := NewBrowserAPI(mock, "chrome")
	_, err := api.ListTabs()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListTabsTransportError(t *testing.T) {
	mock := &mockTransport{
		recvErr: fmt.Errorf("connection lost"),
	}

	api := NewBrowserAPI(mock, "chrome")
	_, err := api.ListTabs()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListTabsUnexpectedFormat(t *testing.T) {
	mock := &mockTransport{
		recvMsg: map[string]interface{}{
			"result": "not an array",
		},
	}

	api := NewBrowserAPI(mock, "chrome")
	_, err := api.ListTabs()
	if err == nil {
		t.Fatal("expected error for non-array result, got nil")
	}
}

func TestGetBrowser(t *testing.T) {
	mock := &mockTransport{
		recvMsg: map[string]interface{}{
			"result": "Brave",
		},
	}

	api := NewBrowserAPI(mock, "")
	browser := api.GetBrowser()
	if browser != "Brave" {
		t.Errorf("expected Brave, got %q", browser)
	}
}

func TestGetBrowserFallback(t *testing.T) {
	mock := &mockTransport{
		recvErr: fmt.Errorf("not connected"),
	}

	api := NewBrowserAPI(mock, "")
	browser := api.GetBrowser()
	if browser != "unknown" {
		t.Errorf("expected unknown, got %q", browser)
	}
}

func TestGetBrowserCached(t *testing.T) {
	mock := &mockTransport{}
	api := NewBrowserAPI(mock, "Firefox")
	browser := api.GetBrowser()
	if browser != "Firefox" {
		t.Errorf("expected Firefox, got %q", browser)
	}
	// Should not have sent a command since browser was already set
	if mock.lastSent != nil {
		t.Error("should not query extension when browser is already known")
	}
}

func TestActivateTab(t *testing.T) {
	mock := &mockTransport{
		recvMsg: map[string]interface{}{
			"result": "OK",
		},
	}

	api := NewBrowserAPI(mock, "chrome")
	err := api.ActivateTab(123, true)
	if err != nil {
		t.Fatalf("ActivateTab failed: %v", err)
	}

	// Verify the command was sent correctly
	cmd, ok := mock.lastSent.(*Command)
	if !ok {
		t.Fatalf("expected *Command, got %T", mock.lastSent)
	}
	if cmd.Command != CmdActivateTab {
		t.Errorf("command: got %q, want %q", cmd.Command, CmdActivateTab)
	}
	if cmd.Args["tab_id"] != 123 {
		t.Errorf("tab_id: got %v, want 123", cmd.Args["tab_id"])
	}
	if cmd.Args["focused"] != true {
		t.Errorf("focused: got %v, want true", cmd.Args["focused"])
	}
}

func TestCloseTabs(t *testing.T) {
	mock := &mockTransport{
		recvMsg: map[string]interface{}{
			"result": "OK",
		},
	}

	api := NewBrowserAPI(mock, "chrome")
	result, err := api.CloseTabs("c.1.1,c.1.2")
	if err != nil {
		t.Fatalf("CloseTabs failed: %v", err)
	}
	if result != "OK" {
		t.Errorf("expected OK, got %q", result)
	}
}

func TestGetText(t *testing.T) {
	mock := &mockTransport{
		recvMsg: map[string]interface{}{
			"result": []interface{}{
				"c.1.1\tPage Title\thttps://example.com\tSome text content",
			},
		},
	}

	api := NewBrowserAPI(mock, "chrome")
	lines, err := api.GetText(`\n|\r`, " ")
	if err != nil {
		t.Fatalf("GetText failed: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}
