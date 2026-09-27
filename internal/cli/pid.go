package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"
)

// PID file identity per IMPLEMENTATION §3.
type pidFile struct {
	PID     int    `json:"pid"`
	Address string `json:"address"`
	Dir     string `json:"dir"`
}

func pidPath() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".local", "share", "lanbox", "lanbox.pid"), nil
}

func writePID(address, dir string) error {
	p, err := pidPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, _ := json.Marshal(pidFile{PID: os.Getpid(), Address: address, Dir: dir})
	return os.WriteFile(p, data, 0o644)
}

func removePID() {
	if p, err := pidPath(); err == nil {
		_ = os.Remove(p)
	}
}

func readPID() (pidFile, error) {
	var pf pidFile
	p, err := pidPath()
	if err != nil {
		return pf, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return pf, err
	}
	if err := json.Unmarshal(data, &pf); err != nil {
		return pf, err
	}
	return pf, nil
}

// alive reports whether the PID is a running process.
func alive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
