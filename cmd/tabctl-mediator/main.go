package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/tabctl/tabctl/internal/mediator"
)

func main() {
	var logFile string
	flag.StringVar(&logFile, "log", "", "Log file path (default: stderr)")
	flag.Parse()

	// Detect browser from arguments passed by native messaging
	browser := detectBrowser()

	// Setup logging
	if logFile == "" && !isTerminal(os.Stdin.Fd()) {
		// Default to file logging when in native messaging mode
		logFile = "/tmp/tabctl-mediator.log"
	}

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			os.Exit(1)
		}
		defer file.Close()
		log.SetOutput(file)
	}

	// Only log startup in debug mode
	if os.Getenv("TABCTL_DEBUG") != "" {
		log.Printf("Starting mediator for %s (pid=%d)", browser, os.Getpid())
	}

	// Create mediator
	m, err := mediator.NewMediator(browser)
	if err != nil {
		log.Fatalf("Failed to create mediator: %v", err)
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run mediator in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- m.Run()
	}()

	// Wait for signal or error
	select {
	case <-sigChan:
		// Signal received, initiate shutdown
	case err := <-errChan:
		if err != nil {
			log.Printf("Mediator error: %v", err)
		}
	}

	// Shutdown
	if err := m.Shutdown(); err != nil {
		// Only log actual errors during shutdown
		if !strings.Contains(err.Error(), "use of closed") {
			log.Printf("Shutdown error: %v", err)
		}
	}
}

func detectBrowser() string {
	// Check command line arguments for browser hints
	args := flag.Args()

	for _, arg := range args {
		// Chrome/Brave pass chrome-extension://
		if strings.HasPrefix(arg, "chrome-extension://") {
			return "Brave" // or detect specific Chrome variant
		}
		// Firefox passes manifest path
		if strings.Contains(arg, ".mozilla/native-messaging-hosts/") {
			return "Firefox"
		}
	}

	// Default
	return "Unknown"
}

func isTerminal(fd uintptr) bool {
	// Simple check if fd is a terminal
	_, err := os.Stdin.Stat()
	return err == nil
}