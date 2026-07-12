package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

type Lock struct {
	file *os.File
}

func Acquire(name string) (*Lock, error) {
	if name == "" || strings.ContainsAny(name, `/\\`) || filepath.Base(name) != name {
		return nil, fmt.Errorf("invalid lock name %q", name)
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("find user cache directory: %w", err)
	}
	dir := filepath.Join(cacheDir, "gh-auth-cli", "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	// #nosec G304 -- name is restricted to one path component above.
	file, err := os.OpenFile(filepath.Join(dir, name+".lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open token lock: %w", err)
	}
	if err := unix.Flock(int(file.Fd()), unix.LOCK_EX); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return nil, fmt.Errorf("lock token: %w (also failed to close lock file: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("lock token: %w", err)
	}
	return &Lock{file: file}, nil
}

func (l *Lock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	unlockErr := unix.Flock(int(l.file.Fd()), unix.LOCK_UN)
	closeErr := l.file.Close()
	if unlockErr != nil {
		return fmt.Errorf("unlock token: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close token lock: %w", closeErr)
	}
	return nil
}
