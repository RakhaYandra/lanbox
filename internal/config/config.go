package config

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
