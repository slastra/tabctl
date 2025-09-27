package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/tabctl/tabctl/internal/utils"
	"github.com/tabctl/tabctl/pkg/api"
	"github.com/tabctl/tabctl/pkg/types"
)

// ParallelClient wraps multiple clients for parallel operations
type ParallelClient struct {
	clients []api.Client
}

// NewParallelClient creates a new parallel client from discovered mediators
func NewParallelClient(host string) *ParallelClient {
	if host == "" {
		host = "localhost"
	}

	mediators := DiscoverAllMediators(host)
	clients := make([]api.Client, 0, len(mediators))

	for _, mediator := range mediators {
		client := NewClient(mediator.Prefix, mediator.Host, mediator.Port)
		clients = append(clients, client)
	}

	return &ParallelClient{
		clients: clients,
	}
}

// GetClients returns all available clients
func (pc *ParallelClient) GetClients() []api.Client {
	return pc.clients
}

// ListAllTabs lists tabs from all browsers in parallel
func (pc *ParallelClient) ListAllTabs() ([]types.Tab, error) {
	if len(pc.clients) == 0 {
		return nil, fmt.Errorf("no mediators available")
	}

	type result struct {
		tabs []types.Tab
		err  error
	}

	results := make(chan result, len(pc.clients))
	var wg sync.WaitGroup

	// Query all clients in parallel
	for _, client := range pc.clients {
		wg.Add(1)
		go func(c api.Client) {
			defer wg.Done()
			tabs, err := c.ListTabs()
			results <- result{tabs: tabs, err: err}
		}(client)
	}

	// Wait for all queries to complete
	wg.Wait()
	close(results)

	// Collect results
	var allTabs []types.Tab
	var errors []error

	for r := range results {
		if r.err != nil {
			errors = append(errors, r.err)
		} else {
			allTabs = append(allTabs, r.tabs...)
		}
	}

	// Return error if all clients failed
	if len(errors) == len(pc.clients) {
		return nil, fmt.Errorf("all mediators failed: %v", errors[0])
	}

	return allTabs, nil
}

// CloseTabsParallel closes tabs across multiple browsers
func (pc *ParallelClient) CloseTabsParallel(tabIDs []string) error {
	// Group tab IDs by client prefix
	clientTabs := make(map[string][]string)
	for _, tabID := range tabIDs {
		prefix, _, _, err := utils.ParseTabID(tabID)
		if err != nil {
			continue
		}
		clientTabs[prefix] = append(clientTabs[prefix], tabID)
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(pc.clients))

	// Close tabs in parallel for each client
	for _, client := range pc.clients {
		// GetPrefix returns "a.", but ParseTabID returns "a", so we need to strip the dot
		prefixWithoutDot := client.GetPrefix()
		if len(prefixWithoutDot) > 0 && prefixWithoutDot[len(prefixWithoutDot)-1] == '.' {
			prefixWithoutDot = prefixWithoutDot[:len(prefixWithoutDot)-1]
		}
		tabs, ok := clientTabs[prefixWithoutDot]
		if !ok || len(tabs) == 0 {
			continue
		}

		wg.Add(1)
		go func(c api.Client, ids []string) {
			defer wg.Done()
			if err := c.CloseTabs(ids); err != nil {
				errors <- fmt.Errorf("%s: %w", c.GetPrefix(), err)
			}
		}(client, tabs)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("failed to close some tabs: %v", allErrors[0])
	}

	return nil
}

