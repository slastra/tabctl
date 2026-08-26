package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenLogFileRotatesOversizedLog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mediator-chrome.log")

	if err := os.WriteFile(path, make([]byte, maxLogSize+1), 0600); err != nil {
		t.Fatal(err)
	}

	f, err := openLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("current log missing after rotation: %v", err)
	}
	if fi.Size() != 0 {
		t.Errorf("current log is %d bytes, want a fresh empty file", fi.Size())
	}

	// The run that caused the growth must stay diagnosable.
	prev, err := os.Stat(path + ".1")
	if err != nil {
		t.Fatalf("previous generation missing: %v", err)
	}
	if prev.Size() != maxLogSize+1 {
		t.Errorf("previous generation is %d bytes, want %d", prev.Size(), maxLogSize+1)
	}
}

// A log under the cap must be appended to, not truncated, or every browser
// restart would discard the previous session's diagnostics.
func TestOpenLogFileAppendsBelowCap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mediator-firefox.log")

	if err := os.WriteFile(path, []byte("earlier session\n"), 0600); err != nil {
		t.Fatal(err)
	}

	f, err := openLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("later session\n"); err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "earlier session") || !strings.Contains(string(got), "later session") {
		t.Errorf("log content = %q, want both sessions", got)
	}
	if _, err := os.Stat(path + ".1"); err == nil {
		t.Error("rotated a log that was under the cap")
	}
}

// Creating the log must never be what stops the mediator from starting.
func TestOpenLogFileCreatesStateDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "mediator-brave.log")
	f, err := openLogFile(path)
	if err != nil {
		t.Fatalf("openLogFile: %v", err)
	}
	f.Close()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("log not created: %v", err)
	}
}
