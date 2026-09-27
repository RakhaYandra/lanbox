package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
)

// Hasher wraps a writer with a running SHA-256 (single pass, no re-read).
type Hasher struct {
	h hash.Hash
}

// NewHasher starts a checksum accumulator.
func NewHasher() *Hasher {
	return &Hasher{h: sha256.New()}
}

// Writer returns a writer that hashes everything written through it.
func (h *Hasher) Writer(dst io.Writer) io.Writer {
	return io.MultiWriter(dst, h.h)
}

// Reader returns a reader that hashes everything read through it.
func (h *Hasher) Reader(src io.Reader) io.Reader {
	return io.TeeReader(src, h.h)
}

// Sum returns the hex digest of everything seen so far.
func (h *Hasher) Sum() string {
	return hex.EncodeToString(h.h.Sum(nil))
}
