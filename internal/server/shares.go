package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/RakhaYandra/lanbox/internal/auth"
	"github.com/RakhaYandra/lanbox/internal/filesystem"
	"github.com/RakhaYandra/lanbox/internal/transfer"
)

// Share is an expiring link to one file, or (AllowUpload) an expiring
// inbox directory. Server clock wins on expiry.
type Share struct {
	Token       string
	Path        string // resolved absolute path
	ExpiresAt   time.Time
	PinRequired bool
	AllowUpload bool
	pin         string
}

// ShareStore is an in-memory map. Shares die with the process (ADR-003).
type ShareStore struct {
	mu sync.Mutex
	m  map[string]Share
}

// NewShareStore starts the store plus a 1-minute expiry sweep.
func NewShareStore() *ShareStore {
	st := &ShareStore{m: map[string]Share{}}
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		for range t.C {
			st.Sweep()
		}
	}()
	return st
}

// Create validates path + expiry and returns a share. allowUpload requires
// path to resolve to a directory (the inbox); otherwise a file.
func (st *ShareStore) Create(root, input string, expiresMin int, pinRequired bool, pin string, allowUpload bool) (Share, error) {
	if expiresMin < 1 || expiresMin > 1440 {
		return Share{}, fmt.Errorf("invalid expiry")
	}
	path, err := filesystem.Resolve(root, input)
	if err != nil {
		return Share{}, err
	}
	if allowUpload {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return Share{}, fmt.Errorf("upload share needs a directory")
		}
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return Share{}, err
	}
	sh := Share{
		Token:       hex.EncodeToString(b[:]),
		Path:        path,
		ExpiresAt:   time.Now().Add(time.Duration(expiresMin) * time.Minute),
		PinRequired: pinRequired,
		AllowUpload: allowUpload,
		pin:         pin,
	}
	st.mu.Lock()
	st.m[sh.Token] = sh
	st.mu.Unlock()
	return sh, nil
}

// Get returns a share or false when missing/expired (same 404, no oracle).
func (st *ShareStore) Get(token string) (Share, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	sh, ok := st.m[token]
	if !ok || time.Now().After(sh.ExpiresAt) {
		delete(st.m, token)
		return Share{}, false
	}
	return sh, true
}

// Sweep drops expired shares.
func (st *ShareStore) Sweep() {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	for k, sh := range st.m {
		if now.After(sh.ExpiresAt) {
			delete(st.m, k)
		}
	}
}

func (s *Server) handleShareCreate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Path        string `json:"path"`
		ExpiresMin  int    `json:"expires_minutes"`
		PinRequired bool   `json:"pin_required"`
		AllowUpload bool   `json:"allow_upload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	if in.ExpiresMin == 0 {
		in.ExpiresMin = 30
	}
	pin := ""
	if in.PinRequired {
		var err error
		pin, err = auth.NewPIN()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Cannot create share")
			return
		}
	}
	sh, err := s.shares.Create(s.root, in.Path, in.ExpiresMin, in.PinRequired, pin, in.AllowUpload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	out := map[string]any{
		"share_token":  sh.Token,
		"url":          "https://" + r.Host + "/api/v1/shares/" + sh.Token,
		"expires_at":   sh.ExpiresAt.Unix(),
		"pin_required": sh.PinRequired,
		"allow_upload": sh.AllowUpload,
	}
	if pin != "" {
		out["pin"] = pin
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) handleShareGet(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.shares.Get(r.PathValue("token"))
	if !ok {
		writeError(w, http.StatusNotFound, "Share expired or not found")
		return
	}
	if sh.PinRequired && r.Header.Get("X-Share-PIN") != sh.pin {
		writeError(w, http.StatusUnauthorized, "Wrong share PIN")
		return
	}
	if sh.AllowUpload {
		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, uploadFormHTML, r.URL.Path)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"upload_to": r.URL.Path + "/files"})
		return
	}
	if info, err := os.Stat(sh.Path); err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "Share expired or not found")
		return
	}
	if _, err := transfer.ServeFileLimit(w, r, sh.Path, s.limit); err != nil {
		s.log.Warn("share download cancelled", "token", sh.Token)
	}
}

// uploadFormHTML is the no-build public upload page for upload shares.
const uploadFormHTML = `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>LANBox upload</title></head>
<body><h1>Send files to LANBox</h1>
<form method="post" action="%s/files" enctype="multipart/form-data">
<input type="file" name="file" multiple><button>Upload</button>
</form></body></html>`

func (s *Server) handleShareUpload(w http.ResponseWriter, r *http.Request) {
	sh, ok := s.shares.Get(r.PathValue("token"))
	if !ok || !sh.AllowUpload {
		writeError(w, http.StatusNotFound, "Share expired or not found")
		return
	}
	if sh.PinRequired && r.Header.Get("X-Share-PIN") != sh.pin {
		writeError(w, http.StatusUnauthorized, "Wrong share PIN")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Missing file field")
		return
	}
	defer file.Close()
	name := filepath.Base(header.Filename)
	if name == "." || name == "/" || name == "" {
		writeError(w, http.StatusBadRequest, "Invalid filename")
		return
	}
	dst, err := filesystem.Resolve(sh.Path, "/"+name)
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
	hasher := transfer.NewHasher()
	if _, err := io.Copy(hasher.Writer(out), file); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		writeError(w, http.StatusBadRequest, "Transfer cancelled")
		return
	}
	_ = out.Close()
	s.log.Info("share upload completed", "path", dst)
	writeJSON(w, http.StatusCreated, map[string]any{"name": name, "checksum": "sha256:" + hasher.Sum()})
}
