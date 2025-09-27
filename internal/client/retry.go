package client

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// RetryConfig contains retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Jitter          bool
	RetryableErrors []int // HTTP status codes to retry
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:      3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		BackoffFactor:   2.0,
		Jitter:          true,
		RetryableErrors: []int{500, 502, 503, 504, 429}, // Server errors and rate limiting
	}
}

// RetryClient wraps a Resty client with retry logic
type RetryClient struct {
	client *resty.Client
	config *RetryConfig
}

// NewRetryClient creates a new retry client
func NewRetryClient(client *resty.Client, config *RetryConfig) *RetryClient {
	if config == nil {
		config = DefaultRetryConfig()
	}
	return &RetryClient{
		client: client,
		config: config,
	}
}

// Execute performs a request with retry logic
func (rc *RetryClient) Execute(ctx context.Context, reqFunc func() (*resty.Response, error)) (*resty.Response, error) {
	var lastErr error
	delay := rc.config.InitialDelay

	for attempt := 0; attempt <= rc.config.MaxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute request
		resp, err := reqFunc()

		// Success case
		if err == nil && (resp == nil || !rc.shouldRetry(resp.StatusCode())) {
			return resp, nil
		}

		// Record error for final return
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
		} else {
			lastErr = fmt.Errorf("attempt %d: HTTP %d", attempt+1, resp.StatusCode())
		}

		// Don't sleep after last attempt
		if attempt < rc.config.MaxRetries {
			// Apply jitter if configured
			actualDelay := delay
			if rc.config.Jitter {
				actualDelay = rc.addJitter(delay)
			}

			select {
			case <-time.After(actualDelay):
				// Continue to next attempt
			case <-ctx.Done():
				return nil, ctx.Err()
			}

			// Calculate next delay with exponential backoff
			delay = rc.nextDelay(delay)
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// shouldRetry checks if a status code should be retried
func (rc *RetryClient) shouldRetry(statusCode int) bool {
	for _, code := range rc.config.RetryableErrors {
		if statusCode == code {
			return true
		}
	}
	return false
}

// nextDelay calculates the next delay with exponential backoff
func (rc *RetryClient) nextDelay(currentDelay time.Duration) time.Duration {
	nextDelay := time.Duration(float64(currentDelay) * rc.config.BackoffFactor)
	if nextDelay > rc.config.MaxDelay {
		return rc.config.MaxDelay
	}
	return nextDelay
}

// addJitter adds random jitter to the delay
func (rc *RetryClient) addJitter(delay time.Duration) time.Duration {
	// Add ±25% jitter
	jitter := rand.Float64()*0.5 - 0.25
	return time.Duration(float64(delay) * (1 + jitter))
}

// ExponentialBackoff performs exponential backoff with configurable parameters
type ExponentialBackoff struct {
	attempt       int
	initialDelay  time.Duration
	maxDelay      time.Duration
	backoffFactor float64
}

// NewExponentialBackoff creates a new exponential backoff
func NewExponentialBackoff(initialDelay, maxDelay time.Duration, factor float64) *ExponentialBackoff {
	return &ExponentialBackoff{
		attempt:       0,
		initialDelay:  initialDelay,
		maxDelay:      maxDelay,
		backoffFactor: factor,
	}
}

// Next returns the next delay and increments the attempt counter
func (eb *ExponentialBackoff) Next() time.Duration {
	delay := time.Duration(float64(eb.initialDelay) * math.Pow(eb.backoffFactor, float64(eb.attempt)))
	if delay > eb.maxDelay {
		delay = eb.maxDelay
	}
	eb.attempt++
	return delay
}

// Reset resets the backoff counter
func (eb *ExponentialBackoff) Reset() {
	eb.attempt = 0
}

// RetryWithBackoff performs an operation with exponential backoff
func RetryWithBackoff(ctx context.Context, operation func() error, maxRetries int) error {
	backoff := NewExponentialBackoff(100*time.Millisecond, 5*time.Second, 2.0)
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try operation
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		// Don't sleep after last attempt
		if i < maxRetries {
			delay := backoff.Next()
			select {
			case <-time.After(delay):
				// Continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// RequestDeduplicator prevents duplicate concurrent requests
type RequestDeduplicator struct {
	pending map[string]chan *dedupResult
	mu      sync.RWMutex
}

type dedupResult struct {
	resp *resty.Response
	err  error
}

// NewRequestDeduplicator creates a new request deduplicator
func NewRequestDeduplicator() *RequestDeduplicator {
	return &RequestDeduplicator{
		pending: make(map[string]chan *dedupResult),
	}
}

// Execute ensures only one request for a given key is in flight
func (rd *RequestDeduplicator) Execute(key string, reqFunc func() (*resty.Response, error)) (*resty.Response, error) {
	rd.mu.Lock()
	if ch, exists := rd.pending[key]; exists {
		// Request already in flight, wait for result
		rd.mu.Unlock()
		result := <-ch
		return result.resp, result.err
	}

	// Create channel for this request
	ch := make(chan *dedupResult, 1)
	rd.pending[key] = ch
	rd.mu.Unlock()

	// Execute request
	resp, err := reqFunc()
	result := &dedupResult{resp: resp, err: err}

	// Send result and cleanup
	rd.mu.Lock()
	delete(rd.pending, key)
	rd.mu.Unlock()

	ch <- result
	close(ch)

	return resp, err
}