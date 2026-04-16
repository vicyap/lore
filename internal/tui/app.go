package tui

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/vicyap/lore/internal/config"
	"github.com/vicyap/lore/internal/git"
)

type viewState int

const (
	viewList viewState = iota
	viewDetail
)

const (
	hashWidth = 8
	dateWidth = 16 // len("2006-01-02 15:04")
	rowGutter = 2  // spaces between columns
)

// noteItem represents a commit with a lore note.
type noteItem struct {
	commitHash string
	subject    string
	note       string
	timestamp  time.Time
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

	// detail view state
	scroll         int
	noteLines      []string
	noteMatchLines []int
	noteMatchIdx   int

	// search
	search    string
	searching bool

	// input state
	lastKey string // tracks prior key for multi-key motions like `gg`
	toast   string // transient status message (e.g. "copied <hash>")

	err error
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

// clearToastMsg clears any transient status toast.
type clearToastMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case clearToastMsg:
		m.toast = ""
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Global keys
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	if m.searching {
		return m.handleSearchKey(msg)
	}

	// Reset multi-key buffer unless the current key is the second half of a
	// known sequence (currently only `gg`).
	prevKey := m.lastKey
	if key != "g" {
		m.lastKey = ""
	}

	switch m.viewState {
	case viewList:
		return m.handleListKey(key, prevKey)
	case viewDetail:
		return m.handleDetailKey(key, prevKey)
	}
	return m, nil
}

// pageSize returns the number of rows the list view can display at once.
func (m model) pageSize() int {
	const headerLines = 3
	const footerLines = 3
	h := m.height - headerLines - footerLines
	if h < 1 {
		return 10
	}
	return h
}

func (m model) handleListKey(key, prevKey string) (tea.Model, tea.Cmd) {
	last := len(m.filtered) - 1
	if last < 0 {
		last = 0
	}
	page := m.pageSize()

	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < last {
			m.cursor++
		}
	case "g":
		if prevKey == "g" {
			m.cursor = 0
			m.lastKey = ""
		} else {
			m.lastKey = "g"
		}
	case "G", "end":
		m.cursor = last
	case "home":
		m.cursor = 0
	case "ctrl+d":
		m.cursor = min(last, m.cursor+page/2)
	case "ctrl+u":
		m.cursor = max(0, m.cursor-page/2)
	case "ctrl+f", "pgdown":
		m.cursor = min(last, m.cursor+page)
	case "ctrl+b", "pgup":
		m.cursor = max(0, m.cursor-page)
	case "enter":
		if len(m.filtered) > 0 {
			m.viewState = viewDetail
			m.scroll = 0
			m.prepareNoteMatches()
			if len(m.noteMatchLines) > 0 {
				m.scroll = m.noteMatchLines[0]
				m.noteMatchIdx = 0
			}
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
	case "y":
		if len(m.filtered) > 0 {
			hash := m.filtered[m.cursor].commitHash
			m.toast = "copied " + hash[:min(12, len(hash))]
			return m, tea.Batch(tea.SetClipboard(hash), clearToastAfter(1500*time.Millisecond))
		}
	}
	return m, nil
}

