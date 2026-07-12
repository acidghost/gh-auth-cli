package cmd

import (
	"fmt"

	"github.com/acidghost/gh-auth-cli/internal/auth"
	"github.com/acidghost/gh-auth-cli/internal/store"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func newLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authorize the GitHub App using a browser and PKCE",
		Args:  cobra.NoArgs,
		RunE:  runLogin,
	}
}

func runLogin(command *cobra.Command, _ []string) error {
	ctx := command.Context()
	cfg, err := store.LoadConfig()
	if err != nil {
		return configureHint(err)
	}
	logger := zerolog.Ctx(ctx)
	logger.Info().Str("profile_id", cfg.ProfileID).Msg("starting GitHub authorization")
	logger.Debug().Msg("loading GitHub App client secret from keyring")
	secret, err := store.GetClientSecret(cfg.ProfileID)
	if err != nil {
		return fmt.Errorf("load GitHub App client secret: %w", err)
	}
	token, err := auth.Login(ctx, cfg, secret)
	if err != nil {
		return err
	}
	if cfg.ActiveUserID != 0 && cfg.ActiveUserID != token.UserID {
		if err := store.DeleteToken(cfg.ProfileID, cfg.ActiveUserID); err != nil {
			return err
		}
	}
	if err := store.SetToken(cfg.ProfileID, token); err != nil {
		return err
	}
	cfg.ActiveUserID = token.UserID
	cfg.ActiveLogin = token.Login
	if err := store.SaveConfig(cfg); err != nil {
		return err
	}
	logger.Info().
		Str("login", token.Login).
		Int64("user_id", token.UserID).
		Time("access_expires_at", token.AccessExpiresAt).
		Time("refresh_expires_at", token.RefreshExpiresAt).
		Msg("GitHub authorization completed")
	fmt.Fprintf(command.ErrOrStderr(), "Authenticated to GitHub as %s.\n", token.Login)
	return nil
}
