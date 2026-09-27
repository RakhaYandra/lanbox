package transfer

import (
	"io"
	"net/http"
	"os"
	"strconv"
)

// ServeFile streams a file to the response without loading it into RAM.
// A cancelled request aborts the copy (writes fail, handler returns).
func ServeFile(w http.ResponseWriter, r *http.Request, path string) {
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	_, _ = io.Copy(w, f)
}
