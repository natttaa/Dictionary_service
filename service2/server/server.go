package server

import (
	"context"
	"dictionary-service/config"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Server представляет основной сервер приложения
type Server struct {
	config *config.Config
	db     *pgxpool.Pool
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
	defer s.db.Close()

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

// connectToDatabase устанавливает подключение к PostgreSQL через pgxpool
func (s *Server) connectToDatabase(cfg *config.Config) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		cfg.Data.Host,
		cfg.Data.Port,
		cfg.Data.User,
		cfg.Data.Password,
		cfg.Data.Dbname,
		cfg.Data.SSL_mode,
		cfg.Data.TimeoutSeconds,
	)

	log.Println("Установка подключения к PostgreSQL...")

	poolCfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфигурации БД: %w", err)
	}

	poolCfg.MaxConns = 25
	poolCfg.MinConns = 5
	poolCfg.MaxConnLifetime = 5 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Data.Timeout)
	defer cancel()

	pool, err := pgxpool.ConnectConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания пула соединений: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("БД не отвечает: %w", err)
	}

	log.Println("Подключение к PostgreSQL успешно установлено")
	return pool, nil
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
