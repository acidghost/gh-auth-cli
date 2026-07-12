package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/acidghost/gh-auth-cli/internal/auth"
	"github.com/acidghost/gh-auth-cli/internal/store"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const defaultCallbackURL = "http://127.0.0.1:17836/callback"

type configureOptions struct {
	clientID    string
	callbackURL string
	secretStdin bool
}

func newConfigureCommand() *cobra.Command {
	options := configureOptions{}
	command := &cobra.Command{
		Use:   "configure",
		Short: "Store GitHub App configuration and client secret",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return runConfigure(command, options)
		},
	}
	command.Flags().StringVar(&options.clientID, "client-id", "", "GitHub App client ID")
	command.Flags().StringVar(&options.callbackURL, "callback-url", defaultCallbackURL, "OAuth callback URL configured on the GitHub App")
	command.Flags().BoolVar(&options.secretStdin, "client-secret-stdin", false, "read the GitHub App client secret from stdin")
	return command
}

func runConfigure(command *cobra.Command, options configureOptions) error {
	ctx := command.Context()
	logger := zerolog.Ctx(ctx)
	logger.Debug().Bool("client_secret_stdin", options.secretStdin).Msg("parsed configure options")

	existing, err := store.LoadConfig()
	if err != nil && !errors.Is(err, store.ErrNotConfigured) {
		return err
	}
	if existing != nil && !command.Flags().Changed("callback-url") {
		options.callbackURL = existing.CallbackURL
	}
	if options.clientID == "" {
		if existing != nil {
			options.clientID = existing.ClientID
		} else {
			fmt.Fprint(command.ErrOrStderr(), "GitHub App client ID: ")
			line, readErr := bufio.NewReader(command.InOrStdin()).ReadString('\n')
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				return fmt.Errorf("read client ID: %w", readErr)
			}
			options.clientID = strings.TrimSpace(line)
		}
	}
	if options.clientID == "" {
		return errors.New("client ID must not be empty")
	}
	if existing != nil && existing.ActiveUserID != 0 && existing.ClientID != options.clientID {
		return errors.New("cannot change the client ID while logged in; run logout first")
	}
	if err := auth.ValidateCallbackURL(options.callbackURL); err != nil {
		return err
	}

	secret, err := readClientSecret(command.InOrStdin(), command.ErrOrStderr(), options.secretStdin)
	if err != nil {
		return err
	}
	if secret == "" {
		return errors.New("client secret must not be empty")
	}

	cfg := existing
	if cfg == nil {
		cfg, err = store.DefaultConfig(options.clientID, options.callbackURL)
		if err != nil {
			return err
		}
	}
	cfg.ClientID = options.clientID
	cfg.CallbackURL = options.callbackURL
	logger.Debug().Str("profile_id", cfg.ProfileID).Msg("storing client secret in keyring")
	if err := store.SetClientSecret(cfg.ProfileID, secret); err != nil {
		return err
	}
	if err := store.SaveConfig(cfg); err != nil {
		return err
	}
	logger.Info().
		Str("profile_id", cfg.ProfileID).
		Str("callback_url", cfg.CallbackURL).
		Msg("GitHub App configuration saved")
	fmt.Fprintln(command.ErrOrStderr(), "GitHub App configuration saved; run `gh-auth-cli login` to authorize it.")
	return nil
}

func readClientSecret(in io.Reader, errOut io.Writer, fromStdin bool) (string, error) {
	if fromStdin {
		data, err := io.ReadAll(io.LimitReader(in, 16*1024))
		if err != nil {
			return "", fmt.Errorf("read client secret: %w", err)
		}
		return strings.TrimRight(string(data), "\r\n"), nil
	}
	input, ok := in.(*os.File)
	if !ok || !term.IsTerminal(int(input.Fd())) {
		return "", errors.New("stdin is not a terminal; pass the client secret using --client-secret-stdin")
	}
	fmt.Fprint(errOut, "GitHub App client secret: ")
	value, err := term.ReadPassword(int(input.Fd()))
	fmt.Fprintln(errOut)
	if err != nil {
		return "", fmt.Errorf("read client secret: %w", err)
	}
	return string(value), nil
}
