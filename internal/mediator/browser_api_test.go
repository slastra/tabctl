package mediator

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tabctl/tabctl/internal/errors"
)

// mockTransport implements Transport for testing
type mockTransport struct {
	mu       sync.Mutex
	sendErr  error
	recvMsg  map[string]interface{}
	recvErr  error
	lastSent interface{}

	// inFlight tracks Send..Recv pairing to detect interleaved commands
	inFlight    atomic.Int32
	interleaved atomic.Bool
	drained     atomic.Int32
}

func (m *mockTransport) Send(message interface{}) error {
	if m.inFlight.Add(1) > 1 {
		m.interleaved.Store(true)
	}
	m.mu.Lock()
	m.lastSent = message
	err := m.sendErr
	m.mu.Unlock()
	return err
}

func (m *mockTransport) Recv(ctx context.Context) (map[string]interface{}, error) {
	defer m.inFlight.Add(-1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	return m.recvMsg, nil
}

func (m *mockTransport) Drain() {
	m.drained.Add(1)
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

// TestSendCommandSerialized verifies concurrent D-Bus handler goroutines
// cannot interleave Send/Recv pairs on the shared transport.
func TestSendCommandSerialized(t *testing.T) {
	mock := &mockTransport{
		recvMsg: map[string]interface{}{"result": "OK"},
	}
	api := NewBrowserAPI(mock, "chrome")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = api.ActivateTab(1, false)
		}()
	}
	wg.Wait()

	if mock.interleaved.Load() {
		t.Fatal("Send called while another command's Recv was outstanding")
	}
}

// TestSendCommandTimeout verifies a Recv timeout marks the pipe stale and
// the next command drains the stale response before sending.
func TestSendCommandTimeout(t *testing.T) {
	mock := &mockTransport{
		recvErr: errors.NewTimeoutError("browser response", "context deadline exceeded"),
	}
	api := NewBrowserAPI(mock, "chrome")

	if _, err := api.sendCommand(NewCommand(CmdListTabs, nil)); err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !api.stale {
		t.Fatal("stale flag not set after Recv timeout")
	}

	// Recover the mock and verify the next command drains first
	mock.mu.Lock()
	mock.recvErr = nil
	mock.recvMsg = map[string]interface{}{"result": "OK"}
	mock.mu.Unlock()

	if _, err := api.sendCommand(NewCommand(CmdListTabs, nil)); err != nil {
		t.Fatalf("sendCommand after stale failed: %v", err)
	}
	if mock.drained.Load() != 1 {
		t.Fatalf("expected 1 drain after stale timeout, got %d", mock.drained.Load())
	}
	if api.stale {
		t.Fatal("stale flag not cleared after drain")
	}
}

