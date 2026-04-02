package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server     ServerConfig     `toml:"server"`
	Scanner    ScannerConfig    `toml:"scanner"`
	Transcoder TranscoderConfig `toml:"transcoder"`
	Safety     SafetyConfig     `toml:"safety"`
	Plex       PlexConfig       `toml:"plex"`
	Auth       AuthConfig       `toml:"auth"`
}

type ServerConfig struct {
	Host    string `toml:"host"`
	Port    int    `toml:"port"`
	DataDir string `toml:"data_dir"`
}

type ScannerConfig struct {
	IntervalHours      int `toml:"interval_hours"`
	WorkerConcurrency  int `toml:"worker_concurrency"`
}

type TranscoderConfig struct {
	TempDir string `toml:"temp_dir"`
}

type SafetyConfig struct {
	QuarantineEnabled      bool   `toml:"quarantine_enabled"`
	QuarantineRetentionDays int   `toml:"quarantine_retention_days"`
	QuarantineDir          string `toml:"quarantine_dir"`
	DiskFreePauseGB        int    `toml:"disk_free_pause_gb"`
}

type PlexConfig struct {
	Enabled bool   `toml:"enabled"`
	BaseURL string `toml:"base_url"`
	Token   string `toml:"token"`
}

type AuthConfig struct {
	PasswordHash string `toml:"password_hash"`
	JWTSecret    string `toml:"jwt_secret"`
}

// Defaults returns a Config with safe default values.
func Defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host:    "127.0.0.1",
			Port:    8080,
			DataDir: "/var/lib/pdarr",
		},
		Scanner: ScannerConfig{
			IntervalHours:     6,
			WorkerConcurrency: 1,
		},
		Safety: SafetyConfig{
			QuarantineEnabled:       true,
			QuarantineRetentionDays: 10,
			DiskFreePauseGB:         50,
		},
	}
}

// Load reads and validates a TOML config file, applying defaults for missing fields.
func Load(path string) (*Config, error) {
	cfg := Defaults()

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Server.DataDir == "" {
		return fmt.Errorf("server.data_dir must not be empty")
	}
	if c.Scanner.WorkerConcurrency < 1 || c.Scanner.WorkerConcurrency > 8 {
		return fmt.Errorf("scanner.worker_concurrency must be between 1 and 8")
	}
	if c.Scanner.IntervalHours < 1 {
		return fmt.Errorf("scanner.interval_hours must be at least 1")
	}
	if c.Plex.Enabled && c.Plex.BaseURL == "" {
		return fmt.Errorf("plex.base_url is required when plex.enabled is true")
	}
	if c.Plex.Enabled && c.Plex.Token == "" {
		return fmt.Errorf("plex.token is required when plex.enabled is true")
	}
	return nil
}

// QuarantineDir returns the resolved quarantine directory path.
func (c *Config) QuarantineDir() string {
	if c.Safety.QuarantineDir != "" {
		return c.Safety.QuarantineDir
	}
	return filepath.Join(c.Server.DataDir, "quarantine")
}

// DBPath returns the path to the SQLite database file.
func (c *Config) DBPath() string {
	return filepath.Join(c.Server.DataDir, "pdarr.db")
}

// Addr returns the host:port listen address.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
