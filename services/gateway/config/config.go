package config

import (
	"fmt"
	"strings"
)

// Config defines minimal Gateway runtime configuration.
type Config struct {
	Addr           string
	AllowedOrigins []string
	AuthToken      string
	EnableAuth     bool
}

// DefaultConfig returns safe defaults without reading environment variables.
func DefaultConfig() Config {
	return Config{
		Addr:           ":8080",
		AllowedOrigins: nil,
		AuthToken:      "",
		EnableAuth:     false,
	}
}

// Validate checks minimal config invariants.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("addr is required")
	}
	if c.EnableAuth && strings.TrimSpace(c.AuthToken) == "" {
		return fmt.Errorf("auth token is required when auth is enabled")
	}
	return nil
}
