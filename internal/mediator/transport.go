// Package mediator provides the native messaging host for browser tab control.
// It handles communication between the browser extension and the CLI tool.
package mediator

import "encoding/json"

// MaxMessageSize is the maximum allowed native-messaging message size (10MB).
const MaxMessageSize = 10 * 1024 * 1024

// Transport is a framed-JSON pipe to the browser extension. It handles the
// 4-byte length prefix and delivers/accepts raw JSON messages; routing and
// interpretation (requests vs responses) belong to the caller.
type Transport interface {
	// Send frames and writes a message to the extension.
	Send(message interface{}) error
	// Incoming delivers each decoded message as it arrives. Closed on
	// disconnect.
	Incoming() <-chan json.RawMessage
	// Errors reports disconnection or fatal read errors. Closed on
	// disconnect.
	Errors() <-chan error
	// Close releases resources.
	Close() error
}
