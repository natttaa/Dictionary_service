package server

import (
	"dictionary-service/models"
	"encoding/json"
	"log/slog"
	"net/http"
)

// handleTranslate обрабатывает запросы на перевод слова
// POST /api/v1/translate
// Принимает: {"source_lang": "ru", "target_lang": "en", "word": "Собака"}
// Возвращает: {"translation": "Dog"}
func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только POST метод", http.StatusMethodNotAllowed)
		return
	}

	var req models.TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("Ошибка декодирования запроса", slog.Any("error", err))
		s.writeError(w, "INVALID_JSON", "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Валидация полей
	if req.SourceLang == "" || req.TargetLang == "" || req.Word == "" {
		s.writeError(w, "MISSING_PARAMS", "Поля source_lang, target_lang и word обязательны", http.StatusBadRequest)
		return
	}

	// Проверяем, что языки поддерживаются
	if !isValidLang(req.SourceLang) || !isValidLang(req.TargetLang) {
		s.writeError(w, "UNSUPPORTED_LANG", "Поддерживаются языки: ru, en, zh", http.StatusBadRequest)
		return
	}

	s.logger.Debug("Запрос на перевод",
		slog.String("source_lang", req.SourceLang),
		slog.String("target_lang", req.TargetLang),
		slog.String("word", req.Word),
	)

	// Динамически подставляем имена колонок (безопасно — только из whitelist isValidLang)
	sourceCol := langToColumn(req.SourceLang)
	targetCol := langToColumn(req.TargetLang)

	query := `SELECT ` + targetCol + ` FROM dictionary.dictionary_table WHERE LOWER(` + sourceCol + `) = LOWER($1)`

	var translation string
	err := s.db.QueryRow(query, req.Word).Scan(&translation)
	if err != nil {
		s.logger.Warn("Слово не найдено",
			slog.String("word", req.Word),
			slog.Any("error", err),
		)
		s.writeError(w, "WORD_NOT_FOUND", "Слово не найдено в словаре", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.TranslateResponse{
		Translation: translation,
	})
}

// handleLanguages возвращает список поддерживаемых языков
// GET /api/v1/languages
func (s *Server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только GET метод", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Запрос списка языков")

	response := models.LanguagesResponse{
		Languages: []models.LanguageInfo{
			{Code: "ru", Name: "Русский"},
			{Code: "en", Name: "English"},
			{Code: "zh", Name: "中文 (Chinese)"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleTopics возвращает список уникальных тем из БД
// GET /api/v1/topics
func (s *Server) handleTopics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только GET метод", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Запрос списка тем")

	rows, err := s.db.Query(`SELECT DISTINCT category FROM dictionary.dictionary_table ORDER BY category`)
	if err != nil {
		s.logger.Error("Ошибка запроса тем", slog.Any("error", err))
		s.writeError(w, "INTERNAL_ERROR", "Ошибка получения тем", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var topics []string
	for rows.Next() {
		var topic string
		if err := rows.Scan(&topic); err != nil {
			s.logger.Error("Ошибка сканирования темы", slog.Any("error", err))
			continue
		}
		topics = append(topics, topic)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.TopicsResponse{Topics: topics})
}

// handleHealth проверяет состояние сервиса и соединения с БД
// GET /api/v1/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только GET метод", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Запрос проверки здоровья")

	if err := s.db.Ping(); err != nil {
		s.logger.Warn("БД недоступна", slog.Any("error", err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(models.HealthResponse{
			Status:   "unhealthy",
			Service2: "database unavailable",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.HealthResponse{
		Status:   "healthy",
		Service2: "ok",
	})
}

// writeError записывает JSON-ошибку в ответ
func (s *Server) writeError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := models.TranslateResponse{
		Error: &models.Error{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Ошибка кодирования ошибки", slog.Any("error", err))
	}
}

// isValidLang проверяет, что язык поддерживается
func isValidLang(lang string) bool {
	switch lang {
	case "ru", "en", "zh":
		return true
	}
	return false
}

// langToColumn возвращает имя колонки в БД для языка
func langToColumn(lang string) string {
	switch lang {
	case "ru":
		return "russian"
	case "en":
		return "english"
	case "zh":
		return "chinese"
	}
	return ""
}
