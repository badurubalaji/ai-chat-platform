package config

import (
	"os"
)

type Config struct {
	DatabaseURL       string
	Port              string
	AIEncryptionKey   string
	AdapterConfigPath string
}

func Load() (*Config, error) {
	// Simple env loader
	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://ai_user:ai_password@localhost:5432/ai_chat_db?sslmode=disable"),
		Port:            getEnv("PORT", "8080"),
		AIEncryptionKey:   getEnv("AI_ENCRYPTION_KEY", "00000000000000000000000000000000"), // Default for dev only
		AdapterConfigPath: getEnv("ADAPTER_CONFIG_PATH", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
