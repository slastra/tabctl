package mediator

import (
	"testing"

	"github.com/tabctl/tabctl/internal/errors"
)

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

func TestValidateCommand(t *testing.T) {
	valid := &Command{Command: "list_tabs"}
	if err := ValidateCommand(valid); err != nil {
		t.Errorf("valid command returned error: %v", err)
	}

	empty := &Command{Command: ""}
	err := ValidateCommand(empty)
	if err == nil {
		t.Error("empty command name should return error")
	}
	if _, ok := err.(*errors.ValidationError); !ok {
		t.Errorf("expected *errors.ValidationError, got %T", err)
	}
}
