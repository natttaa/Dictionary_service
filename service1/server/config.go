package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// Config представляет конфигурацию сервиса
type Config struct {
	Port                 int    `json:"port"`
	LogLevel             string `json:"log_level"`
	DictionaryServiceURL string `json:"dictionary_service_url"`
	TimeoutSeconds       int    `json:"timeout"` // timeout в секундах
	Timeout              time.Duration
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = findConfigFile()
	}

	if configPath == "" {
		return DefaultConfig(), nil
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл конфигурации: %w", err)
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("не удалось распарсить конфигурацию: %w", err)
	}

	config.Timeout = time.Duration(config.TimeoutSeconds) * time.Second

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// findConfigFile ищет файл конфигурации в стандартных местах
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

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Port:                 8080,
		LogLevel:             "info",
		DictionaryServiceURL: "http://localhost:8081",
		TimeoutSeconds:       10,
		Timeout:              10 * time.Second,
	}
}

// validateConfig валидирует конфигурацию
func validateConfig(config *Config) error {
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("неверный порт: %d", config.Port)
	}
	if config.DictionaryServiceURL == "" {
		return fmt.Errorf("dictionary_service_url не может быть пустым")
	}
	if config.TimeoutSeconds <= 0 && config.Timeout <= 0 {
		return fmt.Errorf("таймаут должен быть положительным")
	}

	// Валидация уровня логирования
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[config.LogLevel] {
		return fmt.Errorf("неверный уровень логирования: %s (допустимые: debug, info, warn, error)", config.LogLevel)
	}

	return nil
}

// SetupLogger настраивает логгер на основе конфигурации
func (c *Config) SetupLogger() *slog.Logger {
	var level slog.Level

	switch c.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
