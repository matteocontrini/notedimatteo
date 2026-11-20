// Package config reads the server configuration from environment variables.
package config

import (
	"errors"
	"os"
)

const (
	defaultPort    = "6200"
	defaultSiteURL = "https://notedimatteo.it"
)

type Config struct {
	DatabaseURL string
	Port        string
	// SiteURL is the public address, used for canonical links.
	SiteURL string
	// StoragePublicBaseURL is the public address of uploaded images.
	StoragePublicBaseURL string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		Port:                 os.Getenv("PORT"),
		SiteURL:              os.Getenv("SITE_URL"),
		StoragePublicBaseURL: os.Getenv("STORAGE_PUBLIC_BASE_URL"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL must be set")
	}
	if cfg.StoragePublicBaseURL == "" {
		return Config{}, errors.New("STORAGE_PUBLIC_BASE_URL must be set")
	}
	if cfg.Port == "" {
		cfg.Port = defaultPort
	}
	if cfg.SiteURL == "" {
		cfg.SiteURL = defaultSiteURL
	}

	return cfg, nil
}
