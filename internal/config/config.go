package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds all user preferences. Saved as TOML in the XDG config directory.
type Config struct {
	DownloadDir string      `toml:"download_dir"`
	Itchio      ItchioConfig `toml:"itchio"`
}

type ItchioConfig struct {
	APIKey string `toml:"api_key"`
}

// Defaults returns a config with sane starting values.
func Defaults() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		DownloadDir: filepath.Join(home, "Downloads"),
	}
}

// Path returns the full path to the config file.
func Path() string {
	dir := configDir()
	return filepath.Join(dir, "config.toml")
}

// Exists reports whether a config file has already been written.
func Exists() bool {
	_, err := os.Stat(Path())
	return err == nil
}

// Load reads the config from disk. If the file doesn't exist it returns
// defaults without an error—callers should check Exists() first to decide
// whether to show the setup wizard.
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

	// Expand ~ in the download directory so callers don't have to.
	cfg.DownloadDir = expandHome(cfg.DownloadDir)

	return cfg, nil
}

// Save writes the config to disk with restricted permissions so the API key
// isn't world-readable.
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

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
