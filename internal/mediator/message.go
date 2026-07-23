package mediator

import (
	"encoding/json"

	"github.com/tabctl/tabctl/internal/config"
)

// JSON-RPC 2.0 protocol (subset) spoken over native messaging.
//
// Message kind is disambiguated by shape: a "method" field marks a
// request (or, without an id, a notification); a "result"/"error" field
// marks a response. Requests and their responses correlate by "id".
const JSONRPCVersion = "2.0"

// ProtocolVersion is the native-messaging protocol version this mediator
// speaks. The extension announces its own in the hello handshake; a
// mismatch is surfaced to the CLI rather than allowed to fail silently.
const ProtocolVersion = config.ProtocolVersion

// Method names recognized across the wire.
const (
	MethodHello       = "hello"
	MethodListTabs    = "list_tabs"
	MethodCloseTabs   = "close_tabs"
	MethodActivateTab = "activate_tab"
	MethodOpenURLs    = "open_urls"
	// MethodTabsChanged is an extension→mediator notification (no id, no
	// reply): the browser's tab set changed in some way. Fan-out only —
	// consumers re-pull via ListTabs.
	MethodTabsChanged = "tabs_changed"
)

// Request is a JSON-RPC request sent to the browser extension.
type Request struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// Response is a JSON-RPC response received from the browser extension.
// Exactly one of Result / Error is set.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is the JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return e.Message }

// ErrCodeBrowser is the generic application error code for browser-side
// failures (JSON-RPC reserves -32000..-32099 for implementation errors).
const ErrCodeBrowser = -32000

func newRequest(id int, method string, params map[string]interface{}) *Request {
	return &Request{JSONRPC: JSONRPCVersion, ID: id, Method: method, Params: params}
}

// TabData is a tab as reported by the extension: raw browser-assigned
// numeric IDs, no composed string ID, no TSV.
type TabData struct {
	WindowID int    `json:"windowId"`
	TabID    int    `json:"tabId"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Index    int    `json:"index"`
	Active   bool   `json:"active"`
	Pinned   bool   `json:"pinned"`
}

// HelloParams is what the extension announces on connect.
type HelloParams struct {
	ExtensionVersion string `json:"extensionVersion"`
	ProtocolVersion  int    `json:"protocolVersion"`
}

// HelloResult is the mediator's reply to a hello.
type HelloResult struct {
	MediatorVersion string `json:"mediatorVersion"`
	ProtocolVersion int    `json:"protocolVersion"`
}
