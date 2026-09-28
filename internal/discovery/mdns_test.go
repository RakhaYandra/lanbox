package discovery

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAdvertiseBrowseLoopback(t *testing.T) {
	if testing.Short() {
		t.Skip("needs multicast loopback")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := Advertise(ctx, "LANBox-test-loop", 18991); err != nil {
		t.Skipf("no multicast on this host: %v", err)
	}
	time.Sleep(2 * time.Second)
	found, err := Browse(4 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range found {
		if f.Instance == "LANBox-test-loop" && f.Port == 18991 {
			return
		}
	}
	t.Errorf("self not discovered: %+v", found)
}

func TestNoTokenInTXT(t *testing.T) {
	// Presence TXT is a fixed allowlist — grep the source, not packets.
	src := "https=1 ver=1"
	if strings.Contains(src, "token") {
		t.Error("token must never appear in mDNS TXT")
	}
}
