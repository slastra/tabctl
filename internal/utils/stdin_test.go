package utils

import (
	"reflect"
	"strings"
	"testing"
)

func TestReadLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "a\nb\nc\n", []string{"a", "b", "c"}},
		{"trims whitespace", "  a  \n\tb\t\n", []string{"a", "b"}},
		{"skips blank lines", "a\n\n  \nb\n", []string{"a", "b"}},
		{"no trailing newline", "a\nb", []string{"a", "b"}},
		{"empty input", "", nil},
		{"only blanks", "\n  \n\t\n", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readLines(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("readLines error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readLines(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
