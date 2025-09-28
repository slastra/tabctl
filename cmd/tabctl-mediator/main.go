package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/mediator"
	"golang.org/x/sys/unix"
)

func main() {
	var (
		port    int
		host    string
		logFile string
		debug   bool
	)

	flag.IntVar(&port, "port", 0, "Port to listen on (default: auto-detect from 4625-4627)")
	flag.StringVar(&host, "host", "localhost", "Host to bind to")
	flag.StringVar(&logFile, "log", "", "Log file path (default: stderr)")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	// Always redirect logs when stdin is not a terminal (native messaging mode)
	// to avoid corrupting the native messaging protocol
	if !isTerminal(os.Stdin.Fd()) {
		if logFile == "" {
			logFile = "/tmp/tabctl-mediator.log"
		}
	}

	// Setup logging
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Exit silently to avoid corrupting stdio if in native messaging mode
			os.Exit(1)
		}
		defer file.Close()
		log.SetOutput(file)
		log.Printf("Logging to %s", logFile)
	}

	// Auto-detect port if not specified
	if port == 0 {
		port = findAvailablePort()
		if port == 0 {
			log.Fatal("No available ports found (tried 4625-4627)")
		}
	}

	log.Printf("Starting tabctl mediator on %s:%d (pid=%d)", host, port, os.Getpid())

	// Create mediator config
	cfg := &config.MediatorConfig{
		Host:  host,
		Port:  port,
		Debug: debug,
	}

	// Create simplified mediator that handles Unix socket and stdio
	m, err := mediator.NewMediator(cfg)
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
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down", sig)
		if err := m.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	case err := <-errChan:
		if err != nil {
			log.Fatalf("Mediator error: %v", err)
		}
	}

	log.Println("Mediator shutdown complete")
}

// findAvailablePort finds an available port from the default range
func findAvailablePort() int {
	ports := []int{4625, 4626, 4627}
	for _, port := range ports {
		if !isPortInUse(port) {
			return port
		}
	}
	return 0
}

// isPortInUse checks if a port is already in use
func isPortInUse(port int) bool {
	// TODO: Implement actual port checking
	// For now, return false to allow port selection
	return false
}

// isTerminal checks if a file descriptor is a terminal
func isTerminal(fd uintptr) bool {
	_, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	return err == nil
}