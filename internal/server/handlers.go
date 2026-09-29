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
	if s.dropFile != "" && path != s.dropFile {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	s.log.Info("download started", "path", r.URL.Query().Get("path"))
	if r.URL.Query().Get("preview") == "1" {
		if err := transfer.ServePreview(w, r, path); err != nil {
			s.log.Warn("preview failed", "path", r.URL.Query().Get("path"), "error", err)
		}
		return
	}
	t := s.reg.Start(r.Context(), filepath.Base(path), fileSize(path))
	n, err := transfer.ServeFileCtx(w, r, path, s.limit, t.Context(), t.Add)
	s.reg.Finish(t)
	if err != nil {
		s.log.Warn("download cancelled", "path", r.URL.Query().Get("path"), "sent", n)
	} else {
		s.countDrop()
	}
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if s.dropFile != "" {
		writeError(w, http.StatusForbidden, "Drop mode serves one file")
		return
	}
	dir, err := filesystem.Resolve(s.root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	// Register before FormFile: parsing buffers the whole body, so an entry
	// created later would never be visible mid-receipt.
	t := s.reg.Start(r.Context(), "(receiving)", r.ContentLength)
	defer s.reg.Finish(t)
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
	t.SetMeta(name, header.Size)
	written, err := transfer.CopyCtx(hasher.Writer(out), prog.Reader(transfer.Throttle(file, s.limit)), t.Context(), t.Add)
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

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	path, err := filesystem.Resolve(s.root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	if err := os.Remove(path); err != nil {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	s.log.Info("deleted", "path", r.URL.Query().Get("path"))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": r.URL.Query().Get("path")})
}

// fileSize returns the file size or 0 when unknown.
func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0
	}
	return info.Size()
}

func (s *Server) handleTransfers(w http.ResponseWriter, r *http.Request) {
	items := s.reg.List()
	if items == nil {
		items = []transfer.TView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"transfers": items})
}

func (s *Server) handleTransferCancel(w http.ResponseWriter, r *http.Request) {
	if !s.reg.Cancel(r.PathValue("id")) {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	s.log.Info("transfer cancelled by user", "id", r.PathValue("id"))
	writeJSON(w, http.StatusOK, map[string]any{"cancelled": r.PathValue("id")})
}
