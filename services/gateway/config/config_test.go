package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	t.Setenv("AGENTHUB_API_TOKEN", "test-token")

	cfg := DefaultConfig()
	if cfg.Addr != ":8080" {
		t.Fatalf("expected default addr :8080, got %q", cfg.Addr)
	}
	if cfg.EnableAuth {
		t.Fatalf("expected auth disabled by default")
	}
	if cfg.AuthToken != "" {
		t.Fatalf("expected default auth token empty")
	}
	if len(cfg.AllowedOrigins) != 0 {
		t.Fatalf("expected no default allowed origins")
	}
}

func TestValidateCleanConfig(t *testing.T) {
	cfg := Config{
		Addr:       ":18080",
		EnableAuth: false,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidateEnableAuthWithoutToken(t *testing.T) {
	cfg := Config{
		Addr:       ":18080",
		EnableAuth: true,
		AuthToken:  "   ",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error when auth enabled but token is empty")
	}
}
