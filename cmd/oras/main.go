package main

import (
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/attach"
	"oras.land/oras/cmd/oras/copy"
	"oras.land/oras/cmd/oras/discover"
	"oras.land/oras/cmd/oras/login"
	"oras.land/oras/cmd/oras/logout"
	"oras.land/oras/cmd/oras/manifest"
	"oras.land/oras/cmd/oras/pull"
	"oras.land/oras/cmd/oras/push"
	"oras.land/oras/cmd/oras/version"
)

func main() {
	cmd := &cobra.Command{
		Use:          "oras [command]",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		pull.Cmd(),
		push.Cmd(),
		login.Cmd(),
		logout.Cmd(),
		version.Cmd(),
		discover.Cmd(),
		copy.Cmd(),
		attach.Cmd(),
		manifest.Cmd(),
	)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
