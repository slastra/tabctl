package mediator

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/tabctl/tabctl/internal/config"
	"github.com/tabctl/tabctl/internal/errors"
)

// BrowserAPI handles communication with the browser extension over the
// native-messaging transport.
type BrowserAPI struct {
	transport Transport
	browser   string

	// mu serializes request/response pairs: godbus dispatches each D-Bus
	// method call in its own goroutine, but the extension protocol has no
	// request IDs, so replies can only be matched by strict ordering.
	mu    sync.Mutex
	stale bool // a Recv timed out; the next reply on the pipe may belong to it
}

func NewBrowserAPI(transport Transport, browser string) *BrowserAPI {
	return &BrowserAPI{
		transport: transport,
		browser:   browser,
	}
}

func (r *BrowserAPI) sendCommand(cmd *Command) (interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.stale {
		// A previous command timed out; its reply may still arrive and
		// would be mistaken for ours. Best effort without protocol IDs.
		r.transport.Drain()
		r.stale = false
	}

	if err := r.transport.Send(cmd); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.MediatorBrowserTimeout)
	defer cancel()

	response, err := r.transport.Recv(ctx)
	if err != nil {
		if _, isTimeout := err.(*errors.TimeoutError); isTimeout {
			r.stale = true
		}
		return nil, err
	}

	if errMsg, ok := response["error"].(string); ok && errMsg != "" {
		return nil, fmt.Errorf("browser error: %s", errMsg)
	}

	return response["result"], nil
}

// ListTabs returns a list of TSV-formatted tab descriptors.
func (r *BrowserAPI) ListTabs() ([]string, error) {
	result, err := r.sendCommand(NewCommand(CmdListTabs, nil))
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	tabs, ok := result.([]interface{})
	if !ok {
		return nil, errors.NewTransportError("unexpected response format", nil)
	}

	lines := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		if s, ok := tab.(string); ok {
			lines = append(lines, s)
		}
	}
	return lines, nil
}

// OpenURLs opens new tabs at the given URLs and returns their IDs.
func (r *BrowserAPI) OpenURLs(urls []string, windowID *int) ([]string, error) {
	args := map[string]interface{}{"urls": urls}
	if windowID != nil {
		args["window_id"] = *windowID
	}

	result, err := r.sendCommand(NewCommand(CmdOpenURLs, args))
	if err != nil {
		return nil, fmt.Errorf("failed to communicate with browser extension: %w", err)
	}

	ids, ok := result.([]interface{})
	if !ok {
		return nil, errors.NewTransportError("unexpected response format", nil)
	}

	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if s, ok := id.(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// CloseTabs closes tabs identified by a comma-separated ID list.
func (r *BrowserAPI) CloseTabs(tabIDs string) (string, error) {
	cmd := NewCommand(CmdCloseTabs, map[string]interface{}{
		"tab_ids": strings.Split(tabIDs, ","),
	})
	if _, err := r.sendCommand(cmd); err != nil {
		return "", fmt.Errorf("failed to communicate with browser extension: %w", err)
	}
	return "OK", nil
}

// ActivateTab activates the tab with the given numeric ID.
func (r *BrowserAPI) ActivateTab(tabID int, focused bool) error {
	cmd := NewCommand(CmdActivateTab, map[string]interface{}{
		"tab_id":  tabID,
		"focused": focused,
	})
	if _, err := r.sendCommand(cmd); err != nil {
		return fmt.Errorf("failed to communicate with browser extension: %w", err)
	}
	return nil
}
