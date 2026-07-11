package dbus

import (
	"reflect"
	"testing"
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
