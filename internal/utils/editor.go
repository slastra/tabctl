package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/tabctl/tabctl/pkg/types"
)

// OpenEditor opens an external editor with the given content
func OpenEditor(content string) (string, error) {
	editor := getEditor()

	// Create a temporary file
	tmpFile, err := ioutil.TempFile("", "tabctl-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write content to temporary file
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tmpFile.Close()

	// Open editor
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read modified content
	modifiedContent, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read modified file: %w", err)
	}

	return string(modifiedContent), nil
}

// OpenTabEditor opens an editor for tab manipulation
func OpenTabEditor(tabs []types.Tab) ([]TabEditAction, error) {
	// Format tabs for editing
	content := formatTabsForEditing(tabs)

	// Open editor
	editedContent, err := OpenEditor(content)
	if err != nil {
		return nil, err
	}

	// Parse edited content into actions
	return parseTabEditActions(editedContent, tabs)
}

// TabEditAction represents an action to perform on a tab
type TabEditAction struct {
	Type     string     // "move", "close", "reorder"
	Tab      types.Tab  // Original tab
	NewIndex int        // For reordering
	NewWindow string    // For moving to different window
}

// formatTabsForEditing formats tabs into a text format for editing
func formatTabsForEditing(tabs []types.Tab) string {
	var lines []string
	lines = append(lines, "# Edit tabs below. You can:")
	lines = append(lines, "# - Reorder lines to change tab order")
	lines = append(lines, "# - Delete lines to close tabs")
	lines = append(lines, "# - Change window ID (first part after dot) to move tabs")
	lines = append(lines, "#")

	for _, tab := range tabs {
		line := fmt.Sprintf("%s\t%s\t%s", tab.ID, tab.Title, tab.URL)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// parseTabEditActions parses edited content into tab actions
func parseTabEditActions(content string, originalTabs []types.Tab) ([]TabEditAction, error) {
	var actions []TabEditAction

	lines := strings.Split(content, "\n")
	editedTabs := make(map[int]types.Tab)

	// Parse edited tabs
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tab, err := ParseTabLine(line)
		if err != nil {
			continue // Skip invalid lines
		}

		editedTabs[tab.ID] = tab
	}

	// Find tabs that were deleted (close actions)
	for _, originalTab := range originalTabs {
		if _, exists := editedTabs[originalTab.ID]; !exists {
			actions = append(actions, TabEditAction{
				Type: "close",
				Tab:  originalTab,
			})
		}
	}

	// Find tabs that were moved or reordered
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		tab, err := ParseTabLine(line)
		if err != nil {
			continue
		}

		// Find original tab
		var originalTab types.Tab
		found := false
		for _, t := range originalTabs {
			if t.ID == tab.ID {
				originalTab = t
				found = true
				break
			}
		}

		if !found {
			continue
		}

		// Check if window changed
		// Since tab IDs are now integers, we need to format them for comparison
		originalTabStr := fmt.Sprintf("a.%d.%d", originalTab.WindowID, originalTab.ID)
		newTabStr := fmt.Sprintf("a.%d.%d", tab.WindowID, tab.ID)
		originalPrefix, originalWindow, _, _ := ParseTabID(originalTabStr)
		newPrefix, newWindow, _, _ := ParseTabID(newTabStr)

		if originalWindow != newWindow && originalPrefix == newPrefix {
			actions = append(actions, TabEditAction{
				Type:      "move",
				Tab:       originalTab,
				NewWindow: newWindow,
			})
		}

		// Check if order changed (simple implementation)
		actions = append(actions, TabEditAction{
			Type:     "reorder",
			Tab:      originalTab,
			NewIndex: i,
		})
	}

	return actions, nil
}

// getEditor returns the preferred editor
func getEditor() string {
	// Check environment variables
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}

	// Default fallbacks
	editors := []string{"nano", "vim", "vi", "emacs"}
	for _, editor := range editors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	// Ultimate fallback
	return "vi"
}

// GetTempDir returns the appropriate temporary directory
func GetTempDir() string {
	if tmpDir := os.Getenv("TMPDIR"); tmpDir != "" {
		return tmpDir
	}
	return "/tmp"
}