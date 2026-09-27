package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config holds all server settings.
// Precedence: CLI flags > LANBOX_* env > config file > defaults.
type Config struct {
	Dir        string
	Port       int
	Host       string
	WebDir     string
	WebOrigin  string
	PinEnabled bool
	LimitMbps  int
}

// ExpandDir resolves ~ and returns an absolute path.
func ExpandDir(dir string) (string, error) {
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, dir[2:])
	}
	return filepath.Abs(dir)
}

// Defaults returns the default configuration.
func Defaults() Config {
	return Config{
		Dir:        "~/LANBox",
		Port:       8080,
		Host:       "0.0.0.0",
		WebDir:     "",
		WebOrigin:  "*",
		PinEnabled: true,
		LimitMbps:  0,
	}
}
