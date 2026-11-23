package config

import (
	"strings"
	"testing"
)

func setRequired(t *testing.T) {
	t.Setenv("DATABASE_PATH", "test.db")
	t.Setenv("STORAGE_PUBLIC_BASE_URL", "https://storage.example.com")
	t.Setenv("ADMIN_USERNAME", "admin")
	t.Setenv("ADMIN_PASSWORD", "secret")
}

var s3Variables = []string{
	"STORAGE_S3_ENDPOINT", "STORAGE_S3_REGION", "STORAGE_S3_ACCESS_KEY_ID",
	"STORAGE_S3_SECRET_ACCESS_KEY", "STORAGE_S3_BUCKET",
}

func TestS3IsOptional(t *testing.T) {
	setRequired(t)
	for _, name := range s3Variables {
		t.Setenv(name, "")
	}
	cfg, err := Load()
	if err != nil || cfg.S3 != nil {
		t.Errorf("without S3 variables: %+v, %v", cfg.S3, err)
	}

	for _, name := range s3Variables {
		t.Setenv(name, "value")
	}
	cfg, err = Load()
	if err != nil || cfg.S3 == nil || cfg.S3.Bucket != "value" {
		t.Errorf("with S3 variables: %+v, %v", cfg.S3, err)
	}

	t.Setenv("STORAGE_S3_BUCKET", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "STORAGE_S3_BUCKET") {
		t.Errorf("with a missing S3 variable: %v", err)
	}
}
