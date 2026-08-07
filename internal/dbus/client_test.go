package dbus

import (
	"errors"
	"reflect"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestFilterBrowserNames(t *testing.T) {
	prefix := ServiceNameBase + "."

	tests := []struct {
		name  string
		names []string
		want  []string
	}{
		{
			name:  "single browser",
			names: []string{prefix + "Firefox"},
			want:  []string{"Firefox"},
		},
		{
			name:  "multiple browsers with noise",
			names: []string{"org.freedesktop.DBus", prefix + "Firefox", prefix + "Chrome", ":1.42"},
			want:  []string{"Firefox", "Chrome"},
		},
		{
			name:  "empty suffix and bare base excluded",
			names: []string{prefix, ServiceNameBase},
			want:  nil,
		},
		{
			name:  "unrelated names ignored",
			names: []string{"org.mozilla.firefox", "com.brave.Browser"},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterBrowserNames(tt.names)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterBrowserNames(%v) = %v, want %v", tt.names, got, tt.want)
			}
		})
	}
}

func TestIsUnknownMethod(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"unknown method", dbus.Error{Name: "org.freedesktop.DBus.Error.UnknownMethod"}, true},
		{"unknown interface", dbus.Error{Name: "org.freedesktop.DBus.Error.UnknownInterface"}, true},
		// A mediator that implements the method but fails inside it must NOT
		// trigger the fallback: that would hide a real error behind a silent
		// downgrade to iconless tabs.
		{"method failed", dbus.Error{Name: "org.freedesktop.DBus.Error.Failed"}, false},
		{"no reply", dbus.Error{Name: "org.freedesktop.DBus.Error.NoReply"}, false},
		{"not a dbus error", errors.New("boom"), false},
		{"nil", nil, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUnknownMethod(tt.err); got != tt.want {
				t.Errorf("isUnknownMethod(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
