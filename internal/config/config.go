package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Command struct {
	Name    string
	Aliases []string
	Note    string
	Run     []string
}

func Load() ([]Command, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	return Parse(file)
}

func Parse(r interface{ Read([]byte) (int, error) }) ([]Command, error) {
	scanner := bufio.NewScanner(r)
	var (
		commands []Command
		current  *Command
		lineNo   int
	)

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}

		if line == "[[commands]]" {
			commands = append(commands, Command{})
			current = &commands[len(commands)-1]
			continue
		}

		if current == nil {
			return nil, fmt.Errorf("config line %d: expected [[commands]] before fields", lineNo)
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("config line %d: expected key = value", lineNo)
		}

		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)

		switch key {
		case "name":
			value, err := parseString(rawValue)
			if err != nil {
				return nil, fmt.Errorf("config line %d: %w", lineNo, err)
			}
			current.Name = value
		case "note":
			value, err := parseString(rawValue)
			if err != nil {
				return nil, fmt.Errorf("config line %d: %w", lineNo, err)
			}
			current.Note = value
		case "aliases":
			value, err := parseStringArray(rawValue)
			if err != nil {
				return nil, fmt.Errorf("config line %d: %w", lineNo, err)
			}
			current.Aliases = value
		case "run":
			value, err := parseStringArray(rawValue)
			if err != nil {
				return nil, fmt.Errorf("config line %d: %w", lineNo, err)
			}
			current.Run = value
		default:
			return nil, fmt.Errorf("config line %d: unknown field %q", lineNo, key)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for i, command := range commands {
		if strings.TrimSpace(command.Name) == "" {
			return nil, fmt.Errorf("config command %d: name is required", i+1)
		}
		if len(command.Run) == 0 {
			return nil, fmt.Errorf("config command %q: run is required", command.Name)
		}
	}

	return commands, nil
}

func configPath() (string, error) {
	if configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); configHome != "" {
		return filepath.Join(configHome, "tmcp", "config.toml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "tmcp", "config.toml"), nil
}

func stripComment(line string) string {
	var (
		builder  strings.Builder
		inString bool
		escaped  bool
	)

	for _, r := range line {
		switch {
		case escaped:
			builder.WriteRune(r)
			escaped = false
		case r == '\\' && inString:
			builder.WriteRune(r)
			escaped = true
		case r == '"':
			builder.WriteRune(r)
			inString = !inString
		case r == '#' && !inString:
			return builder.String()
		default:
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func parseString(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", fmt.Errorf("expected quoted string")
	}

	value := raw[1 : len(raw)-1]
	value = strings.ReplaceAll(value, `\"`, `"`)
	value = strings.ReplaceAll(value, `\\`, `\`)
	return value, nil
}

func parseStringArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '[' || raw[len(raw)-1] != ']' {
		return nil, fmt.Errorf("expected string array")
	}

	raw = strings.TrimSpace(raw[1 : len(raw)-1])
	if raw == "" {
		return nil, nil
	}

	var (
		values   []string
		current  strings.Builder
		inString bool
		escaped  bool
	)

	flush := func() error {
		part := strings.TrimSpace(current.String())
		current.Reset()
		if part == "" {
			return fmt.Errorf("expected quoted string")
		}
		value, err := parseString(part)
		if err != nil {
			return err
		}
		values = append(values, value)
		return nil
	}

	for _, r := range raw {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\' && inString:
			current.WriteRune(r)
			escaped = true
		case r == '"':
			current.WriteRune(r)
			inString = !inString
		case r == ',' && !inString:
			if err := flush(); err != nil {
				return nil, err
			}
		default:
			current.WriteRune(r)
		}
	}

	if inString {
		return nil, fmt.Errorf("unterminated string")
	}

	if err := flush(); err != nil {
		return nil, err
	}

	return values, nil
}
