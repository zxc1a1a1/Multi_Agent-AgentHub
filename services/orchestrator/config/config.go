package config

import (
	"fmt"
	"strings"
)

// Config defines minimal Orchestrator runtime configuration.
type Config struct {
	Addr string
}

// DefaultConfig returns safe defaults without reading environment variables.
func DefaultConfig() Config {
	return Config{
		Addr: ":8080",
	}
}

// Validate checks minimal config invariants.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("addr is required")
	}
	return nil
}
