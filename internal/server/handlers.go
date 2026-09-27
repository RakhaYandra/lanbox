package server

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/RakhaYandra/lanbox/internal/filesystem"
	"github.com/RakhaYandra/lanbox/internal/transfer"
)

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	path, err := filesystem.Resolve(s.root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	s.log.Info("download started", "path", r.URL.Query().Get("path"))
	if n, err := transfer.ServeFileLimit(w, r, path, s.limit); err != nil {
		s.log.Warn("download cancelled", "path", r.URL.Query().Get("path"), "sent", n)
	}
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	dir, err := filesystem.Resolve(s.root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		if r.Context().Err() != nil {
			s.log.Warn("upload cancelled", "error", r.Context().Err())
			writeError(w, http.StatusBadRequest, "Transfer cancelled")
			return
		}
		writeError(w, http.StatusBadRequest, "Missing file field")
		return
	}
	defer file.Close()
	name := filepath.Base(header.Filename)
	if name == "." || name == "/" || name == "" {
		writeError(w, http.StatusBadRequest, "Invalid filename")
		return
	}
	dst, err := filesystem.Resolve(dir, "/"+name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid filename")
		return
	}
	out, err := os.Create(dst)
	if err != nil {
		s.log.Error("disk write failed", "path", dst, "error", err)
		writeError(w, http.StatusInsufficientStorage, "Disk full")
		return
	}
	prog := &transfer.Progress{Total: header.Size}
	hasher := transfer.NewHasher()
	written, err := io.Copy(hasher.Writer(out), prog.Reader(file))
	_ = out.Close()
	if err != nil {
		_ = os.Remove(dst)
		s.log.Warn("upload cancelled", "path", dst, "error", err)
		writeError(w, http.StatusBadRequest, "Transfer cancelled")
		return
	}
	sum := hasher.Sum()
	s.log.Info("upload completed", "path", dst, "progress", prog.String(), "sha256", sum)
	writeJSON(w, http.StatusCreated, map[string]any{"name": name, "size": written, "checksum": "sha256:" + sum})
}

func (s *Server) handleChecksum(w http.ResponseWriter, r *http.Request) {
	path, err := filesystem.Resolve(s.root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	f, err := os.Open(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	defer f.Close()
	hasher := transfer.NewHasher()
	if _, err := io.Copy(io.Discard, hasher.Reader(f)); err != nil {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":   r.URL.Query().Get("path"),
		"sha256": hasher.Sum(),
	})
}
