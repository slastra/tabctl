package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// ConnectionPool manages HTTP clients for multiple mediators
type ConnectionPool struct {
	mu      sync.RWMutex
	clients map[string]*PooledClient
	config  *PoolConfig
}

// PoolConfig contains configuration for connection pooling
type PoolConfig struct {
	MaxIdleConns        int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration
	HealthCheckInterval time.Duration
	RequestTimeout      time.Duration
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxIdleConns:        10,
		MaxConnsPerHost:     10,
		IdleConnTimeout:     90 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		RequestTimeout:      10 * time.Second,
	}
}

// PooledClient represents a pooled HTTP client with health checking
type PooledClient struct {
	client      *resty.Client
	host        string
	port        int
	lastHealthy time.Time
	failures    int
	mu          sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PoolConfig) *ConnectionPool {
	if config == nil {
		config = DefaultPoolConfig()
	}
	return &ConnectionPool{
		clients: make(map[string]*PooledClient),
		config:  config,
	}
}

// GetClient returns a client for the given host and port
func (p *ConnectionPool) GetClient(host string, port int) (*resty.Client, error) {
	key := fmt.Sprintf("%s:%d", host, port)

	p.mu.RLock()
	pooled, exists := p.clients[key]
	p.mu.RUnlock()

	if !exists {
		pooled = p.createClient(host, port)
		p.mu.Lock()
		p.clients[key] = pooled
		p.mu.Unlock()
	}

	// Check if circuit breaker is open
	if pooled.IsCircuitOpen() {
		return nil, fmt.Errorf("circuit breaker open for %s:%d (failures: %d)", host, port, pooled.failures)
	}

	return pooled.client, nil
}

// createClient creates a new pooled client
func (p *ConnectionPool) createClient(host string, port int) *PooledClient {
	client := resty.New()
	client.SetTimeout(p.config.RequestTimeout)
	client.SetBaseURL(fmt.Sprintf("http://%s:%d", host, port))

	// Configure connection pooling
	// Note: go-resty v2 handles transport configuration differently
	// These settings are handled automatically by the HTTP client

	// Add request/response middleware for metrics
	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		req.SetHeader("User-Agent", "tabctl/1.0")
		return nil
	})

	pooled := &PooledClient{
		client:      client,
		host:        host,
		port:        port,
		lastHealthy: time.Now(),
		failures:    0,
	}

	// Start health check goroutine
	go p.healthCheck(pooled)

	return pooled
}

// healthCheck performs periodic health checks
func (p *ConnectionPool) healthCheck(pooled *PooledClient) {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		if err := pooled.CheckHealth(); err != nil {
			pooled.RecordFailure()
		} else {
			pooled.RecordSuccess()
		}
	}
}

// CheckHealth checks if the mediator is healthy
func (pc *PooledClient) CheckHealth() error {
	resp, err := pc.client.R().Get("/")
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode())
	}
	return nil
}

// RecordFailure records a connection failure
func (pc *PooledClient) RecordFailure() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.failures++
}

// RecordSuccess records a successful connection
func (pc *PooledClient) RecordSuccess() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.failures = 0
	pc.lastHealthy = time.Now()
}

// IsCircuitOpen checks if circuit breaker is open
func (pc *PooledClient) IsCircuitOpen() bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	// Circuit opens after 3 consecutive failures
	if pc.failures >= 3 {
		// Allow retry after 30 seconds
		if time.Since(pc.lastHealthy) > 30*time.Second {
			pc.mu.RUnlock()
			pc.mu.Lock()
			pc.failures = 0 // Reset for retry
			pc.mu.Unlock()
			pc.mu.RLock()
			return false
		}
		return true
	}
	return false
}

// Close closes all connections in the pool
func (p *ConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Resty clients don't need explicit closing, but we clear the map
	p.clients = make(map[string]*PooledClient)
}

// RemoveClient removes a client from the pool
func (p *ConnectionPool) RemoveClient(host string, port int) {
	key := fmt.Sprintf("%s:%d", host, port)
	p.mu.Lock()
	delete(p.clients, key)
	p.mu.Unlock()
}

// GetHealthyClients returns all healthy clients
func (p *ConnectionPool) GetHealthyClients() []*PooledClient {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var healthy []*PooledClient
	for _, client := range p.clients {
		if !client.IsCircuitOpen() {
			healthy = append(healthy, client)
		}
	}
	return healthy
}