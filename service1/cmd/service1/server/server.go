package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"service1/cmd/service1/config"
	"time"
)

// Server представляет основной сервер приложения
type Server struct {
	config     *config.Config
	httpClient *http.Client
	httpServer *http.Server
	Logger     *slog.Logger
}

// NewServer создает новый экземпляр сервера
func NewServer(config *config.Config) *Server {
	// Настраиваем логгер на основе конфигурации
	logger := config.SetupLogger()

	return &Server{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		Logger: logger,
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

	// Создаём HTTP сервер
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.loggerMiddleware(mux),
		ReadTimeout:  10 * time.Second,  // Таймаут на чтение запроса
		WriteTimeout: 30 * time.Second,  // Таймаут на запись ответа
		IdleTimeout:  120 * time.Second, // Таймаут для keep-alive соединений
	}

	s.Logger.Info("Запуск сервера",
		slog.String("addr", addr),
		slog.String("dictionary_service", s.config.DictionaryServiceURL),
		slog.String("log_level", s.config.LogLevel),
		slog.Duration("timeout", s.config.Timeout),
	)

	// Запускаем сервер
	return s.httpServer.ListenAndServe()
}

// Stop останавливает HTTP сервер gracefully
func (s *Server) Stop(ctx context.Context) error {
	s.Logger.Info("Остановка HTTP сервера...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.Logger.Error("Ошибка при остановке HTTP сервера",
			slog.Any("err", err))
		return err
	}

	s.Logger.Info("HTTP сервер остановлен")
	return nil
}

// loggerMiddleware middleware для логирования всех входящих запросов
func (s *Server) loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Логируем входящий запрос
		s.Logger.Info("Входящий запрос",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("remote_addr", r.RemoteAddr),
		)

		// Создаём responseWriter с захватом статуса для логирования
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)

		// Логируем завершение запроса
		s.Logger.Info("Запрос обработан",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// responseWriter обёртка для захвата HTTP статуса
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
