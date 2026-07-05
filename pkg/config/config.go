package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	Shortcuts ShortcutsConfig           `yaml:"shortcuts"`
	Providers map[string]ProviderConfig `yaml:"providers"`
}

// ShortcutsConfig contains keybindings for global and provider-specific actions.
type ShortcutsConfig struct {
	Global    map[string]string            `yaml:"global"`
	Providers map[string]map[string]string `yaml:"providers"` // maps provider -> action name -> hotkey
}

// ProviderConfig configures where and how a provider is run.
type ProviderConfig struct {
	Type string `yaml:"type"` // "builtin" or "plugin"
	Path string `yaml:"path"` // Path to executable if "plugin"
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Shortcuts: ShortcutsConfig{
			Global: map[string]string{
				"command_palette": ":",
				"quit":            "q",
				"help":            "?",
				"profile_select":  "ctrl-p",
			},
			Providers: map[string]map[string]string{
				"ec2": {
					"SSH Connect": "s",
				},
			},
		},
		Providers: map[string]ProviderConfig{
			"ec2": {
				Type: "builtin",
			},
		},
	}
}

// LoadConfig attempts to load config.yaml from:
// 1. Current working directory
// 2. ~/.config/aws-tui/config.yaml (or OS equivalent)
// If not found, it returns the DefaultConfig().
func LoadConfig() (*Config, error) {
	// 1. Try current working directory
	path := "config.yaml"
	if _, err := os.Stat(path); err == nil {
		return readConfigFile(path)
	}

	// 2. Try user config directory
	userConfigDir, err := os.UserConfigDir()
	if err == nil {
		path = filepath.Join(userConfigDir, "aws-tui", "config.yaml")
		if _, err := os.Stat(path); err == nil {
			return readConfigFile(path)
		}
	}

	// Fallback to default config
	return DefaultConfig(), nil
}

func readConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig() // Start with defaults
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
