package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "lanbox",
	Short: "Local-first file sharing over the LAN",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(versionCmd)
}
