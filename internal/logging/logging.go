package logging

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
)

var ErrUnsupportedPlatform = errors.New("logging is unsupported on this platform")

type Log struct {
	Logger zerolog.Logger
	file   *os.File
	Path   string
}

func Open() (*Log, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find home directory for log file: %w", err)
	}
	path, err := pathFor(runtime.GOOS, home)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	// #nosec G304 -- path is derived only from the home directory and platform.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("secure log file: %w", err)
	}

	level, err := configuredLevel()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	logger := zerolog.New(file).Level(level).With().Timestamp().Logger()
	return &Log{Logger: logger, file: file, Path: path}, nil
}

func (l *Log) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close log file: %w", err)
	}
	return nil
}

func pathFor(goos, home string) (string, error) {
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Logs", "gh-auth-cli.log"), nil
	case "linux", "freebsd", "openbsd", "netbsd", "dragonfly", "solaris":
		return filepath.Join(home, ".local", "share", "gh-auth-cli.log"), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedPlatform, goos)
	}
}

func configuredLevel() (zerolog.Level, error) {
	value := strings.TrimSpace(os.Getenv("GH_AUTH_CLI_LOG_LEVEL"))
	if value == "" {
		return zerolog.InfoLevel, nil
	}
	level, err := zerolog.ParseLevel(value)
	if err != nil {
		return zerolog.NoLevel, fmt.Errorf("invalid GH_AUTH_CLI_LOG_LEVEL %q: %w", value, err)
	}
	return level, nil
}

var _ io.Closer = (*Log)(nil)
