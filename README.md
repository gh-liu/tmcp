# Tmux Command Palette

`tmcp` is a command palette for tmux. It is designed to run inside `tmux display-popup -E` and make tmux commands faster to discover, complete, and execute.

## Features

- Fuzzy command search for tmux commands and aliases
- Positional and flag-value completion for common tmux targets such as sessions, windows, panes, layouts, options, hooks, and key tables
- History search backed by a persistent history file at `$XDG_STATE_HOME/tmcp/history` or `~/.local/state/tmcp/history`
- Inline ghost-text previews that show the currently selected completion without committing it into the input
- Popup-friendly candidate navigation with line-by-line and half-page movement

## Default Keys

- `Enter`: execute the typed tmux command
- `Tab`: accept the currently selected completion
- `Right`: accept the current ghost-text completion
- `Up` / `Down`: move through the candidate list
- `Ctrl-P` / `Ctrl-N`: move through the candidate list
- `Ctrl-U` / `Ctrl-D`: scroll half a page up or down
- `PageUp` / `PageDown`: scroll half a page up or down
- `Ctrl-R`: toggle history search mode
- `Esc` / `Ctrl-C`: close the palette

## Install

```bash
go install github.com/gh-liu/tmcp@latest
```

## Tmux Binding

Add this to `~/.tmux.conf`:

```tmux
bind-key C-p display-popup -E -w 80% -h 70% tmcp
```

Reload tmux config:

```bash
tmux source-file ~/.tmux.conf
```

## Screenshot

`TODO`: add a screenshot of the popup UI.
