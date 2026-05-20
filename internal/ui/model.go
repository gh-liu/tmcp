package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/gh-liu/tmcp/internal/complete"
	"github.com/gh-liu/tmcp/internal/tmux"
)

const (
	defaultVisibleCandidates = 10
)

var getTerminalSize = term.GetSize

type Model struct {
	input      textinput.Model
	commands   []tmux.Command
	completer  complete.Completer
	candidates []complete.Candidate
	width      int
	height     int
	cursor     int
	offset     int
	selection  string
	shouldQuit bool
}

func ReadCommandLine(commands []tmux.Command) (string, error) {
	model := NewModel(commands)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return "", err
	}

	result, ok := finalModel.(Model)
	if !ok {
		return "", fmt.Errorf("unexpected final model type %T", finalModel)
	}

	return result.selection, nil
}

func NewModel(commands []tmux.Command) Model {
	input := textinput.New()
	input.Focus()
	input.Placeholder = "Type a tmux command"
	input.Prompt = "> "

	model := Model{
		input:     input,
		commands:  commands,
		completer: complete.NewCompleter(),
	}
	model.width, model.height = initialTerminalSize()
	model.refreshMatches()
	return model
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.shouldQuit = true
			return m, tea.Quit
		case tea.KeyUp, tea.KeyCtrlP:
			m.moveCursor(-1)
			return m, nil
		case tea.KeyDown, tea.KeyCtrlN:
			m.moveCursor(1)
			return m, nil
		case tea.KeyTab:
			if len(m.candidates) == 0 {
				return m, nil
			}

			m.acceptCandidate(m.candidates[m.cursor])
			return m, nil
		case tea.KeyEnter:
			m.selection = strings.TrimSpace(m.input.Value())
			m.shouldQuit = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refreshMatches()

	return m, cmd
}

func (m *Model) moveCursor(delta int) {
	next := m.cursor + delta
	if next < 0 || next >= len(m.candidates) {
		return
	}

	m.cursor = next
	m.adjustOffset()
}

func (m Model) View() string {
	if m.shouldQuit {
		return ""
	}

	width := m.renderWidth()
	visibleCandidates := m.visibleCandidates()

	var lines []string
	lines = append(lines, fitLine(m.renderInput(), width), fitLine("", width))
	if len(m.candidates) == 0 {
		lines = append(lines, fitLine("  no matches", width))
		lines = append(lines, emptyLines(max(0, visibleCandidates-1), width)...)
		return joinLines(lines)
	}

	start, end := visibleWindow(len(m.candidates), m.offset, visibleCandidates)
	scrollbar := scrollbarColumn(len(m.candidates), m.offset, visibleCandidates)
	contentWidth := width
	if scrollbar != nil {
		contentWidth = max(0, width-2)
	}

	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		line := prefix + m.candidates[i].Display
		var b strings.Builder
		b.WriteString(fitLine(line, contentWidth))
		if scrollbar != nil {
			b.WriteString(" ")
			b.WriteRune(scrollbar[i-start])
		}
		lines = append(lines, b.String())
	}

	lines = append(lines, emptyLines(max(0, visibleCandidates-(end-start)), width)...)
	return joinLines(lines)
}

func visibleWindow(total, offset, maxVisible int) (start, end int) {
	if total <= maxVisible {
		return 0, total
	}

	start = max(0, offset)
	end = start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}

	return start, end
}

func scrollbarColumn(total, offset, maxVisible int) []rune {
	if total <= maxVisible {
		return nil
	}

	height := min(total, maxVisible)
	bar := make([]rune, height)
	for i := range bar {
		bar[i] = '│'
	}

	thumbSize := max(1, maxVisible*height/total)
	maxOffset := total - maxVisible
	maxThumbStart := height - thumbSize

	thumbStart := 0
	if maxOffset > 0 && maxThumbStart > 0 {
		thumbStart = offset * maxThumbStart / maxOffset
	}

	for i := thumbStart; i < thumbStart+thumbSize && i < len(bar); i++ {
		bar[i] = '█'
	}

	return bar
}

func padRight(s string, width int) string {
	current := ansi.StringWidth(s)
	if current >= width {
		return s
	}
	return s + strings.Repeat(" ", width-current)
}

