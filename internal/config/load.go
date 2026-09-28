package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ConfigPath returns ~/.config/lanbox/config.toml.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lanbox", "config.toml"), nil
}

// Load merges defaults, config file, and LANBOX_* env.
// Precedence: file over defaults, env over file. CLI flags override the
// result in serve.go via cobra Changed checks.
func Load() (Config, error) {
	cfg := Defaults()
	path, err := ConfigPath()
	if err != nil {
		return cfg, err
	}
	if data, err := os.ReadFile(path); err == nil {
		if err := parseTOML(data, &cfg); err != nil {
			return cfg, fmt.Errorf("bad config %s: %w", path, err)
		}
	}
	if err := applyEnv(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// parseTOML handles a flat subset: key = value with string/int/bool,
// # comments, one pair per line. Unknown keys and bad types are errors.
func parseTOML(data []byte, cfg *Config) error {
	setString := map[string]*string{
		"dir": &cfg.Dir, "host": &cfg.Host, "web_dir": &cfg.WebDir,
		"web_origin": &cfg.WebOrigin,
	}
	setInt := map[string]*int{"port": &cfg.Port, "limit_mbps": &cfg.LimitMbps}
	setBool := map[string]*bool{"pin_enabled": &cfg.PinEnabled}
	for i, line := range strings.Split(string(data), "\n") {
		line = stripComment(strings.TrimSpace(line))
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("line %d: want key = value", i+1)
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		switch {
		case setString[k] != nil:
			s, err := unquote(v)
			if err != nil {
				return fmt.Errorf("line %d: %w", i+1, err)
			}
			*setString[k] = s
		case setInt[k] != nil:
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("line %d: %q not an int", i+1, v)
			}
			*setInt[k] = n
		case setBool[k] != nil:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return fmt.Errorf("line %d: %q not a bool", i+1, v)
			}
			*setBool[k] = b
		default:
			return fmt.Errorf("line %d: unknown key %q", i+1, k)
		}
	}
	return nil
}

// stripComment cuts # comments outside double quotes.
func stripComment(line string) string {
	inQuotes := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '"':
			inQuotes = !inQuotes
		case '#':
			if !inQuotes {
				return strings.TrimSpace(line[:i])
			}
		}
	}
	return line
}

func unquote(v string) (string, error) {
	if len(v) >= 2 && strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
		return v[1 : len(v)-1], nil
	}
	return "", fmt.Errorf("%q must be quoted", v)
}

// applyEnv overlays LANBOX_* variables onto cfg.
func applyEnv(cfg *Config) error {
	if v := os.Getenv("LANBOX_DIR"); v != "" {
		cfg.Dir = v
	}
	if v := os.Getenv("LANBOX_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("LANBOX_WEBDIR"); v != "" {
		cfg.WebDir = v
	}
	if v := os.Getenv("LANBOX_WEBORIGIN"); v != "" {
		cfg.WebOrigin = v
	}
	if v := os.Getenv("LANBOX_PORT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid LANBOX_PORT %q", v)
		}
		cfg.Port = n
	}
	if v := os.Getenv("LANBOX_LIMIT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid LANBOX_LIMIT %q (Mbps int)", v)
		}
		cfg.LimitMbps = n
	}
	if v := os.Getenv("LANBOX_PIN_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid LANBOX_PIN_ENABLED %q", v)
		}
		cfg.PinEnabled = b
	}
	return nil
}
