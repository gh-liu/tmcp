package config

import (
	"os"
	"strings"
	"testing"
)

func TestParseCommands(t *testing.T) {
	t.Parallel()

	input := strings.NewReader(`
[[commands]]
name = "swap-left"
note = "swap current pane"
aliases = ["sl", "swapl"]
run = ["swap-pane", "-t", "{left}"]

[[commands]]
name = "join-from-right" # trailing comment
run = ["join-pane", "-s", "{right}"]
`)

	got, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("len(Parse()) = %d, want 2", len(got))
	}

	if got[0].Name != "swap-left" {
		t.Fatalf("first command name = %q, want %q", got[0].Name, "swap-left")
	}
	if got[0].Note != "swap current pane" {
		t.Fatalf("first command note = %q, want %q", got[0].Note, "swap current pane")
	}
	if len(got[0].Aliases) != 2 || got[0].Aliases[0] != "sl" || got[0].Aliases[1] != "swapl" {
		t.Fatalf("first command aliases = %#v, want %#v", got[0].Aliases, []string{"sl", "swapl"})
	}
	if len(got[0].Run) != 3 || got[0].Run[2] != "{left}" {
		t.Fatalf("first command run = %#v, want target token preserved", got[0].Run)
	}
	if got[1].Name != "join-from-right" {
		t.Fatalf("second command name = %q, want %q", got[1].Name, "join-from-right")
	}
}

func TestParseRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	_, err := Parse(strings.NewReader(`
[[commands]]
name = "broken"
`))
	if err == nil || !strings.Contains(err.Error(), `run is required`) {
		t.Fatalf("Parse() error = %v, want missing run error", err)
	}
}

func TestConfigPathUsesXDGConfigHome(t *testing.T) {
	previous, hadPrevious := os.LookupEnv("XDG_CONFIG_HOME")
	t.Cleanup(func() {
		if hadPrevious {
			_ = os.Setenv("XDG_CONFIG_HOME", previous)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	_ = os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-config")

	got, err := configPath()
	if err != nil {
		t.Fatalf("configPath() error = %v", err)
	}

	want := "/tmp/xdg-config/tmcp/config.toml"
	if got != want {
		t.Fatalf("configPath() = %q, want %q", got, want)
	}
}
