package transfer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

// ServeFile streams a file to the response without loading it into RAM.
// It returns bytes sent and any copy error (e.g. client disconnect).
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
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	return io.Copy(w, f)
}
