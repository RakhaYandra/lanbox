package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/RakhaYandra/lanbox/internal/auth"
)

// Server wires routes to handlers.
type Server struct {
	root   string
	host   string
	port   int
	webDir string
	token  auth.Token
	pin    string // empty = PIN check disabled (--pin off)
	log    *slog.Logger
	mux    *http.ServeMux
	sem    chan struct{}
	shares *ShareStore
	limit  int64 // bytes/sec, 0 = unlimited
}

// New builds the server without starting it.
func New(root, host string, port int, webDir string, token auth.Token, log *slog.Logger) *Server {
	s := &Server{root: root, host: host, port: port, webDir: webDir, token: token, log: log, mux: http.NewServeMux(), sem: make(chan struct{}, 4), shares: NewShareStore()}
	s.routes()
	return s
}

// SetLimit caps transfer throughput in bytes/sec (0 = unlimited).
func (s *Server) SetLimit(bps int64) {
	s.limit = bps
}

// SetPIN enables the PIN check (empty disables it).
func (s *Server) SetPIN(pin string) {
	s.pin = pin
}

// Handler exposes the mux with logging + auth middleware.
func (s *Server) Handler() http.Handler {
	return s.logging(s.requireToken(s.mux))
}

// Addr returns host:port.
func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

// Run serves until ctx is cancelled, then drains up to 10s.
func (s *Server) Run(ctx context.Context) error {
	httpSrv := &http.Server{Addr: s.Addr(), Handler: s.Handler()}
	go func() {
		<-ctx.Done()
		drain, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(drain)
	}()
	s.log.Info("server started", "addr", s.Addr(), "dir", s.root)
	err := httpSrv.ListenAndServe()
	if err == http.ErrServerClosed {
		s.log.Info("server stopped")
		return nil
	}
	return err
}
