package transfer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"sync"
	"time"
)

// T is one in-flight transfer. Mutable counters are mutex-guarded;
// List returns detached snapshots (TView) so readers never race writers.
type T struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Bytes   int64     `json:"bytes"`
	Total   int64     `json:"total"`
	Started time.Time `json:"started_at"`
	mu      sync.Mutex
	cancel  context.CancelFunc
	ctx     context.Context
}

// TView is a point-in-time copy of a T, safe to encode and share.
type TView struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Bytes   int64     `json:"bytes"`
	Total   int64     `json:"total"`
	Started time.Time `json:"started_at"`
}

// Registry tracks active transfers for listing and cancellation.
// The 4-slot semaphore still 429s excess; this only observes admitted ones.
type Registry struct {
	mu sync.Mutex
	m  map[string]*T
}

// NewRegistry makes an empty registry.
func NewRegistry() *Registry {
	return &Registry{m: map[string]*T{}}
}

// Start registers a transfer, returning its handle. Call Finish when done.
func (r *Registry) Start(parent context.Context, name string, total int64) *T {
	var b [8]byte
	_, _ = rand.Read(b[:])
	ctx, cancel := context.WithCancel(parent)
	t := &T{ID: hex.EncodeToString(b[:]), Name: name, Total: total, Started: time.Now(), cancel: cancel, ctx: ctx}
	r.mu.Lock()
	r.m[t.ID] = t
	r.mu.Unlock()
	return t
}

// Context returns the cancellable context for handler copies.
func (t *T) Context() context.Context {
	return t.ctx
}

// Add records n more bytes.
func (t *T) Add(n int64) {
	t.mu.Lock()
	t.Bytes += n
	t.mu.Unlock()
}

// SetMeta updates name/total once the real values are known (post-parse).
func (t *T) SetMeta(name string, total int64) {
	t.mu.Lock()
	t.Name = name
	t.Total = total
	t.mu.Unlock()
}

// Finish removes the transfer from the registry.
func (r *Registry) Finish(t *T) {
	r.mu.Lock()
	delete(r.m, t.ID)
	r.mu.Unlock()
}

// List snapshots active transfers.
func (r *Registry) List() []TView {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]TView, 0, len(r.m))
	for _, t := range r.m {
		t.mu.Lock()
		out = append(out, TView{ID: t.ID, Name: t.Name, Bytes: t.Bytes, Total: t.Total, Started: t.Started})
		t.mu.Unlock()
	}
	return out
}

// Cancel stops a transfer by id; false when unknown.
func (r *Registry) Cancel(id string) bool {
	r.mu.Lock()
	t, ok := r.m[id]
	r.mu.Unlock()
	if !ok {
		return false
	}
	t.cancel()
	return true
}

// CopyCtx copies like io.Copy but aborts promptly on ctx cancellation,
// reporting progress via onN. Downloads/uploads cancelled through
// Registry.Cancel stop at the next 32 KB chunk, not at client disconnect.
func CopyCtx(dst io.Writer, src io.Reader, ctx context.Context, onN func(int64)) (int64, error) {
	var total int64
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
		n, rerr := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return total, werr
			}
			total += int64(n)
			if onN != nil {
				onN(int64(n))
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				return total, nil
			}
			return total, rerr
		}
	}
}
