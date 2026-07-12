package logging

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestPathFor(t *testing.T) {
	t.Parallel()
	home := filepath.Join("home", "alice")
	tests := []struct {
		goos string
		want string
	}{
		{goos: "darwin", want: filepath.Join(home, "Library", "Logs", "gh-auth-cli.log")},
		{goos: "linux", want: filepath.Join(home, ".local", "share", "gh-auth-cli.log")},
		{goos: "freebsd", want: filepath.Join(home, ".local", "share", "gh-auth-cli.log")},
	}
	for _, test := range tests {
		t.Run(test.goos, func(t *testing.T) {
			t.Parallel()
			got, err := pathFor(test.goos, home)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("pathFor(%q) = %q, want %q", test.goos, got, test.want)
			}
		})
	}
}

func TestPathForUnsupportedPlatform(t *testing.T) {
	t.Parallel()
	_, err := pathFor("windows", "home")
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Fatalf("pathFor(windows) error = %v, want ErrUnsupportedPlatform", err)
	}
}
