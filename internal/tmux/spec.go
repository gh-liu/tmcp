package tmux

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Command struct {
	Name       string   `json:"name"`
	Aliases    []string `json:"aliases,omitempty"`
	Flags      []Flag   `json:"flags,omitempty"`
	Positional []string `json:"positional,omitempty"`
}

type Flag struct {
	Raw string `json:"raw"`
}

func ListCommands(ctx context.Context) ([]Command, error) {
	cmd := exec.CommandContext(ctx, "tmux", "list-commands")

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-commands: %w", err)
	}

	return ParseCommands(out)
}

func ParseCommands(out []byte) ([]Command, error) {
	lines := bytes.Split(out, []byte{'\n'})
	commands := make([]Command, 0, len(lines))

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		command, err := ParseCommandLine(string(line))
		if err != nil {
			return nil, err
		}

		commands = append(commands, command)
	}

	return commands, nil
}

func ParseCommandLine(line string) (Command, error) {
	name, rest, ok := strings.Cut(strings.TrimSpace(line), " ")
	if !ok {
		return Command{Name: name}, nil
	}

	command := Command{Name: name}
	rest = strings.TrimSpace(rest)

	if strings.HasPrefix(rest, "(") {
		end := strings.IndexByte(rest, ')')
		if end == -1 {
			return Command{}, fmt.Errorf("parse command alias: %q", line)
		}

		alias := strings.TrimSpace(rest[1:end])
		if alias != "" {
			command.Aliases = []string{alias}
		}

		rest = strings.TrimSpace(rest[end+1:])
	}

	for _, part := range splitSignature(rest) {
		if len(part) == 0 {
			continue
		}

		if part[0] == '[' && part[len(part)-1] == ']' {
			raw := part[1 : len(part)-1]
			if strings.HasPrefix(raw, "-") {
				command.Flags = append(command.Flags, Flag{Raw: raw})
			} else {
				command.Positional = append(command.Positional, raw)
			}
			continue
		}

		command.Positional = append(command.Positional, part)
	}

	return command, nil
}

func splitSignature(signature string) []string {
	parts := make([]string, 0)

	for i := 0; i < len(signature); {
		switch signature[i] {
		case ' ', '\t':
			i++
		case '[':
			start := i
			depth := 1
			i++

			for i < len(signature) && depth > 0 {
				switch signature[i] {
				case '[':
					depth++
				case ']':
					depth--
				}
				i++
			}

			parts = append(parts, signature[start:i])
		default:
			start := i
			for i < len(signature) && signature[i] != ' ' && signature[i] != '\t' {
				i++
			}
			parts = append(parts, signature[start:i])
		}
	}

	return parts
}
