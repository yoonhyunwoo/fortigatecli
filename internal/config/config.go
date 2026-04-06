package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultDirName  = ".fortigatecli"
	defaultFileName = "config.yaml"
	defaultVDOM     = "root"
	defaultTimeout  = 10 * time.Second
)

type Config struct {
	Host     string        `yaml:"host"`
	Token    string        `yaml:"token"`
	VDOM     string        `yaml:"vdom"`
	Insecure bool          `yaml:"insecure"`
	Timeout  time.Duration `yaml:"timeout"`
}

func Default() Config {
	return Config{
		VDOM:     defaultVDOM,
		Insecure: true,
		Timeout:  defaultTimeout,
	}
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, defaultDirName, defaultFileName), nil
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, ErrNotConfigured
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

func Save(cfg Config) error {
	cfg.applyDefaults()

	if err := cfg.Validate(); err != nil {
		return err
	}

	path, err := Path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func (c *Config) applyDefaults() {
	if c.VDOM == "" {
		c.VDOM = defaultVDOM
	}
	if c.Timeout == 0 {
		c.Timeout = defaultTimeout
	}
}

func (c Config) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Token == "" {
		return errors.New("token is required")
	}
	if c.VDOM == "" {
		return errors.New("vdom is required")
	}
	if c.Timeout < 0 {
		return errors.New("timeout must be >= 0")
	}
	return nil
}

var ErrNotConfigured = errors.New("fortigatecli is not configured")
