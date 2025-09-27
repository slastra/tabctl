package client

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/tabctl/tabctl/pkg/api"
)

// MediatorConfig holds configuration for connecting to mediators
type MediatorConfig struct {
	TargetHosts string
	Timeout     int
}

// CreateClients creates HTTP clients for available mediators
func CreateClients(config MediatorConfig) ([]api.Client, error) {
	if config.TargetHosts != "" {
		return createClientsFromTargets(config.TargetHosts)
	}
	return createClientsFromDiscovery()
}

// createClientsFromTargets creates clients from explicit target hosts
func createClientsFromTargets(targetHosts string) ([]api.Client, error) {
	hosts, ports, err := parseTargetHosts(targetHosts)
	if err != nil {
		return nil, err
	}

	var clients []api.Client
	prefixes := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	for i, host := range hosts {
		if i >= len(prefixes) {
			break
		}

		port := ports[i]
		prefix := prefixes[i] + "."

		if IsPortAcceptingConnections(host, port) {
			client := NewClient(prefix, host, port)
			clients = append(clients, client)
		}
	}

	return clients, nil
}

// createClientsFromDiscovery discovers mediators on default ports
func createClientsFromDiscovery() ([]api.Client, error) {
	// TODO: Implement mediator port discovery
	// Default ports: 4625, 4626, 4627
	defaultPorts := []int{4625, 4626, 4627}
	host := "localhost"
	prefixes := []string{"a", "b", "c"}

	var clients []api.Client

	for i, port := range defaultPorts {
		if IsPortAcceptingConnections(host, port) {
			prefix := prefixes[i] + "."
			client := NewClient(prefix, host, port)
			clients = append(clients, client)
		}
	}

	return clients, nil
}

// parseTargetHosts parses target hosts string into hosts and ports
func parseTargetHosts(targetHosts string) ([]string, []int, error) {
	var hosts []string
	var ports []int

	pairs := strings.Split(targetHosts, ",")
	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			return nil, nil, fmt.Errorf("invalid host:port format: %s", pair)
		}

		host := strings.TrimSpace(parts[0])
		portStr := strings.TrimSpace(parts[1])

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port number: %s", portStr)
		}

		hosts = append(hosts, host)
		ports = append(ports, port)
	}

	return hosts, ports, nil
}


// GetMediatorPorts discovers available mediator ports
func GetMediatorPorts() ([]int, error) {
	// TODO: Implement mediator port discovery logic
	// This should scan for running mediator processes and return their ports
	return []int{4625, 4626, 4627}, nil
}

// CreateMultiClient creates a multi-client from configuration
func CreateMultiClient(config MediatorConfig) (api.MultiClient, error) {
	clients, err := CreateClients(config)
	if err != nil {
		return nil, err
	}

	if len(clients) == 0 {
		return nil, errors.New("no mediator clients available")
	}

	return api.CreateMultiClient(clients), nil
}