package transfer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

// ServeFile streams a file to the response without loading it into RAM.
// It honors a single `Range: bytes=N-` request (206 + Content-Range);
// unsatisfiable ranges get 416. It returns bytes sent and any copy error
// (e.g. client disconnect).
func ServeFile(w http.ResponseWriter, r *http.Request, path string) (int64, error) {
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
		return io.Copy(w, f)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	return io.Copy(w, f)
}
