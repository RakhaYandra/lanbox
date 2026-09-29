package transfer

import (
	"context"
	"sync"
	"testing"
)

func TestRegistryConcurrent(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			h := r.Start(context.Background(), "f", 100)
			for j := 0; j < 50; j++ {
				h.Add(2)
				_ = r.List()
			}
			h.SetMeta("g", 100)
			r.Finish(h)
		}()
	}
	wg.Wait()
	if n := len(r.List()); n != 0 {
		t.Errorf("registry not empty: %d", n)
	}
}

func TestRegistryCancel(t *testing.T) {
	r := NewRegistry()
	h := r.Start(context.Background(), "f", 10)
	if !r.Cancel(h.ID) {
		t.Fatal("Cancel known id = false")
	}
	if r.Cancel("nope") {
		t.Error("Cancel unknown = true")
	}
	select {
	case <-h.Context().Done():
	default:
		t.Error("context not cancelled")
	}
	r.Finish(h)
}
