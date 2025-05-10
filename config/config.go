package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the application configuration
type Config struct {
	Server ServerConfig `json:"server"`
	Auth   AuthConfig   `json:"auth"`
	Cache  CacheConfig  `json:"cache"`
	Log    LogConfig    `json:"log"`
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Port            int    `json:"port"`
	Host            string `json:"host"`
	ReadTimeoutSec  int    `json:"read_timeout_sec"`
	WriteTimeoutSec int    `json:"write_timeout_sec"`
}

// AuthConfig contains authentication-related configuration
type AuthConfig struct {
	Enabled     bool   `json:"enabled"`
	JWTSecret   string `json:"jwt_secret"`
	TokenExpiry int    `json:"token_expiry_minutes"`
}

// CacheConfig contains cache-related configuration
type CacheConfig struct {
	Type          string `json:"type"` // memory or redis
	RedisHost     string `json:"redis_host"`
	RedisPort     int    `json:"redis_port"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`
}

// LogConfig contains logging-related configuration
type LogConfig struct {
	Level      string `json:"level"` // debug, info, warn, error
	Format     string `json:"format"` // json or text
	OutputPath string `json:"output_path"` // stdout, file path, or both
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            8081,
			Host:            "0.0.0.0",
			ReadTimeoutSec:  30,
			WriteTimeoutSec: 30,
		},
		Auth: AuthConfig{
			Enabled:     false,
			JWTSecret:   "change-me-in-production",
			TokenExpiry: 60, // 1 hour
		},
		Cache: CacheConfig{
			Type:          "memory",
			RedisHost:     "localhost",
			RedisPort:     6379,
			RedisPassword: "",
			RedisDB:       0,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "stdout",
		},
	}
}

// LoadConfig loads the configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// Start with default config
	config := DefaultConfig()

	// If config path is specified, load and merge with defaults
	if configPath != "" {
		data, err := ioutil.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}

		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("error parsing config file: %w", err)
		}
	}

	// Override with environment variables
	overrideWithEnv(config)

	return config, nil
}

// overrideWithEnv overrides config values with environment variables
func overrideWithEnv(config *Config) {
	// Server config
	if port := os.Getenv("DGLA_SERVER_PORT"); port != "" {
		var p int
		if _, err := fmt.Sscanf(port, "%d", &p); err == nil {
			config.Server.Port = p
		}
	}
	
	// Auth config
	if enabled := os.Getenv("DGLA_AUTH_ENABLED"); enabled != "" {
		config.Auth.Enabled = strings.ToLower(enabled) == "true"
	}
	if secret := os.Getenv("DGLA_AUTH_JWT_SECRET"); secret != "" {
		config.Auth.JWTSecret = secret
	}
	
	// Cache config
	if cacheType := os.Getenv("DGLA_CACHE_TYPE"); cacheType != "" {
		config.Cache.Type = cacheType
	}
	if redisHost := os.Getenv("DGLA_REDIS_HOST"); redisHost != "" {
		config.Cache.RedisHost = redisHost
	}
	
	// Log config
	if logLevel := os.Getenv("DGLA_LOG_LEVEL"); logLevel != "" {
		config.Log.Level = logLevel
	}
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling config: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
