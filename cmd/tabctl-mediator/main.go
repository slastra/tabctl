package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

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

func openLogFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
}

// detectBrowser identifies which browser launched this mediator.
//
// Firefox/Zen pass their manifest path as argv[1], so we can identify them
// directly. Chromium-family browsers (Chrome, Chromium, Brave, Helium) only
// pass the chrome-extension:// origin, so we identify them by inspecting
// the parent process binary via /proc/<ppid>/exe.
func detectBrowser() string {
	for _, arg := range flag.Args() {
		switch {
		case strings.Contains(arg, "/.mozilla/"):
			return "Firefox"
		case strings.Contains(arg, "/.zen/"):
			return "Zen"
		}
	}

	if exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", os.Getppid())); err == nil {
		base := strings.ToLower(filepath.Base(exe))
		switch {
		case strings.Contains(base, "google-chrome"):
			return "Chrome"
		case strings.Contains(base, "chromium"):
			return "Chromium"
		case strings.Contains(base, "brave"):
			return "Brave"
		case strings.Contains(base, "helium"):
			return "Helium"
		case strings.Contains(base, "chrome"):
			return "Chrome"
		}
	}

	return "Unknown"
}
