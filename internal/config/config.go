package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	SupabaseURL string
	SupabaseKey string
	Port        string
	GinMode     string
	CORSOrigin  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		SupabaseURL: os.Getenv("SUPABASE_URL"),
		SupabaseKey: os.Getenv("SUPABASE_KEY"),
		Port:        os.Getenv("PORT"),
		GinMode:     os.Getenv("GIN_MODE"),
		CORSOrigin:  os.Getenv("CORS_ORIGIN"),
	}

	if cfg.SupabaseURL == "" || cfg.SupabaseKey == "" {
		return nil, fmt.Errorf("SUPABASE_URL and SUPABASE_KEY are required")
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
