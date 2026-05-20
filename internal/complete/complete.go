package complete

import (
	"context"
	"fmt"
	"strings"

	"github.com/gh-liu/tmcp/internal/tmux"
	"github.com/sahilm/fuzzy"
)

type Completer struct {
	providers map[string]Provider
}

func NewCompleter() Completer {
	return Completer{providers: DefaultProviders()}
}

func NewCompleterWithProviders(providers map[string]Provider) Completer {
	return Completer{providers: providers}
}

func (c Completer) Complete(ctx context.Context, commands []tmux.Command, line string) ([]Candidate, error) {
	tokens := strings.Fields(line)
	endsWithSpace := strings.HasSuffix(line, " ")

	switch {
	case len(tokens) == 0:
		return commandCandidates(commands, ""), nil
	case len(tokens) == 1 && !endsWithSpace:
		return commandCandidates(commands, tokens[0]), nil
	}

	command, ok := findCommand(commands, tokens[0])
	if !ok {
		return nil, nil
	}

	switch {
	case endsWithSpace && len(tokens) == 1:
		return flagCandidates(command, "", usedFlags(command, tokens[1:], "")), nil
	case endsWithSpace:
		if flag, ok := findFlag(command, tokens[len(tokens)-1]); ok && flag.Value != "" {
			return c.completeValue(ctx, flag.Value, "")
		}
		return flagCandidates(command, "", usedFlags(command, tokens[1:], "")), nil
	default:
		current := tokens[len(tokens)-1]
		if strings.HasPrefix(current, "-") {
			return flagCandidates(command, current, usedFlags(command, tokens[1:len(tokens)-1], current)), nil
		}

		if len(tokens) >= 2 {
			if flag, ok := findFlag(command, tokens[len(tokens)-2]); ok && flag.Value != "" {
				return c.completeValue(ctx, flag.Value, current)
			}
		}
	}

	return nil, nil
}

func (c Completer) completeValue(ctx context.Context, placeholder, prefix string) ([]Candidate, error) {
	key := providerKey(placeholder)
	if key == "" {
		return nil, nil
	}

	provider, ok := c.providers[key]
	if !ok {
		return specialTokenCandidates(key, prefix), nil
	}

	candidates, err := provider.Candidates(ctx, prefix)
	if err != nil {
		return nil, err
	}

	return mergeCandidates(candidates, specialTokenCandidates(key, prefix)), nil
}

func (c Completer) resolveProvider(placeholder string) (Provider, bool) {
	key := providerKey(placeholder)
	provider, ok := c.providers[key]
	return provider, ok
}

func providerKey(placeholder string) string {
	switch {
	case strings.Contains(placeholder, "session"):
		return "session"
	case strings.Contains(placeholder, "window"):
		return "window"
	case strings.Contains(placeholder, "pane"):
		return "pane"
	case strings.Contains(placeholder, "client"):
		return "client"
	case strings.Contains(placeholder, "buffer-name"), strings.Contains(placeholder, "buffer-index"):
		return "buffer"
	default:
		return ""
	}
}

func specialTokenCandidates(key, prefix string) []Candidate {
	tokens := specialTokens[key]
	candidates := make([]Candidate, 0, len(tokens))

	for _, token := range tokens {
		if prefix != "" && !strings.HasPrefix(token, prefix) {
			continue
		}

		candidates = append(candidates, Candidate{
			Value:   token,
			Display: token,
			Kind:    CandidateValue,
		})
	}

	return candidates
}

func mergeCandidates(primary, secondary []Candidate) []Candidate {
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	merged := make([]Candidate, 0, len(primary)+len(secondary))

	for _, candidate := range primary {
		merged = append(merged, candidate)
		seen[candidate.Value] = struct{}{}
	}

	for _, candidate := range secondary {
		if _, ok := seen[candidate.Value]; ok {
			continue
		}
		merged = append(merged, candidate)
		seen[candidate.Value] = struct{}{}
	}

	return merged
}

func commandCandidates(commands []tmux.Command, query string) []Candidate {
	if strings.TrimSpace(query) == "" {
		result := make([]Candidate, 0, len(commands))
		for _, command := range commands {
			result = append(result, Candidate{
				Value:   command.Name,
				Display: formatCommand(command),
				Kind:    CandidateCommand,
			})
		}
		return result
	}

	values := make(commandValues, len(commands))
	copy(values, commands)

	found := fuzzy.FindFrom(query, values)
	result := make([]Candidate, 0, len(found))
	for _, match := range found {
		command := commands[match.Index]
		result = append(result, Candidate{
			Value:   command.Name,
			Display: formatCommand(command),
			Kind:    CandidateCommand,
		})
	}

	return result
}

