package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/lanbox/internal/config"
)

var serveCfg config.Config

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the file-sharing server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Serving %s on %s:%d\n", serveCfg.Dir, serveCfg.Host, serveCfg.Port)
		return nil
	},
}

func init() {
	defs := config.Defaults()
	serveCmd.Flags().StringVar(&serveCfg.Dir, "dir", defs.Dir, "directory to share")
	serveCmd.Flags().IntVar(&serveCfg.Port, "port", defs.Port, "port to listen on")
	serveCmd.Flags().StringVar(&serveCfg.Host, "host", defs.Host, "address to bind")
	serveCmd.Flags().StringVar(&serveCfg.WebDir, "web-dir", defs.WebDir, "directory with lanbox-web build output")
}
