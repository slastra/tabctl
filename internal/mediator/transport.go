package mediator

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/tabctl/tabctl/internal/errors"
)

// Transport defines the interface for native messaging communication
type Transport interface {
	Send(message interface{}) error
	Recv() (map[string]interface{}, error)
	Close() error
}

// StdTransport implements native messaging over stdin/stdout
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

// Send sends a message to the browser extension
func (t *StdTransport) Send(message interface{}) error {
	// Encode message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.NewTransportError("failed to marshal message", err)
	}

	// Log outgoing message
	if log.Flags() != 0 {
		log.Printf("Transport SENDING: %s", string(jsonData))
	}

	// Write length header (4 bytes, native byte order)
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

	if log.Flags() != 0 {
		log.Printf("Transport SENDING DONE")
	}

	return nil
}

// Recv receives a message from the browser extension
func (t *StdTransport) Recv() (map[string]interface{}, error) {
	if log.Flags() != 0 {
		log.Printf("Transport RECEIVING")
	}

	// Read length header (4 bytes)
	var length uint32
	if err := binary.Read(t.reader, binary.LittleEndian, &length); err != nil {
		if err == io.EOF {
			return nil, errors.NewTransportError("connection closed", err)
		}
		return nil, errors.NewTransportError("failed to read message length", err)
	}

	// Validate message length
	if length > 1024*1024*10 { // 10MB max
		return nil, errors.NewTransportError(fmt.Sprintf("message too large: %d bytes", length), nil)
	}

	// Read message content
	messageData := make([]byte, length)
	if _, err := io.ReadFull(t.reader, messageData); err != nil {
		return nil, errors.NewTransportError("failed to read message content", err)
	}

	if log.Flags() != 0 {
		log.Printf("Transport RECEIVED: %s", string(messageData))
	}

	// Decode JSON message
	var message map[string]interface{}
	if err := json.Unmarshal(messageData, &message); err != nil {
		return nil, errors.NewTransportError("failed to unmarshal message", err)
	}

	return message, nil
}

// Close closes the transport
func (t *StdTransport) Close() error {
	// Don't close stdin/stdout
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