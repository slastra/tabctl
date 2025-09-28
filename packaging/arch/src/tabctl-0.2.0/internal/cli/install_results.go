package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InstallResult represents the result of installing for a single browser
type InstallResult struct {
	Browser BrowserInfo
	Success bool
	Error   error
}

// InstallResultsModel represents the bubbletea model for showing installation results
type InstallResultsModel struct {
	results   []InstallResult
	finished  bool
	hasFirefox bool
	hasChrome  bool
}

// NewInstallResultsModel creates a new results model
func NewInstallResultsModel(results []InstallResult) InstallResultsModel {
	model := InstallResultsModel{
		results: results,
	}

	// Determine which extension types we need to show
	for _, result := range results {
		if result.Success {
			if result.Browser.Type == "firefox" {
				model.hasFirefox = true
			} else if result.Browser.Type == "chromium" {
				model.hasChrome = true
			}
		}
	}

	return model
}

// Init initializes the model
func (m InstallResultsModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m InstallResultsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		m.finished = true
		return m, tea.Quit
	}
	return m, nil
}

// View renders the results interface
func (m InstallResultsModel) View() string {
	if m.finished {
		return ""
	}

	var content strings.Builder

	// Title
	title := titleStyle.Render("Installation Complete")
	content.WriteString(title + "\n\n")

	// Results for each browser
	successCount := 0
	for _, result := range m.results {
		if result.Success {
			content.WriteString(fmt.Sprintf("  %s %s (installed)\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Render("✓"),
				result.Browser.Name))
			successCount++
		} else {
			content.WriteString(fmt.Sprintf("  %s %s (failed)\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗"),
				result.Browser.Name))
		}
	}

	content.WriteString("\n")

	// Extension paths section
	if successCount > 0 {
		content.WriteString(lipgloss.NewStyle().Bold(true).Render("Extensions:") + "\n")

		if m.hasChrome {
			content.WriteString("  • Chrome/Brave/Chromium:\n")
			content.WriteString(fmt.Sprintf("    %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("extensions/chrome/")))
		}

		if m.hasFirefox {
			content.WriteString("  • Firefox/Zen:\n")
			content.WriteString(fmt.Sprintf("    %s\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render("extensions/firefox/")))
		}

		content.WriteString("\n")
		content.WriteString(lipgloss.NewStyle().Italic(true).Render("Load unpacked extensions in browser developer settings, then restart.") + "\n")
	} else {
		content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("No browsers were successfully configured.") + "\n")
	}

	content.WriteString("\n")
	content.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("242")).Render("Press any key to exit") + "\n")

	// Create bordered box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(50)

	return boxStyle.Render(content.String())
}

// showInstallationResults displays the installation results using bubbletea
func showInstallationResults(results []InstallResult) error {
	model := NewInstallResultsModel(results)
	p := tea.NewProgram(model)
	_, err := p.Run()
	return err
}