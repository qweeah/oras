package main

import (
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/attach"
	"oras.land/oras/cmd/oras/manifest"
)

func main() {
	cmd := &cobra.Command{
		Use:          "oras [command]",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		pullCmd(),
		pushCmd(),
		loginCmd(),
		logoutCmd(),
		versionCmd(),
		discoverCmd(),
		copyCmd(),
		attach.Cmd(),
		manifest.Cmd(),
	)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
