package server

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/RakhaYandra/lanbox/internal/auth"
)

// requireToken gates /api/* requests. Static web assets stay open so the
// app (including its login screen) can load; data stays behind auth.
// The public share download carries its own share-token auth.
func (s *Server) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/shares/") {
			next.ServeHTTP(w, r)
			return
		}
		got := auth.Bearer(r.URL.Query().Get("token"), r.Header.Get("Authorization"))
		if !s.token.Check(got) {
			s.log.Warn("unauthorized", "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "Unauthorized — wrong token")
			return
		}
		if s.pin != "" {
			pin := r.Header.Get("X-PIN")
			if pin == "" {
				pin = r.URL.Query().Get("pin")
			}
			if subtle.ConstantTimeCompare([]byte(pin), []byte(s.pin)) != 1 {
				s.log.Warn("wrong PIN", "path", r.URL.Path)
				writeError(w, http.StatusUnauthorized, "Unauthorized — wrong PIN")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// transferSlots bounds concurrent uploads/downloads (ADR-006).
// Excess requests get 429 + Retry-After instead of queuing unbounded.
func (s *Server) limitSlots(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case s.sem <- struct{}{}:
			defer func() { <-s.sem }()
			next.ServeHTTP(w, r)
		default:
			s.log.Warn("server busy", "path", r.URL.Path)
			w.Header().Set("Retry-After", "5")
			writeError(w, http.StatusTooManyRequests, "Server busy — try again")
		}
	})
}

// logging records method, path (query stripped — never log tokens), status,
// and duration.
func (s *Server) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}
