package mediator

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/tabctl/tabctl/internal/errors"
)

// StdTransport implements Transport using channels for non-blocking EOF detection
type StdTransport struct {
	input       io.Reader
	output      io.Writer
	msgChan     chan map[string]interface{}
	errChan     chan error
	closeChan   chan struct{}
	closeOnce   sync.Once
}

// NewStdTransport creates a new transport with automatic EOF detection
func NewStdTransport(input io.Reader, output io.Writer) *StdTransport {
	t := &StdTransport{
		input:     input,
		output:    output,
		msgChan:   make(chan map[string]interface{}, 10), // Buffer for smoother operation
		errChan:   make(chan error, 1),
		closeChan: make(chan struct{}),
	}

	// Start the stdin reader goroutine
	go t.readLoop()

	return t
}

// NewDefaultTransport creates a transport using stdin/stdout
func NewDefaultTransport() *StdTransport {
	return NewStdTransport(os.Stdin, os.Stdout)
}

// readLoop continuously reads from stdin in a goroutine
func (t *StdTransport) readLoop() {
	defer func() {
		close(t.msgChan)
		close(t.errChan)
	}()

	for {
		select {
		case <-t.closeChan:
			return
		default:
			// Continue reading
		}

		// Read message length (4 bytes)
		lengthBytes := make([]byte, 4)
		n, err := io.ReadFull(t.input, lengthBytes)

		if err != nil {
			if err == io.EOF || n == 0 {
				// Browser disconnected cleanly
				t.errChan <- errors.NewTransportError("connection closed", io.EOF)
				return
			}
			// Unexpected error
			t.errChan <- errors.NewTransportError("failed to read message length", err)
			return
		}

		// Parse message length
		var length uint32
		if err := binary.Read(bytes.NewReader(lengthBytes), binary.LittleEndian, &length); err != nil {
			t.errChan <- errors.NewTransportError("failed to parse message length", err)
			return
		}

		// Validate message length
		if length > MaxMessageSize {
			t.errChan <- errors.NewTransportError(fmt.Sprintf("message too large: %d bytes", length), nil)
			return
		}

		// Read message content
		messageData := make([]byte, length)
		if _, err := io.ReadFull(t.input, messageData); err != nil {
			t.errChan <- errors.NewTransportError("failed to read message content", err)
			return
		}

		// Decode JSON
		var message map[string]interface{}
		if err := json.Unmarshal(messageData, &message); err != nil {
			t.errChan <- errors.NewTransportError("failed to unmarshal message", err)
			return
		}

		// Handle internal protocol messages
		if handled := t.handleInternalMessage(message); handled {
			continue // Don't forward ping/health check messages
		}

		// Send message to channel
		select {
		case t.msgChan <- message:
			// Message sent successfully
		case <-t.closeChan:
			return
		}
	}
}

// handleInternalMessage processes ping and health check messages
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

// Send sends a message to the browser
func (t *StdTransport) Send(message interface{}) error {
	// Encode message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.NewTransportError("failed to marshal message", err)
	}

	// Write length header (4 bytes, little-endian)
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

// Recv receives a message from the browser (non-blocking via channel)
func (t *StdTransport) Recv() (map[string]interface{}, error) {
	select {
	case msg, ok := <-t.msgChan:
		if !ok {
			return nil, errors.NewTransportError("transport closed", nil)
		}
		return msg, nil
	case err := <-t.errChan:
		return nil, err
	}
}

// GetErrorChannel returns the error channel for monitoring disconnection
func (t *StdTransport) GetErrorChannel() <-chan error {
	return t.errChan
}

// Close closes the transport
func (t *StdTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.closeChan)
	})
	return nil
}