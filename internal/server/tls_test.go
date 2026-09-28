package server

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"testing"
	"time"
)

func TestNewTLSCert(t *testing.T) {
	c, err := NewTLSCert("192.168.1.10")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(c.Cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if leaf.NotAfter.Sub(leaf.NotBefore) > 25*time.Hour {
		t.Errorf("validity = %v, want ~24h", leaf.NotAfter.Sub(leaf.NotBefore))
	}
	found := false
	for _, ip := range leaf.IPAddresses {
		if ip.String() == "192.168.1.10" {
			found = true
		}
	}
	if !found {
		t.Errorf("SAN IPs = %v, want 192.168.1.10", leaf.IPAddresses)
	}
	sum := sha256.Sum256(c.Cert.Certificate[0])
	plain := hex.EncodeToString(sum[:])
	if len(c.Fingerprint()) != 3*len(plain)/2-1 {
		t.Errorf("fingerprint format = %q", c.Fingerprint())
	}
	stripped := ""
	for _, ch := range c.Fingerprint() {
		if ch != ':' {
			stripped += string(ch)
		}
	}
	if stripped != plain {
		t.Error("fingerprint digest mismatch")
	}
}
