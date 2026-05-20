package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/gh-liu/tmcp/internal/complete"
	"github.com/gh-liu/tmcp/internal/config"
	"github.com/gh-liu/tmcp/internal/tmux"
)

func TestVisibleWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		total      int
		offset     int
		maxVisible int
		wantStart  int
		wantEnd    int
	}{
		{
			name:       "all items fit",
			total:      3,
			offset:     2,
			maxVisible: 10,
			wantStart:  0,
			wantEnd:    3,
		},
		{
			name:       "offset at top keeps first page",
			total:      20,
			offset:     0,
			maxVisible: 10,
			wantStart:  0,
			wantEnd:    10,
		},
		{
			name:       "offset preserves middle page",
			total:      20,
			offset:     1,
			maxVisible: 10,
			wantStart:  1,
			wantEnd:    11,
		},
		{
			name:       "offset near end shows last page",
			total:      20,
			offset:     15,
			maxVisible: 10,
			wantStart:  10,
			wantEnd:    20,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotStart, gotEnd := visibleWindow(tc.total, tc.offset, tc.maxVisible)
			if gotStart != tc.wantStart || gotEnd != tc.wantEnd {
				t.Fatalf("visibleWindow(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tc.total, tc.offset, tc.maxVisible, gotStart, gotEnd, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestAdjustOffset(t *testing.T) {
	t.Parallel()

	model := Model{
		candidates: make([]complete.Candidate, 20),
	}

	model.cursor = 9
	model.adjustOffset()
	if model.offset != 0 {
		t.Fatalf("offset at cursor 9 = %d, want 0", model.offset)
	}

	model.cursor = 10
	model.adjustOffset()
	if model.offset != 1 {
		t.Fatalf("offset at cursor 10 = %d, want 1", model.offset)
	}

	model.cursor = 9
	model.adjustOffset()
	if model.offset != 1 {
		t.Fatalf("offset at cursor 9 after scrolling down = %d, want 1", model.offset)
	}

	model.cursor = 0
	model.adjustOffset()
	if model.offset != 0 {
		t.Fatalf("offset at cursor 0 = %d, want 0", model.offset)
	}
}

func TestMovePage(t *testing.T) {
	t.Parallel()

	model := Model{
		height:     13,
		candidates: make([]complete.Candidate, 20),
	}

	model.movePage(1)
	if model.cursor != 5 {
		t.Fatalf("cursor after movePage(1) = %d, want 5", model.cursor)
	}
	if model.offset != 0 {
		t.Fatalf("offset after movePage(1) = %d, want 0", model.offset)
	}

	model.movePage(1)
	if model.cursor != 10 {
		t.Fatalf("cursor after second movePage(1) = %d, want 10", model.cursor)
	}
	if model.offset != 1 {
		t.Fatalf("offset after second movePage(1) = %d, want 1", model.offset)
	}

	model.movePage(-1)
	if model.cursor != 5 {
		t.Fatalf("cursor after movePage(-1) = %d, want 5", model.cursor)
	}

	model.cursor = 19
	model.adjustOffset()
	model.movePage(1)
	if model.cursor != 19 {
		t.Fatalf("cursor at bottom after movePage(1) = %d, want 19", model.cursor)
	}

	model.cursor = 0
	model.adjustOffset()
	model.movePage(-1)
	if model.cursor != 0 {
		t.Fatalf("cursor at top after movePage(-1) = %d, want 0", model.cursor)
	}
}

func TestScrollbarColumn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		total      int
		offset     int
		maxVisible int
		want       string
	}{
		{
			name:       "no scrollbar when all items fit",
			total:      3,
			offset:     0,
			maxVisible: 10,
			want:       "",
		},
		{
			name:       "thumb at top",
			total:      20,
			offset:     0,
			maxVisible: 10,
			want:       "█████│││││",
		},
		{
			name:       "thumb at bottom",
			total:      20,
			offset:     10,
			maxVisible: 10,
			want:       "│││││█████",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := scrollbarColumn(tc.total, tc.offset, tc.maxVisible)
			if string(got) != tc.want {
				t.Fatalf("scrollbarColumn(%d, %d, %d) = %q, want %q",
					tc.total, tc.offset, tc.maxVisible, string(got), tc.want)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	t.Parallel()

	got := padRight("x", 4)
	if got != "x   " {
		t.Fatalf("padRight() = %q, want %q", got, "x   ")
	}
}

func TestCenterLines(t *testing.T) {
	t.Parallel()

	got := fitLine("abcd", 10)
	want := "abcd      "
	if got != want {
		t.Fatalf("fitLine() = %q, want %q", got, want)
	}
}

func TestFitLineHandlesANSIWidth(t *testing.T) {
	t.Parallel()

	got := fitLine(stylePlaceholder("abcd"), 10)
	if width := ansi.StringWidth(got); width != 10 {
		t.Fatalf("fitLine() display width = %d, want 10", width)
	}
}

func TestRenderWidth(t *testing.T) {
	t.Parallel()

	model := Model{width: 120}
	if got := model.renderWidth(); got != 120 {
		t.Fatalf("renderWidth() = %d, want 120", got)
	}

	model.width = 0
	if got := model.renderWidth(); got != 80 {
		t.Fatalf("renderWidth() = %d, want 80", got)
	}
}

func TestVisibleCandidates(t *testing.T) {
	t.Parallel()

	model := Model{height: 24}
	if got := model.visibleCandidates(); got != 21 {
		t.Fatalf("visibleCandidates() = %d, want 21", got)
	}

	model.height = 0
	if got := model.visibleCandidates(); got != defaultVisibleCandidates {
		t.Fatalf("visibleCandidates() = %d, want %d", got, defaultVisibleCandidates)
	}
}

func TestViewPlacesScrollbarAtRightEdge(t *testing.T) {
	t.Parallel()

	model := Model{
		width:  20,
		height: 6,
		candidates: []complete.Candidate{
			{Display: "a"},
			{Display: "b"},
			{Display: "c"},
			{Display: "d"},
			{Display: "e"},
		},
	}

	lines := strings.Split(strings.TrimSuffix(model.View(), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("View() returned %d lines, want at least 3", len(lines))
	}

	line := lines[2]
	if got := len([]rune(line)); got != 20 {
		t.Fatalf("candidate line width = %d, want 20", got)
	}
	if line[19:] != "█" {
		t.Fatalf("candidate line right edge = %q, want scrollbar at last column", line[19:])
	}
}

func TestViewAlignsFlagNotes(t *testing.T) {
	t.Parallel()

	model := Model{
		width:  80,
		height: 9,
		candidates: []complete.Candidate{
			{Display: "-d", Note: "keep current active pane", Kind: complete.CandidateFlag},
			{Display: "-D", Note: "swap with next pane", Kind: complete.CandidateFlag},
			{Display: "-U", Note: "swap with previous pane", Kind: complete.CandidateFlag},
			{Display: "-Z", Note: "keep zoom", Kind: complete.CandidateFlag},
			{Display: "-s src-pane", Kind: complete.CandidateFlag},
			{Display: "-t dst-pane", Kind: complete.CandidateFlag},
		},
	}

	lines := strings.Split(strings.TrimSuffix(model.View(), "\n"), "\n")
	if len(lines) < 8 {
		t.Fatalf("View() returned %d lines, want at least 8", len(lines))
	}

	first := noteStartColumn(t, lines[2], "keep current active pane")
	second := noteStartColumn(t, lines[3], "swap with next pane")
	third := noteStartColumn(t, lines[4], "swap with previous pane")
	fourth := noteStartColumn(t, lines[5], "keep zoom")
	fifth := noteStartColumn(t, lines[6], "src-pane")
	sixth := noteStartColumn(t, lines[7], "dst-pane")

	if first != second || second != third || third != fourth || fourth != fifth || fifth != sixth {
		t.Fatalf("note columns = %d, %d, %d, %d, %d, %d, want all equal", first, second, third, fourth, fifth, sixth)
	}
}

func noteStartColumn(t *testing.T, line, note string) int {
	t.Helper()

	idx := strings.Index(line, note)
	if idx == -1 {
		t.Fatalf("line %q missing note %q", line, note)
	}

	return ansi.StringWidth(line[:idx])
}

func TestInitialTerminalSize(t *testing.T) {
	previous := getTerminalSize
	previousColumns, hadColumns := os.LookupEnv("COLUMNS")
	previousLines, hadLines := os.LookupEnv("LINES")
	t.Cleanup(func() {
		getTerminalSize = previous
		if hadColumns {
			_ = os.Setenv("COLUMNS", previousColumns)
		} else {
			_ = os.Unsetenv("COLUMNS")
		}
		if hadLines {
			_ = os.Setenv("LINES", previousLines)
		} else {
			_ = os.Unsetenv("LINES")
		}
	})
	_ = os.Unsetenv("COLUMNS")
	_ = os.Unsetenv("LINES")

	var seen []uintptr
	getTerminalSize = func(fd uintptr) (int, int, error) {
		seen = append(seen, fd)
		if fd == os.Stdin.Fd() {
			return 132, 40, nil
		}
		return 0, 0, errors.New("unexpected fd")
	}

	width, height := initialTerminalSize()
	if width != 132 || height != 40 {
		t.Fatalf("initialTerminalSize() = (%d, %d), want (132, 40)", width, height)
	}
	if len(seen) != 1 || seen[0] != os.Stdin.Fd() {
		t.Fatalf("initialTerminalSize() probed %v, want only stdin", seen)
	}
}

func TestInitialTerminalSizeFallsBackOnError(t *testing.T) {
	previous := getTerminalSize
	previousColumns, hadColumns := os.LookupEnv("COLUMNS")
	previousLines, hadLines := os.LookupEnv("LINES")
	t.Cleanup(func() {
		getTerminalSize = previous
		if hadColumns {
			_ = os.Setenv("COLUMNS", previousColumns)
		} else {
			_ = os.Unsetenv("COLUMNS")
		}
		if hadLines {
			_ = os.Setenv("LINES", previousLines)
		} else {
			_ = os.Unsetenv("LINES")
		}
	})
	_ = os.Unsetenv("COLUMNS")
	_ = os.Unsetenv("LINES")

	getTerminalSize = func(uintptr) (int, int, error) {
		return 0, 0, errors.New("boom")
	}

	width, height := initialTerminalSize()
	if width != 0 || height != 0 {
		t.Fatalf("initialTerminalSize() = (%d, %d), want (0, 0)", width, height)
	}
}

func TestInitialTerminalSizeFallsBackToEnvironment(t *testing.T) {
	previous := getTerminalSize
	previousColumns, hadColumns := os.LookupEnv("COLUMNS")
	previousLines, hadLines := os.LookupEnv("LINES")
	t.Cleanup(func() {
		getTerminalSize = previous
		if hadColumns {
			_ = os.Setenv("COLUMNS", previousColumns)
		} else {
			_ = os.Unsetenv("COLUMNS")
		}
		if hadLines {
			_ = os.Setenv("LINES", previousLines)
		} else {
			_ = os.Unsetenv("LINES")
		}
	})

	getTerminalSize = func(uintptr) (int, int, error) {
		return 0, 0, errors.New("boom")
	}
	_ = os.Setenv("COLUMNS", "144")
	_ = os.Setenv("LINES", "52")

	width, height := initialTerminalSize()
	if width != 144 || height != 52 {
		t.Fatalf("initialTerminalSize() = (%d, %d), want (144, 52)", width, height)
	}
}

func TestAcceptCandidateResetsCursorForNextCandidateSet(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{
			Name:    "send-keys",
			Aliases: []string{"send"},
			Flags: []tmux.Flag{
				{Name: "-F"},
				{Name: "-t", Value: "target-pane"},
			},
		},
	})

	model.input.SetValue("send")
	model.refreshMatches()
	model.cursor = 1

	model.acceptCandidate(complete.Candidate{
		Value:   "send-keys",
		Display: "send-keys (send)",
		Kind:    complete.CandidateCommand,
	})

	if model.cursor != 0 {
		t.Fatalf("cursor after acceptCandidate() = %d, want 0", model.cursor)
	}

	if len(model.candidates) == 0 {
		t.Fatalf("candidates after acceptCandidate() = 0, want flags")
	}

	if model.candidates[0].Display != "-F" {
		t.Fatalf("first candidate after acceptCandidate() = %q, want %q", model.candidates[0].Display, "-F")
	}
}

func TestEnterSubmitsTypedInputWithoutAcceptingCandidate(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{Name: "select-layout"},
		{Name: "select-pane"},
		{Name: "select-window"},
	})

	model.input.SetValue("select-")
	model.refreshMatches()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := updated.(Model)

	if got.selection != "select-" {
		t.Fatalf("selection after Enter = %q, want %q", got.selection, "select-")
	}

	if !got.shouldQuit {
		t.Fatalf("shouldQuit after Enter = %v, want true", got.shouldQuit)
	}
}

func TestRenderInputShowsGhostTextForSelectedCandidate(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{Name: "select-layout"},
		{Name: "select-pane"},
		{Name: "select-window"},
	})
	model.input.SetValue("select-")
	model.refreshMatches()

	got := model.renderInput()
	if !strings.Contains(got, "pane") {
		t.Fatalf("renderInput() = %q, want ghost suffix for current candidate", got)
	}
	if got == "> select-pane" {
		t.Fatalf("renderInput() = %q, want styled ghost suffix, not committed input", got)
	}
	if model.input.Value() != "select-" {
		t.Fatalf("input value = %q, want unchanged typed input", model.input.Value())
	}
}

