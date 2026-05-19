package ui

import (
	"testing"

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

func TestCandidateWidthUsesLongestCandidate(t *testing.T) {
	t.Parallel()

	candidates := []complete.Candidate{
		{Display: "short"},
		{Display: "much longer candidate"},
		{Display: "mid"},
	}

	got := candidateWidth(candidates, 0, 0)
	want := len("> much longer candidate")
	if got < want {
		t.Fatalf("candidateWidth() = %d, want at least %d", got, want)
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
