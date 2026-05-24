package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigFrom(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name:    "valid with all fields",
			content: `{"api_key":"key1","token":"tok1","default_board":"b1","allowed_boards":["b1","b2"]}`,
			check: func(t *testing.T, cfg *Config) {
				if cfg.APIKey != "key1" {
					t.Errorf("APIKey = %q, want %q", cfg.APIKey, "key1")
				}
				if cfg.Token != "tok1" {
					t.Errorf("Token = %q, want %q", cfg.Token, "tok1")
				}
				if cfg.DefaultBoard != "b1" {
					t.Errorf("DefaultBoard = %q, want %q", cfg.DefaultBoard, "b1")
				}
				if len(cfg.AllowedBoards) != 2 {
					t.Errorf("AllowedBoards len = %d, want 2", len(cfg.AllowedBoards))
				}
			},
		},
		{
			name:    "valid with only required fields",
			content: `{"api_key":"key1","token":"tok1"}`,
			check: func(t *testing.T, cfg *Config) {
				if cfg.DefaultBoard != "" {
					t.Errorf("DefaultBoard = %q, want empty", cfg.DefaultBoard)
				}
				if cfg.AllowedBoards != nil {
					t.Errorf("AllowedBoards = %v, want nil", cfg.AllowedBoards)
				}
			},
		},
		{
			name:    "empty api_key",
			content: `{"api_key":"","token":"tok1"}`,
			wantErr: "api_key is empty",
		},
		{
			name:    "empty token",
			content: `{"api_key":"key1","token":""}`,
			wantErr: "token is empty",
		},
		{
			name:    "missing api_key field",
			content: `{"token":"tok1"}`,
			wantErr: "api_key is empty",
		},
		{
			name:    "invalid JSON",
			content: `{not valid json}`,
			wantErr: "parse config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("write test config: %v", err)
			}

			cfg, err := LoadConfigFrom(path)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error %q missing %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestPath(t *testing.T) {
	p, err := Path()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(p, filepath.Join("trello-mcp", "config.json")) {
		t.Errorf("Path() = %q, want suffix %q", p, filepath.Join("trello-mcp", "config.json"))
	}
}

func TestLoadConfigFromMissingFile(t *testing.T) {
	_, err := LoadConfigFrom("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("error %q missing expected message", err.Error())
	}
}
