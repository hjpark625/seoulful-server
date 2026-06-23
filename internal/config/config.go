package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	GinMode     string
	CORSOrigin  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		GinMode:     os.Getenv("GIN_MODE"),
		CORSOrigin:  os.Getenv("CORS_ORIGIN"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.GinMode == "" {
		cfg.GinMode = "debug"
	}
	if cfg.CORSOrigin == "" {
		cfg.CORSOrigin = "*"
	}

	return cfg, nil
}
