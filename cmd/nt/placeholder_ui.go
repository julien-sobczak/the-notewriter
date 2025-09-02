package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/julien-sobczak/the-notewriter/internal/core"
)

var (
	titleStyle    = lipgloss.NewStyle().MarginLeft(2)
	itemStyle     = lipgloss.NewStyle().PaddingLeft(4)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	helpStyle     = lipgloss.NewStyle().PaddingLeft(4).PaddingBottom(1)
)

// PlaceholderInputModel handles input for a single placeholder
type PlaceholderInputModel struct {
	placeholder core.Placeholder
	currentURL  string
	textInput   textinput.Model
	list        list.Model
	inputType   string
	value       string
	quitting    bool
}

func NewPlaceholderInputModel(placeholder core.Placeholder, currentURL string) PlaceholderInputModel {
	inputType := getPlaceholderType(placeholder)
	
	model := PlaceholderInputModel{
		placeholder: placeholder,
		currentURL:  currentURL,
		inputType:   inputType,
	}

	switch inputType {
	case "select":
		// Create list for selection
		items := make([]list.Item, len(placeholder.AllowedValues))
		for i, value := range placeholder.AllowedValues {
			items[i] = selectItem{value}
		}
		
		l := list.New(items, selectDelegate{}, 50, 10)
		l.Title = fmt.Sprintf("Select value for %s", placeholder.Name)
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(false)
		l.SetShowPagination(false)
		l.Styles.Title = titleStyle
		l.Styles.HelpStyle = helpStyle
		
		model.list = l
		
	case "input", "autocomplete":
		// Create text input
		ti := textinput.New()
		ti.Placeholder = fmt.Sprintf("Enter value for %s...", placeholder.Name)
		ti.Focus()
		ti.CharLimit = 100
		ti.Width = 50
		
		model.textInput = ti
	}
	
	return model
}

func (m PlaceholderInputModel) Init() tea.Cmd {
	if m.inputType == "select" {
		return nil
	}
	return textinput.Blink
}

func (m PlaceholderInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.inputType == "select" {
				if item, ok := m.list.SelectedItem().(selectItem); ok {
					m.value = item.value
				}
			} else {
				m.value = m.textInput.Value()
			}
			return m, tea.Quit
			
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	if m.inputType == "select" {
		m.list, cmd = m.list.Update(msg)
	} else {
		m.textInput, cmd = m.textInput.Update(msg)
	}
	
	return m, cmd
}

func (m PlaceholderInputModel) View() string {
	var b strings.Builder
	
	// Show current URL
	b.WriteString(fmt.Sprintf("Current URL: %s\n\n", m.currentURL))
	
	// Show placeholder description
	b.WriteString(fmt.Sprintf("Fill placeholder: %s\n\n", m.placeholder.String()))
	
	// Show input component
	switch m.inputType {
	case "select":
		b.WriteString(m.list.View())
	case "autocomplete":
		b.WriteString(m.textInput.View())
		b.WriteString("\n\nSuggestions: ")
		b.WriteString(strings.Join(m.placeholder.AllowedValues, ", "))
	default:
		b.WriteString(m.textInput.View())
	}
	
	b.WriteString("\n\n(Enter to confirm, Esc to cancel)")
	
	return b.String()
}

// selectItem implements list.Item for selection lists
type selectItem struct {
	value string
}

func (i selectItem) FilterValue() string { return i.value }

// selectDelegate implements list.ItemDelegate for selection lists
type selectDelegate struct{}

func (d selectDelegate) Height() int                             { return 1 }
func (d selectDelegate) Spacing() int                            { return 0 }
func (d selectDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d selectDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(selectItem)
	if !ok {
		return
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(i.value))
}

// promptForPlaceholders handles the interactive input for all placeholders
func promptForPlaceholders(url string, placeholders []core.Placeholder) (map[string]string, error) {
	values := make(map[string]string)
	
	for _, placeholder := range placeholders {
		// Create a temporary goto with current URL to use Expand method
		tempGoto := &core.Goto{URL: url}
		currentURL := tempGoto.Expand(values)
		
		model := NewPlaceholderInputModel(placeholder, currentURL)
		p := tea.NewProgram(model)
		
		result, err := p.Run()
		if err != nil {
			return nil, err
		}
		
		finalModel := result.(PlaceholderInputModel)
		if finalModel.quitting || finalModel.value == "" {
			return nil, fmt.Errorf("user cancelled input")
		}
		
		values[placeholder.Name] = finalModel.value
	}
	
	return values, nil
}