// QueryAllTabs queries tabs from all browsers in parallel
func (pc *ParallelClient) QueryAllTabs(query types.TabQuery) ([]types.Tab, error) {
	if len(pc.clients) == 0 {
		return nil, fmt.Errorf("no mediators available")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	type result struct {
		prefix string
		tabs   []types.Tab
		err    error
	}

	results := make(chan result, len(pc.clients))

	// Query all clients concurrently
	for _, client := range pc.clients {
		go func(c api.Client) {
			select {
			case <-ctx.Done():
				return
			default:
				tabs, err := c.QueryTabs(query)
				results <- result{
					prefix: c.GetPrefix(),
					tabs:   tabs,
					err:    err,
				}
			}
		}(client)
	}

	// Collect results with timeout
	var allTabs []types.Tab
	successCount := 0

	for i := 0; i < len(pc.clients); i++ {
		select {
		case r := <-results:
			if r.err == nil {
				allTabs = append(allTabs, r.tabs...)
				successCount++
			}
		case <-ctx.Done():
			return allTabs, ctx.Err()
		}
	}

	if successCount == 0 {
		return nil, fmt.Errorf("all mediators failed to query tabs")
	}

	return allTabs, nil
}

// GetTextParallel gets text from tabs across multiple browsers
func (pc *ParallelClient) GetTextParallel(tabIDs []string) ([]types.TabContent, error) {
	// Group tab IDs by client prefix
	clientTabs := make(map[string][]string)
	for _, tabID := range tabIDs {
		prefix, _, _, err := utils.ParseTabID(tabID)
		if err != nil {
			continue
		}
		clientTabs[prefix] = append(clientTabs[prefix], tabID)
	}

	type result struct {
		content []types.TabContent
		err     error
	}

	results := make(chan result, len(clientTabs))
	var wg sync.WaitGroup

	// Get text in parallel for each client
	for _, client := range pc.clients {
		tabs, ok := clientTabs[client.GetPrefix()]
		if !ok || len(tabs) == 0 {
			continue
		}

		wg.Add(1)
		go func(c api.Client, ids []string) {
			defer wg.Done()
			content, err := c.GetText(ids, types.TextOptions{})
			results <- result{content: content, err: err}
		}(client, tabs)
	}

	wg.Wait()
	close(results)

	// Collect results
	var allContent []types.TabContent
	for r := range results {
		if r.err == nil {
			allContent = append(allContent, r.content...)
		}
	}

	return allContent, nil
}

// OpenURLsParallel opens URLs across multiple browsers
func (pc *ParallelClient) OpenURLsParallel(urlsByBrowser map[string][]string) (map[string][]string, error) {
	type result struct {
		prefix string
		tabIDs []string
		err    error
	}

	results := make(chan result, len(urlsByBrowser))
	var wg sync.WaitGroup

	// Open URLs in parallel for each browser
	for prefix, urls := range urlsByBrowser {
		// Find client with matching prefix
		var client api.Client
		for _, c := range pc.clients {
			if c.GetPrefix() == prefix {
				client = c
				break
			}
		}

		if client == nil {
			continue
		}

		wg.Add(1)
		go func(c api.Client, u []string) {
			defer wg.Done()
			tabIDs, err := c.OpenURLs(u, "")
			results <- result{
				prefix: c.GetPrefix(),
				tabIDs: tabIDs,
				err:    err,
			}
		}(client, urls)
	}

	wg.Wait()
	close(results)

	// Collect results
	allTabIDs := make(map[string][]string)
	for r := range results {
		if r.err == nil {
			allTabIDs[r.prefix] = r.tabIDs
		}
	}

	return allTabIDs, nil
}

// ExecuteParallel executes a function on all clients in parallel
func (pc *ParallelClient) ExecuteParallel(fn func(api.Client) error) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(pc.clients))

	for _, client := range pc.clients {
		wg.Add(1)
		go func(c api.Client) {
			defer wg.Done()
			if err := fn(c); err != nil {
				errors <- fmt.Errorf("%s: %w", c.GetPrefix(), err)
			}
		}(client)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("parallel execution failed: %v", allErrors[0])
	}

	return nil
}

// MapReduce performs a map-reduce operation on all clients
func (pc *ParallelClient) MapReduce(
	mapFn func(api.Client) (interface{}, error),
	reduceFn func([]interface{}) (interface{}, error),
) (interface{}, error) {
	type result struct {
		value interface{}
		err   error
	}

	results := make(chan result, len(pc.clients))
	var wg sync.WaitGroup

	// Map phase
	for _, client := range pc.clients {
		wg.Add(1)
		go func(c api.Client) {
			defer wg.Done()
			value, err := mapFn(c)
			results <- result{value: value, err: err}
		}(client)
	}

	wg.Wait()
	close(results)

	// Collect map results
	var values []interface{}
	for r := range results {
		if r.err == nil && r.value != nil {
			values = append(values, r.value)
		}
	}

	// Reduce phase
	return reduceFn(values)
}

// Close closes all clients
func (pc *ParallelClient) Close() error {
	for _, client := range pc.clients {
		client.Close()
	}
	return nil
}