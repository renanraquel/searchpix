package config

import (
	"os"
	"strings"
)

type BBConfig struct {
	OAuthURL     string
	ApiBaseURL   string
	ClientID     string
	ClientSecret string
	Scope        string
	GwAppKey     string
}

// DBConfig configuração do banco (fidelização)
type DBConfig struct {
	Driver string // "postgres" ou "sqlite3"
	URL    string // DSN completo ou "file::memory:?cache=shared" para SQLite em memória
}

// R2Config Cloudflare R2 (carrossel / mídia pública).
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicBaseURL   string
	Endpoint        string
}

type Config struct {
	BB    BBConfig
	DB    DBConfig
	Email EmailConfig
	R2    R2Config
	Port  string
}

type EmailConfig struct {
	Provider       string
	SMTPHost       string
	SMTPPort       string
	SMTPUser       string
	SMTPPassword   string
	From           string
	PublicUIOrigin string
}

func Load() *Config {
	dbURL := os.Getenv("DATABASE_URL")
	dbDriver := os.Getenv("DATABASE_DRIVER")
	if dbDriver == "" {
		if dbURL != "" {
			dbDriver = "postgres"
		} else {
			// Local/dev: SQLite em memória
			dbDriver = "sqlite3"
			dbURL = "file::memory:?cache=shared"
		}
	}
	if dbDriver == "sqlite3" && dbURL == "" {
		dbURL = "file::memory:?cache=shared"
	}
	emailProvider := strings.ToLower(strings.TrimSpace(os.Getenv("SMTP_PROVIDER")))
	smtpHost := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	smtpPort := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	smtpUser := strings.TrimSpace(os.Getenv("SMTP_USER"))
	emailFrom := strings.TrimSpace(os.Getenv("EMAIL_FROM"))
	if emailProvider == "gmail" {
		if smtpHost == "" {
			smtpHost = "smtp.gmail.com"
		}
		if smtpPort == "" {
			smtpPort = "587"
		}
		if emailFrom == "" {
			emailFrom = smtpUser
		}
	}

	return &Config{
		BB: BBConfig{
			OAuthURL:     os.Getenv("BB_OAUTH_URL"),
			ApiBaseURL:   os.Getenv("BB_API_BASE_URL"),
			ClientID:     os.Getenv("BB_CLIENT_ID"),
			ClientSecret: os.Getenv("BB_CLIENT_SECRET"),
			Scope:        os.Getenv("BB_SCOPE"),
			GwAppKey:     os.Getenv("BB_GW_DEV_APP_KEY"),
		},
		DB: DBConfig{
			Driver: dbDriver,
			URL:    dbURL,
		},
		Email: EmailConfig{
			Provider:       emailProvider,
			SMTPHost:       smtpHost,
			SMTPPort:       smtpPort,
			SMTPUser:       smtpUser,
			SMTPPassword:   os.Getenv("SMTP_PASSWORD"),
			From:           emailFrom,
			PublicUIOrigin: strings.TrimSpace(os.Getenv("PUBLIC_UI_ORIGIN")),
		},
		R2: R2Config{
			AccountID:       strings.TrimSpace(os.Getenv("R2_ACCOUNT_ID")),
			AccessKeyID:     strings.TrimSpace(os.Getenv("R2_ACCESS_KEY_ID")),
			SecretAccessKey: strings.TrimSpace(os.Getenv("R2_SECRET_ACCESS_KEY")),
			Bucket:          strings.TrimSpace(os.Getenv("R2_BUCKET")),
			PublicBaseURL:   strings.TrimSpace(os.Getenv("R2_PUBLIC_BASE_URL")),
			Endpoint:        strings.TrimSpace(os.Getenv("R2_ENDPOINT")),
		},
		Port: os.Getenv("SERVER_PORT"),
	}
}
