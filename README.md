# Tmux Command Palette

`tmcp` 是一个给 tmux 用的命令面板。它通过 `tmux display-popup -E` 拉起，提供：

- tmux 命令的 fuzzy 检索
- 基于命令签名的 flag 和 target 补全
- 回车直接执行当前输入的 tmux 命令

## Build

```bash
go build -o tmcp .
```

## Tmux Binding

把下面这段放进 `~/.tmux.conf`：

```tmux
bind-key C-p display-popup -E -w 80% -h 70% "$HOME/path/to/tmcp"
```

如果你直接在当前仓库里构建，也可以写成：

```tmux
bind-key C-p display-popup -E -w 80% -h 70% "/home/liu/dev/tmcp/tmcp"
```

重载配置：

```bash
tmux source-file ~/.tmux.conf
```

## Screenshot

`TODO`: add screenshot placeholder for the popup UI.
