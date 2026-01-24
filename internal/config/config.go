package config

import "os"

type BBConfig struct {
	OAuthURL     string
	ApiBaseURL   string
	ClientID     string
	ClientSecret string
	Scope        string
	GwAppKey     string
}

type Config struct {
	BB   BBConfig
	Port string
}

func Load() *Config {
	return &Config{
		BB: BBConfig{
			OAuthURL:     os.Getenv("BB_OAUTH_URL"),
			ApiBaseURL:   os.Getenv("BB_API_BASE_URL"),
			ClientID:     os.Getenv("BB_CLIENT_ID"),
			ClientSecret: os.Getenv("BB_CLIENT_SECRET"),
			Scope:        os.Getenv("BB_SCOPE"),
			GwAppKey:     os.Getenv("BB_GW_DEV_APP_KEY"),
		},
		Port: os.Getenv("SERVER_PORT"),
	}
}
