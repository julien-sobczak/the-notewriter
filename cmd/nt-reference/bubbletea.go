package main

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/reference"
)

// FIXME The CLI does not exit when pressing Ctrl+C or ESC keys.

/*
 * The command nt-reference uses Bubble Tea under the hood to provide an interactive CLI.
 * The code is heavily based on examples. It's probably possible better code using richer models.
 * All BubbleTea-related code is present in this file to make easy to refactor or switch to another library someday.
 */

var (
	// List-specific attributes
	listWidth             = 20
	listHeight            = 14
	listTitleStyle        = lipgloss.NewStyle().MarginLeft(2)
	listItemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	listSelectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))

	// Pager-specific attributes
	pagerTitleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	pagerInfoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return listTitleStyle.Copy().BorderStyle(b)
	}()

	// Common attributes
	helpStyle     = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

/*
* Category Selection
 */

func ChooseCategory(categories map[string]*core.ConfigReference) (string, *core.ConfigReference) {
	/* Inspired by https://github.com/charmbracelet/bubbletea/blob/master/examples/list-simple/ */
	res, err := tea.NewProgram(NewCategoryModel(categories)).Run()
	if err != nil {
		log.Fatal(err)
	}
	category := res.(CategoryModel).choice
	return category, categories[category]
}

func NewCategoryModel(categories map[string]*core.ConfigReference) CategoryModel {
	items := []list.Item{}

	// Create a slice to store keys and sort them to have the same predictable order on each execution
	keys := make([]string, 0, len(categories))
	for key := range categories {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		category := categories[key]
		items = append(items, CategoryItem{
			label: category.Title,
			key:   key,
		})
	}

	l := list.New(items, categoryDelegate{}, listWidth, listHeight)
	l.Title = "What do you want to add?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.HelpStyle = helpStyle

	return CategoryModel{list: l}
}

type CategoryItem struct {
	label string
	key   string
}

func (i CategoryItem) FilterValue() string { return "" }

type categoryDelegate struct{}

func (d categoryDelegate) Height() int                             { return 1 }
func (d categoryDelegate) Spacing() int                            { return 0 }
func (d categoryDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d categoryDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(CategoryItem)
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

type CategoryModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m CategoryModel) Init() tea.Cmd {
	return nil
}

func (m CategoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(CategoryItem)
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

func (m CategoryModel) View() string {
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s? Sounds good to me.", m.choice))
	}
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

/*
 * Manager Initialization Progress
 */

func WaitManagerIsReady(manager reference.Manager) {
	// The manager needs some time to start.
	// Show a progress bar
	prog := progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	if _, err := tea.NewProgram(ManagerProgressModel{
		progress: prog,
		manager:  manager,
		padding:  2,
		maxWidth: 80,
	}).Run(); err != nil {
		log.Fatal(err)
	}
}

type ManagerProgressModel struct {
	// The current progress
	percent  float64
	progress progress.Model
	manager  reference.Manager
	padding  int
	maxWidth int
}

func (m ManagerProgressModel) Init() tea.Cmd {
	return tickCmd()
}

func (m ManagerProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - m.padding*2 - 4
		if m.progress.Width > m.maxWidth {
			m.progress.Width = m.maxWidth
		}
		return m, nil

	case time.Time:
		m.percent = 0.25
		ready, err := m.manager.Ready()
		if err != nil {
			log.Fatal(err)
		}
		if ready {
			m.percent = 1.0
			return m, tea.Quit
		}
		return m, tickCmd()

	default:
		return m, nil
	}
}

func (m ManagerProgressModel) View() string {
	pad := strings.Repeat(" ", m.padding)
	return "\n" +
		pad + m.progress.ViewAs(m.percent) + "\n\n" +
		pad + lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render("Press any key to quit")
}

func tickCmd() tea.Cmd {
	// Check readiness every second
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return t
	})
}

/*
 * Search Query Input
 */