func TestRenderInputShowsGhostTextForSelectedValueCandidate(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{
			Name:       "select-layout",
			Positional: []string{"layout-name"},
		},
	})
	model.input.SetValue("select-layout main-")
	model.refreshMatches()

	got := model.renderInput()
	if !strings.Contains(got, "horizontal") {
		t.Fatalf("renderInput() = %q, want ghost suffix for current value candidate", got)
	}
	if model.input.Value() != "select-layout main-" {
		t.Fatalf("input value = %q, want unchanged typed input", model.input.Value())
	}
}

func TestRightAcceptsCurrentGhostCandidate(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{Name: "select-layout"},
		{Name: "select-pane"},
		{Name: "select-window"},
	})
	model.input.SetValue("select-")
	model.refreshMatches()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRight})
	got := updated.(Model)

	if got.input.Value() != "select-pane " {
		t.Fatalf("input after Right = %q, want %q", got.input.Value(), "select-pane ")
	}
}

func TestCtrlRTogglesHistorySearchCandidates(t *testing.T) {
	t.Parallel()

	model := NewModelWithHistory([]tmux.Command{{Name: "send-keys"}}, []string{
		"select-pane -L",
		"send-keys -t main:0 Enter",
	})
	model.input.SetValue("send")

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	got := updated.(Model)

	if !got.historyMode {
		t.Fatalf("historyMode after Ctrl-R = %v, want true", got.historyMode)
	}
	if len(got.candidates) != 1 || got.candidates[0].Value != "send-keys -t main:0 Enter" {
		t.Fatalf("history candidates after Ctrl-R = %#v, want filtered history result", got.candidates)
	}
	if !strings.Contains(got.renderInput(), "r> ") {
		t.Fatalf("renderInput() in history mode = %q, want highlighted history prompt", got.renderInput())
	}
	if strings.HasPrefix(got.renderInput(), "r> ") {
		t.Fatalf("renderInput() in history mode = %q, want styled prompt", got.renderInput())
	}
}

