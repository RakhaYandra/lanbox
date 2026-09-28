package cli

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/RakhaYandra/lanbox/internal/discovery"
)

var discoverTimeout string

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "List LANBox servers on the local network via mDNS",
	RunE: func(cmd *cobra.Command, args []string) error {
		dur, err := time.ParseDuration(discoverTimeout)
		if err != nil || dur <= 0 || dur > time.Minute {
			return fmt.Errorf("invalid --timeout %q (try 5s, max 1m)", discoverTimeout)
		}
		found, err := discovery.Browse(dur)
		if err != nil {
			return err
		}
		if len(found) == 0 {
			fmt.Println("No LANBox servers found (same subnet only — mDNS does not cross routers)")
			return nil
		}
		sort.Slice(found, func(i, j int) bool { return found[i].Instance < found[j].Instance })
		for _, f := range found {
			fmt.Printf("%s\n  https://%s:%d  (host %s)\n", f.Instance, f.IP, f.Port, f.Host)
		}
		return nil
	},
}

func init() {
	discoverCmd.Flags().StringVar(&discoverTimeout, "timeout", "5s", "how long to listen for servers")
	rootCmd.AddCommand(discoverCmd)
}
