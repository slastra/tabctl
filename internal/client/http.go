package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tabctl/tabctl/pkg/types"
)

// HTTPClient implements the TabAPI interface using HTTP
type HTTPClient struct {
	pool         *ConnectionPool
	retryConfig  *RetryConfig
	deduplicator *RequestDeduplicator
	cache        *ResponseCache
	defaultHost  string
	defaultPort  int
}

// NewHTTPClient creates a new HTTP client with resilience features
func NewHTTPClient(host string, port int) *HTTPClient {
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = FindAvailablePort(host)
		if port == 0 {
			port = 4625 // Fallback to default
		}
	}

	return &HTTPClient{
		pool:         NewConnectionPool(DefaultPoolConfig()),
		retryConfig:  DefaultRetryConfig(),
		deduplicator: NewRequestDeduplicator(),
		cache:        NewResponseCache(5 * time.Minute),
		defaultHost:  host,
		defaultPort:  port,
	}
}

// getClient gets an HTTP client for the specified or default host/port
func (c *HTTPClient) getClient() (*resty.Client, error) {
	return c.pool.GetClient(c.defaultHost, c.defaultPort)
}

// executeWithRetry executes a request with retry logic
func (c *HTTPClient) executeWithRetry(ctx context.Context, method, path string, body interface{}) (*resty.Response, error) {
	client, err := c.getClient()
	if err != nil {
		return nil, err
	}

	retryClient := NewRetryClient(client, c.retryConfig)
	return retryClient.Execute(ctx, func() (*resty.Response, error) {
		req := client.R()
		if body != nil {
			req.SetBody(body)
		}

		switch method {
		case "GET":
			return req.Get(path)
		case "POST":
			return req.Post(path)
		case "PUT":
			return req.Put(path)
		case "DELETE":
			return req.Delete(path)
		default:
			return nil, fmt.Errorf("unsupported method: %s", method)
		}
	})
}

// ListTabs lists all tabs from the mediator
func (c *HTTPClient) ListTabs() ([]types.Tab, error) {
	ctx := context.Background()

	// Check cache
	if cached, ok := c.cache.Get("list_tabs"); ok {
		if tabs, ok := cached.([]types.Tab); ok {
			return tabs, nil
		}
	}

	resp, err := c.executeWithRetry(ctx, "GET", "/list_tabs", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tabs: %w", err)
	}

	// Parse TSV response
	lines := strings.Split(string(resp.Body()), "\n")
	var tabs []types.Tab

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		// Parse the tab ID - handle both formats:
		// - With prefix: a.123.456
		// - Without prefix (Python): 123.456
		var windowID, tabID int
		idParts := strings.Split(parts[0], ".")
		if len(idParts) == 3 {
			// Format: prefix.windowID.tabID
			windowID, _ = strconv.Atoi(idParts[1])
			tabID, _ = strconv.Atoi(idParts[2])
		} else if len(idParts) == 2 {
			// Format: windowID.tabID (Python brotab format)
			windowID, _ = strconv.Atoi(idParts[0])
			tabID, _ = strconv.Atoi(idParts[1])
		} else {
			continue
		}

		tab := types.Tab{
			ID:       tabID,
			WindowID: windowID,
			Title:    parts[1],
			URL:      parts[2],
		}
		tabs = append(tabs, tab)
	}

	// Cache the result
	c.cache.Set("list_tabs", tabs, 10*time.Second)

	return tabs, nil
}

// CloseTabs closes specified tabs
func (c *HTTPClient) CloseTabs(tabIDs []string) error {
	ctx := context.Background()

	// Join tab IDs for URL path
	tabIDsStr := strings.Join(tabIDs, ",")
	path := fmt.Sprintf("/close_tabs/%s", tabIDsStr)

	resp, err := c.executeWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed to close tabs: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("close tabs failed with status %d: %s", resp.StatusCode(), resp.String())
	}

	// Invalidate cache
	c.cache.Delete("list_tabs")

	return nil
}

// ActivateTab activates a specific tab
func (c *HTTPClient) ActivateTab(tabID string, focused bool) error {
	ctx := context.Background()

	path := fmt.Sprintf("/activate_tab/%s", tabID)
	if focused {
		path += "?focused=true"
	}

	resp, err := c.executeWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return fmt.Errorf("failed to activate tab: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("activate tab failed with status %d", resp.StatusCode())
	}

	return nil
}

