package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gh-liu/tmcp/internal/tmux"
	"github.com/sahilm/fuzzy"
)

const maxVisibleCandidates = 10

type Model struct {
	input      textinput.Model
	commands   []tmux.Command
	candidates []candidate
	cursor     int
	selection  string
	shouldQuit bool
}

func PickCommand(commands []tmux.Command) (string, error) {
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
		input:    input,
		commands: commands,
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
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor+1 < len(m.candidates) {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			if len(m.candidates) == 0 {
				return m, nil
			}

			m.acceptCandidate(m.candidates[m.cursor])
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.refreshMatches()

	return m, cmd
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

	visible := len(m.candidates)
	if visible > maxVisibleCandidates {
		visible = maxVisibleCandidates
	}

	for i := 0; i < visible; i++ {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		b.WriteString(prefix)
		b.WriteString(m.candidates[i].Display)
		b.WriteString("\n")
	}

	return b.String()
}

func (m *Model) refreshMatches() {
	line := m.input.Value()
	tokens := strings.Fields(line)
	endsWithSpace := strings.HasSuffix(line, " ")

	switch {
	case len(tokens) == 0:
		m.candidates = m.commandCandidates("")
	case len(tokens) == 1 && !endsWithSpace:
		m.candidates = m.commandCandidates(tokens[0])
	default:
		command, ok := findCommand(m.commands, tokens[0])
		if !ok {
			m.candidates = nil
			break
		}

		if endsWithSpace && len(tokens) == 1 {
			m.candidates = flagCandidates(command, "")
			break
		}

		current := tokens[len(tokens)-1]
		if strings.HasPrefix(current, "-") {
			m.candidates = flagCandidates(command, current)
			break
		}

		m.candidates = nil
	}

	if m.cursor >= len(m.candidates) {
		m.cursor = max(0, len(m.candidates)-1)
	}
}

func (m *Model) commandCandidates(query string) []candidate {
	if strings.TrimSpace(query) == "" {
		result := make([]candidate, 0, len(m.commands))
		for _, command := range m.commands {
			result = append(result, candidate{
				Value:   command.Name,
				Display: formatCommand(command),
				Kind:    candidateCommand,
			})
		}
		return result
	}

	values := make(commandValues, len(m.commands))
	copy(values, m.commands)

	found := fuzzy.FindFrom(query, values)
	result := make([]candidate, 0, len(found))
	for _, match := range found {
		command := m.commands[match.Index]
		result = append(result, candidate{
			Value:   command.Name,
			Display: formatCommand(command),
			Kind:    candidateCommand,
		})
	}

	return result
}

func (m *Model) acceptCandidate(item candidate) {
	line := m.input.Value()

	switch item.Kind {
	case candidateCommand:
		m.input.SetValue(item.Value + " ")
	case candidateFlag:
		m.input.SetValue(replaceCurrentToken(line, item.Value+" "))
	}

	m.input.SetCursor(len(m.input.Value()))
	m.refreshMatches()
}

func formatCommand(command tmux.Command) string {
	if len(command.Aliases) == 0 {
		return command.Name
	}

	return fmt.Sprintf("%s (%s)", command.Name, strings.Join(command.Aliases, ", "))
}

type commandValues []tmux.Command

func (c commandValues) Len() int {
	return len(c)
}

func (c commandValues) String(i int) string {
	command := c[i]
	if len(command.Aliases) == 0 {
		return command.Name
	}

	return command.Name + " " + strings.Join(command.Aliases, " ")
}

type candidateKind string

const (
	candidateCommand candidateKind = "command"
	candidateFlag    candidateKind = "flag"
)

type candidate struct {
	Value   string
	Display string
	Kind    candidateKind
}

func findCommand(commands []tmux.Command, token string) (tmux.Command, bool) {
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

func flagCandidates(command tmux.Command, prefix string) []candidate {
	result := make([]candidate, 0, len(command.Flags))
	for _, flag := range command.Flags {
		if prefix != "" && !strings.HasPrefix(flag.Name, prefix) {
			continue
		}

		display := flag.Name
		if flag.Value != "" {
			display += " " + flag.Value
		}

		result = append(result, candidate{
			Value:   flag.Name,
			Display: display,
			Kind:    candidateFlag,
		})
	}

	return result
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