func TestAcceptHistoryCandidateLeavesHistoryMode(t *testing.T) {
	t.Parallel()

	model := NewModelWithHistory(nil, []string{"select-pane -L"})
	model.historyMode = true
	model.candidates = []complete.Candidate{
		{Value: "select-pane -L", Display: "select-pane -L", Kind: complete.CandidateHistory},
	}

	model.acceptCandidate(model.candidates[0])

	if model.historyMode {
		t.Fatalf("historyMode after acceptCandidate() = %v, want false", model.historyMode)
	}
	if model.input.Value() != "select-pane -L" {
		t.Fatalf("input after acceptCandidate() = %q, want %q", model.input.Value(), "select-pane -L")
	}
}

func TestHistoryPrevAndNextRestoreDraft(t *testing.T) {
	t.Parallel()

	model := NewModelWithHistory(nil, []string{
		"kill-pane -t 1",
		"select-pane -L",
	})
	model.input.SetValue("send")

	model.historyPrev()
	if model.input.Value() != "select-pane -L" {
		t.Fatalf("historyPrev() input = %q, want newest history entry", model.input.Value())
	}

	model.historyPrev()
	if model.input.Value() != "kill-pane -t 1" {
		t.Fatalf("second historyPrev() input = %q, want older history entry", model.input.Value())
	}

	model.historyNext()
	if model.input.Value() != "select-pane -L" {
		t.Fatalf("historyNext() input = %q, want newer history entry", model.input.Value())
	}

	model.historyNext()
	if model.input.Value() != "send" {
		t.Fatalf("historyNext() at draft = %q, want %q", model.input.Value(), "send")
	}
}

