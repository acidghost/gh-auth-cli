package cmd

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/acidghost/gh-auth-cli/internal/auth"
	"github.com/acidghost/gh-auth-cli/internal/lock"
	"github.com/acidghost/gh-auth-cli/internal/store"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type tokenOptions struct {
	minValidity    time.Duration
	nonInteractive bool
}

func newTokenCommand() *cobra.Command {
	options := tokenOptions{}
	command := &cobra.Command{
		Use:   "token",
		Short: "Print the current access token for cmd:// capture",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runToken(command, options)
		},
	}
	command.Flags().DurationVar(&options.minValidity, "min-validity", 20*time.Minute, "minimum required access-token lifetime")
	command.Flags().BoolVar(&options.nonInteractive, "non-interactive", false, "fail instead of starting an interactive login (currently always enabled)")
	return command
}

func runToken(command *cobra.Command, options tokenOptions) error {
	if options.minValidity < 0 {
		return errors.New("min-validity must not be negative")
	}
	ctx := command.Context()
	cfg, err := store.LoadConfig()
	if err != nil {
		return configureHint(err)
	}
	if cfg.ActiveUserID == 0 {
		return errors.New("not authenticated; run `gh-auth-cli login`")
	}
	logger := zerolog.Ctx(ctx)
	logger.Debug().
		Int64("user_id", cfg.ActiveUserID).
		Dur("min_validity", options.minValidity).
		Bool("non_interactive", options.nonInteractive).
		Msg("acquiring token refresh lock")
	tokenLock, err := lock.Acquire(cfg.ProfileID + "-" + strconv.FormatInt(cfg.ActiveUserID, 10))
	if err != nil {
		return err
	}
	defer tokenLock.Close()

	token, err := store.GetToken(cfg.ProfileID, cfg.ActiveUserID)
	if err != nil {
		return fmt.Errorf("load GitHub token: %w; run `gh-auth-cli login`", err)
	}
	refreshed := false
	if !auth.Usable(token, options.minValidity) {
		logger.Info().
			Str("login", token.Login).
			Time("access_expires_at", token.AccessExpiresAt).
			Msg("refreshing GitHub access token")
		secret, secretErr := store.GetClientSecret(cfg.ProfileID)
		if secretErr != nil {
			return fmt.Errorf("load GitHub App client secret for refresh: %w", secretErr)
		}
		token, err = auth.Refresh(ctx, cfg, secret, token)
		if err != nil {
			return err
		}
		if err := store.SetToken(cfg.ProfileID, token); err != nil {
			return fmt.Errorf("refreshed token but could not persist the rotated credentials: %w; run login again before the access token expires", err)
		}
		refreshed = true
	} else {
		logger.Debug().Time("access_expires_at", token.AccessExpiresAt).Msg("stored access token has sufficient validity")
	}
	logger.Info().
		Str("login", token.Login).
		Bool("refreshed", refreshed).
		Time("access_expires_at", token.AccessExpiresAt).
		Msg("serving GitHub access token")
	_, err = io.WriteString(command.OutOrStdout(), token.AccessToken+"\n")
	return err
}
