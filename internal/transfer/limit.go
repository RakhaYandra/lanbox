package transfer

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// Throttle wraps r so reads proceed at most bytesPerSec (0 = unlimited).
func Throttle(r io.Reader, bytesPerSec int64) io.Reader {
	if bytesPerSec <= 0 {
		return r
	}
	return &throttled{src: r, lim: rate.NewLimiter(rate.Limit(bytesPerSec), int(bytesPerSec))}
}

type throttled struct {
	src io.Reader
	lim *rate.Limiter
}

func (t *throttled) Read(b []byte) (int, error) {
	n, err := t.src.Read(b)
	if n > 0 {
		_ = t.lim.WaitN(context.Background(), n)
	}
	return n, err
}
