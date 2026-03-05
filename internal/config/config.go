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
	UploadDir        string
	QBOClientID      string
	QBOClientSecret  string
	QBORedirectURL   string
	QBOSandbox       bool
	MSSQLMigrationDSN string
	MigrationsDir     string
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
		UploadDir:       getEnv("UPLOAD_DIR", "./data/uploads"),
		QBOClientID:     getEnv("QBO_CLIENT_ID", ""),
		QBOClientSecret: getEnv("QBO_CLIENT_SECRET", ""),
		QBORedirectURL:  getEnv("QBO_REDIRECT_URL", "http://localhost:8080/integrations/qbo/callback"),
		QBOSandbox:      getEnv("QBO_SANDBOX", "true") == "true",
		MSSQLMigrationDSN: getEnv("MSSQL_MIGRATION_DSN", "sqlserver://sa:ATLinks2024!@localhost:1433?encrypt=disable"),
		MigrationsDir:     getEnv("MIGRATIONS_DIR", "./data/migrations"),
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
