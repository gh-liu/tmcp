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
	"github.com/sahilm/fuzzy"
)

const (
	defaultVisibleCandidates = 10
)

var getTerminalSize = term.GetSize

type Model struct {
	input       textinput.Model
	commands    []tmux.Command
	completer   complete.Completer
	candidates  []complete.Candidate
	history     []string
	width       int
	height      int
	cursor      int
	offset      int
	historyPos  int
	historyTmp  string
	historyMode bool
	selection   string
	shouldQuit  bool
}

func ReadCommandLine(commands []tmux.Command) (string, error) {
	return ReadCommandLineWithHistory(commands, nil)
}

func ReadCommandLineWithHistory(commands []tmux.Command, history []string) (string, error) {
	model := NewModelWithHistory(commands, history)
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
	return NewModelWithHistory(commands, nil)
}

func NewModelWithHistory(commands []tmux.Command, history []string) Model {
	input := textinput.New()
	input.Focus()
	input.Placeholder = "Type a tmux command"
	input.Prompt = "> "

	model := Model{
		input:      input,
		commands:   commands,
		completer:  complete.NewCompleter(),
		history:    append([]string(nil), history...),
		historyPos: len(history),
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
		case tea.KeyUp:
			m.moveCursor(-1)
			return m, nil
		case tea.KeyDown:
			m.moveCursor(1)
			return m, nil
		case tea.KeyCtrlP:
			m.moveCursor(-1)
			return m, nil
		case tea.KeyCtrlN:
			m.moveCursor(1)
			return m, nil
		case tea.KeyCtrlR:
			m.toggleHistoryMode()
			return m, nil
		case tea.KeyTab:
			if len(m.candidates) == 0 {
				return m, nil
			}

			m.acceptCandidate(m.candidates[m.cursor])
			return m, nil
		case tea.KeyEnter:
			if len(m.candidates) > 0 {
				m.acceptCandidate(m.candidates[m.cursor])
			}
			m.selection = strings.TrimSpace(m.input.Value())
			m.shouldQuit = true
			return m, tea.Quit
		}
	}

	previousValue := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if !m.historyMode && m.input.Value() != previousValue {
		m.resetHistoryNavigation()
	}
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
	labelWidth := candidateLabelWidth(m.candidates[start:end])

	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		label, note := candidateDisplayParts(m.candidates[i])
		line := prefix + label
		if note != "" {
			line = prefix + padRight(label, labelWidth) + "  " + stylePlaceholder(note)
		}
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
	if m.historyMode {
		m.candidates = historyCandidates(m.history, m.input.Value())
		if m.cursor >= len(m.candidates) {
			m.cursor = max(0, len(m.candidates)-1)
		}
		m.adjustOffset()
		return
	}

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

func (m *Model) toggleHistoryMode() {
	m.historyMode = !m.historyMode
	m.cursor = 0
	m.offset = 0
	m.refreshMatches()
}

func (m *Model) resetHistoryNavigation() {
	m.historyPos = len(m.history)
	m.historyTmp = ""
}

func (m *Model) historyPrev() {
	if len(m.history) == 0 {
		return
	}

	if m.historyPos == len(m.history) {
		m.historyTmp = m.input.Value()
		m.historyPos = len(m.history) - 1
	} else if m.historyPos > 0 {
		m.historyPos--
	} else {
		return
	}

	m.historyMode = false
	m.input.SetValue(m.history[m.historyPos])
	m.input.SetCursor(len(m.input.Value()))
	m.cursor = 0
	m.offset = 0
	m.refreshMatches()
}