func TestHistoryNavigationResetAfterEditing(t *testing.T) {
	t.Parallel()

	model := NewModelWithHistory(nil, []string{"select-pane -L"})
	model.input.SetValue("draft")
	model.historyPrev()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	got := updated.(Model)

	if got.historyPos != len(got.history) {
		t.Fatalf("historyPos after editing = %d, want %d", got.historyPos, len(got.history))
	}
}

func TestCtrlPNStillNavigateCandidatesOutsideHistoryMode(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{Name: "select-layout"},
		{Name: "select-pane"},
		{Name: "select-window"},
	})
	model.input.SetValue("select-")
	model.refreshMatches()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	got := updated.(Model)
	if got.cursor != 1 {
		t.Fatalf("cursor after Ctrl-N = %d, want 1", got.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	got = updated.(Model)
	if got.cursor != 0 {
		t.Fatalf("cursor after Ctrl-P = %d, want 0", got.cursor)
	}
}

func TestCtrlDUHalfPageScrollCandidates(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{Name: "select-layout"},
		{Name: "select-pane"},
		{Name: "select-window"},
		{Name: "send-keys"},
		{Name: "split-window"},
		{Name: "switch-client"},
		{Name: "show-options"},
		{Name: "show-hooks"},
		{Name: "show-messages"},
		{Name: "show-buffer"},
		{Name: "show-environment"},
		{Name: "show-window-options"},
	})
	model.height = 13

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	got := updated.(Model)
	if got.cursor != 5 {
		t.Fatalf("cursor after Ctrl-D = %d, want 5", got.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	got = updated.(Model)
	if got.cursor != 0 {
		t.Fatalf("cursor after Ctrl-U = %d, want 0", got.cursor)
	}
}

func TestPageUpDownHalfPageScrollCandidates(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{Name: "select-layout"},
		{Name: "select-pane"},
		{Name: "select-window"},
		{Name: "send-keys"},
		{Name: "split-window"},
		{Name: "switch-client"},
		{Name: "show-options"},
		{Name: "show-hooks"},
		{Name: "show-messages"},
		{Name: "show-buffer"},
		{Name: "show-environment"},
		{Name: "show-window-options"},
	})
	model.height = 13

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	got := updated.(Model)
	if got.cursor != 5 {
		t.Fatalf("cursor after PageDown = %d, want 5", got.cursor)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	got = updated.(Model)
	if got.cursor != 0 {
		t.Fatalf("cursor after PageUp = %d, want 0", got.cursor)
	}
}

