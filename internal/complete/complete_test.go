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

	if len(got) != 1 {
		t.Fatalf("len(Complete()) = %d, want 1", len(got))
	}

	if got[0].Value != "main:editor.0" {
		t.Fatalf("got[0].Value = %q, want %q", got[0].Value, "main:editor.0")
	}
}
