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

// StdTransport implements Transport over stdin/stdout using channels for
// non-blocking EOF detection.
type StdTransport struct {
	input     io.Reader
	output    io.Writer
	writeMu   sync.Mutex // serializes framed writes
	inChan    chan json.RawMessage
	errChan   chan error
	closeChan chan struct{}
	closeOnce sync.Once
}

// NewStdTransport creates a transport with automatic browser-disconnect
// detection.
func NewStdTransport(input io.Reader, output io.Writer) *StdTransport {
	t := &StdTransport{
		input:     input,
		output:    output,
		inChan:    make(chan json.RawMessage, 16),
		errChan:   make(chan error, 1),
		closeChan: make(chan struct{}),
	}
	go t.readLoop()
	return t
}

// NewStdioTransport creates a transport bound to os.Stdin/os.Stdout.
func NewStdioTransport() *StdTransport {
	return NewStdTransport(os.Stdin, os.Stdout)
}

// readLoop continuously reads framed messages from the input.
func (t *StdTransport) readLoop() {
	defer func() {
		close(t.inChan)
		close(t.errChan)
	}()

	for {
		select {
		case <-t.closeChan:
			return
		default:
		}

		// Read the 4-byte little-endian length header.
		lengthBytes := make([]byte, 4)
		n, err := io.ReadFull(t.input, lengthBytes)
		if err != nil {
			if err == io.EOF || n == 0 {
				t.errChan <- errors.NewTransportError("connection closed", io.EOF)
				return
			}
			t.errChan <- errors.NewTransportError("failed to read message length", err)
			return
		}

		var length uint32
		if err := binary.Read(bytes.NewReader(lengthBytes), binary.LittleEndian, &length); err != nil {
			t.errChan <- errors.NewTransportError("failed to parse message length", err)
			return
		}
		if length > MaxMessageSize {
			t.errChan <- errors.NewTransportError(fmt.Sprintf("message too large: %d bytes", length), nil)
			return
		}

		messageData := make([]byte, length)
		if _, err := io.ReadFull(t.input, messageData); err != nil {
			t.errChan <- errors.NewTransportError("failed to read message content", err)
			return
		}

		select {
		case t.inChan <- json.RawMessage(messageData):
		case <-t.closeChan:
			return
		}
	}
}

// Send frames and writes a message to the extension.
func (t *StdTransport) Send(message interface{}) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return errors.NewTransportError("failed to marshal message", err)
	}

	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	length := uint32(len(jsonData))
	if err := binary.Write(t.output, binary.LittleEndian, length); err != nil {
		return errors.NewTransportError("failed to write message length", err)
	}
	if _, err := t.output.Write(jsonData); err != nil {
		return errors.NewTransportError("failed to write message content", err)
	}
	if flusher, ok := t.output.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			return errors.NewTransportError("failed to flush output", err)
		}
	}
	return nil
}

// Incoming returns the channel of decoded incoming messages.
func (t *StdTransport) Incoming() <-chan json.RawMessage { return t.inChan }

// Errors returns the channel that reports disconnection.
func (t *StdTransport) Errors() <-chan error { return t.errChan }

// Close closes the transport.
func (t *StdTransport) Close() error {
	t.closeOnce.Do(func() { close(t.closeChan) })
	return nil
}
