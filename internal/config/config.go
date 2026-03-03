package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DownloadDir string      `toml:"download_dir"`
	Itchio      ItchioConfig `toml:"itchio"`
}

type ItchioConfig struct {
	APIKey string `toml:"api_key"`
}

func Defaults() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		DownloadDir: filepath.Join(home, "Downloads"),
	}
}

func Path() string {
	dir := configDir()
	return filepath.Join(dir, "config.toml")
}

func Exists() bool {
	_, err := os.Stat(Path())
	return err == nil
}

// Load reads the config from disk, returning defaults if no file exists yet.
func Load() (*Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(Path())
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.DownloadDir = ExpandHome(cfg.DownloadDir)

	return cfg, nil
}

// Save writes the config to disk with 0600 permissions.
func Save(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	f, err := os.OpenFile(Path(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening config: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "flashgrab")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "flashgrab")
}

func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// ValidateDir expands ~ and checks the path is usable as a download directory.
func ValidateDir(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("download directory can't be empty")
	}

	expanded := ExpandHome(raw)
	expanded, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(expanded)
	if err == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("path exists but is not a directory")
		}
		return expanded, nil
	}

	parent := filepath.Dir(expanded)
	if _, err := os.Stat(parent); err != nil {
		return "", fmt.Errorf("parent directory %s doesn't exist", parent)
	}

	return expanded, nil
}
