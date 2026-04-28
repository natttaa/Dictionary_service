package server

import (
	"database/sql"
	"dictionary-service/config"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

// Server представляет основной сервер приложения
type Server struct {
	config *config.Config
	db     *sql.DB
	logger *slog.Logger
}

// NewServer создает новый экземпляр сервера
func NewServer(cfg *config.Config) *Server {
	logger := cfg.SetupLogger()

	return &Server{
		config: cfg,
		logger: logger,
	}
}

// Start запускает HTTP сервер
func (s *Server) Start(cfg *config.Config) error {
	db, err := s.connectToDatabase(cfg)
	if err != nil {
		return fmt.Errorf("ошибка подключения к БД: %w", err)
	}
	s.db = db

	mux := http.NewServeMux()

	// Регистрация маршрутов
	mux.HandleFunc("/api/v1/translate", s.handleTranslate)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/languages", s.handleLanguages)
	mux.HandleFunc("/api/v1/topics", s.handleTopics)
	mux.HandleFunc("/api/v1/topics/words", s.handleTopicWords)
	mux.HandleFunc("/api/v1/check-translation", s.handleCheckTranslation)

	addr := fmt.Sprintf(":%d", s.config.Port)
	s.logger.Info("Запуск сервера",
		slog.String("addr", addr),
		slog.String("log_level", s.config.LogLevel),
		slog.Duration("timeout", s.config.Timeout),
	)

	return http.ListenAndServe(addr, s.loggerMiddleware(mux))
}

// connectToDatabase устанавливает подключение к PostgreSQL и возвращает *sql.DB
func (s *Server) connectToDatabase(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Data.Host,
		cfg.Data.Port,
		cfg.Data.User,
		cfg.Data.Password,
		cfg.Data.Dbname,
		cfg.Data.SSL_mode,
	)

	log.Println("Установка подключения к PostgreSQL...")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия БД: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("БД не отвечает: %w", err)
	}

	log.Println("Подключение к PostgreSQL успешно установлено")
	return db, nil
}

// loggerMiddleware middleware для логирования всех входящих запросов
func (s *Server) loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		s.logger.Info("Входящий запрос",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		next.ServeHTTP(w, r)

		s.logger.Info("Запрос обработан",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("duration", time.Since(start)),
		)
	})
}
