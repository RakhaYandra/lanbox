package discovery

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/grandcat/zeroconf"
)

// Service is the mDNS service type LANBox advertises and browses.
const Service = "_lanbox._tcp"

// Domain is the local mDNS domain.
const Domain = "local."

// InstanceName returns the mDNS instance name. Port is included so two
// serves on one host never fight over the same name (a lost probing
// conflict renders the loser invisible).
func InstanceName(port int) string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "lanbox"
	}
	return fmt.Sprintf("LANBox-%s-%d", host, port)
}

// Advertise registers instance on port until ctx ends, then unpublishes.
// TXT carries presence only (https hint + version) — never the token.
func Advertise(ctx context.Context, instance string, port int) error {
	text := []string{"https=1", "ver=1"}
	srv, err := zeroconf.Register(instance, Service, Domain, port, text, nil)
	if err != nil {
		return fmt.Errorf("mDNS advertise: %w", err)
	}
	go func() {
		<-ctx.Done()
		srv.Shutdown()
	}()
	return nil
}

// Found is one discovered server.
type Found struct {
	Instance string
	Host     string
	IP       string
	Port     int
}

// Browse lists servers seen within timeout.
func Browse(timeout time.Duration) ([]Found, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("mDNS browse: %w", err)
	}
	entries := make(chan *zeroconf.ServiceEntry, 16)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := resolver.Browse(ctx, Service, Domain, entries); err != nil {
		return nil, fmt.Errorf("mDNS browse: %w", err)
	}
	seen := map[string]Found{}
	for {
		select {
		case e := <-entries:
			if e == nil {
				continue
			}
			ip := ""
			if len(e.AddrIPv4) > 0 {
				ip = e.AddrIPv4[0].String()
			}
			// Key by instance+port: same hostname may serve twice.
			key := fmt.Sprintf("%s|%d", e.Instance, e.Port)
			seen[key] = Found{
				Instance: e.Instance,
				Host:     e.HostName,
				IP:       ip,
				Port:     e.Port,
			}
		case <-ctx.Done():
			out := make([]Found, 0, len(seen))
			for _, f := range seen {
				out = append(out, f)
			}
			return out, nil
		}
	}
}
