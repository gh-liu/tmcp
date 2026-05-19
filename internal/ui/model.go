package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gh-liu/tmcp/internal/complete"
	"github.com/gh-liu/tmcp/internal/tmux"
	"github.com/mattn/go-runewidth"
)

const maxVisibleCandidates = 10

type Model struct {
	input      textinput.Model
	commands   []tmux.Command
	completer  complete.Completer
	candidates []complete.Candidate
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
	model.refreshMatches()
	return model
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

	var b strings.Builder
	b.WriteString("tmcp\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	if len(m.candidates) == 0 {
		b.WriteString("  no matches\n")
		return b.String()
	}

	start, end := visibleWindow(len(m.candidates), m.offset, maxVisibleCandidates)
	scrollbar := scrollbarColumn(len(m.candidates), m.offset, maxVisibleCandidates)
	contentWidth := candidateWidth(m.candidates, m.cursor, 0)

	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		line := prefix + m.candidates[i].Display
		b.WriteString(padRight(line, contentWidth))
		if scrollbar != nil {
			b.WriteString(" ")
			b.WriteRune(scrollbar[i-start])
		}
		b.WriteString("\n")
	}

	return b.String()
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

func candidateWidth(candidates []complete.Candidate, cursor, start int) int {
	width := 0
	for i, candidate := range candidates {
		prefix := "  "
		if start+i == cursor {
			prefix = "> "
		}
		lineWidth := runewidth.StringWidth(prefix + candidate.Display)
		if lineWidth > width {
			width = lineWidth
		}
	}
	return width
}

func padRight(s string, width int) string {
	current := runewidth.StringWidth(s)
	if current >= width {
		return s
	}
	return s + strings.Repeat(" ", width-current)
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
	if len(m.candidates) <= maxVisibleCandidates {
		m.offset = 0
		return
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
		return
	}

	bottom := m.offset + maxVisibleCandidates - 1
	if m.cursor > bottom {
		m.offset = m.cursor - maxVisibleCandidates + 1
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
	m.refreshMatches()
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