func TestHistoryPathUsesXDGStateHome(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	previous, had := os.LookupEnv("XDG_STATE_HOME")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("XDG_STATE_HOME", previous)
		} else {
			_ = os.Unsetenv("XDG_STATE_HOME")
		}
	})

	_ = os.Setenv("XDG_STATE_HOME", tmp)

	got, err := historyPath()
	if err != nil {
		t.Fatalf("historyPath() error = %v", err)
	}

	want := filepath.Join(tmp, "tmcp", "history")
	if got != want {
		t.Fatalf("historyPath() = %q, want %q", got, want)
	}
}

func TestSaveAndLoadHistory(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	previous, had := os.LookupEnv("XDG_STATE_HOME")
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("XDG_STATE_HOME", previous)
		} else {
			_ = os.Unsetenv("XDG_STATE_HOME")
		}
	})
	_ = os.Setenv("XDG_STATE_HOME", tmp)

	history := []string{"select-pane -L", "send-keys Enter"}
	if err := SaveHistory(history); err != nil {
		t.Fatalf("SaveHistory() error = %v", err)
	}

	got, err := LoadHistory()
	if err != nil {
		t.Fatalf("LoadHistory() error = %v", err)
	}

	if len(got) != len(history) || got[0] != history[0] || got[1] != history[1] {
		t.Fatalf("LoadHistory() = %#v, want %#v", got, history)
	}
}

