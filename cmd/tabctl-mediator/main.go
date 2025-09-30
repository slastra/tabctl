package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/tabctl/tabctl/internal/mediator"
)

func main() {
	var logFile string
	flag.StringVar(&logFile, "log", "", "Log file path (default: stderr)")
	flag.Parse()

	// Detect browser from arguments passed by native messaging
	browser := detectBrowser()

	// Always log to file for debugging browser lifecycle
	if logFile == "" {
		// Use timestamp in filename to track different sessions
		logFile = fmt.Sprintf("/tmp/tabctl-mediator-%d.log", os.Getpid())
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		os.Exit(1)
	}
	defer file.Close()
	log.SetOutput(file)

	// Always log startup and PID for debugging
	log.Printf("Starting mediator for %s (pid=%d)", browser, os.Getpid())

	// Create mediator
	m, err := mediator.NewMediator(browser)
	if err != nil {
		log.Fatalf("Failed to create mediator: %v", err)
	}

	// Setup signal handling - catch ALL signals for debugging
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGPIPE)

	log.Printf("Signal handlers registered at %s", time.Now().Format("15:04:05.000"))

	// Run mediator in goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting mediator.Run() at %s", time.Now().Format("15:04:05.000"))
		err := m.Run()
		log.Printf("mediator.Run() returned: err=%v at %s", err, time.Now().Format("15:04:05.000"))
		errChan <- err
	}()

	// Wait for signal or error
	log.Printf("Main thread waiting for signals or errors...")
	select {
	case sig := <-sigChan:
		// Signal received, initiate shutdown
		log.Printf("SIGNAL RECEIVED: %v at %s", sig, time.Now().Format("15:04:05.000"))
	case err := <-errChan:
		if err != nil {
			log.Printf("Mediator error: %v at %s", err, time.Now().Format("15:04:05.000"))
		} else {
			log.Printf("Mediator exited normally at %s", time.Now().Format("15:04:05.000"))
		}
	}

	// Shutdown
	log.Printf("Beginning shutdown sequence at %s", time.Now().Format("15:04:05.000"))
	if err := m.Shutdown(); err != nil {
		// Only log actual errors during shutdown
		if !strings.Contains(err.Error(), "use of closed") {
			log.Printf("Shutdown error: %v at %s", err, time.Now().Format("15:04:05.000"))
		}
	}
	log.Printf("Shutdown complete, exiting at %s", time.Now().Format("15:04:05.000"))
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