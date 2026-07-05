package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*AppConfig, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}

	cfg.DataDir = ExpandPlaceholders(cfg.DataDir)
	cfg.Database.Path = ExpandPlaceholders(cfg.Database.Path)
	cfg.Backends.LlamaCpp.BinaryPath = ExpandPlaceholders(cfg.Backends.LlamaCpp.BinaryPath)
	cfg.Backends.LlamaCpp.CacheDir = ExpandPlaceholders(cfg.Backends.LlamaCpp.CacheDir)

	if cfg.DataDir == "" {
		cfg.DataDir = envDataDir()
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = envDBPath(cfg.DataDir)
	}

	applyEnvOverrides(&cfg)

	return &cfg, nil
}

func ConfigPath() string {
	if v := os.Getenv("LLAMA_ADMIN_CONFIG_PATH"); v != "" {
		return v
	}
	return "config.yaml"
}

func DataDirPath() string {
	if v := os.Getenv("LLAMA_ADMIN_DATA_DIR"); v != "" {
		return v
	}
	return dataDir()
}

func DBDirPath(dataDir string) string {
	return filepath.Dir(envDBPath(dataDir))
}
