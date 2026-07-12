package cmd

import (
	"errors"
	"fmt"

	"github.com/acidghost/gh-auth-cli/internal/store"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type logoutOptions struct {
	all bool
}

func newLogoutCommand() *cobra.Command {
	options := logoutOptions{}
	command := &cobra.Command{
		Use:   "logout",
		Short: "Delete the active user's stored authentication",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runLogout(command, options)
		},
	}
	command.Flags().BoolVar(&options.all, "all", false, "also delete the GitHub App client secret and configuration")
	return command
}

func runLogout(command *cobra.Command, options logoutOptions) error {
	ctx := command.Context()
	cfg, err := store.LoadConfig()
	if errors.Is(err, store.ErrNotConfigured) {
		zerolog.Ctx(ctx).Debug().Msg("logout requested without existing configuration")
		return nil
	}
	if err != nil {
		return err
	}
	loggedOutUserID := cfg.ActiveUserID
	if loggedOutUserID != 0 {
		if err := store.DeleteToken(cfg.ProfileID, loggedOutUserID); err != nil {
			return err
		}
	}
	logger := zerolog.Ctx(ctx)
	if options.all {
		logger.Debug().Str("profile_id", cfg.ProfileID).Msg("deleting client secret and configuration")
		if err := store.DeleteClientSecret(cfg.ProfileID); err != nil {
			return err
		}
		if err := store.DeleteConfig(); err != nil {
			return err
		}
		logger.Info().Msg("authentication and GitHub App configuration removed")
		fmt.Fprintln(command.ErrOrStderr(), "Authentication and GitHub App configuration removed.")
		return nil
	}
	cfg.ActiveUserID = 0
	cfg.ActiveLogin = ""
	if err := store.SaveConfig(cfg); err != nil {
		return err
	}
	logger.Info().Int64("user_id", loggedOutUserID).Msg("GitHub authentication removed")
	fmt.Fprintln(command.ErrOrStderr(), "GitHub authentication removed.")
	return nil
}
