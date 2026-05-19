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
	matches    []tmux.Command
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
			if m.cursor+1 < len(m.matches) {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			if len(m.matches) == 0 {
				m.selection = strings.TrimSpace(m.input.Value())
			} else {
				m.selection = m.matches[m.cursor].Name
			}
			m.shouldQuit = true
			return m, tea.Quit
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

	if len(m.matches) == 0 {
		b.WriteString("  no matches\n")
		return b.String()
	}

	visible := len(m.matches)
	if visible > maxVisibleCandidates {
		visible = maxVisibleCandidates
	}

	for i := 0; i < visible; i++ {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		b.WriteString(prefix)
		b.WriteString(formatCommand(m.matches[i]))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *Model) refreshMatches() {
	query := strings.TrimSpace(m.input.Value())
	if query == "" {
		m.matches = append([]tmux.Command(nil), m.commands...)
		if m.cursor >= len(m.matches) {
			m.cursor = max(0, len(m.matches)-1)
		}
		return
	}

	values := make(commandValues, len(m.commands))
	copy(values, m.commands)

	found := fuzzy.FindFrom(query, values)
	m.matches = m.matches[:0]
	for _, match := range found {
		m.matches = append(m.matches, m.commands[match.Index])
	}

	if m.cursor >= len(m.matches) {
		m.cursor = max(0, len(m.matches)-1)
	}
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
