package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	APIToken          string
	CORSAllowOrigins  []string
	Agents            map[string]AgentConfig
}

type AgentConfig struct {
	Name string
	URL  string
}

func Load() (*Config, error) {
	dbURL, err := loadDatabaseURL()
	if err != nil {
		return nil, err
	}

	return &Config{
		Port:              getEnv("GATEWAY_PORT", "8080"),
		DatabaseURL:       dbURL,
		DBMaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SECONDS", 300)) * time.Second,
		APIToken:          os.Getenv("AGENTHUB_API_TOKEN"),
		CORSAllowOrigins:  loadCORSAllowOrigins(),
		Agents: map[string]AgentConfig{
			"code-agent": {
				Name: "code-agent",
				URL:  getEnv("AGENT_CODE_URL", "http://localhost:8081"),
			},
		},
	}, nil
}

func loadDatabaseURL() (string, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}

	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "root")
	name := getEnv("DB_NAME", "agenthub")

	// Backward-compatible fallback for DB password variable name.
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = os.Getenv("MYSQL_PASSWORD")
	}
	if password == "" {
		return "", fmt.Errorf("missing database password: set DATABASE_URL or DB_PASSWORD (or MYSQL_PASSWORD)")
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user,
		password,
		host,
		port,
		name,
	), nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func loadCORSAllowOrigins() []string {
	const rawDefault = "http://localhost:3000,http://localhost:5173"
	raw := getEnv("CORS_ALLOW_ORIGINS", rawDefault)

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		origin := strings.TrimSpace(p)
		if origin == "" {
			continue
		}
		origins = append(origins, origin)
	}
	if len(origins) == 0 {
		return []string{"http://localhost:3000", "http://localhost:5173"}
	}
	return origins
}
