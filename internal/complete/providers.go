package complete

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func DefaultProviders() map[string]Provider {
	return map[string]Provider{
		"session": providerFunc(tmuxListProvider("list-sessions", "#S")),
		"window":  providerFunc(tmuxListProvider("list-windows", "#S:#W", "-a")),
		"pane":    providerFunc(tmuxListProvider("list-panes", "#S:#W.#P", "-a")),
		"client":  providerFunc(tmuxListProvider("list-clients", "#{client_tty}")),
		"buffer":  providerFunc(tmuxListProvider("list-buffers", "#{buffer_name}")),
	}
}

type providerFunc func(context.Context, string) ([]Candidate, error)

func (p providerFunc) Candidates(ctx context.Context, prefix string) ([]Candidate, error) {
	return p(ctx, prefix)
}

func tmuxListProvider(command, format string, extraArgs ...string) providerFunc {
	return func(ctx context.Context, prefix string) ([]Candidate, error) {
		args := []string{command, "-F", format}
		args = append(args, extraArgs...)

		cmd := exec.CommandContext(ctx, "tmux", args...)
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("tmux %s: %w", command, err)
		}

		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		candidates := make([]Candidate, 0)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			if prefix != "" && !strings.HasPrefix(line, prefix) {
				continue
			}

			candidates = append(candidates, Candidate{
				Value:   line,
				Display: line,
				Kind:    CandidateValue,
			})
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		return candidates, nil
	}
}
