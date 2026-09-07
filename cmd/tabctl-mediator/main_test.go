package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// Profiles of one browser share a log file, so several mediators can rotate at
// once. Exactly one generation must survive: the pre-2.3.0 stat/rename pair
// let a second mediator rename the empty file the first had just created over
// the 8MB the first had just saved, losing both.
func TestOpenLogFileConcurrentRotationKeepsOneGeneration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mediator-chrome.log")

	oversized := make([]byte, maxLogSize+1)
	copy(oversized, "the run that caused the growth")
	if err := os.WriteFile(path, oversized, 0600); err != nil {
		t.Fatal(err)
	}

	const mediators = 8
	var wg sync.WaitGroup
	files := make([]*os.File, mediators)
	errs := make([]error, mediators)
	start := make(chan struct{})
	for i := range mediators {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			files[i], errs[i] = openLogFile(path)
		}()
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("mediator %d: %v", i, err)
		}
		files[i].Close()
	}

	prev, err := os.Stat(path + ".1")
	if err != nil {
		t.Fatalf("previous generation lost to a concurrent rotation: %v", err)
	}
	if prev.Size() != maxLogSize+1 {
		t.Errorf("previous generation is %d bytes, want the %d that were saved", prev.Size(), maxLogSize+1)
	}
	if _, err := os.Stat(path + ".1.1"); err == nil {
		t.Error("rotated a generation that had already been rotated")
	}
}

// Whichever mediator loses the race must still end up writing to the live log
// rather than to the inode that was just moved aside.
func TestOpenLogFileWritesReachTheLiveLogAfterRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mediator-chrome.log")

	if err := os.WriteFile(path, make([]byte, maxLogSize+1), 0600); err != nil {
		t.Fatal(err)
	}

	first, err := openLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()

	// A sibling starting after the rotation sees a small log and must not
	// rotate again; both handles then describe the same live file.
	second, err := openLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	if _, err := second.WriteString("later mediator\n"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "later mediator") {
		t.Errorf("live log = %q, want the later mediator's line", got)
	}
}
