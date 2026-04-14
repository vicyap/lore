package tui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
)

type viewState int

const (
	viewList viewState = iota
	viewDetail
)

// noteItem represents a commit with a lore note.
type noteItem struct {
	commitHash string
	subject    string
	note       string
}

// model is the top-level bubbletea model.
type model struct {
	cfg       config.Config
	items     []noteItem
	filtered  []noteItem
	cursor    int
	viewState viewState
	width     int
	height    int
	scroll    int // scroll offset for detail view
	search    string
	searching bool
	err       error
}

// Run launches the TUI.
func Run() error {
	cfg := config.Load()

	items, err := loadNotes(cfg)
	if err != nil {
		return fmt.Errorf("load notes: %w", err)
	}

	if len(items) == 0 {
		fmt.Println("No lore notes found. Notes are created when you commit during Claude Code sessions with lore enabled.")
		return nil
	}

	initial := model{
		cfg:       cfg,
		items:     items,
		filtered:  items,
		viewState: viewList,
	}

	p := tea.NewProgram(initial)
	_, err = p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	}

	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch m.viewState {
	case viewList:
		return m.handleListKey(key)
	case viewDetail:
		return m.handleDetailKey(key)
	}
	return m, nil
}

func (m model) handleListKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.filtered) > 0 {
			m.viewState = viewDetail
			m.scroll = 0
		}
	case "/":
		m.searching = true
		m.search = ""
	case "escape":
		if m.search != "" {
			m.search = ""
			m.filtered = m.items
			m.cursor = 0
		}
	}
	return m, nil
}

func (m model) handleDetailKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "escape":
		m.viewState = viewList
		m.scroll = 0
	case "up", "k":
		if m.scroll > 0 {
			m.scroll--
		}
	case "down", "j":
		m.scroll++
	}
	return m, nil
}

func (m model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter", "escape":
		m.searching = false
	case "backspace":
		if len(m.search) > 0 {
			m.search = m.search[:len(m.search)-1]
			m.applyFilter()
		}
	default:
		if len(key) == 1 {
			m.search += key
			m.applyFilter()
		}
	}
	return m, nil
}

func (m *model) applyFilter() {
	if m.search == "" {
		m.filtered = m.items
		m.cursor = 0
		return
	}

	query := strings.ToLower(m.search)
	var filtered []noteItem
	for _, item := range m.items {
		if strings.Contains(strings.ToLower(item.subject), query) ||
			strings.Contains(strings.ToLower(item.note), query) {
			filtered = append(filtered, item)
		}
	}
	m.filtered = filtered
	m.cursor = 0
}

func (m model) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}

	var content string
	switch m.viewState {
	case viewList:
		content = m.renderList()
	case viewDetail:
		content = m.renderDetail()
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m model) renderList() string {
	var buf strings.Builder

	// Header
	buf.WriteString("  lore — decision notes\n")
	if m.search != "" || m.searching {
		buf.WriteString(fmt.Sprintf("  search: %s", m.search))
		if m.searching {
			buf.WriteString("_")
		}
		buf.WriteString(fmt.Sprintf("  (%d results)\n", len(m.filtered)))
	}
	buf.WriteString("\n")

	// Available height for items
	headerLines := 3
	footerLines := 3
	availableHeight := m.height - headerLines - footerLines
	if availableHeight < 1 {
		availableHeight = 10
	}

	// Calculate visible window
	start := 0
	if m.cursor >= availableHeight {
		start = m.cursor - availableHeight + 1
	}
	end := start + availableHeight
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for idx := start; idx < end; idx++ {
		item := m.filtered[idx]
		cursor := "  "
		if idx == m.cursor {
			cursor = "> "
		}
		// Truncate subject to fit width
		subject := item.subject
		maxWidth := m.width - 20
		if maxWidth < 20 {
			maxWidth = 40
		}
		if len(subject) > maxWidth {
			subject = subject[:maxWidth-3] + "..."
		}
		buf.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, item.commitHash[:min(8, len(item.commitHash))], subject))
	}

	// Footer
	buf.WriteString("\n")
	buf.WriteString("  j/k: navigate  enter: view  /: search  q: quit\n")

	return buf.String()
}

func (m model) renderDetail() string {
	if m.cursor >= len(m.filtered) {
		return "No item selected"
	}

	item := m.filtered[m.cursor]

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("  %s — %s\n\n", item.commitHash[:min(12, len(item.commitHash))], item.subject))

	// Render markdown with glamour
	rendered := renderMarkdown(item.note, m.width-4)

	lines := strings.Split(rendered, "\n")

	// Apply scroll
	start := m.scroll
	if start >= len(lines) {
		start = max(0, len(lines)-1)
	}
	availableHeight := m.height - 6
	if availableHeight < 5 {
		availableHeight = 20
	}
	end := start + availableHeight
	if end > len(lines) {
		end = len(lines)
	}

	for _, line := range lines[start:end] {
		buf.WriteString("  " + line + "\n")
	}

	buf.WriteString("\n  j/k: scroll  q/esc: back\n")

	return buf.String()
}

func renderMarkdown(content string, width int) string {
	// Plain text rendering — glamour has version conflicts with bubbletea v2.
	// The lore notes are simple markdown that reads well as-is.
	if width < 20 {
		width = 80
	}
	var lines []string
	for _, line := range strings.Split(content, "\n") {
		if len(line) > width {
			lines = append(lines, line[:width])
		} else {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func loadNotes(cfg config.Config) ([]noteItem, error) {
	pairs, err := git.ListNotes(cfg.NotesRef)
	if err != nil {
		return nil, err
	}

	var items []noteItem
	for _, pair := range pairs {
		commitHash := pair[1]
		note, err := git.ReadNote(cfg.NotesRef, commitHash)
		if err != nil || note == "" {
			continue
		}

		subject, _ := git.GetCommitSubject(commitHash)
		items = append(items, noteItem{
			commitHash: commitHash,
			subject:    subject,
			note:       note,
		})
	}
	return items, nil
}

// RunWithFallback runs the TUI, or falls back to stdout if not a terminal.
func RunWithFallback() error {
	fi, _ := os.Stdout.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		// Not a terminal — fall back to plain output
		cfg := config.Load()
		output, err := git.GetCommitsWithNotes(cfg.NotesRef, 20)
		if err != nil {
			return err
		}
		fmt.Println(output)
		return nil
	}
	return Run()
}
