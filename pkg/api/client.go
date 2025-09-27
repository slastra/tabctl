package api

import (
	"errors"

	"github.com/tabctl/tabctl/pkg/types"
)

// ClientConfig holds configuration for creating clients
type ClientConfig struct {
	TargetHosts string
	Timeout     int
}

// CreateClients creates clients based on configuration
func CreateClients(config ClientConfig) ([]Client, error) {
	// TODO: Implement client creation logic
	return nil, errors.New("not implemented yet")
}

// CreateMultiClient creates a multi-client from individual clients
func CreateMultiClient(clients []Client) MultiClient {
	return &multiClient{
		clients: clients,
	}
}

// multiClient implements MultiClient interface
type multiClient struct {
	clients []Client
}

func (mc *multiClient) GetClients() []Client {
	return mc.clients
}

func (mc *multiClient) AddClient(client Client) {
	mc.clients = append(mc.clients, client)
}

func (mc *multiClient) RemoveClient(prefix string) {
	for i, client := range mc.clients {
		if client.GetPrefix() == prefix {
			mc.clients = append(mc.clients[:i], mc.clients[i+1:]...)
			break
		}
	}
}

// Implement TabAPI by delegating to all clients
func (mc *multiClient) ListTabs() ([]types.Tab, error) {
	var allTabs []types.Tab
	for _, client := range mc.clients {
		tabs, err := client.ListTabs()
		if err != nil {
			continue // Skip failed clients
		}
		allTabs = append(allTabs, tabs...)
	}
	return allTabs, nil
}

func (mc *multiClient) CloseTabs(tabIDs []string) error {
	// Group tab IDs by client prefix
	clientTabs := make(map[string][]string)
	for _, tabID := range tabIDs {
		prefix := getTabPrefix(tabID)
		clientTabs[prefix] = append(clientTabs[prefix], tabID)
	}

	// Execute close on each client
	for prefix, tabs := range clientTabs {
		client := mc.getClientByPrefix(prefix)
		if client != nil {
			client.CloseTabs(tabs)
		}
	}
	return nil
}

func (mc *multiClient) ActivateTab(tabID string, focused bool) error {
	prefix := getTabPrefix(tabID)
	client := mc.getClientByPrefix(prefix)
	if client == nil {
		return errors.New("client not found")
	}
	return client.ActivateTab(tabID, focused)
}

func (mc *multiClient) MoveTabs() error {
	// TODO: Implement move tabs across all clients
	return errors.New("not implemented yet")
}

func (mc *multiClient) UpdateTabs(updates []types.TabUpdate) error {
	// Group updates by client prefix
	clientUpdates := make(map[string][]types.TabUpdate)
	for _, update := range updates {
		prefix := getTabPrefix(update.TabID)
		clientUpdates[prefix] = append(clientUpdates[prefix], update)
	}

	// Execute updates on each client
	for prefix, updates := range clientUpdates {
		client := mc.getClientByPrefix(prefix)
		if client != nil {
			client.UpdateTabs(updates)
		}
	}
	return nil
}

func (mc *multiClient) QueryTabs(query types.TabQuery) ([]types.Tab, error) {
	var allTabs []types.Tab
	for _, client := range mc.clients {
		tabs, err := client.QueryTabs(query)
		if err != nil {
			continue
		}
		allTabs = append(allTabs, tabs...)
	}
	return allTabs, nil
}

func (mc *multiClient) NavigateURLs(pairs []types.TabURLPair) error {
	// Group pairs by client prefix
	clientPairs := make(map[string][]types.TabURLPair)
	for _, pair := range pairs {
		prefix := getTabPrefix(pair.TabID)
		clientPairs[prefix] = append(clientPairs[prefix], pair)
	}

	// Execute navigation on each client
	for prefix, pairs := range clientPairs {
		client := mc.getClientByPrefix(prefix)
		if client != nil {
			client.NavigateURLs(pairs)
		}
	}
	return nil
}

