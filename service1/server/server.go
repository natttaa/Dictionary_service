package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// Server представляет основной сервер приложения
type Server struct {
	config     *Config
	httpClient *http.Client
	logger     *slog.Logger
}

// NewServer создает новый экземпляр сервера
func NewServer(config *Config) *Server {
	// Настраиваем логгер на основе конфигурации
	logger := config.SetupLogger()

	return &Server{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// Start запускает HTTP сервер
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Регистрация маршрутов
	mux.HandleFunc("/api/v1/translate", s.handleTranslate)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/languages", s.handleLanguages)
	mux.HandleFunc("/api/v1/topics", s.handleTopics)

	addr := fmt.Sprintf(":%d", s.config.Port)
	s.logger.Info("Запуск сервера",
		slog.String("addr", addr),
		slog.String("dictionary_service", s.config.DictionaryServiceURL),
		slog.String("log_level", s.config.LogLevel),
		slog.Duration("timeout", s.config.Timeout),
	)

	return http.ListenAndServe(addr, s.loggerMiddleware(mux))
}

// loggerMiddleware middleware для логирования всех входящих запросов
func (s *Server) loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Логируем входящий запрос
		s.logger.Info("Входящий запрос",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		next.ServeHTTP(w, r)

		// Логируем завершение запроса
		s.logger.Info("Запрос обработан",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
