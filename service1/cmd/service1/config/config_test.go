package config

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		configData  interface{}
		expected    *Config
		expectError bool
	}{
		{
			name:       "load from existing file",
			configPath: "test_config.json",
			configData: Config{
				Port:                 9090,
				LogLevel:             "debug",
				DictionaryServiceURL: "http://localhost:9090",
				TimeoutSeconds:       20,
			},
			expected: &Config{
				Port:                 9090,
				LogLevel:             "debug",
				DictionaryServiceURL: "http://localhost:9090",
				TimeoutSeconds:       20,
				Timeout:              20 * time.Second,
			},
			expectError: false,
		},
		{
			name:        "file not found - should return error",
			configPath:  "nonexistent.json",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid json",
			configPath:  "invalid.json",
			configData:  "invalid json content",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.configData != nil {
				// Создаем временный файл для теста
				tmpFile, err := os.CreateTemp("", tt.configPath)
				require.NoError(t, err)
				defer os.Remove(tmpFile.Name())

				if data, ok := tt.configData.(Config); ok {
					jsonData, err := json.Marshal(data)
					require.NoError(t, err)
					_, err = tmpFile.Write(jsonData)
					require.NoError(t, err)
				} else if data, ok := tt.configData.(string); ok {
					_, err = tmpFile.Write([]byte(data))
					require.NoError(t, err)
				}
				tmpFile.Close()

				config, err := LoadConfig(tmpFile.Name())
				if tt.expectError {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.expected.Port, config.Port)
					assert.Equal(t, tt.expected.LogLevel, config.LogLevel)
					assert.Equal(t, tt.expected.DictionaryServiceURL, config.DictionaryServiceURL)
					assert.Equal(t, tt.expected.TimeoutSeconds, config.TimeoutSeconds)
					assert.Equal(t, tt.expected.Timeout, config.Timeout)
				}
			} else {
				config, err := LoadConfig(tt.configPath)
				if tt.expectError {
					assert.Error(t, err)
					assert.Nil(t, config)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, config)
				}
			}
		})
	}
}

func TestLoadConfigWithDefault(t *testing.T) {
	// Сохраняем оригинальное значение
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Тестируем загрузку без указания пути к конфигу
	t.Run("no config path - use default", func(t *testing.T) {
		// Создаем временную директорию и делаем ее рабочей
		tempDir, err := os.MkdirTemp("", "test_configs")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Меняем рабочую директорию
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Создаем стандартный файл конфигурации
		configDir := "configs"
		err = os.Mkdir(configDir, 0755)
		require.NoError(t, err)

		defaultConfig := Config{
			Port:                 9999,
			LogLevel:             "warn",
			DictionaryServiceURL: "http://test:8080",
			TimeoutSeconds:       30,
		}

		configFile, err := os.Create(configDir + "/service.json")
		require.NoError(t, err)

		jsonData, err := json.Marshal(defaultConfig)
		require.NoError(t, err)
		_, err = configFile.Write(jsonData)
		require.NoError(t, err)
		configFile.Close()

		// Загружаем конфиг без указания пути
		config, err := LoadConfig("")
		require.NoError(t, err)

		assert.Equal(t, defaultConfig.Port, config.Port)
		assert.Equal(t, defaultConfig.LogLevel, config.LogLevel)
		assert.Equal(t, defaultConfig.DictionaryServiceURL, config.DictionaryServiceURL)
		assert.Equal(t, defaultConfig.TimeoutSeconds, config.TimeoutSeconds)
		assert.Equal(t, time.Duration(defaultConfig.TimeoutSeconds)*time.Second, config.Timeout)
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Port:                 8081,
				DictionaryServiceURL: "http://localhost:8083",
				TimeoutSeconds:       10,
				LogLevel:             "info",
			},
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			config: &Config{
				Port:                 0,
				DictionaryServiceURL: "http://localhost:8083",
				TimeoutSeconds:       10,
				LogLevel:             "info",
			},
			wantErr: true,
			errMsg:  "неверный порт",
		},
		{
			name: "invalid port - too high",
			config: &Config{
				Port:                 70000,
				DictionaryServiceURL: "http://localhost:8083",
				TimeoutSeconds:       10,
				LogLevel:             "info",
			},
			wantErr: true,
			errMsg:  "неверный порт",
		},
		{
			name: "empty dictionary url",
			config: &Config{
				Port:                 8081,
				DictionaryServiceURL: "",
				TimeoutSeconds:       10,
				LogLevel:             "info",
			},
			wantErr: true,
			errMsg:  "dictionary_service_url не может быть пустым",
		},
		{
			name: "invalid timeout - zero",
			config: &Config{
				Port:                 8081,
				DictionaryServiceURL: "http://localhost:8083",
				TimeoutSeconds:       0,
				LogLevel:             "info",
			},
			wantErr: true,
			errMsg:  "таймаут должен быть положительным",
		},
		{
			name: "invalid log level",
			config: &Config{
				Port:                 8081,
				DictionaryServiceURL: "http://localhost:8083",
				TimeoutSeconds:       10,
				LogLevel:             "invalid",
			},
			wantErr: true,
			errMsg:  "неверный уровень логирования",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 8081, config.Port)
	assert.Equal(t, "info", config.LogLevel)
	assert.Equal(t, "http://localhost:8083", config.DictionaryServiceURL)
	assert.Equal(t, 10, config.TimeoutSeconds)
	assert.Equal(t, 10*time.Second, config.Timeout)
}

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		wantLevel string
	}{
		{"debug level", "debug", "debug"},
		{"info level", "info", "info"},
		{"warn level", "warn", "warn"},
		{"error level", "error", "error"},
		{"invalid level - default to info", "invalid", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				LogLevel: tt.logLevel,
			}
			logger := config.SetupLogger()
			assert.NotNil(t, logger)

			// Проверяем, что логгер создан без ошибок
			assert.NotNil(t, logger.Handler())
		})
	}
}

func TestTryToLoadStandardConfigFile(t *testing.T) {
	// Сохраняем оригинальную рабочую директорию
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Создаем временную директорию
	tempDir, err := os.MkdirTemp("", "test_config_load")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	t.Run("no config file exists", func(t *testing.T) {
		result := tryToLoadStandartConfigFile()
		assert.Equal(t, "", result)
	})

	t.Run("config file exists", func(t *testing.T) {
		// Создаем директорию configs
		err := os.Mkdir("configs", 0755)
		require.NoError(t, err)

		// Создаем файл конфигурации
		configFile, err := os.Create("configs/service.json")
		require.NoError(t, err)
		configFile.WriteString("{}")
		configFile.Close()

		result := tryToLoadStandartConfigFile()
		assert.Equal(t, "configs/service.json", result)
	})
}

// Бенчмарк тесты
func BenchmarkValidateConfig(b *testing.B) {
	cfg := &Config{
		Port:                 8081,
		DictionaryServiceURL: "http://localhost:8083",
		TimeoutSeconds:       10,
		LogLevel:             "info",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validateConfig(cfg)
	}
}

func BenchmarkDefaultConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}