func AskSearchQuery() string {
	// Inspired by https://github.com/charmbracelet/bubbletea/blob/master/examples/textinput/main.go
	model := NewSearchModel()
	p := tea.NewProgram(model)
	if m, err := p.Run(); err != nil {
		log.Fatal(err)
	} else {
		model = m.(SearchModel)
	}
	return model.textInput.Value()
}

type SearchModel struct {
	textInput textinput.Model
	err       error
}

func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "isbn, title, ..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80

	return SearchModel{
		textInput: ti,
		err:       nil,
	}
}

func (m SearchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case error:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m SearchModel) View() string {
	return fmt.Sprintf(
		"Search?\n\n%s\n\n%s",
		m.textInput.View(),
		"(esc to quit)",
	) + "\n"
}

/*
 * Result Selection
 */
func SelectSearchResult(results []reference.Result) reference.Result {
	/* Inspired by https://github.com/charmbracelet/bubbletea/blob/master/examples/list-simple/ */
	res, err := tea.NewProgram(NewResultModel(results)).Run()
	if err != nil {
		log.Fatal(err)
	}
	resultIndex := res.(ResultModel).choice
	i, err := strconv.Atoi(resultIndex)
	if err != nil {
		log.Fatalf("Invalid result index %q", resultIndex)
	}
	return results[i]
}

func NewResultModel(results []reference.Result) ResultModel {
	items := []list.Item{}

	for i, result := range results {
		items = append(items, ResultItem{
			label: result.Description(),
			key:   fmt.Sprintf("%d", i),
		})
	}

	l := list.New(items, resultDelegate{}, listWidth, listHeight)
	l.Title = "Which?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.Styles.Title = listTitleStyle
	l.Styles.HelpStyle = helpStyle

	return ResultModel{list: l}
}

type ResultItem struct {
	label string
	key   string
}

func (i ResultItem) FilterValue() string { return "" }

type resultDelegate struct{}

func (d resultDelegate) Height() int                             { return 1 }
func (d resultDelegate) Spacing() int                            { return 0 }
func (d resultDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d resultDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(ResultItem)
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

type ResultModel struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m ResultModel) Init() tea.Cmd {
	return nil
}

func (m ResultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			i, ok := m.list.SelectedItem().(ResultItem)
			if ok {
				m.choice = string(i.key)
			}
			fmt.Println("Selected", m.choice)
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ResultModel) View() string {
	if m.choice != "" {
		return quitTextStyle.Render("OK")
	}
	if m.quitting {
		return ""
	}
	return "\n" + m.list.View()
}

/*
 * Result Pager
 */

func ReviewResult(text string) {
	/* Inspired by https://github.com/charmbracelet/bubbletea/blob/master/examples/pager/ */
	p := tea.NewProgram(
		TemplateModel{content: text},
		tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	_, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}
}

type TemplateModel struct {
	content  string
	ready    bool
	viewport viewport.Model
}

func (m TemplateModel) Init() tea.Cmd {
	return nil
}

func (m TemplateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" || k == "enter" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = false
			m.viewport.SetContent(m.content)
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m TemplateModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m TemplateModel) headerView() string {
	title := pagerTitleStyle.Render("Mr. Pager")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m TemplateModel) footerView() string {
	info := pagerInfoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

/*
 * Save Input
 */

func AskFilename(defaultPath string) string {
	/* Inspired by https://github.com/charmbracelet/bubbletea/blob/master/examples/textinput/ */
	model := NewSaveModel(defaultPath)
	p := tea.NewProgram(
		model,
	)
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
	return model.textInput.Value()
}

type SaveModel struct {
	textInput textinput.Model
	err       error
}

func NewSaveModel(path string) SaveModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 80
	ti.SetValue(path)

	return SaveModel{
		textInput: ti,
		err:       nil,
	}
}

func (m SaveModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SaveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			// Clear default value
			m.textInput.SetValue("")
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case error:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m SaveModel) View() string {
	return fmt.Sprintf(
		"Save?\n\n%s\n\n%s",
		m.textInput.View(),
		"(esc to quit)",
	) + "\n"
}
