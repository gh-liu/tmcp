package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gh-liu/tmcp/internal/config"
	"github.com/gh-liu/tmcp/internal/tmux"
	"github.com/gh-liu/tmcp/internal/ui"
)

var executeTokens = tmux.ExecuteTokens

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		customCommands, err := config.Load()
		if err != nil {
			return err
		}

		commands, err := tmux.ListCommands(ctx)
		if err != nil {
			return err
		}

		history, err := ui.LoadHistory()
		if err != nil {
			return err
		}

		line, err := ui.ReadCommandLineWithHistoryAndCommands(commands, customCommands, history)
		if err != nil {
			return err
		}

		if err := executeLine(ctx, commands, customCommands, line); err != nil {
			return err
		}

		history = ui.AppendHistory(history, line)
		if err := ui.SaveHistory(history); err != nil {
			return err
		}

		return nil
	}

	switch args[0] {
	case "ls":
		commands, err := tmux.ListCommands(ctx)
		if err != nil {
			return err
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(commands)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func executeLine(ctx context.Context, commands []tmux.Command, customCommands []config.Command, line string) error {
	tokens := strings.Fields(line)
	if len(tokens) == 0 {
		return nil
	}

	if command, ok := findCustomCommand(customCommands, tokens[0]); ok {
		argv := make([]string, 0, len(command.Run)+len(tokens)-1)
		argv = append(argv, command.Run...)
		argv = append(argv, tokens[1:]...)
		return executeTokens(ctx, argv)
	}

	return executeTokens(ctx, tokens)
}

func findCustomCommand(commands []config.Command, token string) (config.Command, bool) {
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

	return config.Command{}, false
}
