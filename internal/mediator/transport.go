// Package mediator provides the native messaging host for browser tab control.
// It handles communication between the browser extension and the CLI tool.
package mediator

// Message type constants for the native messaging protocol
const (
	// MsgTypePing is sent by the browser to check if the host is alive
	MsgTypePing = "ping"
	// MsgTypePong is the response to a ping message
	MsgTypePong = "pong"
	// MsgTypeHealthCheck is a health check request from the browser
	MsgTypeHealthCheck = "health_check"
	// MsgTypeHealthCheckResponse is the response to a health check
	MsgTypeHealthCheckResponse = "health_check_response"

	// MaxMessageSize is the maximum allowed message size (10MB)
	MaxMessageSize = 10 * 1024 * 1024
)

// Transport defines the interface for native messaging communication.
// Implementations handle the low-level protocol details including
// message framing (4-byte length header) and JSON encoding.
type Transport interface {
	// Send encodes and sends a message to the browser
	Send(message interface{}) error
	// Recv receives and decodes a message from the browser.
	// It automatically handles ping/health check messages internally.
	Recv() (map[string]interface{}, error)
	// Close cleans up any resources (no-op for stdio)
	Close() error
}


