package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

/*
 * The command nt-journal uses Bubble Tea under the hood to provide an interactive CLI.
 * The code is heavily based on examples. It's probably possible to write a better code using richer models.
 * All BubbleTea-related code is present in this file to make easy to refactor or switch to another library someday.
 */

var (
	// List-specific attributes
	listWidth             = 20
	listHeight            = 14
	listTitleStyle        = lipgloss.NewStyle().MarginLeft(2)
	listItemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	listSelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))

	// Common attributes
	helpStyle     = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

/*
* Emotion Selection
 */

func ChooseEmotion(emotions []*Emotion) *Emotion {
	/* Inspired by https://github.com/charmbracelet/bubbletea/blob/master/examples/list-simple/ */
	res, err := tea.NewProgram(NewEmotionModel(emotions)).Run()
	if err != nil {
		log.Fatal(err)
	}
	// Retrieve the user selection
	typedRes := res.(EmotionModel)
	emotionKey := typedRes.choice
	if emotionKey == "" {
		// Abort on premature exit
		os.Exit(0)
	}

	// Retrieve the corresponding emotion from its key
	for _, emotion := range emotions {
		if emotion.Key == emotionKey {
			return emotion
		}
	}

	panic("You are living without emotions 😱")
}

func NewEmotionModel(emotions []*Emotion) EmotionModel {
	items := []list.Item{}

	for _, emotion := range emotions {
		items = append(items, EmotionItem{
			label: emotion.Emoji + " " + emotion.Title,
			key:   emotion.Key,
		})
	}

	l := list.New(items, emotionDelegate{}, listWidth, listHeight)
	l.Title = "How to you feel?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.HelpStyle = helpStyle

	return EmotionModel{list: l}
}

type EmotionItem struct {
	label string
	key   string
}

func (i EmotionItem) FilterValue() string { return "" }

type emotionDelegate struct{}

func (d emotionDelegate) Height() int                             { return 1 }
func (d emotionDelegate) Spacing() int                            { return 0 }
func (d emotionDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d emotionDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(EmotionItem)
	if !ok {
		return
	}

	fn := listItemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return listSelectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(i.label))
}

type EmotionModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m EmotionModel) Init() tea.Cmd {
	return nil
}

func (m EmotionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(EmotionItem)
			if ok {
				m.choice = string(i.key)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m EmotionModel) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice))
	}
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

/////////////////

type editorModel struct {
	choice   string
	quitting bool
}

func initialEditorModel() editorModel {
	return editorModel{
		choice: "yes", // Set default to encourage opening the entry in the editor
	}
}

func (m editorModel) Init() tea.Cmd {
	return nil
}

func (m editorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.choice = "yes"
			m.quitting = true
			return m, tea.Quit
		case "n", "N":
			m.choice = "no"
			m.quitting = true
			return m, tea.Quit
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m editorModel) View() string {
	if m.quitting {
		return ""
	}
	return "Open in the editor? (y/n)\n"
}

func AskToOpenInEditor() bool {
	p := tea.NewProgram(initialEditorModel())
	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if m, ok := m.(editorModel); ok {
		return strings.EqualFold(m.choice, "yes")
	}
	return false
}
