package cmd

import (
	"fmt"
	"time"

	"github.com/acidghost/gh-auth-cli/internal/store"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show configuration and token expiry without revealing secrets",
		Args:  cobra.NoArgs,
		RunE:  runStatus,
	}
}

func runStatus(command *cobra.Command, _ []string) error {
	ctx := command.Context()
	cfg, err := store.LoadConfig()
	if err != nil {
		return configureHint(err)
	}
	logger := zerolog.Ctx(ctx)
	logger.Debug().Str("profile_id", cfg.ProfileID).Msg("loading authentication status")
	_, secretErr := store.GetClientSecret(cfg.ProfileID)
	out := command.OutOrStdout()
	fmt.Fprintf(out, "GitHub host:  %s\nClient ID:    %s\n", cfg.GitHubHost, cfg.ClientID)
	if secretErr == nil {
		fmt.Fprintln(out, "Client secret: stored in keyring")
	} else {
		fmt.Fprintln(out, "Client secret: unavailable")
	}
	if cfg.ActiveUserID == 0 {
		logger.Info().Msg("authentication status checked: not authenticated")
		fmt.Fprintln(out, "Status:       not authenticated")
		return nil
	}
	token, err := store.GetToken(cfg.ProfileID, cfg.ActiveUserID)
	if err != nil {
		return fmt.Errorf("load token metadata: %w", err)
	}
	fmt.Fprintf(out, "Account:      %s (%d)\n", token.Login, token.UserID)
	if token.AccessExpiresAt.IsZero() {
		fmt.Fprintln(out, "Access expiry: does not expire")
	} else {
		fmt.Fprintf(out, "Access expiry: %s\n", token.AccessExpiresAt.Local().Format(time.RFC3339))
	}
	logger.Info().
		Str("login", token.Login).
		Int64("user_id", token.UserID).
		Time("access_expires_at", token.AccessExpiresAt).
		Msg("authentication status checked")
	if token.RefreshToken == "" {
		fmt.Fprintln(out, "Refresh:      unavailable")
	} else if token.RefreshExpiresAt.IsZero() {
		fmt.Fprintln(out, "Refresh:      available")
	} else {
		fmt.Fprintf(out, "Refresh expiry: %s\n", token.RefreshExpiresAt.Local().Format(time.RFC3339))
	}
	return nil
}
