package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/lanbox/internal/config"
	"github.com/RakhaYandra/lanbox/internal/discovery"
	"github.com/RakhaYandra/lanbox/internal/server"
)

var serveCfg config.Config

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the file-sharing server",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.ExpandDir(serveCfg.Dir)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(root, 0o755); err != nil {
			return fmt.Errorf("cannot access %s: %w", root, err)
		}
		log := slog.New(slog.NewTextHandler(os.Stdout, nil))
		srv := server.New(root, serveCfg.Host, serveCfg.Port, serveCfg.WebDir, log)
		fmt.Printf("Serving %s\nLocal: http://localhost:%d\nNetwork: http://%s:%d\n",
			root, serveCfg.Port, discovery.LANIP(), serveCfg.Port)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		return srv.Run(ctx)
	},
}

func init() {
	defs := config.Defaults()
	serveCmd.Flags().StringVar(&serveCfg.Dir, "dir", defs.Dir, "directory to share")
	serveCmd.Flags().IntVar(&serveCfg.Port, "port", defs.Port, "port to listen on")
	serveCmd.Flags().StringVar(&serveCfg.Host, "host", defs.Host, "address to bind")
	serveCmd.Flags().StringVar(&serveCfg.WebDir, "web-dir", defs.WebDir, "directory with lanbox-web build output")
}
