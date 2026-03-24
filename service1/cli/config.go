package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// CLIConfig представляет конфигурацию CLI
type CLIConfig struct {
	ServerURL      string `json:"translation_server_url"`
	TimeoutSeconds int    `json:"timeout"` // timeout в секундах
	DefaultFormat  string `json:"default_format"`
	LogLevel       string `json:"log_level"` // уровень логирования
	Timeout        time.Duration
}

// UnmarshalJSON кастомный парсинг JSON для CLIConfig
func (c *CLIConfig) UnmarshalJSON(data []byte) error {
	// Временная структура для парсинга
	type Alias struct {
		ServerURL      string `json:"translation_server_url"`
		TimeoutSeconds int    `json:"timeout"`
		DefaultFormat  string `json:"default_format"`
		LogLevel       string `json:"log_level"`
	}

	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	// Заполняем поля
	c.ServerURL = alias.ServerURL
	c.TimeoutSeconds = alias.TimeoutSeconds
	c.DefaultFormat = alias.DefaultFormat
	c.LogLevel = alias.LogLevel

	// Устанавливаем значения по умолчанию
	if c.ServerURL == "" {
		c.ServerURL = "http://localhost:8081"
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = 30
	}
	if c.DefaultFormat == "" {
		c.DefaultFormat = "table"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}

	// Конвертируем секунды в time.Duration
	c.Timeout = time.Duration(c.TimeoutSeconds) * time.Second

	return nil
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(configPath string) (*CLIConfig, error) {
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

	var config CLIConfig
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, fmt.Errorf("не удалось распарсить конфигурацию: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// findConfigFile ищет файл конфигурации в стандартных местах
func findConfigFile() string {
	paths := []string{
		"configs/cli.json",
		"../configs/cli.json",
		"./cli.json",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *CLIConfig {
	return &CLIConfig{
		ServerURL:      "http://localhost:8081",
		TimeoutSeconds: 30,
		Timeout:        30 * time.Second,
		DefaultFormat:  "table",
		LogLevel:       "info",
	}
}

// validateConfig валидирует конфигурацию
func validateConfig(config *CLIConfig) error {
	if config.ServerURL == "" {
		return fmt.Errorf("translation_server_url не может быть пустым")
	}
	if config.TimeoutSeconds <= 0 {
		return fmt.Errorf("таймаут должен быть положительным")
	}
	if config.DefaultFormat != "table" && config.DefaultFormat != "json" {
		return fmt.Errorf("default_format должен быть 'table' или 'json'")
	}

	// Валидация уровня логирования
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[config.LogLevel] {
		return fmt.Errorf("неверный уровень логирования: %s (допустимые: debug, info, warn, error)", config.LogLevel)
	}

	return nil
}

// SetupLogger настраивает логгер на основе конфигурации
func (c *CLIConfig) SetupLogger() *slog.Logger {
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
