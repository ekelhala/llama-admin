package config

import "time"

type AppConfig struct {
	Server    ServerConfig
	Backends  BackendConfig
	Instances InstancesConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	DataDir   string
	Version   string
	Commit    string
	BuildTime string
}

type ServerConfig struct {
	Host           string
	Port           int
	AllowedOrigins []string
	EnableSwagger  bool
}

type BackendConfig struct {
	LlamaCpp LlamaCppBackendConfig
}

type LlamaCppBackendConfig struct {
	BinaryPath      string
	CacheDir        string
	DownloadTimeout time.Duration
}

type InstancesConfig struct {
	PortRange            PortRange
	OnDemandStartTimeout time.Duration
	LogRotationEnabled   bool
	LogRotationMaxSize   int
	LogRotationCompress  bool
	TimeoutCheckInterval time.Duration
	EnableLRUEviction    bool
	MaxRunningInstances  int
	GroupLimits          map[string]int
}

type PortRange struct {
	Min, Max int
}

type DatabaseConfig struct {
	Path string
}

type AuthConfig struct {
	Session       SessionConfig
	Providers     map[string]ProviderConfig
	AllowedEmails []string
}

type SessionConfig struct {
	TTL time.Duration
}

type ProviderConfig struct {
	Enabled                     bool     `yaml:"enabled"`
	ClientID                    string   `yaml:"clientId"`
	ClientSecret                string   `yaml:"clientSecret"`
	Scopes                      []string `yaml:"scopes"`
	DeviceAuthorizationEndpoint string   `yaml:"deviceAuthorizationEndpoint"`
	TokenEndpoint               string   `yaml:"tokenEndpoint"`
	UserEndpoint                string   `yaml:"userEndpoint"`
}
