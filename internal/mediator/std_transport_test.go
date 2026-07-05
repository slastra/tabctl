package mediator

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"

	"github.com/tabctl/tabctl/internal/errors"
)

// writeNativeMessage writes a length-prefixed JSON message to a writer
func writeNativeMessage(w io.Writer, msg map[string]interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	length := uint32(len(data))
	if err := binary.Write(w, binary.LittleEndian, length); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// readNativeMessage reads a length-prefixed JSON message from a reader
func readNativeMessage(r io.Reader) (map[string]interface{}, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func TestSendFraming(t *testing.T) {
	var buf bytes.Buffer
	// Use a pipe for input (won't be read in this test)
	inputR, inputW := io.Pipe()
	defer inputW.Close()

	transport := NewStdTransport(inputR, &buf)
	defer transport.Close()

	msg := map[string]interface{}{"name": "list_tabs"}
	if err := transport.Send(msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Read the length header
	var length uint32
	if err := binary.Read(&buf, binary.LittleEndian, &length); err != nil {
		t.Fatalf("failed to read length: %v", err)
	}

	// Read the JSON body
	data := make([]byte, length)
	if _, err := io.ReadFull(&buf, data); err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if decoded["name"] != "list_tabs" {
		t.Errorf("name: got %v, want list_tabs", decoded["name"])
	}
}

func TestRecvFraming(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	// Write a message to the input pipe
	msg := map[string]interface{}{"result": "hello"}
	go func() {
		writeNativeMessage(inputW, msg)
	}()

	received, err := transport.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received["result"] != "hello" {
		t.Errorf("result: got %v, want hello", received["result"])
	}
}

func TestEOFDetection(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	// Close the writer to trigger EOF
	inputW.Close()

	_, err := transport.Recv(context.Background())
	if err == nil {
		t.Fatal("expected error on EOF, got nil")
	}

	// Should be a TransportError
	if _, ok := err.(*errors.TransportError); !ok {
		t.Errorf("expected *TransportError, got %T: %v", err, err)
	}
}

func TestMaxMessageSize(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	// Write a length header claiming more than MaxMessageSize
	go func() {
		length := uint32(MaxMessageSize + 1)
		binary.Write(inputW, binary.LittleEndian, length)
		inputW.Close()
	}()

	_, err := transport.Recv(context.Background())
	if err == nil {
		t.Fatal("expected error for oversized message, got nil")
	}

	transportErr, ok := err.(*errors.TransportError)
	if !ok {
		t.Fatalf("expected *TransportError, got %T", err)
	}
	if transportErr.Message == "" {
		t.Error("expected non-empty error message")
	}
}

func TestPingFiltering(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	// Write a ping message followed by a real message
	go func() {
		writeNativeMessage(inputW, map[string]interface{}{"type": "ping"})
		writeNativeMessage(inputW, map[string]interface{}{"result": "real_data"})
	}()

	// Recv should skip the ping and return the real message
	received, err := transport.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received["result"] != "real_data" {
		t.Errorf("expected real_data, got %v", received["result"])
	}

	// Check that a pong was written to output
	pong, err := readNativeMessage(&outputBuf)
	if err != nil {
		t.Fatalf("failed to read pong: %v", err)
	}
	if pong["type"] != "pong" {
		t.Errorf("expected pong response, got %v", pong["type"])
	}
}

func TestHealthCheckFiltering(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	// Write a health check followed by a real message
	go func() {
		writeNativeMessage(inputW, map[string]interface{}{"type": "health_check"})
		writeNativeMessage(inputW, map[string]interface{}{"result": "after_health"})
	}()

	// Recv should skip the health check and return the real message
	received, err := transport.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received["result"] != "after_health" {
		t.Errorf("expected after_health, got %v", received["result"])
	}

	// Check the health check response
	response, err := readNativeMessage(&outputBuf)
	if err != nil {
		t.Fatalf("failed to read health check response: %v", err)
	}
	if response["type"] != "health_check_response" {
		t.Errorf("expected health_check_response, got %v", response["type"])
	}
	if response["status"] != "alive" {
		t.Errorf("expected status alive, got %v", response["status"])
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	inputR, inputW := io.Pipe()
	defer inputW.Close()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)

	// Calling Close multiple times should not panic
	if err := transport.Close(); err != nil {
		t.Errorf("first Close returned error: %v", err)
	}
	if err := transport.Close(); err != nil {
		t.Errorf("second Close returned error: %v", err)
	}
}

func TestMultipleMessages(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	go func() {
		writeNativeMessage(inputW, map[string]interface{}{"seq": float64(1)})
		writeNativeMessage(inputW, map[string]interface{}{"seq": float64(2)})
		writeNativeMessage(inputW, map[string]interface{}{"seq": float64(3)})
	}()

	for i := float64(1); i <= 3; i++ {
		msg, err := transport.Recv(context.Background())
		if err != nil {
			t.Fatalf("Recv %v failed: %v", i, err)
		}
		if msg["seq"] != i {
			t.Errorf("expected seq %v, got %v", i, msg["seq"])
		}
	}
}
