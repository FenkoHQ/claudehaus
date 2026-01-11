package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Server   ServerConfig            `json:"server"`
	Tokens   []Token                 `json:"tokens"`
	Sessions map[string]SessionMeta  `json:"sessions"`
	Settings Settings                `json:"settings"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type Token struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Value      string `json:"value"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at"`
}

type SessionMeta struct {
	Nickname string `json:"nickname"`
}

type Settings struct {
	ApprovalTimeoutSeconds  int    `json:"approval_timeout_seconds"`
	ApprovalTimeoutBehavior string `json:"approval_timeout_behavior"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8420,
		},
		Tokens:   []Token{},
		Sessions: make(map[string]SessionMeta),
		Settings: Settings{
			ApprovalTimeoutSeconds:  300,
			ApprovalTimeoutBehavior: "passthrough",
		},
	}
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".claudehaus"), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := cfg.Save(); err != nil {
			return nil, fmt.Errorf("saving default config: %w", err)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Sessions == nil {
		cfg.Sessions = make(map[string]SessionMeta)
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("writing temp config: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming config: %w", err)
	}

	return nil
}
