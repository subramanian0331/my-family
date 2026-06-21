package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL        string
	GoogleClientID     string
	GoogleClientSecret string
	JWTSecret          string
	UploadDir          string
	FrontendURL        string
	Port               string
	SiteAdminEmail     string
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPassword       string
	SMTPFrom           string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		UploadDir:          getenv("UPLOAD_DIR", "./uploads"),
		FrontendURL:        getenv("FRONTEND_URL", "http://localhost"),
		Port:               getenv("PORT", "8080"),
		SiteAdminEmail:     os.Getenv("SITE_ADMIN_EMAIL"),
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           getenv("SMTP_PORT", "587"),
		SMTPUser:           os.Getenv("SMTP_USER"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:           os.Getenv("SMTP_FROM"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}