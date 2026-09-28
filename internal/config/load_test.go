package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTOML(t *testing.T, body string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "lanbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{"LANBOX_DIR", "LANBOX_PORT", "LANBOX_HOST", "LANBOX_WEBDIR", "LANBOX_WEBORIGIN", "LANBOX_LIMIT", "LANBOX_PIN_ENABLED"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir()) // no config file
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 8080 || cfg.Host != "0.0.0.0" || !cfg.PinEnabled || cfg.LimitMbps != 0 {
		t.Errorf("defaults = %+v", cfg)
	}
}

func TestLoadFile(t *testing.T) {
	clearEnv(t)
	writeTOML(t, "# comment\nport = 9090\ndir = \"/data\"\npin_enabled = false\nlimit_mbps = 20\n")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9090 || cfg.Dir != "/data" || cfg.PinEnabled || cfg.LimitMbps != 20 {
		t.Errorf("file = %+v", cfg)
	}
}

func TestLoadEnvBeatsFile(t *testing.T) {
	clearEnv(t)
	writeTOML(t, "port = 9090\n")
	t.Setenv("LANBOX_PORT", "7070")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 7070 {
		t.Errorf("port = %d, want 7070 (env wins)", cfg.Port)
	}
}

func TestLoadBadFile(t *testing.T) {
	clearEnv(t)
	for _, body := range []string{
		"bogus = 1\n",
		"port = \"abc\"\n",
		"pin_enabled = maybe\n",
		"dir = /unquoted\n",
		"just words\n",
	} {
		writeTOML(t, body)
		if _, err := Load(); err == nil {
			t.Errorf("body %q: want error", body)
		}
	}
}

func TestLoadBadEnv(t *testing.T) {
	clearEnv(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LANBOX_PORT", "abc")
	if _, err := Load(); err == nil {
		t.Error("bad LANBOX_PORT: want error")
	}
}