func flagCandidates(command tmux.Command, prefix string, used map[string]struct{}) []Candidate {
	result := make([]Candidate, 0, len(command.Flags))
	for _, flag := range command.Flags {
		if _, ok := used[flag.Name]; ok {
			continue
		}

		if prefix != "" && !strings.HasPrefix(flag.Name, prefix) {
			continue
		}

		display := flag.Name
		if flag.Value != "" {
			display += " " + flag.Value
		}

		result = append(result, Candidate{
			Value:   flag.Name,
			Display: display,
			Note:    flagNote(command.Name, flag.Name),
			Kind:    CandidateFlag,
		})
	}

	return result
}

func flagNote(commandName, flagName string) string {
	switch commandName {
	case "split-window":
		switch flagName {
		case "-b":
			return "create before or above"
		case "-d":
			return "do not select new pane"
		case "-f":
			return "span the full window"
		case "-h":
			return "split horizontally"
		case "-I":
			return "forward stdin to empty pane"
		case "-P":
			return "print created pane info"
		case "-v":
			return "split vertically"
		case "-Z":
			return "keep or enable zoom"
		}
	case "join-pane", "move-pane":
		switch flagName {
		case "-b":
			return "place before or above"
		case "-d":
			return "do not select target pane"
		case "-f":
			return "span the full window"
		case "-h":
			return "split horizontally"
		case "-v":
			return "split vertically"
		}
	case "break-pane":
		switch flagName {
		case "-a":
			return "place after destination index"
		case "-b":
			return "place before destination index"
		case "-d":
			return "do not select new window"
		case "-P":
			return "print created window info"
		}
	case "swap-pane":
		switch flagName {
		case "-d":
			return "keep current active pane"
		case "-D":
			return "swap with next pane"
		case "-U":
			return "swap with previous pane"
		case "-Z":
			return "keep zoom"
		}
	case "resize-pane":
		switch flagName {
		case "-D":
			return "resize downward"
		case "-L":
			return "resize left"
		case "-M":
			return "resize with mouse"
		case "-R":
			return "resize right"
		case "-T":
			return "trim below cursor"
		case "-U":
			return "resize upward"
		case "-Z":
			return "toggle zoom"
		}
	case "resize-window":
		switch flagName {
		case "-A":
			return "use largest session size"
		case "-D":
			return "resize downward"
		case "-L":
			return "resize left"
		case "-R":
			return "resize right"
		case "-U":
			return "resize upward"
		case "-a":
			return "use smallest session size"
		}
	case "rotate-window":
		switch flagName {
		case "-D":
			return "rotate downward"
		case "-U":
			return "rotate upward"
		case "-Z":
			return "keep zoom"
		}
	case "select-layout":
		switch flagName {
		case "-E":
			return "spread panes evenly"
		case "-n":
			return "next layout"
		case "-o":
			return "restore previous layout"
		case "-p":
			return "previous layout"
		}
	case "select-pane":
		switch flagName {
		case "-D":
			return "select pane below"
		case "-L":
			return "select pane on the left"
		case "-R":
			return "select pane on the right"
		case "-U":
			return "select pane above"
		case "-d":
			return "disable input"
		case "-e":
			return "enable input"
		case "-l":
			return "select previous pane"
		case "-m":
			return "set marked pane"
		case "-M":
			return "clear marked pane"
		case "-Z":
			return "keep zoom"
		}
	case "select-window":
		switch flagName {
		case "-l":
			return "select previous window"
		case "-n":
			return "select next window"
		case "-p":
			return "select previous window by index"
		case "-T":
			return "toggle to last window if current"
		}
	}

	return ""
}

func usedFlags(command tmux.Command, tokens []string, keep string) map[string]struct{} {
	used := make(map[string]struct{})
	for _, token := range tokens {
		flag, ok := findFlag(command, token)
		if !ok {
			continue
		}
		if flag.Name == keep {
			continue
		}
		used[flag.Name] = struct{}{}
	}
	return used
}

func findCommand(commands []tmux.Command, token string) (tmux.Command, bool) {
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

func findFlag(command tmux.Command, token string) (tmux.Flag, bool) {
	for _, flag := range command.Flags {
		if flag.Name == token {
			return flag, true
		}
	}

	return tmux.Flag{}, false
}

func formatCommand(command tmux.Command) string {
	if len(command.Aliases) == 0 {
		return command.Name
	}

	return fmt.Sprintf("%s (%s)", command.Name, strings.Join(command.Aliases, ", "))
}

type commandValues []tmux.Command

func (c commandValues) Len() int {
	return len(c)
}

func (c commandValues) String(i int) string {
	command := c[i]
	if len(command.Aliases) == 0 {
		return command.Name
	}

	return command.Name + " " + strings.Join(command.Aliases, " ")
}
