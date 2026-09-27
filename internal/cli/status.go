package cli

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show running server status from the PID file",
	RunE: func(cmd *cobra.Command, args []string) error {
		pf, err := readPID()
		if err != nil {
			return fmt.Errorf("not running (no PID file)")
		}
		if !alive(pf.PID) {
			return fmt.Errorf("not running (stale PID %d, remove %s or start serve)", pf.PID, mustPidPath())
		}
		fmt.Printf("Status: running\nPID: %d\nDirectory: %s\nAddress: %s\n", pf.PID, pf.Dir, pf.Address)
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running server via the PID file",
	RunE: func(cmd *cobra.Command, args []string) error {
		pf, err := readPID()
		if err != nil {
			return fmt.Errorf("not running (no PID file)")
		}
		proc, err := os.FindProcess(pf.PID)
		if err != nil || !alive(pf.PID) {
			removePID()
			return fmt.Errorf("not running (stale PID %d cleaned up)", pf.PID)
		}
		if err := proc.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("cannot stop PID %d: %w", pf.PID, err)
		}
		fmt.Printf("Stopped PID %d\n", pf.PID)
		return nil
	},
}

func mustPidPath() string {
	p, err := pidPath()
	if err != nil {
		return "lanbox.pid"
	}
	return p
}

func init() {
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(stopCmd)
}
