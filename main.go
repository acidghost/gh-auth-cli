package main

import (
	"context"
	"fmt"
	"os"

	"github.com/acidghost/gh-auth-cli/cmd"
	"github.com/acidghost/gh-auth-cli/internal/logging"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	output, err := logging.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gh-auth-cli: initialize logging: %v\n", err)
		return 1
	}
	defer func() {
		if closeErr := output.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "gh-auth-cli: %v\n", closeErr)
		}
	}()

	ctx := output.Logger.WithContext(context.Background())
	logger := zerolog.Ctx(ctx)
	logger.Debug().
		Str("version", buildVersion).
		Str("commit", buildCommit).
		Str("log_path", output.Path).
		Msg("application initialized")

	root := cmd.NewRootCommand(cmd.BuildInfo{
		Version: buildVersion,
		Commit:  buildCommit,
		Date:    buildDate,
	})
	root.SetArgs(os.Args[1:])
	root.SetIn(os.Stdin)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	command := commandName(root, os.Args[1:])
	logger.Info().Str("command", command).Msg("command started")

	if err := root.ExecuteContext(ctx); err != nil {
		logger.Error().Err(err).Str("command", command).Msg("command failed")
		fmt.Fprintf(os.Stderr, "gh-auth-cli: %v\n", err)
		return 1
	}

	logger.Info().Str("command", command).Msg("command completed")
	return 0
}

func commandName(root *cobra.Command, args []string) string {
	command, _, err := root.Find(args)
	if err != nil || command == nil {
		return root.CommandPath()
	}
	return command.CommandPath()
}
