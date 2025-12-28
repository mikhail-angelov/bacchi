package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	content := `
s3:
  bucket: "test-bucket"
  region: "us-east-1"
backups:
  - name: "test"
    folders: ["/tmp"]
retention:
  daily: 5
`
	tmpfile, err := os.CreateTemp("", "config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.WriteString(content)
	if err != nil {
		t.Fatal(err)
	}
	if err = tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.S3.Bucket != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %s", cfg.S3.Bucket)
	}

	if cfg.Retention.Daily != 5 {
		t.Errorf("expected daily 5, got %d", cfg.Retention.Daily)
	}

	// Test defaults
	if cfg.Retention.Monthly != 1 {
		t.Errorf("expected default monthly 1, got %d", cfg.Retention.Monthly)
	}
}
