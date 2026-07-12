package cmd

import (
	"errors"

	"github.com/acidghost/gh-auth-cli/internal/store"
)

func configureHint(err error) error {
	if errors.Is(err, store.ErrNotConfigured) {
		return errors.New("not configured; run `gh-auth-cli configure --client-id <id>`")
	}
	return err
}
