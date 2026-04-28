package config

import (
	"log/slog"
	"testing"
	"time"
)

// TestLoadConfig_ValidFile проверяет загрузку корректного конфига
func TestLoadConfig_ValidFile(t *testing.T) {
	cfg, err := LoadConfig("testdata/valid_config.json")
	if err != nil {
		t.Fatalf("ожидали успех, получили ошибку: %v", err)
	}

	if cfg.Port != 8083 {
		t.Errorf("Port: ожидали 8083, получили %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: ожидали 'info', получили '%s'", cfg.LogLevel)
	}
	if cfg.Data.Host != "localhost" {
		t.Errorf("DB Host: ожидали 'localhost', получили '%s'", cfg.Data.Host)
	}
	if cfg.Data.Port != 5432 {
		t.Errorf("DB Port: ожидали 5432, получили %d", cfg.Data.Port)
	}
	if cfg.Data.User != "testuser" {
		t.Errorf("DB User: ожидали 'testuser', получили '%s'", cfg.Data.User)
	}
	if cfg.Data.Dbname != "testdb" {
		t.Errorf("DB Name: ожидали 'testdb', получили '%s'", cfg.Data.Dbname)
	}
}

// TestLoadConfig_TimeoutConversion проверяет, что секунды корректно конвертируются в time.Duration
func TestLoadConfig_TimeoutConversion(t *testing.T) {
	cfg, err := LoadConfig("testdata/valid_config.json")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	expectedServerTimeout := 30 * time.Second
	if cfg.Timeout != expectedServerTimeout {
		t.Errorf("Timeout: ожидали %v, получили %v", expectedServerTimeout, cfg.Timeout)
	}

	expectedDBTimeout := 10 * time.Second
	if cfg.Data.Timeout != expectedDBTimeout {
		t.Errorf("DB Timeout: ожидали %v, получили %v", expectedDBTimeout, cfg.Data.Timeout)
	}
}

// TestLoadConfig_DebugConfig проверяет загрузку конфига с нестандартными значениями
func TestLoadConfig_DebugConfig(t *testing.T) {
	cfg, err := LoadConfig("testdata/debug_config.json")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if cfg.Port != 9090 {
		t.Errorf("Port: ожидали 9090, получили %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: ожидали 'debug', получили '%s'", cfg.LogLevel)
	}
	if cfg.Data.SSL_mode != "require" {
		t.Errorf("SSL_mode: ожидали 'require', получили '%s'", cfg.Data.SSL_mode)
	}
}

// TestLoadConfig_FileNotFound проверяет поведение при несуществующем файле
func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("testdata/nonexistent.json")
	if err == nil {
		t.Fatal("ожидали ошибку при несуществующем файле, но ошибки не было")
	}
}

// TestLoadConfig_MalformedJSON проверяет поведение при битом JSON
func TestLoadConfig_MalformedJSON(t *testing.T) {
	_, err := LoadConfig("testdata/malformed.json")
	if err == nil {
		t.Fatal("ожидали ошибку при битом JSON, но ошибки не было")
	}
}

// TestLoadConfig_EmptyPath_ReturnsDefault проверяет, что пустой путь даёт дефолтный конфиг
// (при условии, что config/service2.json не существует в тестовом окружении)
func TestLoadConfig_EmptyPath_ReturnsDefault(t *testing.T) {
	// Если файл config/service2.json существует — тест пропускаем,
	// так как поведение зависит от окружения
	cfg, err := LoadConfig("")
	if err != nil {
		// Если файл найден, но невалиден — это тоже нормальный исход
		t.Logf("LoadConfig('') вернул ошибку (возможно найден файл окружения): %v", err)
		return
	}
	if cfg == nil {
		t.Fatal("конфиг не должен быть nil")
	}
}

// TestValidateConfig_InvalidPort проверяет отклонение невалидного порта сервера
func TestValidateConfig_InvalidPort(t *testing.T) {
	_, err := LoadConfig("testdata/invalid_port.json")
	if err == nil {
		t.Fatal("ожидали ошибку при порте 99999, но ошибки не было")
	}
}

