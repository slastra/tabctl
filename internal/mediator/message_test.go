package mediator

import (
	"encoding/json"
	"testing"
)

func TestNewRequest(t *testing.T) {
	req := newRequest(7, MethodActivateTab, map[string]interface{}{"tab_id": 123})

	if req.JSONRPC != JSONRPCVersion {
		t.Errorf("JSONRPC: got %q, want %q", req.JSONRPC, JSONRPCVersion)
	}
	if req.ID != 7 || req.Method != MethodActivateTab {
		t.Errorf("unexpected request: %+v", req)
	}
	if req.Params["tab_id"] != 123 {
		t.Errorf("Params[tab_id]: got %v, want 123", req.Params["tab_id"])
	}
}

func TestRequestMarshalsJSONRPC(t *testing.T) {
	req := newRequest(1, MethodListTabs, nil)
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if m["jsonrpc"] != "2.0" || m["method"] != "list_tabs" {
		t.Errorf("unexpected wire form: %s", b)
	}
	// params omitted when nil
	if _, ok := m["params"]; ok {
		t.Errorf("params should be omitted when nil: %s", b)
	}
}
