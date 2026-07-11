package mediator

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// mockTransport implements Transport for testing the dispatch model.
type mockTransport struct {
	inChan  chan json.RawMessage
	errChan chan error

	responder func(req *Request) json.RawMessage // optional auto-reply

	mu   sync.Mutex
	sent []interface{}
}

func newMockTransport(responder func(*Request) json.RawMessage) *mockTransport {
	return &mockTransport{
		inChan:    make(chan json.RawMessage, 32),
		errChan:   make(chan error, 1),
		responder: responder,
	}
}

func (m *mockTransport) Send(message interface{}) error {
	m.mu.Lock()
	m.sent = append(m.sent, message)
	m.mu.Unlock()
	if req, ok := message.(*Request); ok && m.responder != nil {
		if raw := m.responder(req); raw != nil {
			m.inChan <- raw
		}
	}
	return nil
}

func (m *mockTransport) Incoming() <-chan json.RawMessage { return m.inChan }
func (m *mockTransport) Errors() <-chan error             { return m.errChan }
func (m *mockTransport) Close() error                     { return nil }

func (m *mockTransport) sentRequests() []*Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	var reqs []*Request
	for _, s := range m.sent {
		if r, ok := s.(*Request); ok {
			reqs = append(reqs, r)
		}
	}
	return reqs
}

func rpcResult(id int, result interface{}) json.RawMessage {
	raw, _ := json.Marshal(result)
	b, _ := json.Marshal(Response{JSONRPC: JSONRPCVersion, ID: id, Result: raw})
	return b
}

func rpcError(id int, msg string) json.RawMessage {
	b, _ := json.Marshal(Response{JSONRPC: JSONRPCVersion, ID: id, Error: &RPCError{Code: ErrCodeBrowser, Message: msg}})
	return b
}

// echoResult replies to each request with the given result.
func echoResult(result interface{}) func(*Request) json.RawMessage {
	return func(req *Request) json.RawMessage { return rpcResult(req.ID, result) }
}

func TestListTabs(t *testing.T) {
	want := []TabData{
		{WindowID: 1, TabID: 1, Title: "Google", URL: "https://google.com"},
		{WindowID: 1, TabID: 2, Title: "GitHub", URL: "https://github.com"},
	}
	mock := newMockTransport(echoResult(want))
	api := NewBrowserAPI(mock)

	tabs, err := api.ListTabs()
	if err != nil {
		t.Fatalf("ListTabs failed: %v", err)
	}
	if len(tabs) != 2 || tabs[1].Title != "GitHub" || tabs[1].TabID != 2 {
		t.Fatalf("unexpected tabs: %+v", tabs)
	}
}

func TestListTabsError(t *testing.T) {
	mock := newMockTransport(func(req *Request) json.RawMessage {
		return rpcError(req.ID, "extension not ready")
	})
	api := NewBrowserAPI(mock)

	if _, err := api.ListTabs(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestActivateTab(t *testing.T) {
	mock := newMockTransport(echoResult(nil))
	api := NewBrowserAPI(mock)

	if err := api.ActivateTab(123, true); err != nil {
		t.Fatalf("ActivateTab failed: %v", err)
	}
	reqs := mock.sentRequests()
	if len(reqs) != 1 || reqs[0].Method != MethodActivateTab {
		t.Fatalf("unexpected request: %+v", reqs)
	}
	if reqs[0].Params["tab_id"] != 123 || reqs[0].Params["focused"] != true {
		t.Errorf("params: got %+v", reqs[0].Params)
	}
}

func TestCloseTabs(t *testing.T) {
	mock := newMockTransport(echoResult(nil))
	api := NewBrowserAPI(mock)

	if err := api.CloseTabs([]int{1, 2}); err != nil {
		t.Fatalf("CloseTabs failed: %v", err)
	}
	reqs := mock.sentRequests()
	if len(reqs) != 1 || reqs[0].Method != MethodCloseTabs {
		t.Fatalf("unexpected request: %+v", reqs)
	}
}

// TestOutOfOrderCorrelation proves responses are matched to requests by id,
// not by arrival order.
func TestOutOfOrderCorrelation(t *testing.T) {
	mock := newMockTransport(nil) // no auto-reply; we inject manually
	api := NewBrowserAPI(mock)

	type out struct {
		raw json.RawMessage
		err error
	}
	c1 := make(chan out, 1)
	c2 := make(chan out, 1)
	go func() { r, e := api.sendCommand(MethodListTabs, nil); c1 <- out{r, e} }()
	go func() { r, e := api.sendCommand(MethodCloseTabs, nil); c2 <- out{r, e} }()

	// Wait until both requests have been sent.
	var reqs []*Request
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		reqs = mock.sentRequests()
		if len(reqs) == 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests sent, got %d", len(reqs))
	}

	// Identify which request is which, respond in REVERSE order.
	byMethod := map[string]int{}
	for _, r := range reqs {
		byMethod[r.Method] = r.ID
	}
	mock.inChan <- rpcResult(byMethod[MethodCloseTabs], "close-result")
	mock.inChan <- rpcResult(byMethod[MethodListTabs], "list-result")

	o1 := <-c1
	o2 := <-c2
	if o1.err != nil || string(o1.raw) != `"list-result"` {
		t.Errorf("list caller got %s (err %v), want \"list-result\"", o1.raw, o1.err)
	}
	if o2.err != nil || string(o2.raw) != `"close-result"` {
		t.Errorf("close caller got %s (err %v), want \"close-result\"", o2.raw, o2.err)
	}
}

// TestHelloHandshake verifies the mediator records the extension version and
// replies to a hello.
func TestHelloHandshake(t *testing.T) {
	mock := newMockTransport(nil)
	api := NewBrowserAPI(mock)

	hello, _ := json.Marshal(Request{
		JSONRPC: JSONRPCVersion, ID: 1, Method: MethodHello,
		Params: map[string]interface{}{"extensionVersion": "2.0.0", "protocolVersion": ProtocolVersion},
	})
	mock.inChan <- hello

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if api.Info().ExtensionVersion == "2.0.0" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	info := api.Info()
	if info.ExtensionVersion != "2.0.0" {
		t.Fatalf("extension version not recorded: %+v", info)
	}
	if !info.Compatible {
		t.Errorf("expected compatible, got %+v", info)
	}
}

// TestVersionGuard verifies an incompatible extension protocol fails commands
// loudly.
func TestVersionGuard(t *testing.T) {
	mock := newMockTransport(echoResult(nil))
	api := NewBrowserAPI(mock)

	hello, _ := json.Marshal(Request{
		JSONRPC: JSONRPCVersion, ID: 1, Method: MethodHello,
		Params: map[string]interface{}{"extensionVersion": "99.0.0", "protocolVersion": ProtocolVersion + 1},
	})
	mock.inChan <- hello

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if api.Info().ExtensionProtocol != 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	_, err := api.ListTabs()
	if err == nil {
		t.Fatal("expected version-guard error, got nil")
	}
}
