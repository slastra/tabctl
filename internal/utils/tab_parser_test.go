package utils

import (
	"testing"

	"github.com/tabctl/tabctl/pkg/types"
)

func TestParseTabID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		prefix    string
		windowID  string
		tabIDNum  string
		expectErr bool
	}{
		{"valid chrome tab", "c.1.123", "c", "1", "123", false},
		{"valid firefox tab", "f.5.999", "f", "5", "999", false},
		{"valid unknown prefix", "u.0.0", "u", "0", "0", false},
		{"too few parts", "bad", "", "", "", true},
		{"two parts only", "c.1", "", "", "", true},
		{"too many parts", "a.b.c.d", "", "", "", true},
		{"empty string", "", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, windowID, tabIDNum, err := ParseTabID(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
				return
			}
			if prefix != tt.prefix {
				t.Errorf("prefix: got %q, want %q", prefix, tt.prefix)
			}
			if windowID != tt.windowID {
				t.Errorf("windowID: got %q, want %q", windowID, tt.windowID)
			}
			if tabIDNum != tt.tabIDNum {
				t.Errorf("tabIDNum: got %q, want %q", tabIDNum, tt.tabIDNum)
			}
		})
	}
}

func TestParseTabLine(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  types.Tab
		expectErr bool
	}{
		{
			"full 6-field line",
			"c.1.123\tGoogle\thttps://google.com\t0\ttrue\tfalse",
			types.Tab{ID: "c.1.123", Title: "Google", URL: "https://google.com", WindowID: 1, Index: 0, Active: true, Pinned: false},
			false,
		},
		{
			"minimal 3-field line",
			"f.5.42\tExample\thttps://example.com",
			types.Tab{ID: "f.5.42", Title: "Example", URL: "https://example.com", WindowID: 5},
			false,
		},
		{
			"4 fields with index",
			"c.2.10\tTitle\thttps://url.com\t3",
			types.Tab{ID: "c.2.10", Title: "Title", URL: "https://url.com", WindowID: 2, Index: 3},
			false,
		},
		{
			"too few fields",
			"c.1.123\tTitle",
			types.Tab{},
			true,
		},
		{
			"invalid tab ID format",
			"bad\tTitle\thttps://url.com",
			types.Tab{},
			true,
		},
		{
			"non-numeric window ID",
			"c.abc.123\tTitle\thttps://url.com",
			types.Tab{},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tab, err := ParseTabLine(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tab.ID != tt.expected.ID {
				t.Errorf("ID: got %q, want %q", tab.ID, tt.expected.ID)
			}
			if tab.Title != tt.expected.Title {
				t.Errorf("Title: got %q, want %q", tab.Title, tt.expected.Title)
			}
			if tab.URL != tt.expected.URL {
				t.Errorf("URL: got %q, want %q", tab.URL, tt.expected.URL)
			}
			if tab.WindowID != tt.expected.WindowID {
				t.Errorf("WindowID: got %d, want %d", tab.WindowID, tt.expected.WindowID)
			}
			if tab.Index != tt.expected.Index {
				t.Errorf("Index: got %d, want %d", tab.Index, tt.expected.Index)
			}
			if tab.Active != tt.expected.Active {
				t.Errorf("Active: got %v, want %v", tab.Active, tt.expected.Active)
			}
			if tab.Pinned != tt.expected.Pinned {
				t.Errorf("Pinned: got %v, want %v", tab.Pinned, tt.expected.Pinned)
			}
		})
	}
}