// OpenURLs opens URLs in new tabs
func (c *HTTPClient) OpenURLs(urls []string, windowID string) ([]string, error) {
	ctx := context.Background()

	request := map[string]interface{}{
		"urls": urls,
	}
	if windowID != "" {
		request["window_id"] = windowID
	}

	resp, err := c.executeWithRetry(ctx, "POST", "/open_urls", request)
	if err != nil {
		return nil, fmt.Errorf("failed to open URLs: %w", err)
	}

	var tabIDs []string
	if err := json.Unmarshal(resp.Body(), &tabIDs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate cache
	c.cache.Delete("list_tabs")

	return tabIDs, nil
}

// UpdateTabs updates multiple tabs
func (c *HTTPClient) UpdateTabs(updates []types.TabUpdate) error {
	ctx := context.Background()

	resp, err := c.executeWithRetry(ctx, "POST", "/update_tabs", updates)
	if err != nil {
		return fmt.Errorf("failed to update tabs: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("update tabs failed with status %d", resp.StatusCode())
	}

	// Invalidate cache
	c.cache.Delete("list_tabs")

	return nil
}

// QueryTabs queries tabs with filters
func (c *HTTPClient) QueryTabs(query types.TabQuery) ([]types.Tab, error) {
	ctx := context.Background()

	// Build cache key from query
	cacheKey := fmt.Sprintf("query_%v", query)
	if cached, ok := c.cache.Get(cacheKey); ok {
		if tabs, ok := cached.([]types.Tab); ok {
			return tabs, nil
		}
	}

	resp, err := c.executeWithRetry(ctx, "POST", "/query_tabs", query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tabs: %w", err)
	}

	var tabs []types.Tab
	if err := json.Unmarshal(resp.Body(), &tabs); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache the result
	c.cache.Set(cacheKey, tabs, 30*time.Second)

	return tabs, nil
}

// GetText gets text content from tabs
func (c *HTTPClient) GetText(tabIDs []string) ([]types.TabContent, error) {
	ctx := context.Background()

	tabIDsStr := strings.Join(tabIDs, ",")
	path := fmt.Sprintf("/get_text/%s", tabIDsStr)

	resp, err := c.executeWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get text: %w", err)
	}

	var content []types.TabContent
	if err := json.Unmarshal(resp.Body(), &content); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return content, nil
}

// GetHTML gets HTML content from tabs
func (c *HTTPClient) GetHTML(tabIDs []string) ([]types.TabContent, error) {
	ctx := context.Background()

	tabIDsStr := strings.Join(tabIDs, ",")
	path := fmt.Sprintf("/get_html/%s", tabIDsStr)

	resp, err := c.executeWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	var content []types.TabContent
	if err := json.Unmarshal(resp.Body(), &content); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return content, nil
}

// GetWords gets words from all tabs for autocomplete
func (c *HTTPClient) GetWords(tabIDs []string) ([]string, error) {
	ctx := context.Background()

	// Check cache
	if cached, ok := c.cache.Get("words"); ok {
		if words, ok := cached.([]string); ok {
			return words, nil
		}
	}

	resp, err := c.executeWithRetry(ctx, "GET", "/get_words", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get words: %w", err)
	}

	var words []string
	if err := json.Unmarshal(resp.Body(), &words); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Cache for longer as words don't change often
	c.cache.Set("words", words, 5*time.Minute)

	return words, nil
}

// MoveTabs moves tabs to different positions
func (c *HTTPClient) MoveTabs(moves []types.TabMove) error {
	ctx := context.Background()

	resp, err := c.executeWithRetry(ctx, "POST", "/move_tabs", moves)
	if err != nil {
		return fmt.Errorf("failed to move tabs: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("move tabs failed with status %d", resp.StatusCode())
	}

	// Invalidate cache
	c.cache.Delete("list_tabs")

	return nil
}

// CreateWindow creates a new browser window
func (c *HTTPClient) CreateWindow(urls []string) (string, error) {
	ctx := context.Background()

	request := map[string]interface{}{
		"urls": urls,
	}

	resp, err := c.executeWithRetry(ctx, "POST", "/create_window", request)
	if err != nil {
		return "", fmt.Errorf("failed to create window: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Invalidate cache
	c.cache.Delete("list_tabs")

	return result["window_id"], nil
}

// GetScreenshot gets a screenshot of a tab
func (c *HTTPClient) GetScreenshot(tabID string) ([]byte, error) {
	ctx := context.Background()

	path := fmt.Sprintf("/get_screenshot/%s", tabID)

	resp, err := c.executeWithRetry(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get screenshot: %w", err)
	}

	return resp.Body(), nil
}

// Close closes the HTTP client and its connections
func (c *HTTPClient) Close() error {
	c.pool.Close()
	return nil
}

// ResponseCache provides simple caching for responses
type ResponseCache struct {
	cache map[string]*cacheEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// NewResponseCache creates a new response cache
func NewResponseCache(defaultTTL time.Duration) *ResponseCache {
	rc := &ResponseCache{
		cache: make(map[string]*cacheEntry),
		ttl:   defaultTTL,
	}

	// Start cleanup goroutine
	go rc.cleanup()

	return rc
}

// Get retrieves a cached value
func (rc *ResponseCache) Get(key string) (interface{}, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	entry, ok := rc.cache[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.value, true
}

// Set stores a value in the cache
func (rc *ResponseCache) Set(key string, value interface{}, ttl time.Duration) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache
func (rc *ResponseCache) Delete(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	delete(rc.cache, key)
}

// cleanup periodically removes expired entries
func (rc *ResponseCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		for key, entry := range rc.cache {
			if now.After(entry.expiresAt) {
				delete(rc.cache, key)
			}
		}
		rc.mu.Unlock()
	}
}