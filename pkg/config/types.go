package config

import "time"

type AppConfig struct {
	Server    ServerConfig    `yaml:"server"`
	Backends  BackendConfig   `yaml:"backends"`
	Instances InstancesConfig `yaml:"instances"`
	Database  DatabaseConfig  `yaml:"database"`
	Auth      AuthConfig      `yaml:"auth"`
	DataDir   string          `yaml:"dataDir"`
	Version   string          `yaml:"-"`
	Commit    string          `yaml:"-"`
	BuildTime string          `yaml:"-"`
}

type ServerConfig struct {
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	AllowedOrigins []string `yaml:"allowedOrigins"`
	EnableSwagger  bool     `yaml:"enableSwagger"`
}

type BackendConfig struct {
	LlamaCpp LlamaCppBackendConfig `yaml:"llamaCpp"`
}

type LlamaCppBackendConfig struct {
	BinaryPath      string        `yaml:"binaryPath"`
	CacheDir        string        `yaml:"cacheDir"`
	DownloadTimeout time.Duration `yaml:"downloadTimeout"`
}

type InstancesConfig struct {
	PortRange            PortRange        `yaml:"portRange"`
	StartTimeout         time.Duration    `yaml:"startTimeout"`
	LogRotationEnabled   bool             `yaml:"logRotationEnabled"`
	LogRotationMaxSize   int              `yaml:"logRotationMaxSize"`
	LogRotationCompress  bool             `yaml:"logRotationCompress"`
	TimeoutCheckInterval time.Duration    `yaml:"timeoutCheckInterval"`
	EnableLRUEviction    bool             `yaml:"enableLRUEviction"`
	MaxRunningInstances  int              `yaml:"maxRunningInstances"`
	GroupLimits          map[string]int   `yaml:"groupLimits"`
}

type PortRange struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type AuthConfig struct {
	Session       SessionConfig               `yaml:"session"`
	Providers     map[string]ProviderConfig   `yaml:"providers"`
	AllowedEmails []string                    `yaml:"allowedEmails"`
}

type SessionConfig struct {
	TTL time.Duration `yaml:"ttl"`
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
