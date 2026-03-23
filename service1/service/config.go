package service

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config represents the service configuration
type Config struct {
	Port                 int           `json:"port"`
	LogLevel             string        `json:"log_level"`
	DictionaryServiceURL string        `json:"dictionary_service_url"`
	Timeout              time.Duration `json:"timeout"`
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = findConfigFile()
	}

	if configPath == "" {
		return DefaultConfig(), nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// findConfigFile searches for config file in standard locations
func findConfigFile() string {
	paths := []string{
		"configs/service.json",
		"../configs/service.json",
		"./service.json",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Port:                 8080,
		LogLevel:             "info",
		DictionaryServiceURL: "http://localhost:8081",
		Timeout:              10 * time.Second,
	}
}

// validateConfig validates configuration
func validateConfig(config *Config) error {
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}
	if config.DictionaryServiceURL == "" {
		return fmt.Errorf("dictionary_service_url cannot be empty")
	}
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}
