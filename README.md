# Tmux Command Palette

`tmcp` is a command palette for tmux. It is designed to run inside `tmux display-popup -E` and provides:

- fuzzy search for tmux commands
- flag and target completion based on tmux command signatures
- direct execution of the typed tmux command on Enter

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
