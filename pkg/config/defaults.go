package config

import (
	"os"
	"os/user"
	"path/filepath"
	"time"
)

func defaults() AppConfig {
	return AppConfig{
		Server: ServerConfig{
			Host:           "127.0.0.1",
			Port:           8080,
			AllowedOrigins: []string{"*"},
			EnableSwagger:  false,
		},
		Backends: BackendConfig{
			LlamaCpp: LlamaCppBackendConfig{
				BinaryPath:      "",
				CacheDir:        "",
				DownloadTimeout: 10 * time.Minute,
			},
		},
		Instances: InstancesConfig{
			PortRange: PortRange{
				Min: 8100,
				Max: 9000,
			},
			OnDemandStartTimeout: 30 * time.Second,
			LogRotationEnabled:   true,
			LogRotationMaxSize:   50 * 1024 * 1024,
			LogRotationCompress:  true,
			TimeoutCheckInterval: 10 * time.Second,
			EnableLRUEviction:    false,
			MaxRunningInstances:  10,
			GroupLimits:          map[string]int{},
		},
		Database: DatabaseConfig{
			Path: "",
		},
		Auth: AuthConfig{
			Session: SessionConfig{
				TTL: 24 * time.Hour,
			},
			Providers:     map[string]ProviderConfig{},
			AllowedEmails: []string{},
		},
	}
}

func dataDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "llama-admin")
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "llama-admin")
	}
	if us, err := user.Current(); err == nil {
		return filepath.Join(us.HomeDir, ".llama-admin")
	}
	return "llama-admin"
}

func defaultDBPath(dataDir string) string {
	return filepath.Join(dataDir, "data", "llama-admin.db")
}
