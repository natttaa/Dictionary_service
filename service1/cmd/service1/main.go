package main

import (
	"flag"
	"log"
	"log/slog"
	"service1/cmd/service1/config"
	"service1/cmd/service1/server"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	config, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	s := server.NewServer(config)

	if err := s.Start(); err != nil {
		s.Logger.Error("Ошибка запуска сервера",
			slog.Any("err", err),
		)
	}
}
