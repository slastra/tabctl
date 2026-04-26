package mediator

import "testing"

func TestNewCommand(t *testing.T) {
	args := map[string]interface{}{"tab_id": 123}
	cmd := NewCommand("activate_tab", args)

	if cmd.Command != "activate_tab" {
		t.Errorf("Command: got %q, want %q", cmd.Command, "activate_tab")
	}
	if cmd.Args["tab_id"] != 123 {
		t.Errorf("Args[tab_id]: got %v, want 123", cmd.Args["tab_id"])
	}
}

func TestNewCommandNilArgs(t *testing.T) {
	cmd := NewCommand("list_tabs", nil)

	if cmd.Command != "list_tabs" {
		t.Errorf("Command: got %q, want %q", cmd.Command, "list_tabs")
	}
	if cmd.Args != nil {
		t.Errorf("Args: got %v, want nil", cmd.Args)
	}
}
