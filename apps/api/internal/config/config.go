package config

import "os"

// Config contains runtime settings for the API server.
type Config struct {
	Host      string
	Port      string
	WebOrigin string
}

// Load reads configuration from environment variables.
func Load() Config {
	return Config{
		Host:      getEnv("API_HOST", "0.0.0.0"),
		Port:      getEnv("API_PORT", "8080"),
		WebOrigin: getEnv("WEB_ORIGIN", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
