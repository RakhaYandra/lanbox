package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/RakhaYandra/lanbox/internal/auth"
	"github.com/RakhaYandra/lanbox/internal/filesystem"
	"github.com/RakhaYandra/lanbox/internal/transfer"
)

// Share is an expiring link to one file. Server clock wins on expiry.
type Share struct {
	Token       string
	Path        string // resolved absolute path
	ExpiresAt   time.Time
	PinRequired bool
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

// Create validates path + expiry and returns a share.
func (st *ShareStore) Create(root, input string, expiresMin int, pinRequired bool, pin string) (Share, error) {
	if expiresMin < 1 || expiresMin > 1440 {
		return Share{}, fmt.Errorf("invalid expiry")
	}
	path, err := filesystem.Resolve(root, input)
	if err != nil {
		return Share{}, err
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
	sh, err := s.shares.Create(s.root, in.Path, in.ExpiresMin, in.PinRequired, pin)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	out := map[string]any{
		"share_token":  sh.Token,
		"url":          "http://" + r.Host + "/api/v1/shares/" + sh.Token,
		"expires_at":   sh.ExpiresAt.Unix(),
		"pin_required": sh.PinRequired,
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
	if info, err := os.Stat(sh.Path); err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "Share expired or not found")
		return
	}
	if _, err := transfer.ServeFileLimit(w, r, sh.Path, s.limit); err != nil {
		s.log.Warn("share download cancelled", "token", sh.Token)
	}
}
