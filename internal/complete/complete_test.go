package complete

import (
	"context"
	"testing"

	"github.com/gh-liu/tmcp/internal/tmux"
)

func TestCompleteListsFlagsAfterCommand(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name: "send-keys",
			Flags: []tmux.Flag{
				{Name: "-F"},
				{Name: "-t", Value: "target-pane"},
			},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "send-keys -")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(Complete()) = %d, want 2", len(got))
	}

	if got[1].Display != "-t target-pane" {
		t.Fatalf("got[1].Display = %q, want %q", got[1].Display, "-t target-pane")
	}
}

func TestCompleteDelegatesToProviderForFlagValue(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(map[string]Provider{
		"pane": providerFunc(func(context.Context, string) ([]Candidate, error) {
			return []Candidate{
				{Value: "main:editor.0", Display: "main:editor.0", Kind: CandidateValue},
			}, nil
		}),
	})

	commands := []tmux.Command{
		{
			Name: "send-keys",
			Flags: []tmux.Flag{
				{Name: "-t", Value: "target-pane"},
			},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "send-keys -t ")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(got) < 1 {
		t.Fatalf("len(Complete()) = %d, want at least 1", len(got))
	}

	if got[0].Value != "main:editor.0" {
		t.Fatalf("got[0].Value = %q, want %q", got[0].Value, "main:editor.0")
	}
}

func TestCompleteMergesSpecialTokens(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(map[string]Provider{
		"pane": providerFunc(func(context.Context, string) ([]Candidate, error) {
			return []Candidate{
				{Value: "main:editor.0", Display: "main:editor.0", Kind: CandidateValue},
			}, nil
		}),
	})

	commands := []tmux.Command{
		{
			Name: "kill-pane",
			Flags: []tmux.Flag{
				{Name: "-t", Value: "target-pane"},
			},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "kill-pane -t ")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(got) < 2 {
		t.Fatalf("len(Complete()) = %d, want at least 2", len(got))
	}

	if got[0].Value != "main:editor.0" {
		t.Fatalf("got[0].Value = %q, want %q", got[0].Value, "main:editor.0")
	}

	found := false
	for _, candidate := range got {
		if candidate.Value == "{last}" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected {last} in candidates, got %#v", got)
	}
}

func TestCompleteExcludesAlreadySelectedFlags(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name: "send-keys",
			Flags: []tmux.Flag{
				{Name: "-F"},
				{Name: "-N", Value: "repeat-count"},
				{Name: "-t", Value: "target-pane"},
			},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "send-keys -F ")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	for _, candidate := range got {
		if candidate.Value == "-F" {
			t.Fatalf("unexpected reused flag in candidates: %#v", got)
		}
	}
}

func TestCompleteEditingNewFlagPrefixStillExcludesUsedFlags(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name: "send-keys",
			Flags: []tmux.Flag{
				{Name: "-F"},
				{Name: "-N", Value: "repeat-count"},
				{Name: "-t", Value: "target-pane"},
			},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "send-keys -F -")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	for _, candidate := range got {
		if candidate.Value == "-F" {
			t.Fatalf("unexpected reused flag in candidates: %#v", got)
		}
		if candidate.Value == "-N" && candidate.Display != "-N repeat-count" {
			t.Fatalf("unexpected display for -N: %q", candidate.Display)
		}
	}
}

func TestCompleteListsKnownPositionalValuesAfterCommand(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name: "select-layout",
			Flags: []tmux.Flag{
				{Name: "-E"},
				{Name: "-t", Value: "target-pane"},
			},
			Positional: []string{"layout-name"},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "select-layout ")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	want := map[string]bool{
		"-E":              false,
		"even-horizontal": false,
		"tiled":           false,
	}
	for _, candidate := range got {
		if _, ok := want[candidate.Value]; ok {
			want[candidate.Value] = true
		}
	}

	for value, found := range want {
		if !found {
			t.Fatalf("expected %q in candidates, got %#v", value, got)
		}
	}
}

func TestCompleteFiltersKnownPositionalValues(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name:       "select-layout",
			Positional: []string{"layout-name"},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "select-layout main-v")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(Complete()) = %d, want 2: %#v", len(got), got)
	}

	for _, candidate := range got {
		if candidate.Value != "main-vertical" && candidate.Value != "main-vertical-mirrored" {
			t.Fatalf("unexpected candidate %#v", candidate)
		}
	}
}

func TestCompleteListsKnownPositionalValuesAfterFlagValue(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name: "select-layout",
			Flags: []tmux.Flag{
				{Name: "-t", Value: "target-pane"},
			},
			Positional: []string{"layout-name"},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "select-layout -t main:editor.0 ")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	found := false
	for _, candidate := range got {
		if candidate.Value == "tiled" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected tiled in candidates, got %#v", got)
	}
}

func TestCompleteDelegatesToProviderForKnownPositionalValue(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(map[string]Provider{
		"option": providerFunc(func(context.Context, string) ([]Candidate, error) {
			return []Candidate{
				{Value: "status-style", Display: "status-style", Kind: CandidateValue},
			}, nil
		}),
	})
	commands := []tmux.Command{
		{
			Name:       "set-option",
			Positional: []string{"option", "value"},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "set-option status-")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(got) != 1 || got[0].Value != "status-style" {
		t.Fatalf("Complete() = %#v, want status-style", got)
	}
}

func TestCompleteListsSpecialTokensForKnownFlagValue(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name: "bind-key",
			Flags: []tmux.Flag{
				{Name: "-T", Value: "key-table"},
			},
			Positional: []string{"key", "command [argument ...]"},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "bind-key -T ")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	found := false
	for _, candidate := range got {
		if candidate.Value == "copy-mode-vi" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected copy-mode-vi in candidates, got %#v", got)
	}
}

func TestCompleteListsKnownCommandsForCommandPositional(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name:       "list-commands",
			Positional: []string{"command"},
		},
		{
			Name: "display-message",
		},
		{
			Name: "display-panes",
		},
	}

	got, err := completer.Complete(context.Background(), commands, "list-commands displ")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(Complete()) = %d, want 2: %#v", len(got), got)
	}

	for _, candidate := range got {
		if candidate.Value != "display-message" && candidate.Value != "display-panes" {
			t.Fatalf("unexpected candidate %#v", candidate)
		}
	}
}

func TestCompleteAddsLayoutFlagNotes(t *testing.T) {
	t.Parallel()

	completer := NewCompleterWithProviders(nil)
	commands := []tmux.Command{
		{
			Name: "split-window",
			Flags: []tmux.Flag{
				{Name: "-b"},
				{Name: "-h"},
				{Name: "-v"},
				{Name: "-Z"},
			},
		},
	}

	got, err := completer.Complete(context.Background(), commands, "split-window -")
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	want := map[string]string{
		"-b": "create before or above",
		"-h": "split horizontally",
		"-v": "split vertically",
		"-Z": "keep or enable zoom",
	}
	for _, candidate := range got {
		if note, ok := want[candidate.Value]; ok && candidate.Note != note {
			t.Fatalf("candidate %q note = %q, want %q", candidate.Value, candidate.Note, note)
		}
	}
}
