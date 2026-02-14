package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const apiKeyEnvName = "HOLDED_CONFIG_PATH"

type Config struct {
	APIKey string `yaml:"api_key"`
}

func DefaultPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv(apiKeyEnvName)); override != "" {
		return override, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "holdedcli", "config.yaml"), nil
}

func Load(path string) (Config, error) {
	var cfg Config

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}

	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	return cfg, nil
}

func Save(path string, cfg Config) error {
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	b, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0o600)
}
