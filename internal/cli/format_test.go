package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/tabctl/tabctl/pkg/types"
)

var sampleTabs = []types.Tab{
	{ID: "firefox.1.10", Title: "One", URL: "https://one.example", WindowID: 1, Index: 0, Active: true},
	{ID: "firefox.1.11", Title: "Two", URL: "https://two.example", WindowID: 1, Index: 1},
}

func TestWriteTSV(t *testing.T) {
	var buf bytes.Buffer
	if err := writeTabList(&buf, sampleTabs, "tsv", "\t"); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.HasPrefix(got, "firefox.1.10\tOne\thttps://one.example\n") {
		t.Errorf("unexpected TSV: %q", got)
	}
	if strings.Count(got, "\n") != 2 {
		t.Errorf("expected 2 lines, got %q", got)
	}
}

func TestWriteTSVCustomDelimiter(t *testing.T) {
	var buf bytes.Buffer
	if err := writeTabList(&buf, sampleTabs, "tsv", ","); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "firefox.1.10,One,https://one.example") {
		t.Errorf("custom delimiter not applied: %q", buf.String())
	}
}

func TestWriteJSONRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	if err := writeTabList(&buf, sampleTabs, "json", "\t"); err != nil {
		t.Fatal(err)
	}
	var back []types.Tab
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(back) != 2 || back[0].ID != "firefox.1.10" || back[1].Title != "Two" {
		t.Errorf("round-trip mismatch: %+v", back)
	}
}

func TestWriteSimpleTitlesOnly(t *testing.T) {
	var buf bytes.Buffer
	if err := writeTabList(&buf, sampleTabs, "simple", "\t"); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "One\nTwo\n" {
		t.Errorf("simple output: got %q, want \"One\\nTwo\\n\"", buf.String())
	}
}

func TestWriteEmpty(t *testing.T) {
	var buf bytes.Buffer
	writeTabList(&buf, nil, "tsv", "\t")
	if strings.TrimSpace(buf.String()) != "No tabs found" {
		t.Errorf("empty tsv: got %q", buf.String())
	}

	buf.Reset()
	writeTabList(&buf, nil, "json", "\t")
	if strings.TrimSpace(buf.String()) != "[]" {
		t.Errorf("empty json: got %q", buf.String())
	}
}