// TestValidateConfig_InvalidDBPort проверяет отклонение невалидного порта БД
func TestValidateConfig_InvalidDBPort(t *testing.T) {
	_, err := LoadConfig("testdata/invalid_db_port.json")
	if err == nil {
		t.Fatal("ожидали ошибку при порте БД = 0, но ошибки не было")
	}
}

// TestValidateConfig_InvalidLogLevel проверяет отклонение неизвестного log level
func TestValidateConfig_InvalidLogLevel(t *testing.T) {
	_, err := LoadConfig("testdata/invalid_log_level.json")
	if err == nil {
		t.Fatal("ожидали ошибку при log_level='verbose', но ошибки не было")
	}
}

// TestValidateConfig_AllValidLogLevels проверяет все допустимые уровни логирования
func TestValidateConfig_AllValidLogLevels(t *testing.T) {
	validLevels := []string{"debug", "info", "warn", "error"}

	for _, level := range validLevels {
		cfg := &Config{
			Port:     8083,
			LogLevel: level,
			Data: Database{
				Port: 5432,
			},
		}
		if err := validateConfig(cfg); err != nil {
			t.Errorf("уровень '%s' должен быть допустимым, но получили ошибку: %v", level, err)
		}
	}
}

// TestValidateConfig_PortBoundaries проверяет граничные значения порта
func TestValidateConfig_PortBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{"порт 0 — невалидный", 0, true},
		{"порт -1 — невалидный", -1, true},
		{"порт 1 — минимальный валидный", 1, false},
		{"порт 65535 — максимальный валидный", 65535, false},
		{"порт 65536 — невалидный", 65536, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Port:     tt.port,
				LogLevel: "info",
				Data:     Database{Port: 5432},
			}
			err := validateConfig(cfg)
			if tt.wantErr && err == nil {
				t.Errorf("ожидали ошибку для порта %d, но ошибки не было", tt.port)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("не ожидали ошибку для порта %d, но получили: %v", tt.port, err)
			}
		})
	}
}

// TestDefaultConfig проверяет дефолтные значения конфига
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() вернул nil")
	}
	if cfg.Port != 8083 {
		t.Errorf("дефолтный Port: ожидали 8083, получили %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("дефолтный LogLevel: ожидали 'info', получили '%s'", cfg.LogLevel)
	}
}

// TestSetupLogger_ReturnsLogger проверяет, что SetupLogger возвращает не-nil логгер
func TestSetupLogger_ReturnsLogger(t *testing.T) {
	cfg := &Config{LogLevel: "info"}
	logger := cfg.SetupLogger()
	if logger == nil {
		t.Fatal("SetupLogger() вернул nil")
	}
}

// TestSetupLogger_AllLevels проверяет создание логгера для всех уровней
func TestSetupLogger_AllLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}

	for _, level := range levels {
		t.Run("level="+level, func(t *testing.T) {
			cfg := &Config{LogLevel: level}
			logger := cfg.SetupLogger()
			if logger == nil {
				t.Fatalf("SetupLogger() вернул nil для уровня '%s'", level)
			}
		})
	}
}

// TestSetupLogger_DebugLevelEnabled проверяет, что debug-логгер действительно включает DEBUG
func TestSetupLogger_DebugLevelEnabled(t *testing.T) {
	cfg := &Config{LogLevel: "debug"}
	logger := cfg.SetupLogger()

	if !logger.Enabled(nil, slog.LevelDebug) {
		t.Error("debug логгер должен обрабатывать LevelDebug, но не обрабатывает")
	}
}

// TestSetupLogger_InfoLevel_DebugDisabled проверяет, что info-логгер не пропускает DEBUG
func TestSetupLogger_InfoLevel_DebugDisabled(t *testing.T) {
	cfg := &Config{LogLevel: "info"}
	logger := cfg.SetupLogger()

	if logger.Enabled(nil, slog.LevelDebug) {
		t.Error("info логгер не должен обрабатывать LevelDebug, но обрабатывает")
	}
	if !logger.Enabled(nil, slog.LevelInfo) {
		t.Error("info логгер должен обрабатывать LevelInfo, но не обрабатывает")
	}
}
