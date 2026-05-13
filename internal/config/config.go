// Package config handles loading and validating the trello-mcp config file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the trello-mcp configuration.
type Config struct {
	APIKey        string   `json:"api_key"`
	Token         string   `json:"token"`
	DefaultBoard  string   `json:"default_board"`
	AllowedBoards []string `json:"allowed_boards"`
}

func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine config directory: %w", err)
	}
	return filepath.Join(dir, "trello-mcp"), nil
}

// Path returns the path to the config file.
func Path() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfig loads the config from the default path.
func LoadConfig() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	return LoadConfigFrom(path)
}

// LoadConfigFrom loads and validates the config from a specific path.
func LoadConfigFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"config file not found at %s -- see SETUP.md for instructions", path,
			)
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf(
			"api_key is empty in %s -- add your Trello API key", path,
		)
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf(
			"token is empty in %s -- add your Trello API token", path,
		)
	}

	return &cfg, nil
}
