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

	"github.com/tabctl/tabctl/internal/mediator"
)

func main() {
	var logFile string
	flag.StringVar(&logFile, "log", "/tmp/tabctl-mediator.log", "Log file path")
	flag.Parse()

	browser := detectBrowser()

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		os.Exit(1)
	}
	defer file.Close()
	log.SetOutput(file)

	log.Printf("Starting mediator for %s (pid=%d)", browser, os.Getpid())

	m, err := mediator.NewMediator(browser)
	if err != nil {
		log.Fatalf("Failed to create mediator: %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGPIPE)

	errChan := make(chan error, 1)
	go func() { errChan <- m.Run() }()

	select {
	case sig := <-sigChan:
		log.Printf("Signal received: %v", sig)
	case err := <-errChan:
		if err != nil {
			log.Printf("Mediator error: %v", err)
		}
	}

	if err := m.Shutdown(); err != nil && !strings.Contains(err.Error(), "use of closed") {
		log.Printf("Shutdown error: %v", err)
	}
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