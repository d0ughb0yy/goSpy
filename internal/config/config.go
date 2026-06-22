package config

import "time"

// Config holds all configuration for a scan session.
type Config struct {
	Workers   int
	Width     int
	Height    int
	Timeout   time.Duration
	OutputDir string
	Profile   string
	Resume    bool
	Verbose   bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Workers:   4,
		Width:     1920,
		Height:    1080,
		Timeout:   30 * time.Second,
		OutputDir: "output",
		Profile:   "desktop",
	}
}
