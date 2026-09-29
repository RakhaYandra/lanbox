package transfer

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// ServeFile streams a file to the response without loading it into RAM.
// It honors a single `Range: bytes=N-` request (206 + Content-Range);
// unsatisfiable ranges get 416. It returns bytes sent and any copy error
// (e.g. client disconnect).
func ServeFile(w http.ResponseWriter, r *http.Request, path string) (int64, error) {
	return ServeFileLimit(w, r, path, 0)
}

// ServeFileLimit streams with an optional bytes-per-second cap (0 = unlimited).
func ServeFileLimit(w http.ResponseWriter, r *http.Request, path string, bps int64) (int64, error) {
	return ServeFileCtx(w, r, path, bps, r.Context(), nil)
}

// ServeFileCtx streams like ServeFileLimit but aborts on ctx (registry
// cancel) and reports progress via onN. Range + throttle supported.
func ServeFileCtx(w http.ResponseWriter, r *http.Request, path string, bps int64, ctx context.Context, onN func(int64)) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return 0, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "Not found", http.StatusNotFound)
		return 0, fmt.Errorf("not a file")
	}
	size := info.Size()
	start := int64(0)
	if rng := r.Header.Get("Range"); rng != "" {
		var n int64
		if _, err := fmt.Sscanf(rng, "bytes=%d-", &n); err != nil || n < 0 || n >= size {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
			http.Error(w, "Range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
			return 0, fmt.Errorf("bad range")
		}
		start = n
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return 0, err
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, size-1, size))
		w.Header().Set("Content-Length", strconv.FormatInt(size-start, 10))
		w.WriteHeader(http.StatusPartialContent)
		return CopyCtx(w, Throttle(f, bps), ctx, onN)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	return CopyCtx(w, Throttle(f, bps), ctx, onN)
}

// MaxPreviewBytes caps inline previews; larger files must download.
const MaxPreviewBytes = 20 * 1024 * 1024

// ServePreview streams a file for inline browser display (images, text,
// PDF). Only safe types are inlined; anything else falls back to download.
// Files over the cap get 413.
func ServePreview(w http.ResponseWriter, r *http.Request, path string) error {
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "Not found", http.StatusNotFound)
		return fmt.Errorf("not a file")
	}
	if info.Size() > MaxPreviewBytes {
		http.Error(w, "Too large to preview", http.StatusRequestEntityTooLarge)
		return fmt.Errorf("too large")
	}
	ctype := mime.TypeByExtension(filepath.Ext(path))
	if ctype == "" {
		var head [512]byte
		n, _ := f.Read(head[:])
		ctype = http.DetectContentType(head[:n])
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return err
		}
	}
	if !previewable(ctype) {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(path)))
	} else {
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filepath.Base(path)))
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	_, err = io.Copy(w, f)
	return err
}

// previewable allowlists types safe to render inline.
func previewable(ctype string) bool {
	for _, p := range []string{"image/", "text/plain", "text/markdown", "application/pdf", "application/json"} {
		if len(ctype) >= len(p) && ctype[:len(p)] == p {
			return true
		}
	}
	return false
}
