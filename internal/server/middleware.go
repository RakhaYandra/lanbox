package server

import (
	"net/http"
	"time"

	"github.com/RakhaYandra/lanbox/internal/auth"
)

// requireToken gates every /api/* request. The static web UI ("/")
// stays open so the login screen can load.
func (s *Server) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			next.ServeHTTP(w, r)
			return
		}
		got := auth.Bearer(r.URL.Query().Get("token"), r.Header.Get("Authorization"))
		if !s.token.Check(got) {
			s.log.Warn("unauthorized", "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "Unauthorized — wrong token")
			return
		}
		next.ServeHTTP(w, r)
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
