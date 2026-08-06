package dbus

import (
	"fmt"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestInstanceName(t *testing.T) {
	tests := []struct {
		browser string
		n       int
		want    string
	}{
		{"Chrome", 1, "Chrome"},
		{"Chrome", 2, "Chrome2"},
		{"Chrome", 16, "Chrome16"},
		{"Firefox", 3, "Firefox3"},
		// Defensive: callers count from 1, but 0 must not yield "Chrome0".
		{"Chrome", 0, "Chrome"},
	}

	for _, tt := range tests {
		if got := InstanceName(tt.browser, tt.n); got != tt.want {
			t.Errorf("InstanceName(%q, %d) = %q, want %q", tt.browser, tt.n, got, tt.want)
		}
	}
}

func TestBaseBrowser(t *testing.T) {
	tests := []struct {
		instance string
		want     string
	}{
		{"Chrome", "Chrome"},
		{"Chrome2", "Chrome"},
		{"Chrome16", "Chrome"},
		{"Firefox", "Firefox"},
		{"", ""},
		// All-digit names have no browser part to recover; leave them alone
		// rather than returning empty.
		{"123", "123"},
	}

	for _, tt := range tests {
		if got := BaseBrowser(tt.instance); got != tt.want {
			t.Errorf("BaseBrowser(%q) = %q, want %q", tt.instance, got, tt.want)
		}
	}
}

// fakeBus answers RequestName from a set of names already owned by someone
// else, recording every name that was attempted.
type fakeBus struct {
	taken     map[string]bool
	requested []string
	err       error
}

func (f *fakeBus) requestName(name string, _ dbus.RequestNameFlags) (dbus.RequestNameReply, error) {
	f.requested = append(f.requested, name)
	if f.err != nil {
		return 0, f.err
	}
	if f.taken[name] {
		return dbus.RequestNameReplyExists, nil
	}
	f.taken[name] = true
	return dbus.RequestNameReplyPrimaryOwner, nil
}

func TestClaimName(t *testing.T) {
	t.Run("first mediator keeps the bare browser name", func(t *testing.T) {
		bus := &fakeBus{taken: map[string]bool{}}
		got, err := claimName(bus.requestName, "Chrome")
		if err != nil {
			t.Fatalf("claimName failed: %v", err)
		}
		if got != "Chrome" {
			t.Errorf("claimed %q, want %q", got, "Chrome")
		}
	})

	t.Run("second profile falls through to the next slot", func(t *testing.T) {
		bus := &fakeBus{taken: map[string]bool{ServiceName("Chrome"): true}}
		got, err := claimName(bus.requestName, "Chrome")
		if err != nil {
			t.Fatalf("claimName failed: %v", err)
		}
		if got != "Chrome2" {
			t.Errorf("claimed %q, want %q", got, "Chrome2")
		}
	})

	t.Run("three profiles get three distinct names", func(t *testing.T) {
		bus := &fakeBus{taken: map[string]bool{}}
		var claimed []string
		for i := 0; i < 3; i++ {
			name, err := claimName(bus.requestName, "Chrome")
			if err != nil {
				t.Fatalf("claim %d failed: %v", i, err)
			}
			claimed = append(claimed, name)
		}
		want := []string{"Chrome", "Chrome2", "Chrome3"}
		for i := range want {
			if claimed[i] != want[i] {
				t.Fatalf("claimed %v, want %v", claimed, want)
			}
		}
	})

	t.Run("a freed slot is reused", func(t *testing.T) {
		bus := &fakeBus{taken: map[string]bool{
			ServiceName("Chrome"):  true,
			ServiceName("Chrome2"): true,
		}}
		delete(bus.taken, ServiceName("Chrome")) // first profile's browser closed
		got, err := claimName(bus.requestName, "Chrome")
		if err != nil {
			t.Fatalf("claimName failed: %v", err)
		}
		if got != "Chrome" {
			t.Errorf("claimed %q, want the freed %q", got, "Chrome")
		}
	})

	t.Run("all slots taken", func(t *testing.T) {
		bus := &fakeBus{taken: map[string]bool{}}
		for i := 1; i <= MaxInstances; i++ {
			bus.taken[ServiceName(InstanceName("Chrome", i))] = true
		}
		_, err := claimName(bus.requestName, "Chrome")
		if err == nil {
			t.Fatal("expected an error when every instance name is taken")
		}
		if len(bus.requested) != MaxInstances {
			t.Errorf("tried %d names, want %d", len(bus.requested), MaxInstances)
		}
	})

	t.Run("bus error is not retried", func(t *testing.T) {
		bus := &fakeBus{taken: map[string]bool{}, err: fmt.Errorf("bus is gone")}
		_, err := claimName(bus.requestName, "Chrome")
		if err == nil {
			t.Fatal("expected the bus error to propagate")
		}
		if len(bus.requested) != 1 {
			t.Errorf("made %d requests, want 1 (a broken bus must not be walked)", len(bus.requested))
		}
	})
}
