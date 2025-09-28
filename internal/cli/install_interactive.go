package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// browserItem represents a browser option with selection state and implements list.Item
type browserItem struct {
	browser  BrowserInfo
	selected bool
}

func (i browserItem) FilterValue() string { return i.browser.Name }

func (i browserItem) Title() string       { return i.browser.Name }
func (i browserItem) Description() string { return i.browser.Type + " browser" }

// browserItemDelegate handles rendering of list items
type browserItemDelegate struct{}

func (d browserItemDelegate) Height() int                             { return 1 }
func (d browserItemDelegate) Spacing() int                            { return 0 }
func (d browserItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d browserItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(browserItem)
	if !ok {
		return
	}

	checkbox := "[ ]"
	if i.selected {
		checkbox = "[✓]"
	}

	str := fmt.Sprintf("%s %s", checkbox, i.browser.Name)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

// browserSelectionModel implements the bubbletea Model interface
type browserSelectionModel struct {
	list     list.Model
	choice   string
	finished bool
	cancelled bool
}

var (
	itemStyle = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

// selectBrowsersInteractive shows an interactive browser selection interface
func selectBrowsersInteractive(available []BrowserInfo) ([]BrowserInfo, error) {
	// Create items with all browsers selected by default
	items := make([]list.Item, len(available))
	for i, browser := range available {
		items[i] = browserItem{
			browser:  browser,
			selected: true, // Default to selected
		}
	}

	// Create the list
	const defaultWidth = 20
	const listHeight = 14

	l := list.New(items, browserItemDelegate{}, defaultWidth, listHeight)
	l.Title = "Select Browsers to Install"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// Create the model
	model := browserSelectionModel{list: l}

	// Run the program
	p := tea.NewProgram(model)
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("error running interactive selection: %w", err)
	}

	// Extract results
	result := finalModel.(browserSelectionModel)
	if result.cancelled {
		return nil, fmt.Errorf("selection cancelled")
	}

	// Build the selected browsers list
	var selectedBrowsers []BrowserInfo
	for _, item := range result.list.Items() {
		if browserItem, ok := item.(browserItem); ok && browserItem.selected {
			selectedBrowsers = append(selectedBrowsers, browserItem.browser)
		}
	}

	return selectedBrowsers, nil
}

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4")).
	Padding(0, 1)

// Init initializes the model
func (m browserSelectionModel) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m browserSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			m.finished = true
			return m, tea.Quit

		case " ": // Space to toggle selection
			i := m.list.Index()
			items := m.list.Items()
			if i < len(items) {
				if browserItem, ok := items[i].(browserItem); ok {
					// Toggle selection
					browserItem.selected = !browserItem.selected
					// Update the item in the list
					items[i] = browserItem
					m.list.SetItems(items)
				}
			}
			return m, nil

		case "a": // Select all
			items := m.list.Items()
			for i, item := range items {
				if browserItem, ok := item.(browserItem); ok {
					browserItem.selected = true
					items[i] = browserItem
				}
			}
			m.list.SetItems(items)
			return m, nil

		case "n": // Select none
			items := m.list.Items()
			for i, item := range items {
				if browserItem, ok := item.(browserItem); ok {
					browserItem.selected = false
					items[i] = browserItem
				}
			}
			m.list.SetItems(items)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the interface
func (m browserSelectionModel) View() string {
	if m.finished {
		return quitTextStyle.Render("Installing...")
	}
	if m.cancelled {
		return quitTextStyle.Render("Cancelled.")
	}

	return "\n" + m.list.View() + "\n\nSpace: Toggle • a: Select all • n: Select none • Enter: Install • q: Quit"
}