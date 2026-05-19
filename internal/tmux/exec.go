package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func ExecuteLine(ctx context.Context, line string) error {
	tokens := strings.Fields(line)
	if len(tokens) == 0 {
		return nil
	}

	return ExecuteTokens(ctx, tokens)
}

func ExecuteTokens(ctx context.Context, tokens []string) error {
	cmd := exec.CommandContext(ctx, "tmux", tokens...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	message := strings.TrimSpace(string(out))
	if message == "" {
		message = err.Error()
	}

	_ = displayMessage(ctx, message)

	return fmt.Errorf("tmux %s: %s", strings.Join(tokens, " "), message)
}

func displayMessage(ctx context.Context, message string) error {
	cmd := exec.CommandContext(ctx, "tmux", "display-message", message)
	return cmd.Run()
}