func TestAppendHistorySkipsEmptyAndConsecutiveDuplicates(t *testing.T) {
	t.Parallel()

	history := AppendHistory(nil, "")
	history = AppendHistory(history, "select-pane -L")
	history = AppendHistory(history, "select-pane -L")

	if len(history) != 1 || history[0] != "select-pane -L" {
		t.Fatalf("AppendHistory() = %#v, want one deduplicated entry", history)
	}
}

func TestRenderInputStylesPlaceholder(t *testing.T) {
	t.Parallel()

	model := NewModel(nil)
	got := model.renderInput()

	if !strings.HasPrefix(got, "> ") {
		t.Fatalf("renderInput() = %q, want prefix %q", got, "> ")
	}
	if !strings.Contains(got, "Type a command") {
		t.Fatalf("renderInput() = %q, want placeholder text", got)
	}
	if got == "> Type a command" {
		t.Fatalf("renderInput() = %q, want styled placeholder", got)
	}
}

func TestRenderInputShowsPendingFlagValuePlaceholder(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{
			Name: "send-keys",
			Flags: []tmux.Flag{
				{Name: "-t", Value: "target-pane"},
			},
		},
	})
	model.input.SetValue("send-keys -t ")

	got := model.renderInput()
	if !strings.Contains(got, "send-keys -t ") {
		t.Fatalf("renderInput() = %q, want command and flag prefix", got)
	}
	if !strings.Contains(got, "target-pane") {
		t.Fatalf("renderInput() = %q, want pending value placeholder", got)
	}
	if got == "> send-keys -t target-pane" {
		t.Fatalf("renderInput() = %q, want styled pending value placeholder", got)
	}
}

func TestRenderInputDoesNotShowPendingFlagValuePlaceholderForUnknownFlag(t *testing.T) {
	t.Parallel()

	model := NewModel([]tmux.Command{
		{
			Name: "send-keys",
			Flags: []tmux.Flag{
				{Name: "-t", Value: "target-pane"},
			},
		},
	})
	model.input.SetValue("send-keys -x ")

	got := model.renderInput()
	if strings.Contains(got, "target-pane") {
		t.Fatalf("renderInput() = %q, did not expect pending value placeholder", got)
	}
}

