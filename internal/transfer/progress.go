package transfer

import (
	"fmt"
	"io"
)

// Progress tracks bytes copied for logging (percent needs total set).
type Progress struct {
	Total int64
	Done  int64
}

// Reader wraps src and counts bytes read into p.
func (p *Progress) Reader(src io.Reader) io.Reader {
	return &countReader{src: src, p: p}
}

type countReader struct {
	src io.Reader
	p   *Progress
}

func (c *countReader) Read(b []byte) (int, error) {
	n, err := c.src.Read(b)
	c.p.Done += int64(n)
	return n, err
}

// String renders "done / total (pct%)".
func (p *Progress) String() string {
	if p.Total <= 0 {
		return fmt.Sprintf("%d bytes", p.Done)
	}
	pct := p.Done * 100 / p.Total
	return fmt.Sprintf("%d / %d (%d%%)", p.Done, p.Total, pct)
}
