package cli

import (
	"net/http"
	"os"
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