func (mc *multiClient) GetText(tabIDs []string, options types.TextOptions) ([]types.TabContent, error) {
	var allContent []types.TabContent
	clientTabs := mc.groupTabsByClient(tabIDs)

	for prefix, tabs := range clientTabs {
		client := mc.getClientByPrefix(prefix)
		if client != nil {
			content, err := client.GetText(tabs, options)
			if err == nil {
				allContent = append(allContent, content...)
			}
		}
	}
	return allContent, nil
}

func (mc *multiClient) GetHTML(tabIDs []string, options types.TextOptions) ([]types.TabContent, error) {
	var allContent []types.TabContent
	clientTabs := mc.groupTabsByClient(tabIDs)

	for prefix, tabs := range clientTabs {
		client := mc.getClientByPrefix(prefix)
		if client != nil {
			content, err := client.GetHTML(tabs, options)
			if err == nil {
				allContent = append(allContent, content...)
			}
		}
	}
	return allContent, nil
}

func (mc *multiClient) GetWords(tabIDs []string, options types.WordsOptions) ([]string, error) {
	var allWords []string
	clientTabs := mc.groupTabsByClient(tabIDs)

	for prefix, tabs := range clientTabs {
		client := mc.getClientByPrefix(prefix)
		if client != nil {
			words, err := client.GetWords(tabs, options)
			if err == nil {
				allWords = append(allWords, words...)
			}
		}
	}
	return allWords, nil
}

func (mc *multiClient) GetWindows() ([]types.Window, error) {
	var allWindows []types.Window
	for _, client := range mc.clients {
		windows, err := client.GetWindows()
		if err != nil {
			continue
		}
		allWindows = append(allWindows, windows...)
	}
	return allWindows, nil
}

func (mc *multiClient) GetActiveTab() (string, error) {
	// Return first active tab found
	for _, client := range mc.clients {
		tab, err := client.GetActiveTab()
		if err == nil && tab != "" {
			return tab, nil
		}
	}
	return "", errors.New("no active tab found")
}

func (mc *multiClient) GetActiveTabs() ([]string, error) {
	var allActive []string
	for _, client := range mc.clients {
		tabs, err := client.GetActiveTabs()
		if err != nil {
			continue
		}
		allActive = append(allActive, tabs...)
	}
	return allActive, nil
}

func (mc *multiClient) OpenURLs(urls []string, windowID string) ([]string, error) {
	prefix := getWindowPrefix(windowID)
	client := mc.getClientByPrefix(prefix)
	if client == nil {
		return nil, errors.New("client not found")
	}
	return client.OpenURLs(urls, windowID)
}

func (mc *multiClient) GetScreenshot() (*types.Screenshot, error) {
	// Return screenshot from first available client
	for _, client := range mc.clients {
		screenshot, err := client.GetScreenshot()
		if err == nil {
			return screenshot, nil
		}
	}
	return nil, errors.New("no screenshot available")
}

// Helper methods
func (mc *multiClient) getClientByPrefix(prefix string) Client {
	for _, client := range mc.clients {
		if client.GetPrefix() == prefix {
			return client
		}
	}
	return nil
}

func (mc *multiClient) groupTabsByClient(tabIDs []string) map[string][]string {
	clientTabs := make(map[string][]string)
	for _, tabID := range tabIDs {
		prefix := getTabPrefix(tabID)
		clientTabs[prefix] = append(clientTabs[prefix], tabID)
	}
	return clientTabs
}

// Utility functions
func getTabPrefix(tabID string) string {
	// Extract prefix from tab ID (format: prefix.window.tab)
	if len(tabID) > 0 && tabID[1] == '.' {
		return tabID[:2] // Return "a.", "b.", etc.
	}
	return ""
}

func getWindowPrefix(windowID string) string {
	// Extract prefix from window ID (format: prefix.window)
	if len(windowID) > 0 && windowID[1] == '.' {
		return windowID[:2] // Return "a.", "b.", etc.
	}
	return ""
}