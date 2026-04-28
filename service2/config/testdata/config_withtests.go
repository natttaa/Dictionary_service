package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

type Database struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	Password       string `json:"password"`
	Dbname         string `json:"dbname"`
	SSL_mode       string `json:"ssl_mode"`
	TimeoutSeconds int    `json:"timeout"`
	Timeout        time.Duration
}

// Config представляет конфигурацию сервиса
type Config struct {
	Port           int      `json:"port"`
	LogLevel       string   `json:"log_level"`
	Data           Database `json:"database"`
	TimeoutSeconds int      `json:"timeout"`
	Timeout        time.Duration
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

	config.Data.Timeout = time.Duration(config.Data.TimeoutSeconds) * time.Second

	if err := validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// findConfigFile ищет файл конфигурации в стандартных местах
func findConfigFile() string {
	paths := []string{
		"config/service2.json",
		// тестовые конфиги
		"config/testdata/valid_config.json",
		"config/testdata/debug_config.json",
		"config/testdata/invalid_port.json",
		"config/testdata/invalid_db_port.json",
		"config/testdata/invalid_log_level.json",
		"config/testdata/malformed.json",
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
		Port:     8083,
		LogLevel: "info",
	}
}

// validateConfig валидирует конфигурацию
func validateConfig(config *Config) error {
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("неверный порт: %d", config.Port)
	}
	if config.Data.Port <= 0 || config.Data.Port > 65535 {
		return fmt.Errorf("неверный порт для database: %d", config.Data.Port)
	}

	// Валидация таймаута сервера (0 допустимо — таймаут не задан)
	if config.TimeoutSeconds < 0 {
		return fmt.Errorf("таймаут не может быть отрицательным: %d", config.TimeoutSeconds)
	}
	// Валидация таймаута БД
	if config.Data.TimeoutSeconds < 0 {
		return fmt.Errorf("таймаут БД не может быть отрицательным: %d", config.Data.TimeoutSeconds)
	}

	// Валидация уровня логирования
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

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
