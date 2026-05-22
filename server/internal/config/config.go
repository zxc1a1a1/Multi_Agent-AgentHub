package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	Agents      map[string]AgentConfig
}

type AgentConfig struct {
	Name string
	URL  string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("GATEWAY_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "root:agenthub123@tcp(localhost:3306)/agenthub?charset=utf8mb4&parseTime=True&loc=Local"),
		Agents: map[string]AgentConfig{
			"code-agent": {
				Name: "code-agent",
				URL:  getEnv("AGENT_CODE_URL", "http://localhost:8081"),
			},
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
