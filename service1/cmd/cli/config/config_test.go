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
		expected    *CLIConfig
		expectError bool
	}{
		{
			name:       "load from existing file",
			configPath: "test_cli_config.json",
			configData: map[string]interface{}{
				"translation_server_url": "http://localhost:9090",
				"timeout":                60,
				"default_format":         "json",
				"log_level":              "debug",
			},
			expected: &CLIConfig{
				ServerURL:      "http://localhost:9090",
				TimeoutSeconds: 60,
				Timeout:        60 * time.Second,
				DefaultFormat:  "json",
				LogLevel:       "debug",
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
		{
			name:       "empty config - use defaults",
			configPath: "empty.json",
			configData: map[string]interface{}{},
			expected: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 30,
				Timeout:        30 * time.Second,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			expectError: false,
		},
		{
			name:       "partial config - fill missing with defaults",
			configPath: "partial.json",
			configData: map[string]interface{}{
				"translation_server_url": "http://custom:8080",
			},
			expected: &CLIConfig{
				ServerURL:      "http://custom:8080",
				TimeoutSeconds: 30,
				Timeout:        30 * time.Second,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.configData != nil {
				// Создаем временный файл для теста
				tmpFile, err := os.CreateTemp("", tt.configPath)
				require.NoError(t, err)
				defer os.Remove(tmpFile.Name())

				// Записываем данные в файл
				jsonData, err := json.Marshal(tt.configData)
				require.NoError(t, err)
				_, err = tmpFile.Write(jsonData)
				require.NoError(t, err)
				tmpFile.Close()

				config, err := LoadConfig(tmpFile.Name())
				if tt.expectError {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.expected.ServerURL, config.ServerURL)
					assert.Equal(t, tt.expected.TimeoutSeconds, config.TimeoutSeconds)
					assert.Equal(t, tt.expected.Timeout, config.Timeout)
					assert.Equal(t, tt.expected.DefaultFormat, config.DefaultFormat)
					assert.Equal(t, tt.expected.LogLevel, config.LogLevel)
				}
			} else {
				config, err := LoadConfig(tt.configPath)
				if tt.expectError {
					assert.Error(t, err)
					assert.Nil(t, config)
				} else {
					require.NoError(t, err)
					assert.NotNil(t, config)
					if tt.expected != nil {
						assert.Equal(t, tt.expected.ServerURL, config.ServerURL)
						assert.Equal(t, tt.expected.TimeoutSeconds, config.TimeoutSeconds)
						assert.Equal(t, tt.expected.Timeout, config.Timeout)
						assert.Equal(t, tt.expected.DefaultFormat, config.DefaultFormat)
						assert.Equal(t, tt.expected.LogLevel, config.LogLevel)
					}
				}
			}
		})
	}
}

