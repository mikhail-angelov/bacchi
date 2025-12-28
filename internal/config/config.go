package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	S3 struct {
		Bucket          string `yaml:"bucket"`
		Region          string `yaml:"region"`
		Endpoint        string `yaml:"endpoint"`
		AccessKeyID     string `yaml:"access_key_id"`
		SecretAccessKey string `yaml:"secret_access_key"`
		Prefix          string `yaml:"prefix"`
	} `yaml:"s3"`
	Backups []struct {
		Name    string   `yaml:"name"`
		Folders []string `yaml:"folders"`
		Exclude []string `yaml:"exclude"`
	} `yaml:"backups"`
	Encryption struct {
		Passphrase string `yaml:"passphrase"`
		Enabled    bool   `yaml:"enabled"`
	} `yaml:"encryption"`
	Retention struct {
		Daily   int `yaml:"daily"`
		Monthly int `yaml:"monthly"`
	} `yaml:"retention"`
	Telegram struct {
		BotToken string `yaml:"bot_token"`
		ChatID   string `yaml:"chat_id"`
		Enabled  bool   `yaml:"enabled"`
	} `yaml:"telegram"`
	Schedule string `yaml:"schedule"` // Cron format
}

// LoadConfig loads the configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	if cfg.Retention.Daily == 0 {
		cfg.Retention.Daily = 10
	}
	if cfg.Retention.Monthly == 0 {
		cfg.Retention.Monthly = 1
	}
	if cfg.Schedule == "" {
		cfg.Schedule = "0 0 * * *" // Daily at midnight
	}

	return &cfg, nil
}
