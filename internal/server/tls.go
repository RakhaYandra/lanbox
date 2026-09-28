package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"math/big"
	"net"
	"time"
)

// TLSCert holds a per-boot self-signed certificate (in memory, never disk).
type TLSCert struct {
	Cert   tls.Certificate
	Digest string // SHA-256 of raw DER, for TOFU verification
}

// NewTLSCert generates an ECDSA P-256 self-signed cert valid for 24h with
// SANs for the LAN IP and localhost. Restart rotates it, like the token.
func NewTLSCert(lanIP string) (*TLSCert, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "LANBox"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	if ip := net.ParseIP(lanIP); ip != nil {
		tmpl.IPAddresses = []net.IP{ip, net.ParseIP("127.0.0.1")}
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(der)
	return &TLSCert{
		Cert:   tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key},
		Digest: hex.EncodeToString(sum[:]),
	}, nil
}

// Fingerprint formats the digest as colon-separated octets (openssl style).
func (c *TLSCert) Fingerprint() string {
	out := ""
	for i := 0; i < len(c.Digest); i += 2 {
		if i > 0 {
			out += ":"
		}
		out += c.Digest[i : i+2]
	}
	return out
}
