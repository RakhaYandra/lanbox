package server

import (
	"context"
	"sync/atomic"
	"time"
)

// SetDrop restricts the server to one file and auto-shutdown after count
// completed downloads. Empty file disables drop mode.
func (s *Server) SetDrop(file string, count int) {
	s.dropFile = file
	s.dropLeft = int32(count)
}

// countDrop decrements the drop counter on each completed download and
// initiates shutdown at zero. Failed transfers never count.
func (s *Server) countDrop() {
	if s.dropFile == "" {
		return
	}
	if atomic.AddInt32(&s.dropLeft, -1) <= 0 && s.shutdown != nil {
		s.log.Info("drop complete, shutting down")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.shutdown(ctx)
		}()
	}
}
