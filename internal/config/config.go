package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	Port          string
	InviteCode    string
	ResendAPIKey  string
	AppBaseURL    string
	FromEmail     string
	APIKey        string
	UploadDir     string
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://atlinks:atlinks_dev@localhost:5432/atlinks?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		Port:         getEnv("PORT", "8080"),
		InviteCode:   getEnv("INVITE_CODE", "BETA2024"),
		ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		AppBaseURL:   getEnv("APP_BASE_URL", "http://localhost:8080"),
		FromEmail:    getEnv("FROM_EMAIL", "noreply@atlinks.app"),
		APIKey:       getEnv("API_KEY", ""),
		UploadDir:    getEnv("UPLOAD_DIR", "./data/uploads"),
	}

	if cfg.JWTSecret == "dev-secret-change-in-production" {
		fmt.Fprintln(os.Stderr, "WARNING: using default JWT secret — set JWT_SECRET in production")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
