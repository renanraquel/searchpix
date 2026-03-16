package config

import (
	"os"
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

type Config struct {
	BB   BBConfig
	DB   DBConfig
	Port string
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
		Port: os.Getenv("SERVER_PORT"),
	}
}
