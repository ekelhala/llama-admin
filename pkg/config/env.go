package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func applyEnvOverrides(cfg *AppConfig) {
	if v := os.Getenv("LLAMA_ADMIN_SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("LLAMA_ADMIN_SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_SERVER_ALLOWED_ORIGINS"); v != "" {
		cfg.Server.AllowedOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("LLAMA_ADMIN_SERVER_ENABLE_SWAGGER"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Server.EnableSwagger = b
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_BACKEND_LLAMACPP_BINARY_PATH"); v != "" {
		cfg.Backends.LlamaCpp.BinaryPath = v
	}
	if v := os.Getenv("LLAMA_ADMIN_BACKEND_LLAMACPP_CACHE_DIR"); v != "" {
		cfg.Backends.LlamaCpp.CacheDir = v
	}
	if v := os.Getenv("LLAMA_ADMIN_BACKEND_LLAMACPP_DOWNLOAD_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Backends.LlamaCpp.DownloadTimeout = d
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_PORT_RANGE_MIN"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Instances.PortRange.Min = i
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_PORT_RANGE_MAX"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Instances.PortRange.Max = i
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_START_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Instances.StartTimeout = d
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_LOG_ROTATION_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Instances.LogRotationEnabled = b
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_LOG_ROTATION_MAX_SIZE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Instances.LogRotationMaxSize = i
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_LOG_ROTATION_COMPRESS"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Instances.LogRotationCompress = b
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_TIMEOUT_CHECK_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Instances.TimeoutCheckInterval = d
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_ENABLE_LRU_EVICTION"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Instances.EnableLRUEviction = b
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_INSTANCES_MAX_RUNNING_INSTANCES"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Instances.MaxRunningInstances = i
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_DATABASE_PATH"); v != "" {
		cfg.Database.Path = v
	}
	if v := os.Getenv("LLAMA_ADMIN_AUTH_SESSION_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Auth.Session.TTL = d
		}
	}
	if v := os.Getenv("LLAMA_ADMIN_AUTH_ALLOWED_EMAILS"); v != "" {
		cfg.Auth.AllowedEmails = strings.Split(v, ",")
	}
	if v := os.Getenv("LLAMA_ADMIN_DATA_DIR"); v != "" {
		cfg.DataDir = v
	}
}

func envConfigPath() string {
	if v := os.Getenv("LLAMA_ADMIN_CONFIG_PATH"); v != "" {
		return v
	}
	return "config.yaml"
}

func envDataDir() string {
	if v := os.Getenv("LLAMA_ADMIN_DATA_DIR"); v != "" {
		return v
	}
	return dataDir()
}

func envDBPath(dataDir string) string {
	if v := os.Getenv("LLAMA_ADMIN_DATABASE_PATH"); v != "" {
		return v
	}
	return defaultDBPath(dataDir)
}
