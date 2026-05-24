package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadCORSAllowOriginsUsesDefault(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", "")

	got := loadCORSAllowOrigins()
	want := []string{"http://localhost:3000", "http://localhost:5173"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default origins mismatch: got=%v want=%v", got, want)
	}
}

func TestLoadCORSAllowOriginsFromEnv(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", "http://localhost:3000, https://example.com ,http://localhost:5173")

	got := loadCORSAllowOrigins()
	want := []string{"http://localhost:3000", "https://example.com", "http://localhost:5173"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed origins mismatch: got=%v want=%v", got, want)
	}
}

func TestLoadCORSAllowOriginsSkipsEmptyParts(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", " ,  ,http://localhost:3000,,")

	got := loadCORSAllowOrigins()
	want := []string{"http://localhost:3000"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("origins with empties mismatch: got=%v want=%v", got, want)
	}
}

func TestGetEnvIntUsesFallbackOnMissingInvalidOrNonPositive(t *testing.T) {
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 20 {
		t.Fatalf("missing env should fallback to 20, got=%d", got)
	}

	t.Setenv("DB_MAX_OPEN_CONNS", "abc")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 20 {
		t.Fatalf("invalid env should fallback to 20, got=%d", got)
	}

	t.Setenv("DB_MAX_OPEN_CONNS", "0")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 20 {
		t.Fatalf("non-positive env should fallback to 20, got=%d", got)
	}

	t.Setenv("DB_MAX_OPEN_CONNS", "-1")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 20 {
		t.Fatalf("negative env should fallback to 20, got=%d", got)
	}

	t.Setenv("DB_MAX_OPEN_CONNS", "33")
	if got := getEnvInt("DB_MAX_OPEN_CONNS", 20); got != 33 {
		t.Fatalf("valid env should be used, got=%d", got)
	}
}

func TestLoadParsesDBPoolConfigWithFallbacks(t *testing.T) {
	t.Setenv("DATABASE_URL", "root:pw@tcp(localhost:3306)/agenthub?charset=utf8mb4&parseTime=True&loc=Local")
	t.Setenv("AGENTHUB_API_TOKEN", "test-token")
	t.Setenv("DB_MAX_OPEN_CONNS", "44")
	t.Setenv("DB_MAX_IDLE_CONNS", "xyz")
	t.Setenv("DB_CONN_MAX_LIFETIME_SECONDS", "600")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DBMaxOpenConns != 44 {
		t.Fatalf("expected DBMaxOpenConns=44, got=%d", cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns != 5 {
		t.Fatalf("invalid DB_MAX_IDLE_CONNS should fallback to 5, got=%d", cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetime != 600*time.Second {
		t.Fatalf("expected DBConnMaxLifetime=600s, got=%s", cfg.DBConnMaxLifetime)
	}
}

func TestLoadDBPoolDefaultsWhenEnvMissing(t *testing.T) {
	t.Setenv("DATABASE_URL", "root:pw@tcp(localhost:3306)/agenthub?charset=utf8mb4&parseTime=True&loc=Local")
	t.Setenv("AGENTHUB_API_TOKEN", "test-token")
	t.Setenv("DB_MAX_OPEN_CONNS", "")
	t.Setenv("DB_MAX_IDLE_CONNS", "")
	t.Setenv("DB_CONN_MAX_LIFETIME_SECONDS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DBMaxOpenConns != 20 {
		t.Fatalf("expected default DBMaxOpenConns=20, got=%d", cfg.DBMaxOpenConns)
	}
	if cfg.DBMaxIdleConns != 5 {
		t.Fatalf("expected default DBMaxIdleConns=5, got=%d", cfg.DBMaxIdleConns)
	}
	if cfg.DBConnMaxLifetime != 300*time.Second {
		t.Fatalf("expected default DBConnMaxLifetime=300s, got=%s", cfg.DBConnMaxLifetime)
	}
}
