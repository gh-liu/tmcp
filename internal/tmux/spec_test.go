package tmux

import "testing"

func TestParseCommandLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want Command
	}{
		{
			name: "alias and optional flags",
			line: "attach-session (attach) [-dErx] [-c working-directory] [-f flags] [-t target-session]",
			want: Command{
				Name:    "attach-session",
				Aliases: []string{"attach"},
				Flags: []Flag{
					{Raw: "-dErx"},
					{Raw: "-c working-directory"},
					{Raw: "-f flags"},
					{Raw: "-t target-session"},
				},
			},
		},
		{
			name: "variadic positionals",
			line: "display-menu (menu) [-O] [-b border-lines] [-c target-client] [-C starting-choice] [-H selected-style] [-s style] [-S border-style] [-t target-pane][-T title] [-x position] [-y position] name key command ...",
			want: Command{
				Name:    "display-menu",
				Aliases: []string{"menu"},
				Flags: []Flag{
					{Raw: "-O"},
					{Raw: "-b border-lines"},
					{Raw: "-c target-client"},
					{Raw: "-C starting-choice"},
					{Raw: "-H selected-style"},
					{Raw: "-s style"},
					{Raw: "-S border-style"},
					{Raw: "-t target-pane"},
					{Raw: "-T title"},
					{Raw: "-x position"},
					{Raw: "-y position"},
				},
				Positional: []string{"name", "key", "command", "..."},
			},
		},
		{
			name: "mixed required and optional arguments",
			line: "send-keys (send) [-FHKlMRX] [-c target-client] [-N repeat-count] [-t target-pane] key ...",
			want: Command{
				Name:    "send-keys",
				Aliases: []string{"send"},
				Flags: []Flag{
					{Raw: "-FHKlMRX"},
					{Raw: "-c target-client"},
					{Raw: "-N repeat-count"},
					{Raw: "-t target-pane"},
				},
				Positional: []string{"key", "..."},
			},
		},
		{
			name: "optional positional wrapped in brackets",
			line: "bind-key (bind) [-nr] [-T key-table] [-N note] key [command [arguments]]",
			want: Command{
				Name:    "bind-key",
				Aliases: []string{"bind"},
				Flags: []Flag{
					{Raw: "-nr"},
					{Raw: "-T key-table"},
					{Raw: "-N note"},
				},
				Positional: []string{"key", "command [arguments]"},
			},
		},
		{
			name: "command without signature",
			line: "kill-server ",
			want: Command{
				Name: "kill-server",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseCommandLine(tc.line)
			if err != nil {
				t.Fatalf("ParseCommandLine() error = %v", err)
			}

			if got.Name != tc.want.Name {
				t.Fatalf("Name = %q, want %q", got.Name, tc.want.Name)
			}

			if !equalStrings(got.Aliases, tc.want.Aliases) {
				t.Fatalf("Aliases = %#v, want %#v", got.Aliases, tc.want.Aliases)
			}

			if !equalFlags(got.Flags, tc.want.Flags) {
				t.Fatalf("Flags = %#v, want %#v", got.Flags, tc.want.Flags)
			}

			if !equalStrings(got.Positional, tc.want.Positional) {
				t.Fatalf("Positional = %#v, want %#v", got.Positional, tc.want.Positional)
			}
		})
	}
}

func TestParseCommands(t *testing.T) {
	t.Parallel()

	out := []byte("kill-server \nlist-sessions (ls) [-F format] [-f filter]\n")

	got, err := ParseCommands(out)
	if err != nil {
		t.Fatalf("ParseCommands() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(ParseCommands()) = %d, want 2", len(got))
	}

	if got[1].Name != "list-sessions" {
		t.Fatalf("got[1].Name = %q, want %q", got[1].Name, "list-sessions")
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func equalFlags(got, want []Flag) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
