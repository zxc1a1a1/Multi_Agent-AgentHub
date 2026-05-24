package config

import (
	"reflect"
	"testing"
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
