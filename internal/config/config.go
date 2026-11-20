// Package config reads the server configuration from environment variables.
package config

import (
	"errors"
	"os"
)

const defaultPort = "6200"

type Config struct {
	DatabaseURL string
	Port        string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL must be set")
	}
	if cfg.Port == "" {
		cfg.Port = defaultPort
	}

	return cfg, nil
}
