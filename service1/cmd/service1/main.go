package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"service1/cmd/service1/config"
	"service1/cmd/service1/server"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	config, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	s := server.NewServer(config)

	// Запускаем сервер в горутине
	go func() {
		if err := s.Start(); err != nil {
			// http.ErrServerClosed - это не ошибка, а завершение по сигналу остановки
			if errors.Is(err, http.ErrServerClosed) {
				s.Logger.Info("Сервер остановлен")
			} else {
				s.Logger.Error("Ошибка запуска сервера",
					slog.Any("err", err),
				)
				os.Exit(1)
			}
		}
	}()

	// Ожидание сигнала остановки
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.Logger.Info("Получен сигнал завершения, начинаем graceful shutdown")

	// Контекст с таймаутом 30 секунд на завершение всех операций
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем сервер gracefully
	if err := s.Stop(shutdownCtx); err != nil {
		s.Logger.Error("Ошибка при graceful shutdown",
			slog.Any("err", err),
		)
		return
	}

	s.Logger.Info("Сервер успешно остановлен")
}
