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
		"hook":    providerFunc(tmuxFieldProvider([]string{"show-hooks", "-g"}, hookName)),
		"option":  providerFunc(tmuxOptionProvider()),
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

func tmuxOptionProvider() providerFunc {
	return func(ctx context.Context, prefix string) ([]Candidate, error) {
		var candidates []Candidate
		seen := make(map[string]struct{})
		for _, args := range [][]string{
			{"show-options", "-g"},
			{"show-options", "-gw"},
			{"show-options", "-gs"},
		} {
			items, err := tmuxFieldProvider(args, firstField)(ctx, prefix)
			if err != nil {
				return nil, err
			}

			for _, item := range items {
				if _, ok := seen[item.Value]; ok {
					continue
				}
				seen[item.Value] = struct{}{}
				candidates = append(candidates, item)
			}
		}

		return candidates, nil
	}
}

func tmuxFieldProvider(args []string, field func(string) string) providerFunc {
	return func(ctx context.Context, prefix string) ([]Candidate, error) {
		cmd := exec.CommandContext(ctx, "tmux", args...)
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("tmux %s: %w", strings.Join(args, " "), err)
		}

		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		candidates := make([]Candidate, 0)
		for scanner.Scan() {
			value := field(scanner.Text())
			if value == "" {
				continue
			}
			if prefix != "" && !strings.HasPrefix(value, prefix) {
				continue
			}

			candidates = append(candidates, Candidate{
				Value:   value,
				Display: value,
				Kind:    CandidateValue,
			})
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}

		return candidates, nil
	}
}

func firstField(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func hookName(line string) string {
	name := firstField(line)
	if before, _, ok := strings.Cut(name, "["); ok {
		return before
	}
	return name
}
