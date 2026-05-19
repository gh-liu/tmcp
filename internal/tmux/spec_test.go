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
					{Name: "-d"},
					{Name: "-E"},
					{Name: "-r"},
					{Name: "-x"},
					{Name: "-c", Value: "working-directory"},
					{Name: "-f", Value: "flags"},
					{Name: "-t", Value: "target-session"},
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
					{Name: "-O"},
					{Name: "-b", Value: "border-lines"},
					{Name: "-c", Value: "target-client"},
					{Name: "-C", Value: "starting-choice"},
					{Name: "-H", Value: "selected-style"},
					{Name: "-s", Value: "style"},
					{Name: "-S", Value: "border-style"},
					{Name: "-t", Value: "target-pane"},
					{Name: "-T", Value: "title"},
					{Name: "-x", Value: "position"},
					{Name: "-y", Value: "position"},
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
					{Name: "-F"},
					{Name: "-H"},
					{Name: "-K"},
					{Name: "-l"},
					{Name: "-M"},
					{Name: "-R"},
					{Name: "-X"},
					{Name: "-c", Value: "target-client"},
					{Name: "-N", Value: "repeat-count"},
					{Name: "-t", Value: "target-pane"},
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
					{Name: "-n"},
					{Name: "-r"},
					{Name: "-T", Value: "key-table"},
					{Name: "-N", Value: "note"},
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

func TestParseFlagGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []Flag
	}{
		{
			name: "combined short flags",
			raw:  "-abdP",
			want: []Flag{{Name: "-a"}, {Name: "-b"}, {Name: "-d"}, {Name: "-P"}},
		},
		{
			name: "flag with value",
			raw:  "-t target-pane",
			want: []Flag{{Name: "-t", Value: "target-pane"}},
		},
		{
			name: "mutually exclusive flags",
			raw:  "-L|-S|-U",
			want: []Flag{{Name: "-L"}, {Name: "-S"}, {Name: "-U"}},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := parseFlagGroup(tc.raw)
			if !equalFlags(got, tc.want) {
				t.Fatalf("parseFlagGroup() = %#v, want %#v", got, tc.want)
			}
		})
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