func TestFormatTabLine(t *testing.T) {
	tab := types.Tab{ID: "c.1.2", Title: "Test", URL: "https://test.com"}
	got := FormatTabLine(tab)
	want := "c.1.2\tTest\thttps://test.com"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSplitTabIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"space separated", "c.1.1 c.1.2", []string{"c.1.1", "c.1.2"}},
		{"newline separated", "c.1.1\nc.1.2", []string{"c.1.1", "c.1.2"}},
		{"tab separated", "c.1.1\tc.1.2", []string{"c.1.1", "c.1.2"}},
		{"mixed separators", "c.1.1 c.1.2\nf.1.3\tf.2.4", []string{"c.1.1", "c.1.2", "f.1.3", "f.2.4"}},
		{"extra whitespace", "  c.1.1   c.1.2  ", []string{"c.1.1", "c.1.2"}},
		{"empty string", "", []string{}},
		{"single ID", "c.1.1", []string{"c.1.1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitTabIDs(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("length: got %d, want %d", len(got), len(tt.expected))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGroupTabsByPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string][]string
	}{
		{
			"mixed browsers",
			[]string{"c.1.1", "c.1.2", "f.1.3"},
			map[string][]string{
				"c.": {"c.1.1", "c.1.2"},
				"f.": {"f.1.3"},
			},
		},
		{
			"single browser",
			[]string{"f.1.1", "f.2.2"},
			map[string][]string{
				"f.": {"f.1.1", "f.2.2"},
			},
		},
		{
			"empty input",
			[]string{},
			map[string][]string{},
		},
		{
			"invalid prefix skipped",
			[]string{"c.1.1", "x"},
			map[string][]string{
				"c.": {"c.1.1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupTabsByPrefix(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("group count: got %d, want %d", len(got), len(tt.expected))
				return
			}
			for prefix, expectedIDs := range tt.expected {
				gotIDs, ok := got[prefix]
				if !ok {
					t.Errorf("missing prefix %q", prefix)
					continue
				}
				if len(gotIDs) != len(expectedIDs) {
					t.Errorf("prefix %q: got %d IDs, want %d", prefix, len(gotIDs), len(expectedIDs))
					continue
				}
				for i := range gotIDs {
					if gotIDs[i] != expectedIDs[i] {
						t.Errorf("prefix %q index %d: got %q, want %q", prefix, i, gotIDs[i], expectedIDs[i])
					}
				}
			}
		})
	}
}

func TestValidateTabID(t *testing.T) {
	if err := ValidateTabID("c.1.123"); err != nil {
		t.Errorf("valid ID returned error: %v", err)
	}
	if err := ValidateTabID("bad"); err == nil {
		t.Error("invalid ID returned nil error")
	}
}

func TestGetTabPrefix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"c.1.123", "c."},
		{"f.5.42", "f."},
		{"x", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := GetTabPrefix(tt.input)
		if got != tt.expected {
			t.Errorf("GetTabPrefix(%q): got %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestGetWindowID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"c.1.123", "1"},
		{"f.5.42", "5"},
		{"nodots", ""},
	}

	for _, tt := range tests {
		got := GetWindowID(tt.input)
		if got != tt.expected {
			t.Errorf("GetWindowID(%q): got %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMakeTabID(t *testing.T) {
	got := MakeTabID("c", "1", "123")
	want := "c.1.123"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParsePrefixAndWindowID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		prefix   string
		windowID string
		expectErr bool
	}{
		{"prefix and window", "c.1", "c.", "1", false},
		{"prefix only with dot", "c.", "c.", "", false},
		{"prefix only no dot", "c", "c.", "", false},
		{"full tab ID", "c.1.123", "c.", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, windowID, err := ParsePrefixAndWindowID(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if prefix != tt.prefix {
				t.Errorf("prefix: got %q, want %q", prefix, tt.prefix)
			}
			if windowID != tt.windowID {
				t.Errorf("windowID: got %q, want %q", windowID, tt.windowID)
			}
		})
	}
}

func TestFilterTabsByURL(t *testing.T) {
	tabs := []types.Tab{
		{ID: "c.1.1", URL: "https://google.com"},
		{ID: "c.1.2", URL: "https://github.com"},
		{ID: "c.1.3", URL: "https://example.com"},
	}

	got := FilterTabsByURL(tabs, "github")
	if len(got) != 1 || got[0].ID != "c.1.2" {
		t.Errorf("expected 1 match for 'github', got %d", len(got))
	}

	got = FilterTabsByURL(tabs, "notfound")
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}

func TestFilterTabsByTitle(t *testing.T) {
	tabs := []types.Tab{
		{ID: "c.1.1", Title: "Google Search"},
		{ID: "c.1.2", Title: "GitHub"},
		{ID: "c.1.3", Title: "Example Page"},
	}

	got := FilterTabsByTitle(tabs, "Git")
	if len(got) != 1 || got[0].ID != "c.1.2" {
		t.Errorf("expected 1 match for 'Git', got %d", len(got))
	}
}
