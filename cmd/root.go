package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func NewRootCommand(build BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:           "gh-auth-cli",
		Short:         "Authenticate to GitHub through a personal GitHub App",
		Version:       build.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetVersionTemplate(fmt.Sprintf("gh-auth-cli {{.Version}}\ncommit: %s\ndate: %s\n", build.Commit, build.Date))
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(
		newConfigureCommand(),
		newLoginCommand(),
		newTokenCommand(),
		newStatusCommand(),
		newLogoutCommand(),
		newVersionCommand(build),
	)
	return root
}
