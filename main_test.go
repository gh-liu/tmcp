package main

import (
	"context"
	"testing"

	"github.com/gh-liu/tmcp/internal/config"
)

func TestFindCustomCommandMatchesNameAndAlias(t *testing.T) {
	t.Parallel()

	commands := []config.Command{
		{Name: "swap-left", Aliases: []string{"sl"}, Run: []string{"swap-pane", "-t", "{left}"}},
	}

	if got, ok := findCustomCommand(commands, "swap-left"); !ok || got.Name != "swap-left" {
		t.Fatalf("findCustomCommand(name) = (%#v, %v), want swap-left", got, ok)
	}

	if got, ok := findCustomCommand(commands, "sl"); !ok || got.Name != "swap-left" {
		t.Fatalf("findCustomCommand(alias) = (%#v, %v), want swap-left", got, ok)
	}
}

func TestExecuteLineReturnsNilForEmptyInput(t *testing.T) {
	t.Parallel()

	if err := executeLine(context.Background(), nil, nil, "   "); err != nil {
		t.Fatalf("executeLine() error = %v, want nil", err)
	}
}

func TestExecuteLineExpandsCustomCommandRunAndExtraArgs(t *testing.T) {
	previous := executeTokens
	t.Cleanup(func() {
		executeTokens = previous
	})

	var got []string
	executeTokens = func(_ context.Context, tokens []string) error {
		got = append([]string(nil), tokens...)
		return nil
	}

	err := executeLine(context.Background(), nil, []config.Command{
		{Name: "send-left", Run: []string{"send-keys", "-t", "{left}"}},
	}, "send-left C-c")
	if err != nil {
		t.Fatalf("executeLine() error = %v", err)
	}

	want := []string{"send-keys", "-t", "{left}", "C-c"}
	if len(got) != len(want) {
		t.Fatalf("executeLine() tokens = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("executeLine() tokens = %#v, want %#v", got, want)
		}
	}
}