func TestCandidateDisplayPartsMovesFlagValuePlaceholderToNote(t *testing.T) {
	t.Parallel()

	label, note := candidateDisplayParts(complete.Candidate{
		Display: "-t target-pane",
		Kind:    complete.CandidateFlag,
	})

	if label != "-t" {
		t.Fatalf("candidateDisplayParts() label = %q, want flag name", label)
	}
	if note != "target-pane" {
		t.Fatalf("candidateDisplayParts() note = %q, want placeholder text", note)
	}
}

func TestCandidateDisplayPartsLeavesBareFlagUnchanged(t *testing.T) {
	t.Parallel()

	label, note := candidateDisplayParts(complete.Candidate{
		Display: "-F",
		Kind:    complete.CandidateFlag,
	})
	if label != "-F" || note != "" {
		t.Fatalf("candidateDisplayParts() = (%q, %q), want (%q, %q)", label, note, "-F", "")
	}
}

func TestCandidateDisplayPartsAddsBareFlagNote(t *testing.T) {
	t.Parallel()

	label, note := candidateDisplayParts(complete.Candidate{
		Display: "-h",
		Note:    "split horizontally",
		Kind:    complete.CandidateFlag,
	})
	if label != "-h" || note != "split horizontally" {
		t.Fatalf("candidateDisplayParts() = (%q, %q), want (%q, %q)", label, note, "-h", "split horizontally")
	}
}

func TestCandidateDisplayPartsUsesFlagValuePlaceholderAsNote(t *testing.T) {
	t.Parallel()

	label, note := candidateDisplayParts(complete.Candidate{
		Display: "-F format",
		Kind:    complete.CandidateFlag,
	})

	if label != "-F" || note != "format" {
		t.Fatalf("candidateDisplayParts() = (%q, %q), want flag and placeholder note", label, note)
	}
}

func TestPlaceholderNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		placeholder string
		want        string
		ok          bool
	}{
		{placeholder: "working-directory", want: "working directory", ok: true},
		{placeholder: "window-name", want: "window name", ok: true},
		{placeholder: "key-table", want: "key table", ok: true},
		{placeholder: "flags", want: "comma-separated flags", ok: true},
		{placeholder: "position", want: "popup or menu position", ok: true},
		{placeholder: "match-string", want: "search pattern", ok: true},
		{placeholder: "environment", want: "VARIABLE=value", ok: true},
		{placeholder: "format", want: "tmux format", ok: true},
		{placeholder: "filter", want: "format expression", ok: true},
		{placeholder: "path", want: "filesystem path", ok: true},
		{placeholder: "shell-command", want: "shell command", ok: true},
		{placeholder: "layout-name", want: "layout preset", ok: true},
		{placeholder: "target-pane", want: "", ok: false},
		{placeholder: "repeat-count", want: "", ok: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.placeholder, func(t *testing.T) {
			t.Parallel()

			got, ok := placeholderNote(tc.placeholder)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("placeholderNote(%q) = (%q, %v), want (%q, %v)", tc.placeholder, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestCandidateDisplayPartsAddsCommandNote(t *testing.T) {
	t.Parallel()

	label, note := candidateDisplayParts(complete.Candidate{
		Value:   "send-keys",
		Display: "send-keys (send)",
		Kind:    complete.CandidateCommand,
	})

	if label != "send-keys (send)" || note != "send keys to a pane or client" {
		t.Fatalf("candidateDisplayParts() = (%q, %q), want command note", label, note)
	}
}

func TestCandidateDisplayPartsPrefersCustomCommandNote(t *testing.T) {
	t.Parallel()

	label, note := candidateDisplayParts(complete.Candidate{
		Value:   "swap-left",
		Display: "swap-left (sl)",
		Note:    "swap current pane with the left pane",
		Kind:    complete.CandidateCommand,
	})

	if label != "swap-left (sl)" || note != "swap current pane with the left pane" {
		t.Fatalf("candidateDisplayParts() = (%q, %q), want custom command note", label, note)
	}
}

func TestCustomCommandCandidatesAppearAtTopLevel(t *testing.T) {
	t.Parallel()

	model := NewModelWithHistoryAndCommands(
		[]tmux.Command{{Name: "send-keys"}},
		[]config.Command{{Name: "swap-left", Aliases: []string{"sl"}, Note: "swap current pane", Run: []string{"swap-pane", "-t", "{left}"}}},
		nil,
	)
	model.input.SetValue("sw")
	model.refreshMatches()

	if len(model.candidates) != 1 {
		t.Fatalf("len(candidates) = %d, want 1", len(model.candidates))
	}
	if model.candidates[0].Value != "swap-left" {
		t.Fatalf("candidate value = %q, want %q", model.candidates[0].Value, "swap-left")
	}
	if model.candidates[0].Note != "swap current pane" {
		t.Fatalf("candidate note = %q, want %q", model.candidates[0].Note, "swap current pane")
	}
}

func TestCandidateLabelWidthUsesVisibleLabels(t *testing.T) {
	t.Parallel()

	width := candidateLabelWidth([]complete.Candidate{
		{Display: "-d", Note: "keep current active pane", Kind: complete.CandidateFlag},
		{Display: "-D", Note: "swap with next pane", Kind: complete.CandidateFlag},
		{Display: "-t target-pane", Kind: complete.CandidateFlag},
	})
	if width != ansi.StringWidth("-d") {
		t.Fatalf("candidateLabelWidth() = %d, want flag name width", width)
	}
}

func TestCommandNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		command string
		want    string
		ok      bool
	}{
		{command: "bind-key", want: "bind a key to a tmux command", ok: true},
		{command: "command-prompt", want: "open the tmux command prompt", ok: true},
		{command: "choose-client", want: "choose a client from a list", ok: true},
		{command: "choose-tree", want: "choose a session, window, or pane from a tree", ok: true},
		{command: "display-menu", want: "show an interactive tmux menu", ok: true},
		{command: "move-pane", want: "move a pane into another split", ok: true},
		{command: "respawn-window", want: "restart a dead window command", ok: true},
		{command: "send-prefix", want: "send the tmux prefix key to a pane", ok: true},
		{command: "unbind-key", want: "remove a key binding", ok: true},
		{command: "choose-buffer", want: "choose a paste buffer from a list", ok: true},
		{command: "set-environment", want: "set or unset an environment variable", ok: true},
		{command: "set-option", want: "set a tmux option", ok: true},
		{command: "show-hooks", want: "show hooks", ok: true},
		{command: "run-shell", want: "run a shell or tmux command in the background", ok: true},
		{command: "attach-session", want: "attach or switch to a session", ok: true},
		{command: "detach-client", want: "detach one or more clients", ok: true},
		{command: "new-session", want: "create a new session", ok: true},
		{command: "switch-client", want: "switch a client to another session", ok: true},
		{command: "source-file", want: "load tmux commands from a file", ok: true},
		{command: "send-keys", want: "send keys to a pane or client", ok: true},
		{command: "split-window", want: "split a pane and create a new one", ok: true},
		{command: "display-popup", want: "show a popup running a shell command", ok: true},
		{command: "find-window", want: "search window names, titles, or contents", ok: true},
		{command: "wait-for", want: "wait on, signal, or lock a channel", ok: true},
		{command: "foobar", want: "", ok: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.command, func(t *testing.T) {
			t.Parallel()

			got, ok := commandNote(tc.command)
			if ok != tc.ok || got != tc.want {
				t.Fatalf("commandNote(%q) = (%q, %v), want (%q, %v)", tc.command, got, ok, tc.want, tc.ok)
			}
		})
	}
}
