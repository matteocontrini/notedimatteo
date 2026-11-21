// Package config reads the server configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/matteocontrini/notedimatteo/internal/storage"
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
	// AdminUsername and AdminPassword are the only login credentials.
	AdminUsername string
	AdminPassword string
	// S3 is the bucket for uploaded images, or nil if the STORAGE_S3_*
	// variables aren't set, which turns uploads off.
	S3 *storage.Config
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		Port:                 os.Getenv("PORT"),
		SiteURL:              os.Getenv("SITE_URL"),
		StoragePublicBaseURL: os.Getenv("STORAGE_PUBLIC_BASE_URL"),
		AdminUsername:        os.Getenv("ADMIN_USERNAME"),
		AdminPassword:        os.Getenv("ADMIN_PASSWORD"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL must be set")
	}
	if cfg.StoragePublicBaseURL == "" {
		return Config{}, errors.New("STORAGE_PUBLIC_BASE_URL must be set")
	}
	if cfg.AdminUsername == "" || cfg.AdminPassword == "" {
		return Config{}, errors.New("ADMIN_USERNAME and ADMIN_PASSWORD must be set")
	}
	s3, err := loadS3()
	if err != nil {
		return Config{}, err
	}
	cfg.S3 = s3

	if cfg.Port == "" {
		cfg.Port = defaultPort
	}
	if cfg.SiteURL == "" {
		cfg.SiteURL = defaultSiteURL
	}

	return cfg, nil
}

// loadS3 reads the STORAGE_S3_* variables, which must be all set or all unset.
func loadS3() (*storage.Config, error) {
	cfg := storage.Config{
		Endpoint:        os.Getenv("STORAGE_S3_ENDPOINT"),
		Region:          os.Getenv("STORAGE_S3_REGION"),
		AccessKeyID:     os.Getenv("STORAGE_S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("STORAGE_S3_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("STORAGE_S3_BUCKET"),
	}
	values := map[string]string{
		"STORAGE_S3_ENDPOINT":          cfg.Endpoint,
		"STORAGE_S3_REGION":            cfg.Region,
		"STORAGE_S3_ACCESS_KEY_ID":     cfg.AccessKeyID,
		"STORAGE_S3_SECRET_ACCESS_KEY": cfg.SecretAccessKey,
		"STORAGE_S3_BUCKET":            cfg.Bucket,
	}

	var missing []string
	for name, value := range values {
		if value == "" {
			missing = append(missing, name)
		}
	}
	switch len(missing) {
	case 0:
		return &cfg, nil
	case len(values):
		return nil, nil
	default:
		slices.Sort(missing)
		return nil, fmt.Errorf("set all STORAGE_S3_* variables or none; missing %s", strings.Join(missing, ", "))
	}
}
