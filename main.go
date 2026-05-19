package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gh-liu/tmcp/internal/tmux"
	"github.com/gh-liu/tmcp/internal/ui"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		commands, err := tmux.ListCommands(ctx)
		if err != nil {
			return err
		}

		selection, err := ui.PickCommand(commands)
		if err != nil {
			return err
		}

		if selection != "" {
			fmt.Println(selection)
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
