package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(build BuildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show build information",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(command.OutOrStdout(), "gh-auth-cli %s\ncommit: %s\ndate: %s\n", build.Version, build.Commit, build.Date)
			return err
		},
	}
}
