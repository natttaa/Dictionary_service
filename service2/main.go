package main

import (
	"dictionary-service/config"
	"dictionary-service/server"
	"log"
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

func main() {
	// Шаг 1: загрузка конфигурации
	cfg, err := config.LoadConfig("config/service2.json")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	s := server.NewServer(cfg)
	s.Start(cfg)

}
