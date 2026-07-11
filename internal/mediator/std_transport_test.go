package mediator

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/tabctl/tabctl/internal/errors"
)

// writeNativeMessage writes a length-prefixed JSON message to a writer.
func writeNativeMessage(w io.Writer, msg map[string]interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// recvDecoded reads one message from the transport's Incoming channel and
// decodes it, or fails the test on timeout.
func recvDecoded(t *testing.T, transport *StdTransport) map[string]interface{} {
	t.Helper()
	select {
	case raw, ok := <-transport.Incoming():
		if !ok {
			t.Fatal("Incoming channel closed unexpectedly")
		}
		var m map[string]interface{}
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		return m
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
		return nil
	}
}

func TestSendFraming(t *testing.T) {
	var buf bytes.Buffer
	inputR, inputW := io.Pipe()
	defer inputW.Close()

	transport := NewStdTransport(inputR, &buf)
	defer transport.Close()

	if err := transport.Send(map[string]interface{}{"method": "list_tabs"}); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	var length uint32
	if err := binary.Read(&buf, binary.LittleEndian, &length); err != nil {
		t.Fatalf("failed to read length: %v", err)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(&buf, data); err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if decoded["method"] != "list_tabs" {
		t.Errorf("method: got %v, want list_tabs", decoded["method"])
	}
}

func TestIncomingFraming(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	go writeNativeMessage(inputW, map[string]interface{}{"result": "hello"})

	received := recvDecoded(t, transport)
	if received["result"] != "hello" {
		t.Errorf("result: got %v, want hello", received["result"])
	}
}

func TestEOFDetection(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	inputW.Close()

	select {
	case err := <-transport.Errors():
		if _, ok := err.(*errors.TransportError); !ok {
			t.Errorf("expected *TransportError, got %T: %v", err, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for EOF error")
	}
}

func TestMaxMessageSize(t *testing.T) {
	inputR, inputW := io.Pipe()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)
	defer transport.Close()

	go func() {
		binary.Write(inputW, binary.LittleEndian, uint32(MaxMessageSize+1))
		inputW.Close()
	}()

	select {
	case err := <-transport.Errors():
		te, ok := err.(*errors.TransportError)
		if !ok {
			t.Fatalf("expected *TransportError, got %T", err)
		}
		if te.Message == "" {
			t.Error("expected non-empty error message")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for oversize error")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	inputR, inputW := io.Pipe()
	defer inputW.Close()
	var outputBuf bytes.Buffer

	transport := NewStdTransport(inputR, &outputBuf)

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
		msg := recvDecoded(t, transport)
		if msg["seq"] != i {
			t.Errorf("expected seq %v, got %v", i, msg["seq"])
		}
	}
}