func fitLine(s string, width int) string {
	if width <= 0 {
		return s
	}

	s = ansi.Truncate(s, width, "")
	return padRight(s, width)
}

func emptyLines(n, width int) []string {
	if n <= 0 {
		return nil
	}

	lines := make([]string, n)
	for i := range lines {
		lines[i] = fitLine("", width)
	}
	return lines
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}

func (m *Model) refreshMatches() {
	candidates, err := m.completer.Complete(context.Background(), m.commands, m.input.Value())
	if err != nil {
		m.candidates = nil
	} else {
		m.candidates = candidates
	}

	if m.cursor >= len(m.candidates) {
		m.cursor = max(0, len(m.candidates)-1)
	}

	m.adjustOffset()
}

func (m *Model) adjustOffset() {
	visibleCandidates := m.visibleCandidates()
	if len(m.candidates) <= visibleCandidates {
		m.offset = 0
		return
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
		return
	}

	bottom := m.offset + visibleCandidates - 1
	if m.cursor > bottom {
		m.offset = m.cursor - visibleCandidates + 1
	}
}

func (m *Model) acceptCandidate(item complete.Candidate) {
	line := m.input.Value()

	switch item.Kind {
	case complete.CandidateCommand:
		m.input.SetValue(item.Value + " ")
	case complete.CandidateFlag, complete.CandidateValue:
		m.input.SetValue(replaceCurrentToken(line, item.Value+" "))
	}

	m.input.SetCursor(len(m.input.Value()))
	m.cursor = 0
	m.offset = 0
	m.refreshMatches()
}

func (m Model) renderWidth() int {
	if m.width > 0 {
		return m.width
	}

	return 80
}

func (m Model) visibleCandidates() int {
	if m.height <= 0 {
		return defaultVisibleCandidates
	}

	visible := m.height - 3
	if visible < 0 {
		return 0
	}

	return visible
}

func (m Model) renderInput() string {
	value := m.input.Value()
	if value == "" {
		return "> " + stylePlaceholder(m.input.Placeholder)
	}

	if placeholder, ok := m.pendingValuePlaceholder(value); ok {
		return "> " + value + stylePlaceholder(placeholder)
	}

	return "> " + value
}

func (m Model) pendingValuePlaceholder(line string) (string, bool) {
	if !strings.HasSuffix(line, " ") {
		return "", false
	}

	tokens := strings.Fields(line)
	if len(tokens) < 2 {
		return "", false
	}

	command, ok := findCommandSpec(m.commands, tokens[0])
	if !ok {
		return "", false
	}

	flag, ok := findCommandFlag(command, tokens[len(tokens)-1])
	if !ok || flag.Value == "" {
		return "", false
	}

	return flag.Value, true
}

func replaceCurrentToken(line, replacement string) string {
	if line == "" {
		return replacement
	}

	trimmed := strings.TrimRight(line, " \t")
	if trimmed == "" || strings.HasSuffix(line, " ") {
		return line + replacement
	}

	lastSpace := strings.LastIndexAny(trimmed, " \t")
	if lastSpace == -1 {
		return replacement
	}

	return trimmed[:lastSpace+1] + replacement
}

func stylePlaceholder(s string) string {
	return "\x1b[90m" + s + "\x1b[0m"
}

func findCommandSpec(commands []tmux.Command, token string) (tmux.Command, bool) {
	for _, command := range commands {
		if command.Name == token {
			return command, true
		}

		for _, alias := range command.Aliases {
			if alias == token {
				return command, true
			}
		}
	}

	return tmux.Command{}, false
}

func findCommandFlag(command tmux.Command, token string) (tmux.Flag, bool) {
	for _, flag := range command.Flags {
		if flag.Name == token {
			return flag, true
		}
	}

	return tmux.Flag{}, false
}

func initialTerminalSize() (int, int) {
	for _, fd := range []uintptr{
		os.Stdin.Fd(),
		os.Stdout.Fd(),
		os.Stderr.Fd(),
	} {
		width, height, err := getTerminalSize(fd)
		if err == nil && width > 0 && height > 0 {
			return width, height
		}
	}

	width, _ := strconv.Atoi(os.Getenv("COLUMNS"))
	height, _ := strconv.Atoi(os.Getenv("LINES"))
	if width > 0 && height > 0 {
		return width, height
	}

	return 0, 0
}
