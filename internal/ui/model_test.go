package ui

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/gh-liu/tmcp/internal/complete"
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

func TestInitialTerminalSize(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestRenderInputStylesPlaceholder(t *testing.T) {
	t.Parallel()

	model := NewModel(nil)
	got := model.renderInput()

	if !strings.HasPrefix(got, "> ") {
		t.Fatalf("renderInput() = %q, want prefix %q", got, "> ")
	}
	if !strings.Contains(got, "Type a tmux command") {
		t.Fatalf("renderInput() = %q, want placeholder text", got)
	}
	if got == "> Type a tmux command" {
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

func TestRenderCandidateDisplayStylesFlagValuePlaceholder(t *testing.T) {
	t.Parallel()

	got := renderCandidateDisplay(complete.Candidate{
		Display: "-t target-pane",
		Kind:    complete.CandidateFlag,
	})

	if !strings.Contains(got, "-t ") {
		t.Fatalf("renderCandidateDisplay() = %q, want flag prefix", got)
	}
	if !strings.Contains(got, "target-pane") {
		t.Fatalf("renderCandidateDisplay() = %q, want placeholder text", got)
	}
	if got == "-t target-pane" {
		t.Fatalf("renderCandidateDisplay() = %q, want styled placeholder", got)
	}
}

func TestRenderCandidateDisplayLeavesBareFlagUnchanged(t *testing.T) {
	t.Parallel()

	got := renderCandidateDisplay(complete.Candidate{
		Display: "-F",
		Kind:    complete.CandidateFlag,
	})
	if got != "-F" {
		t.Fatalf("renderCandidateDisplay() = %q, want %q", got, "-F")
	}
}

func TestRenderCandidateDisplayAddsPlaceholderNote(t *testing.T) {
	t.Parallel()

	got := renderCandidateDisplay(complete.Candidate{
		Display: "-t target-pane",
		Kind:    complete.CandidateFlag,
	})

	if !strings.Contains(got, "pane target") {
		t.Fatalf("renderCandidateDisplay() = %q, want placeholder note", got)
	}
}

func TestPlaceholderNote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		placeholder string
		want        string
		ok          bool
	}{
		{placeholder: "target-pane", want: "pane target", ok: true},
		{placeholder: "target-window", want: "window target", ok: true},
		{placeholder: "target-session", want: "session target", ok: true},
		{placeholder: "target-client", want: "client target", ok: true},
		{placeholder: "format", want: "tmux format", ok: true},
		{placeholder: "filter", want: "format expression", ok: true},
		{placeholder: "path", want: "filesystem path", ok: true},
		{placeholder: "shell-command", want: "shell command", ok: true},
		{placeholder: "layout-name", want: "layout preset", ok: true},
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