func TestLoadConfigWithDefault(t *testing.T) {
	// Сохраняем оригинальную директорию
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	t.Run("no config path - use default from standard location", func(t *testing.T) {
		// Создаем временную директорию
		tempDir, err := os.MkdirTemp("", "test_cli_configs")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Меняем рабочую директорию
		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Создаем стандартный файл конфигурации
		configDir := "configs"
		err = os.Mkdir(configDir, 0755)
		require.NoError(t, err)

		// Создаем конфиг с нестандартными значениями
		testConfig := map[string]interface{}{
			"translation_server_url": "http://test-server:8080",
			"timeout":                45,
			"default_format":         "json",
			"log_level":              "warn",
		}

		configFile, err := os.Create(configDir + "/cli.json")
		require.NoError(t, err)

		jsonData, err := json.Marshal(testConfig)
		require.NoError(t, err)
		_, err = configFile.Write(jsonData)
		require.NoError(t, err)
		configFile.Close()

		// Загружаем конфиг без указания пути
		config, err := LoadConfig("")
		require.NoError(t, err)

		// Проверяем, что загрузились значения из файла, а не дефолтные
		assert.Equal(t, "http://test-server:8080", config.ServerURL)
		assert.Equal(t, 45, config.TimeoutSeconds)
		assert.Equal(t, 45*time.Second, config.Timeout)
		assert.Equal(t, "json", config.DefaultFormat)
		assert.Equal(t, "warn", config.LogLevel)
	})

	t.Run("no config path and no standard file - use hardcoded defaults", func(t *testing.T) {
		// Создаем временную директорию без файла конфигурации
		tempDir, err := os.MkdirTemp("", "test_cli_configs_empty")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Загружаем конфиг без указания пути
		config, err := LoadConfig("")
		require.NoError(t, err)

		expected := DefaultConfig()
		assert.Equal(t, expected.ServerURL, config.ServerURL)
		assert.Equal(t, expected.TimeoutSeconds, config.TimeoutSeconds)
		assert.Equal(t, expected.Timeout, config.Timeout)
		assert.Equal(t, expected.DefaultFormat, config.DefaultFormat)
		assert.Equal(t, expected.LogLevel, config.LogLevel)
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *CLIConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 30,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			wantErr: false,
		},
		{
			name: "valid config with json format",
			config: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 30,
				DefaultFormat:  "json",
				LogLevel:       "debug",
			},
			wantErr: false,
		},
		{
			name: "empty server url",
			config: &CLIConfig{
				ServerURL:      "",
				TimeoutSeconds: 30,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			wantErr: true,
			errMsg:  "translation_server_url не может быть пустым",
		},
		{
			name: "invalid timeout - zero",
			config: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 0,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			wantErr: true,
			errMsg:  "таймаут должен быть положительным",
		},
		{
			name: "invalid timeout - negative",
			config: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: -10,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			wantErr: true,
			errMsg:  "таймаут должен быть положительным",
		},
		{
			name: "invalid default format",
			config: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 30,
				DefaultFormat:  "xml",
				LogLevel:       "info",
			},
			wantErr: true,
			errMsg:  "default_format должен быть 'table' или 'json'",
		},
		{
			name: "invalid log level",
			config: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 30,
				DefaultFormat:  "table",
				LogLevel:       "invalid",
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

	assert.Equal(t, "http://localhost:8081", config.ServerURL)
	assert.Equal(t, 30, config.TimeoutSeconds)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "table", config.DefaultFormat)
	assert.Equal(t, "info", config.LogLevel)
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
			config := &CLIConfig{
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
	tempDir, err := os.MkdirTemp("", "test_cli_config_load")
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
		configFile, err := os.Create("configs/cli.json")
		require.NoError(t, err)
		configFile.WriteString("{}")
		configFile.Close()

		result := tryToLoadStandartConfigFile()
		assert.Equal(t, "configs/cli.json", result)
	})
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expected    *CLIConfig
		expectError bool
	}{
		{
			name: "full config",
			jsonData: `{
				"translation_server_url": "http://custom:8080",
				"timeout": 45,
				"default_format": "json",
				"log_level": "error"
			}`,
			expected: &CLIConfig{
				ServerURL:      "http://custom:8080",
				TimeoutSeconds: 45,
				Timeout:        45 * time.Second,
				DefaultFormat:  "json",
				LogLevel:       "error",
			},
			expectError: false,
		},
		{
			name:     "empty config - use defaults",
			jsonData: `{}`,
			expected: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 30,
				Timeout:        30 * time.Second,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			expectError: false,
		},
		{
			name: "partial config - override some values",
			jsonData: `{
				"translation_server_url": "http://new:9090",
				"timeout": 60
			}`,
			expected: &CLIConfig{
				ServerURL:      "http://new:9090",
				TimeoutSeconds: 60,
				Timeout:        60 * time.Second,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			expectError: false,
		},
		{
			name: "zero timeout - use default",
			jsonData: `{
				"timeout": 0
			}`,
			expected: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 30,
				Timeout:        30 * time.Second,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			expectError: false,
		},
		{
			name: "negative timeout - use default",
			jsonData: `{
				"timeout": -10
			}`,
			expected: &CLIConfig{
				ServerURL:      "http://localhost:8081",
				TimeoutSeconds: 30,
				Timeout:        30 * time.Second,
				DefaultFormat:  "table",
				LogLevel:       "info",
			},
			expectError: false,
		},
		{
			name:        "invalid json",
			jsonData:    `invalid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var config CLIConfig
			err := json.Unmarshal([]byte(tt.jsonData), &config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected.ServerURL, config.ServerURL)
				assert.Equal(t, tt.expected.TimeoutSeconds, config.TimeoutSeconds)
				assert.Equal(t, tt.expected.Timeout, config.Timeout)
				assert.Equal(t, tt.expected.DefaultFormat, config.DefaultFormat)
				assert.Equal(t, tt.expected.LogLevel, config.LogLevel)
			}
		})
	}
}

// Edge cases тесты
func TestLoadConfigEdgeCases(t *testing.T) {
	t.Run("config with very large timeout", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "large_timeout.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		configData := `{
			"translation_server_url": "http://localhost:8081",
			"timeout": 3600,
			"default_format": "table",
			"log_level": "info"
		}`
		_, err = tmpFile.Write([]byte(configData))
		require.NoError(t, err)
		tmpFile.Close()

		config, err := LoadConfig(tmpFile.Name())
		require.NoError(t, err)
		assert.Equal(t, 3600, config.TimeoutSeconds)
		assert.Equal(t, 3600*time.Second, config.Timeout)
	})

	t.Run("config with special characters in URL", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "special_url.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		configData := `{
			"translation_server_url": "http://localhost:8081/path?param=value",
			"timeout": 30,
			"default_format": "table",
			"log_level": "info"
		}`
		_, err = tmpFile.Write([]byte(configData))
		require.NoError(t, err)
		tmpFile.Close()

		config, err := LoadConfig(tmpFile.Name())
		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8081/path?param=value", config.ServerURL)
	})
}

// Бенчмарк тесты
func BenchmarkValidateConfig(b *testing.B) {
	cfg := &CLIConfig{
		ServerURL:      "http://localhost:8081",
		TimeoutSeconds: 30,
		DefaultFormat:  "table",
		LogLevel:       "info",
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

func BenchmarkLoadConfig(b *testing.B) {
	// Создаем временный файл для бенчмарка
	tmpFile, err := os.CreateTemp("", "bench_config.json")
	require.NoError(b, err)
	defer os.Remove(tmpFile.Name())

	configData := `{
		"translation_server_url": "http://localhost:8081",
		"timeout": 30,
		"default_format": "table",
		"log_level": "info"
	}`
	_, err = tmpFile.Write([]byte(configData))
	require.NoError(b, err)
	tmpFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadConfig(tmpFile.Name())
	}
}

func BenchmarkUnmarshalJSON(b *testing.B) {
	jsonData := []byte(`{
		"translation_server_url": "http://localhost:8081",
		"timeout": 30,
		"default_format": "table",
		"log_level": "info"
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var config CLIConfig
		_ = json.Unmarshal(jsonData, &config)
	}
}
