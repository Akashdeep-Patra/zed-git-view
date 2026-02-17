package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the resolved application configuration.
type Config struct {
	// Theme name: "dark" (default), "light", or path to custom theme.
	Theme string `mapstructure:"theme"`
	// Editor to use for commit messages (falls back to $EDITOR).
	Editor string `mapstructure:"editor"`
	// MaxLogEntries is the default number of log entries to load.
	MaxLogEntries int `mapstructure:"max_log_entries"`
	// ConfirmDestructive prompts before force push, discard, etc.
	ConfirmDestructive bool `mapstructure:"confirm_destructive"`
	// DiffContextLines is the number of context lines in diffs.
	DiffContextLines int `mapstructure:"diff_context_lines"`
	// SideBySideDiff enables side-by-side diff mode by default.
	SideBySideDiff bool `mapstructure:"side_by_side_diff"`
}

// Load reads configuration from ~/.config/zgv/config.yaml (or TOML/JSON).
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	configDir := configDirectory()
	v.AddConfigPath(configDir)
	v.AddConfigPath(".")

	setDefaults(v)

	v.SetEnvPrefix("ZGV")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		// Config file not found is fine — use defaults.
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("theme", "dark")
	v.SetDefault("editor", "")
	v.SetDefault("max_log_entries", 200)
	v.SetDefault("confirm_destructive", true)
	v.SetDefault("diff_context_lines", 3)
	v.SetDefault("side_by_side_diff", false)
}

func configDirectory() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "zgv")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zgv")
}