func (m model) handleDetailKey(key, prevKey string) (tea.Model, tea.Cmd) {
	last := len(m.noteLines) - 1
	if last < 0 {
		last = 0
	}
	page := m.pageSize()

	switch key {
	case "q", "escape":
		// Intentionally do NOT reset m.cursor or m.search so returning to the
		// list preserves selection and active filter.
		m.viewState = viewList
		m.scroll = 0
		m.noteLines = nil
		m.noteMatchLines = nil
		m.noteMatchIdx = 0
	case "up", "k":
		if m.scroll > 0 {
			m.scroll--
		}
	case "down", "j":
		if m.scroll < last {
			m.scroll++
		}
	case "g":
		if prevKey == "g" {
			m.scroll = 0
			m.lastKey = ""
		} else {
			m.lastKey = "g"
		}
	case "G", "end":
		m.scroll = last
	case "home":
		m.scroll = 0
	case "ctrl+d":
		m.scroll = min(last, m.scroll+page/2)
	case "ctrl+u":
		m.scroll = max(0, m.scroll-page/2)
	case "ctrl+f", "pgdown", " ":
		m.scroll = min(last, m.scroll+page)
	case "ctrl+b", "pgup":
		m.scroll = max(0, m.scroll-page)
	case "n":
		if len(m.noteMatchLines) > 0 {
			m.noteMatchIdx = (m.noteMatchIdx + 1) % len(m.noteMatchLines)
			m.scroll = m.noteMatchLines[m.noteMatchIdx]
		}
	case "N":
		if len(m.noteMatchLines) > 0 {
			m.noteMatchIdx = (m.noteMatchIdx - 1 + len(m.noteMatchLines)) % len(m.noteMatchLines)
			m.scroll = m.noteMatchLines[m.noteMatchIdx]
		}
	case "y":
		hash := m.filtered[m.cursor].commitHash
		m.toast = "copied " + hash[:min(12, len(hash))]
		return m, tea.Batch(tea.SetClipboard(hash), clearToastAfter(1500*time.Millisecond))
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

// prepareNoteMatches renders the current note's lines and records which lines
// contain the active search query (case-insensitive).
func (m *model) prepareNoteMatches() {
	if m.cursor >= len(m.filtered) {
		m.noteLines = nil
		m.noteMatchLines = nil
		m.noteMatchIdx = 0
		return
	}
	rendered := renderMarkdown(m.filtered[m.cursor].note, m.width-4)
	m.noteLines = strings.Split(rendered, "\n")
	m.noteMatchLines = nil
	m.noteMatchIdx = 0
	if m.search == "" {
		return
	}
	query := strings.ToLower(m.search)
	for idx, line := range m.noteLines {
		if strings.Contains(strings.ToLower(line), query) {
			m.noteMatchLines = append(m.noteMatchLines, idx)
		}
	}
}

func clearToastAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return clearToastMsg{} })
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
	header := "  lore — decision notes"
	if len(m.filtered) > 0 {
		header += fmt.Sprintf("  (%d/%d)", m.cursor+1, len(m.filtered))
	} else {
		header += fmt.Sprintf("  (0/%d)", len(m.items))
	}
	if m.toast != "" {
		header += "  — " + m.toast
	}
	buf.WriteString(header + "\n")
	if m.search != "" || m.searching {
		buf.WriteString(fmt.Sprintf("  search: %s", m.search))
		if m.searching {
			buf.WriteString("_")
		}
		buf.WriteString(fmt.Sprintf("  (%d results)\n", len(m.filtered)))
	}
	buf.WriteString("\n")

	availableHeight := m.pageSize()

	// Calculate visible window
	start := 0
	if m.cursor >= availableHeight {
		start = m.cursor - availableHeight + 1
	}
	end := start + availableHeight
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	// cursor(2) + hash(8) + gutter(2) + date(16) + gutter(2) = 30 chars of fixed prefix
	fixedPrefix := 2 + hashWidth + rowGutter + dateWidth + rowGutter
	subjectWidth := m.width - fixedPrefix
	if subjectWidth < 20 {
		subjectWidth = 40
	}

	for idx := start; idx < end; idx++ {
		item := m.filtered[idx]
		cursor := "  "
		if idx == m.cursor {
			cursor = "> "
		}
		subject := item.subject
		if len(subject) > subjectWidth {
			subject = subject[:subjectWidth-3] + "..."
		}
		date := item.timestamp.Local().Format("2006-01-02 15:04")
		buf.WriteString(fmt.Sprintf("%s%s  %s  %s\n",
			cursor,
			item.commitHash[:min(hashWidth, len(item.commitHash))],
			date,
			subject,
		))
	}

	// Footer
	buf.WriteString("\n")
	buf.WriteString("  j/k · gg/G · ctrl-d/u · /: search · y: copy hash · enter: view · q: quit\n")

	return buf.String()
}

func (m model) renderDetail() string {
	if m.cursor >= len(m.filtered) {
		return "No item selected"
	}

	item := m.filtered[m.cursor]

	var buf strings.Builder
	header := fmt.Sprintf("  %s  %s  —  %s",
		item.commitHash[:min(12, len(item.commitHash))],
		item.timestamp.Local().Format("2006-01-02 15:04"),
		item.subject,
	)
	if m.toast != "" {
		header += "  (" + m.toast + ")"
	}
	if len(m.noteMatchLines) > 0 {
		header += fmt.Sprintf("  [match %d/%d]", m.noteMatchIdx+1, len(m.noteMatchLines))
	}
	buf.WriteString(header + "\n\n")

	lines := m.noteLines
	if lines == nil {
		rendered := renderMarkdown(item.note, m.width-4)
		lines = strings.Split(rendered, "\n")
	}

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

	buf.WriteString("\n  j/k · gg/G · ctrl-d/u · n/N: next match · y: copy hash · q/esc: back\n")

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

		subject, ts, err := git.GetCommitMeta(commitHash)
		if err != nil {
			// Commit missing or unreadable — skip (matches prior behaviour).
			continue
		}
		items = append(items, noteItem{
			commitHash: commitHash,
			subject:    subject,
			note:       note,
			timestamp:  ts,
		})
	}

	slices.SortFunc(items, func(a, b noteItem) int {
		return b.timestamp.Compare(a.timestamp) // most recent first
	})

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
