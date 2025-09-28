package client

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tabctl/tabctl/internal/config"
)

// PortStatus represents the status of a port
type PortStatus struct {
	Port      int
	Available bool
	Host      string
}

// DiscoverMediators discovers available mediator ports
func DiscoverMediators(host string) []PortStatus {
	if host == "" {
		host = "localhost"
	}

	ports := config.DefaultMediatorPorts
	results := make([]PortStatus, len(ports))

	// Check ports in parallel
	var wg sync.WaitGroup
	for i, port := range ports {
		wg.Add(1)
		go func(idx int, p int) {
			defer wg.Done()
			results[idx] = PortStatus{
				Port:      p,
				Host:      host,
				Available: IsPortAcceptingConnections(host, p),
			}
		}(i, port)
	}
	wg.Wait()

	return results
}

// IsPortAcceptingConnections checks if a port is accepting connections
func IsPortAcceptingConnections(host string, port int) bool {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	// Try to connect with a short timeout
	conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
	if err != nil {
		return false
	}

	// Successfully connected, close immediately
	conn.Close()
	return true
}

// IsUnixSocketAvailable checks if a Unix socket exists for the given port
func IsUnixSocketAvailable(port int) bool {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/tmp"
	}

	socketPath := filepath.Join(runtimeDir, fmt.Sprintf("tabctl-%d.sock", port))

	// Check if socket file exists
	info, err := os.Stat(socketPath)
	if err != nil {
		return false
	}

	// Verify it's a socket
	return info.Mode()&os.ModeSocket != 0
}

// FindAvailablePort finds the first available port from the default range
func FindAvailablePort(host string) int {
	if host == "" {
		host = "localhost"
	}

	// For Unix sockets, check if the socket file exists
	for _, port := range config.DefaultMediatorPorts {
		if IsUnixSocketAvailable(port) {
			return port
		}
	}

	return 0
}

// FindAllAvailablePorts returns all available mediator ports
func FindAllAvailablePorts(host string) []int {
	if host == "" {
		host = "localhost"
	}

	var available []int

	// For Unix sockets, check directly
	for _, port := range config.DefaultMediatorPorts {
		if IsUnixSocketAvailable(port) {
			available = append(available, port)
		}
	}

	return available
}

// MediatorInfo represents information about a discovered mediator
type MediatorInfo struct {
	Host    string
	Port    int
	Browser string
	PID     int
	Prefix  string
}

// DiscoverAllMediators discovers all available mediators with their info
func DiscoverAllMediators(host string) []MediatorInfo {
	if host == "" {
		host = "localhost"
	}

	var mediators []MediatorInfo
	ports := FindAllAvailablePorts(host)

	prefixes := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	for i, port := range ports {
		if i >= len(prefixes) {
			break
		}

		mediator := MediatorInfo{
			Host:   host,
			Port:   port,
			Prefix: prefixes[i] + ".",
		}

		// TODO: Query mediator for actual browser and PID info
		// For now, use defaults
		switch port {
		case 4625:
			mediator.Browser = "firefox"
		case 4626:
			mediator.Browser = "chrome"
		case 4627:
			mediator.Browser = "chromium"
		default:
			mediator.Browser = "unknown"
		}

		mediators = append(mediators, mediator)
	}

	return mediators
}

// WaitForPort waits for a port to become available
func WaitForPort(host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if IsPortAcceptingConnections(host, port) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("port %d not available after %v", port, timeout)
}

// CheckMediatorHealth checks if a mediator is healthy
func CheckMediatorHealth(host string, port int) error {
	if !IsPortAcceptingConnections(host, port) {
		return fmt.Errorf("mediator at %s:%d is not responding", host, port)
	}

	// TODO: Make actual health check Unix socket request
	return nil
}