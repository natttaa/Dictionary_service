package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Port     int    `json:"port"`
	LogLevel string `json:"log_level"`
	Database struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		DBName   string `json:"dbname"`
		SSLMode  string `json:"ssl_mode"`
	} `json:"database"`
}

// loadConfig читает настройки конфигурации из JSON файла
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения файла конфигурации %s: %w", filename, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("Ошибка парсинга файла конфигурации: %w", err)
	}
	return &cfg, nil
}

func main() {
	// Шаг 1: загрузка конфигурации
	cfg, err := loadConfig("configs/service2.json")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Шаг 2: сборка строки подключения к БД
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	log.Println("Установка подключения к PostgreSQL...")

	// Шаг 3: установка соединения с БД
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка открытия БД: %v", err)
	}
	defer db.Close()

	// Шаг 4: проверка, что БД отвечает
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("БД не отвечает: %v", err)
	}
	log.Println("Подключение к PostgreSQL успешно установлено")

	// Шаг 5: настройка HTTP сервера
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "сервер работает",
			"db":     "подключена",
		})
	})

	// Шаг 6: запуск сервера для работы
	portStr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Сервис-2 запущен на порту %s", portStr)
	log.Fatal(http.ListenAndServe(portStr, mux))
}
