package mediator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/errors"
)

// BrowserAPI drives the browser extension over the native-messaging
// transport, correlating requests and responses by id. Concurrent D-Bus
// handler goroutines can each have a command in flight; replies are routed
// back to the right caller by the dispatch loop.
type BrowserAPI struct {
	transport Transport

	mu      sync.Mutex
	nextID  int
	pending map[int]chan *Response

	infoMu            sync.RWMutex
	helloReceived     bool
	extensionVersion  string
	extensionProtocol int

	tabsChangedMu sync.RWMutex
	onTabsChanged func()
}

// commandTimeout bounds how long sendCommand waits for a reply. It is a
// variable (not the config constant directly) so tests can shorten it.
var commandTimeout = config.MediatorBrowserTimeout

// NewBrowserAPI creates a BrowserAPI and starts its dispatch loop.
func NewBrowserAPI(transport Transport) *BrowserAPI {
	r := &BrowserAPI{
		transport: transport,
		pending:   make(map[int]chan *Response),
	}
	go r.dispatch()
	return r
}

// dispatch routes each incoming message: responses go to their waiting
// caller by id; requests (the hello handshake) are handled inline.
func (r *BrowserAPI) dispatch() {
	for raw := range r.transport.Incoming() {
		var probe struct {
			Method string          `json:"method"`
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *RPCError       `json:"error"`
		}
		if err := json.Unmarshal(raw, &probe); err != nil {
			continue // ignore malformed frames
		}

		if probe.Method != "" {
			r.handleRequest(probe.Method, raw)
			continue
		}

		// Response: hand off to the waiter registered for this id.
		r.mu.Lock()
		ch, ok := r.pending[probe.ID]
		if ok {
			delete(r.pending, probe.ID)
		}
		r.mu.Unlock()
		if ok {
			ch <- &Response{ID: probe.ID, Result: probe.Result, Error: probe.Error}
		}
		// Unmatched (late/unknown) responses are dropped.
	}
}

// SetTabsChangedHandler registers a callback for the extension's tabs_changed
// notification. The callback runs on the dispatch goroutine — keep it fast.
func (r *BrowserAPI) SetTabsChangedHandler(f func()) {
	r.tabsChangedMu.Lock()
	r.onTabsChanged = f
	r.tabsChangedMu.Unlock()
}

// handleRequest processes an extension-initiated request or notification.
func (r *BrowserAPI) handleRequest(method string, raw json.RawMessage) {
	if method == MethodTabsChanged {
		r.tabsChangedMu.RLock()
		f := r.onTabsChanged
		r.tabsChangedMu.RUnlock()
		if f != nil {
			f()
		}
		return
	}
	if method != MethodHello {
		return
	}
	var req struct {
		ID     int         `json:"id"`
		Params HelloParams `json:"params"`
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}

	r.infoMu.Lock()
	r.helloReceived = true
	r.extensionVersion = req.Params.ExtensionVersion
	r.extensionProtocol = req.Params.ProtocolVersion
	r.infoMu.Unlock()

	result, _ := json.Marshal(HelloResult{
		MediatorVersion: config.Version,
		ProtocolVersion: ProtocolVersion,
	})
	_ = r.transport.Send(&Response{
		JSONRPC: JSONRPCVersion,
		ID:      req.ID,
		Result:  result,
	})
}

// sendCommand sends a request and waits for its correlated response.
func (r *BrowserAPI) sendCommand(method string, params map[string]interface{}) (json.RawMessage, error) {
	// Version guard: if the extension announced an incompatible protocol,
	// fail loud rather than let commands misbehave.
	r.infoMu.RLock()
	helloReceived := r.helloReceived
	extProto := r.extensionProtocol
	r.infoMu.RUnlock()
	if helloReceived && extProto != ProtocolVersion {
		return nil, fmt.Errorf(
			"browser extension speaks protocol v%d but this mediator speaks v%d — update tabctl and the extension to matching versions",
			extProto, ProtocolVersion)
	}

	r.mu.Lock()
	r.nextID++
	id := r.nextID
	ch := make(chan *Response, 1)
	r.pending[id] = ch
	r.mu.Unlock()

	if err := r.transport.Send(newRequest(id, method, params)); err != nil {
		r.mu.Lock()
		delete(r.pending, id)
		r.mu.Unlock()
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	select {
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("browser error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	case <-ctx.Done():
		r.mu.Lock()
		delete(r.pending, id)
		r.mu.Unlock()
		if !helloReceived {
			return nil, fmt.Errorf("no response from browser extension; it may be missing or incompatible — run `tabctl status`")
		}
		return nil, errors.NewTimeoutError("browser response", commandTimeout.String())
	}
}

// ListTabs returns the browser's tabs as structured data.
func (r *BrowserAPI) ListTabs() ([]TabData, error) {
	raw, err := r.sendCommand(MethodListTabs, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}
	var tabs []TabData
	if err := json.Unmarshal(raw, &tabs); err != nil {
		return nil, errors.NewTransportError("unexpected list_tabs response", err)
	}
	return tabs, nil
}

// CloseTabs closes the tabs with the given browser-assigned numeric IDs.
func (r *BrowserAPI) CloseTabs(tabIDs []int) error {
	if _, err := r.sendCommand(MethodCloseTabs, map[string]interface{}{"tab_ids": tabIDs}); err != nil {
		return fmt.Errorf("failed to communicate with browser extension: %w", err)
	}
	return nil
}

// ActivateTab activates the tab with the given numeric ID.
func (r *BrowserAPI) ActivateTab(tabID int, focused bool) error {
	if _, err := r.sendCommand(MethodActivateTab, map[string]interface{}{"tab_id": tabID, "focused": focused}); err != nil {
		return fmt.Errorf("failed to communicate with browser extension: %w", err)
	}
	return nil
}

// OpenURLs opens the given URLs in new tabs and returns their data.
func (r *BrowserAPI) OpenURLs(urls []string) ([]TabData, error) {
	raw, err := r.sendCommand(MethodOpenURLs, map[string]interface{}{"urls": urls})
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}
	var tabs []TabData
	if err := json.Unmarshal(raw, &tabs); err != nil {
		return nil, errors.NewTransportError("unexpected open_urls response", err)
	}
	return tabs, nil
}

// Info reports version/handshake state for the D-Bus GetInfo method.
type Info struct {
	MediatorVersion   string
	ExtensionVersion  string
	MediatorProtocol  int
	ExtensionProtocol int
	Compatible        bool
}

// Info returns the current handshake state.
func (r *BrowserAPI) Info() Info {
	r.infoMu.RLock()
	defer r.infoMu.RUnlock()
	return Info{
		MediatorVersion:   config.Version,
		ExtensionVersion:  r.extensionVersion,
		MediatorProtocol:  ProtocolVersion,
		ExtensionProtocol: r.extensionProtocol,
		Compatible:        r.helloReceived && r.extensionProtocol == ProtocolVersion,
	}
}
