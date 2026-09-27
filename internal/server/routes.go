package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/RakhaYandra/lanbox/internal/filesystem"
)

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/v1/info", s.handleInfo)
	s.mux.HandleFunc("GET /api/v1/files", s.handleList)
	s.mux.Handle("GET /api/v1/files/download", s.limitSlots(http.HandlerFunc(s.handleDownload)))
	s.mux.Handle("POST /api/v1/files/upload", s.limitSlots(http.HandlerFunc(s.handleUpload)))
	s.mux.HandleFunc("GET /api/v1/files/checksum", s.handleChecksum)
	s.mux.HandleFunc("POST /api/v1/shares", s.handleShareCreate)
	s.mux.Handle("GET /api/v1/shares/{token}", s.limitSlots(http.HandlerFunc(s.handleShareGet)))
	if s.webDir != "" {
		s.mux.Handle("/", http.FileServer(http.Dir(s.webDir)))
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     "LANBox",
		"hostname": hostname,
		"address":  s.host,
		"port":     s.port,
	})
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	dir, err := filesystem.Resolve(s.root, r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	entries, err := filesystem.Browse(dir)
	if err != nil {
		writeError(w, http.StatusNotFound, "Not found")
		return
	}
	if entries == nil {
		entries = []filesystem.Entry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":    r.URL.Query().Get("path"),
		"entries": entries,
	})
}
