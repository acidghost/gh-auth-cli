package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommands(t *testing.T) {
	t.Parallel()
	root := NewRootCommand(BuildInfo{Version: "test", Commit: "abc123", Date: "today"})
	for _, name := range []string{"configure", "login", "token", "status", "logout", "version"} {
		command, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatalf("find command %q: %v", name, err)
		}
		if command.Name() != name {
			t.Fatalf("Find(%q) returned %q", name, command.Name())
		}
	}
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	root := NewRootCommand(BuildInfo{Version: "v1.2.3", Commit: "abc123", Date: "today"})
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	want := "gh-auth-cli v1.2.3\ncommit: abc123\ndate: today\n"
	if output.String() != want {
		t.Fatalf("version output = %q, want %q", output.String(), want)
	}
}

func TestRootVersionFlag(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	root := NewRootCommand(BuildInfo{Version: "v1.2.3", Commit: "abc123", Date: "today"})
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs([]string{"--version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"v1.2.3", "abc123", "today"} {
		if !strings.Contains(output.String(), value) {
			t.Fatalf("version output %q does not contain %q", output.String(), value)
		}
	}
}
