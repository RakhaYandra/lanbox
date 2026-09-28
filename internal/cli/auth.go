package cli

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// pinFor resolves --pin flag over LANBOX_PIN env.
func pinFor(flag string) string {
	if flag != "" {
		return flag
	}
	return os.Getenv("LANBOX_PIN")
}

// setAuth sets Bearer token and X-PIN headers when present.
func setAuth(req *http.Request, token, pin string) {
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if pin != "" {
		req.Header.Set("X-PIN", pin)
	}
}

// apiURL builds an https URL for host:port + path. The server uses a
// per-boot self-signed cert, so clients skip chain verification; the real
// auth is token + PIN. Never downgrade to http.
func apiURL(host, apiPath string) string {
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	return fmt.Sprintf("https://%s%s", host, apiPath)
}

// tlsClient skips verification of the self-signed cert (see apiURL).
var tlsClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	},
}
