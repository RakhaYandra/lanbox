package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/lanbox/internal/auth"
	"github.com/RakhaYandra/lanbox/internal/config"
	"github.com/RakhaYandra/lanbox/internal/discovery"
	"github.com/RakhaYandra/lanbox/internal/server"
)

var serveLimit string

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
		token, err := auth.NewToken()
		if err != nil {
			return fmt.Errorf("cannot generate token: %w", err)
		}
		pin, err := auth.NewPIN()
		if err != nil {
			return fmt.Errorf("cannot generate PIN: %w", err)
		}
		srv := server.New(root, serveCfg.Host, serveCfg.Port, serveCfg.WebDir, token, log)
		if serveLimit != "" {
			bps, err := parseLimit(serveLimit)
			if err != nil {
				return err
			}
			srv.SetLimit(bps)
		}
		lanIP := discovery.LANIP()
		// Ruling: block-art "QR" is unscannable theater, so M2 prints the
		// full URL instead. Scannable QR moves to M3 via a small lib.
		fmt.Printf(`
LANBox %s
Serving: %s
Local:   http://localhost:%d/?token=%s
Network: http://%s:%d/?token=%s
PIN:     %s
Press Ctrl+C to stop.
`, Version, root, serveCfg.Port, token, lanIP, serveCfg.Port, token, pin)
		if err := writePID(fmt.Sprintf("%s:%d", lanIP, serveCfg.Port), root); err != nil {
			log.Warn("cannot write PID file", "error", err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		err = srv.Run(ctx)
		removePID()
		return err
	},
}

func init() {
	defs := config.Defaults()
	serveCmd.Flags().StringVar(&serveCfg.Dir, "dir", defs.Dir, "directory to share")
	serveCmd.Flags().IntVar(&serveCfg.Port, "port", defs.Port, "port to listen on")
	serveCmd.Flags().StringVar(&serveCfg.Host, "host", defs.Host, "address to bind")
	serveCmd.Flags().StringVar(&serveCfg.WebDir, "web-dir", defs.WebDir, "directory with lanbox-web build output")
	serveCmd.Flags().StringVar(&serveLimit, "limit", "", "cap throughput, e.g. 20MB/s (0 = unlimited)")
}

// parseLimit accepts "20", "20M", "20MB/s" (megabytes/sec) into bytes/sec.
func parseLimit(s string) (int64, error) {
	s = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(s), "/s"), "B")
	s = strings.TrimSuffix(s, "M")
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid --limit %q (try 20MB/s)", s)
	}
	return int64(v * 1024 * 1024), nil
}
