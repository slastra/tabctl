package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/tabctl/tabctl/internal/browsers"
	"github.com/tabctl/tabctl/internal/dbus"
	"github.com/tabctl/tabctl/internal/mediator"
)

func main() {
	var logFile string
	flag.StringVar(&logFile, "log", "", "Log file path (default: $XDG_STATE_HOME/tabctl/mediator-<browser>.log)")
	flag.Parse()

	browser := detectBrowser()

	if logFile == "" {
		logFile = defaultLogPath(browser)
	}
	if file, err := openLogFile(logFile); err != nil {
		// Never die over logging: the browser owns our stdio, but stderr
		// still reaches the browser's console/journal.
		log.SetOutput(os.Stderr)
		log.Printf("Cannot open log file %s: %v; logging to stderr", logFile, err)
	} else {
		defer file.Close()
		log.SetOutput(file)
	}

	log.Printf("Starting mediator for %s (pid=%d)", browser, os.Getpid())

	m, err := mediator.NewMediator(browser)
	if err != nil {
		log.Fatalf("Failed to create mediator: %v", err)
	}
	if err := m.Start(); err != nil {
		log.Fatalf("Failed to start mediator: %v", err)
	}
	// Profiles of the same browser share one log file, so tag every line
	// with the instance that wrote it.
	log.SetPrefix("[" + m.BusName() + "] ")
	log.Printf("Serving on D-Bus as %s", dbus.ServiceName(m.BusName()))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGPIPE)

	errChan := make(chan error, 1)
	go func() { errChan <- m.Wait() }()

	select {
	case sig := <-sigChan:
		log.Printf("Signal received: %v", sig)
	case err := <-errChan:
		if err != nil {
			log.Printf("Mediator error: %v", err)
		} else {
			log.Printf("Browser disconnected, shutting down")
		}
	}

	if err := m.Shutdown(); err != nil && !strings.Contains(err.Error(), "use of closed") {
		log.Printf("Shutdown error: %v", err)
	}
}

// defaultLogPath returns the per-browser log location under the XDG state
// directory, so concurrent mediators don't interleave into one file.
func defaultLogPath(browser string) string {
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "tabctl-mediator.log")
		}
		stateDir = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateDir, "tabctl", fmt.Sprintf("mediator-%s.log", strings.ToLower(browser)))
}

// maxLogSize caps a single mediator log before it is rotated.
//
// The mediator normally writes two lines per browser session, so this is
// generous. It exists because a mediator that dies on startup is relaunched by
// the browser immediately and forever: the D-Bus name collision fixed in 2.2.0
// produced 284k start/fail cycles and a 39MB log over one month, on a machine
// whose owner never noticed. Any future startup failure has the same shape, so
// cap the damage rather than trusting that it cannot recur.
const maxLogSize = 8 << 20

// openLogFile opens the mediator log, rotating it first if it has grown past
// maxLogSize. One previous generation is kept as "<path>.1" so the run that
// caused the growth is still diagnosable.
//
// The log is named per browser, not per profile, so every profile's mediator
// shares this file and several can arrive here at once. Rotation therefore
// takes an exclusive lock on the file and re-checks that the inode it locked
// is still the one at path: without that, two mediators in the same startup
// loop both see an oversized log, and the second renames the empty file the
// first just created over the generation the first just saved, losing both.
//
// Every failure below is deliberately ignored. Logging must never stop the
// mediator from starting, and the worst case is an oversized log.
func openLogFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	rotated, err := rotate(f, path)
	if err != nil {
		return f, nil
	}
	return rotated, nil
}

// rotate renames path to "<path>.1" and returns a handle on the fresh file if
// f has outgrown maxLogSize. It returns an error when it did not rotate, so
// the caller keeps the handle it already has.
func rotate(f *os.File, path string) (*os.File, error) {
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return nil, err
	}
	// Closing f below releases the lock, so unlocking here would be wrong;
	// the only path that keeps f unlocks explicitly.
	unlock := func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }

	fi, err := f.Stat()
	if err != nil || fi.Size() <= maxLogSize {
		unlock()
		return nil, errNoRotation
	}
	// A sibling that rotated while we waited for the lock has already moved
	// our inode aside. Renaming now would clobber what it saved.
	if cur, err := os.Stat(path); err != nil || !os.SameFile(cur, fi) {
		unlock()
		return nil, errNoRotation
	}
	if err := os.Rename(path, path+".1"); err != nil {
		unlock()
		return nil, err
	}
	fresh, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		unlock()
		return nil, err
	}
	f.Close()
	return fresh, nil
}

// errNoRotation reports that the log did not need rotating. It is never shown
// to the user; openLogFile only distinguishes rotated from not.
var errNoRotation = errors.New("log does not need rotation")

// detectBrowser identifies which browser launched this mediator. The rules
// live in internal/browsers, alongside the table that decides where `tabctl
// install` writes each browser's manifest, so the two cannot disagree about
// what a browser is called.
func detectBrowser() string {
	return browsers.Detect(flag.Args())
}
