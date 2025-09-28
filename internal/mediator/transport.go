// Package mediator provides the native messaging host for browser tab control.
// It handles communication between the browser extension and the CLI tool.
package mediator

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/tabctl/tabctl/internal/errors"
)

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

// StdTransport implements the native messaging protocol over stdin/stdout.
// This is the standard way browser extensions communicate with native hosts.
type StdTransport struct {
	input  io.Reader
	output io.Writer
	reader *bufio.Reader
}

// NewStdTransport creates a new standard transport
func NewStdTransport(input io.Reader, output io.Writer) Transport {
	return &StdTransport{
		input:  input,
		output: output,
		reader: bufio.NewReader(input),
	}
}

// NewDefaultTransport creates a transport using stdin/stdout
func NewDefaultTransport() Transport {
	return NewStdTransport(os.Stdin, os.Stdout)
}

// Send sends a message to the browser extension using the native messaging protocol:
// 4-byte little-endian length header followed by JSON message body
func (t *StdTransport) Send(message interface{}) error {
	// Sending message
	// Encode message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.NewTransportError("failed to marshal message", err)
	}

	// Write length header (4 bytes, little-endian per native messaging spec)
	length := uint32(len(jsonData))
	if err := binary.Write(t.output, binary.LittleEndian, length); err != nil {
		return errors.NewTransportError("failed to write message length", err)
	}

	// Write message content
	if _, err := t.output.Write(jsonData); err != nil {
		return errors.NewTransportError("failed to write message content", err)
	}

	// Flush output if it's a buffered writer
	if flusher, ok := t.output.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			return errors.NewTransportError("failed to flush output", err)
		}
	}

	return nil
}

// Recv receives a message from the browser extension.
// It automatically handles ping and health check messages internally,
// only returning actual command messages to the caller.
func (t *StdTransport) Recv() (map[string]interface{}, error) {
	for {
		// Read the next message
		message, err := t.readMessage()
		if err != nil {
			return nil, err
		}

		// Handle internal protocol messages
		if handled := t.handleInternalMessage(message); handled {
			continue // Read next message
		}

		// Return non-internal messages to caller
		return message, nil
	}
}

// readMessage reads a single message from the input stream
func (t *StdTransport) readMessage() (map[string]interface{}, error) {
	// Read length header (4 bytes)
	lengthBytes := make([]byte, 4)
	_, err := io.ReadFull(t.reader, lengthBytes)
	if err != nil {
		if err == io.EOF {
			return nil, errors.NewTransportError("connection closed", err)
		}
		return nil, errors.NewTransportError("failed to read message length", err)
	}

	var length uint32
	if err := binary.Read(bytes.NewReader(lengthBytes), binary.LittleEndian, &length); err != nil {
		return nil, errors.NewTransportError("failed to parse message length", err)
	}

	// Validate message length
	if length > MaxMessageSize {
		return nil, errors.NewTransportError(fmt.Sprintf("message too large: %d bytes", length), nil)
	}

	// Read message content
	messageData := make([]byte, length)
	if _, err := io.ReadFull(t.reader, messageData); err != nil {
		return nil, errors.NewTransportError("failed to read message content", err)
	}

	// Raw message received

	// Decode JSON message
	var message map[string]interface{}
	if err := json.Unmarshal(messageData, &message); err != nil {
		return nil, errors.NewTransportError("failed to unmarshal message", err)
	}

	// Message received and decoded

	return message, nil
}

// handleInternalMessage processes ping and health check messages.
// Returns true if the message was handled internally.
func (t *StdTransport) handleInternalMessage(message map[string]interface{}) bool {
	msgType, ok := message["type"].(string)
	if !ok {
		return false
	}

	switch msgType {
	case MsgTypePing:
		// Respond to ping with pong
		t.Send(map[string]interface{}{"type": MsgTypePong})
		return true
	case MsgTypeHealthCheck:
		// Respond to health check
		t.Send(map[string]interface{}{
			"type":   MsgTypeHealthCheckResponse,
			"status": "alive",
		})
		return true
	default:
		return false
	}
}

// Close closes the transport (no-op for stdio as we don't own the streams)
func (t *StdTransport) Close() error {
	return nil
}

// TimeoutTransport wraps a transport with timeout capabilities
type TimeoutTransport struct {
	transport Transport
	timeout   time.Duration
}

// NewTimeoutTransport creates a new timeout transport
func NewTimeoutTransport(transport Transport, timeout time.Duration) Transport {
	return &TimeoutTransport{
		transport: transport,
		timeout:   timeout,
	}
}

// Send sends a message with timeout
func (t *TimeoutTransport) Send(message interface{}) error {
	done := make(chan error, 1)
	go func() {
		done <- t.transport.Send(message)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(t.timeout):
		return errors.NewTimeoutError("send", t.timeout.String())
	}
}

// Recv receives a message with timeout
func (t *TimeoutTransport) Recv() (map[string]interface{}, error) {
	done := make(chan struct {
		msg map[string]interface{}
		err error
	}, 1)

	go func() {
		msg, err := t.transport.Recv()
		done <- struct {
			msg map[string]interface{}
			err error
		}{msg, err}
	}()

	select {
	case result := <-done:
		return result.msg, result.err
	case <-time.After(t.timeout):
		return nil, errors.NewTimeoutError("receive", t.timeout.String())
	}
}

// Close closes the underlying transport
func (t *TimeoutTransport) Close() error {
	return t.transport.Close()
}