func (m *Model) historyNext() {
	if len(m.history) == 0 || m.historyPos == len(m.history) {
		return
	}

	m.historyPos++
	if m.historyPos == len(m.history) {
		m.input.SetValue(m.historyTmp)
		m.historyTmp = ""
	} else {
		m.input.SetValue(m.history[m.historyPos])
	}

	m.historyMode = false
	m.input.SetCursor(len(m.input.Value()))
	m.cursor = 0
	m.offset = 0
	m.refreshMatches()
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
		m.historyMode = false
		m.resetHistoryNavigation()
		m.input.SetValue(item.Value + " ")
	case complete.CandidateHistory:
		m.historyMode = false
		m.resetHistoryNavigation()
		m.input.SetValue(item.Value)
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
	prompt := "> "
	if m.historyMode {
		prompt = styleHistoryPrompt("r> ")
	}

	value := m.input.Value()
	if value == "" {
		return prompt + stylePlaceholder(m.input.Placeholder)
	}

	if placeholder, ok := m.pendingValuePlaceholder(value); ok {
		return prompt + value + stylePlaceholder(placeholder)
	}

	return prompt + value
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

func styleHistoryPrompt(s string) string {
	return "\x1b[36m" + s + "\x1b[0m"
}

func candidateDisplayParts(candidate complete.Candidate) (label, note string) {
	if candidate.Kind == complete.CandidateCommand {
		if note, ok := commandNote(candidate.Value); ok {
			return candidate.Display, note
		}
		return candidate.Display, ""
	}

	if candidate.Kind == complete.CandidateHistory {
		return candidate.Display, ""
	}

	if candidate.Kind != complete.CandidateFlag {
		return candidate.Display, ""
	}

	flag, value, ok := strings.Cut(candidate.Display, " ")
	if !ok || value == "" {
		return candidate.Display, candidate.Note
	}

	if note := candidate.Note; note != "" {
		return flag, note
	}
	return flag, value
}

func historyCandidates(history []string, query string) []complete.Candidate {
	if len(history) == 0 {
		return nil
	}

	values := make([]string, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		values = append(values, history[i])
	}

	if strings.TrimSpace(query) == "" {
		candidates := make([]complete.Candidate, 0, len(values))
		for _, value := range values {
			candidates = append(candidates, complete.Candidate{
				Value:   value,
				Display: value,
				Kind:    complete.CandidateHistory,
			})
		}
		return candidates
	}

	matches := fuzzy.Find(query, values)
	candidates := make([]complete.Candidate, 0, len(matches))
	for _, match := range matches {
		candidates = append(candidates, complete.Candidate{
			Value:   values[match.Index],
			Display: values[match.Index],
			Kind:    complete.CandidateHistory,
		})
	}

	return candidates
}

func candidateLabelWidth(candidates []complete.Candidate) int {
	width := 0
	for _, candidate := range candidates {
		label, _ := candidateDisplayParts(candidate)
		if w := ansi.StringWidth(label); w > width {
			width = w
		}
	}
	return width
}

func commandNote(command string) (string, bool) {
	switch command {
	case "choose-buffer":
		return "choose a paste buffer from a list", true
	case "clear-history":
		return "clear pane history", true
	case "delete-buffer":
		return "delete a paste buffer", true
	case "list-buffers":
		return "list paste buffers", true
	case "load-buffer":
		return "load a paste buffer from a file", true
	case "paste-buffer":
		return "paste a buffer into a pane", true
	case "save-buffer":
		return "save a paste buffer to a file", true
	case "set-buffer":
		return "set or rename a paste buffer", true
	case "show-buffer":
		return "show paste buffer contents", true
	case "set-environment":
		return "set or unset an environment variable", true
	case "show-environment":
		return "show environment variables", true
	case "set-hook":
		return "set, unset, or run a hook", true
	case "show-hooks":
		return "show hooks", true
	case "set-option":
		return "set a tmux option", true
	case "set-window-option":
		return "set a window option", true
	case "show-options":
		return "show tmux options", true
	case "show-window-options":
		return "show window options", true
	case "clock-mode":
		return "show a large clock", true
	case "if-shell":
		return "run commands based on a shell result", true
	case "lock-server":
		return "lock all clients using lock-command", true
	case "run-shell":
		return "run a shell or tmux command in the background", true
	case "wait-for":
		return "wait on, signal, or lock a channel", true
	case "bind-key":
		return "bind a key to a tmux command", true
	case "clear-prompt-history":
		return "clear command prompt history", true
	case "command-prompt":
		return "open the tmux command prompt", true
	case "confirm-before":
		return "ask for confirmation before running a command", true
	case "copy-mode":
		return "enter copy mode", true
	case "customize-mode":
		return "browse and edit options and key bindings", true
	case "display-menu":
		return "show an interactive tmux menu", true
	case "display-panes":
		return "show numbered pane indicators", true
	case "attach-session":
		return "attach or switch to a session", true
	case "detach-client":
		return "detach one or more clients", true
	case "has-session":
		return "check whether a session exists", true
	case "kill-server":
		return "stop the tmux server", true
	case "kill-session":
		return "destroy a session", true
	case "list-clients":
		return "list connected clients", true
	case "list-commands":
		return "list tmux command syntax", true
	case "lock-client":
		return "lock a client", true
	case "lock-session":
		return "lock all clients in a session", true
	case "new-session":
		return "create a new session", true
	case "refresh-client":
		return "refresh a client display", true
	case "rename-session":
		return "rename a session", true
	case "server-access":
		return "change tmux socket access permissions", true
	case "show-messages":
		return "show server messages and debug info", true
	case "source-file":
		return "load tmux commands from a file", true
	case "start-server":
		return "start the tmux server", true
	case "suspend-client":
		return "suspend a client", true
	case "switch-client":
		return "switch a client to another session", true
	case "send-keys":
		return "send keys to a pane or client", true
	case "split-window":
		return "split a pane and create a new one", true
	case "new-window":
		return "create a new window", true
	case "kill-pane":
		return "destroy a pane", true
	case "kill-window":
		return "destroy a window", true
	case "select-pane":
		return "make a pane active", true
	case "select-window":
		return "switch to a window", true
	case "display-popup":
		return "show a popup running a shell command", true
	case "display-message":
		return "show or print a tmux message", true
	case "choose-client":
		return "choose a client from a list", true
	case "choose-tree":
		return "choose a session, window, or pane from a tree", true
	case "list-panes":
		return "list panes", true
	case "list-windows":
		return "list windows", true
	case "list-sessions":
		return "list sessions", true
	case "list-keys":
		return "list key bindings", true
	case "capture-pane":
		return "capture pane contents", true
	case "pipe-pane":
		return "pipe pane output to or from a command", true
	case "join-pane":
		return "move a pane into another split", true
	case "move-pane":
		return "move a pane into another split", true
	case "break-pane":
		return "move a pane into its own window", true
	case "link-window":
		return "link a window into another session", true
	case "move-window":
		return "move a window to a new index", true
	case "next-layout":
		return "switch to the next layout", true
	case "next-window":
		return "switch to the next window", true
	case "previous-layout":
		return "switch to the previous layout", true
	case "previous-window":
		return "switch to the previous window", true
	case "last-pane":
		return "switch to the previous pane", true
	case "last-window":
		return "switch to the previous window", true
	case "swap-pane":
		return "swap two panes", true
	case "swap-window":
		return "swap two windows", true
	case "resize-pane":
		return "resize a pane", true
	case "resize-window":
		return "resize a window", true
	case "respawn-pane":
		return "restart a dead pane command", true
	case "respawn-window":
		return "restart a dead window command", true
	case "rotate-window":
		return "rotate pane positions in a window", true
	case "rename-window":
		return "rename a window", true
	case "select-layout":
		return "apply a window layout", true
	case "find-window":
		return "search window names, titles, or contents", true
	case "send-prefix":
		return "send the tmux prefix key to a pane", true
	case "show-prompt-history":
		return "show command prompt history", true
	case "unlink-window":
		return "unlink a window from a session", true
	case "unbind-key":
		return "remove a key binding", true
	}

	return "", false
}

func placeholderNote(placeholder string) (string, bool) {
	switch placeholder {
	case "working-directory", "start-directory":
		return "working directory", true
	case "session-name":
		return "session name", true
	case "window-name":
		return "window name", true
	case "buffer-name", "new-buffer-name":
		return "buffer name", true
	case "key-table":
		return "key table", true
	case "key-format":
		return "key label format", true
	case "sort-order":
		return "sort field", true
	case "border-lines":
		return "border line style", true
	case "border-style":
		return "border style", true
	case "selected-style", "style":
		return "tmux style", true
	case "title":
		return "title text", true
	case "template":
		return "command template", true
	case "prompt", "prompts":
		return "prompt text", true
	case "type":
		return "prompt type", true
	case "note":
		return "key binding note", true
	case "flags":
		return "comma-separated flags", true
	case "inputs":
		return "initial prompt values", true
	case "confirm_key":
		return "confirmation key", true
	case "starting-choice":
		return "initial menu selection", true
	case "position":
		return "popup or menu position", true
	case "height":
		return "height", true
	case "width":
		return "width", true
	case "delay", "duration":
		return "time delay", true
	case "buffer-index":
		return "buffer index", true
	case "start-line":
		return "start line", true
	case "end-line":
		return "end line", true
	case "match-string":
		return "search pattern", true
	case "size":
		return "pane or window size", true
	case "separator":
		return "separator text", true
	case "pane:state":
		return "pane state", true
	case "name:what:format":
		return "subscription spec", true
	case "XxY":
		return "client size", true
	case "adjustment":
		return "size adjustment", true
	case "new-name":
		return "new name", true
	case "data":
		return "buffer contents", true
	case "hook":
		return "hook name", true
	case "option":
		return "option name", true
	case "name":
		return "variable name", true
	case "value":
		return "value", true
	case "format":
		return "tmux format", true
	case "filter":
		return "format expression", true
	case "path":
		return "filesystem path", true
	case "shell-command":
		return "shell command", true
	case "layout-name":
		return "layout preset", true
	case "command":
		return "tmux command", true
	case "arguments":
		return "command arguments", true
	case "key":
		return "key name", true
	case "user":
		return "user name", true
	case "channel":
		return "channel name", true
	case "environment":
		return "VARIABLE=value", true
	}

	return "", false
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
