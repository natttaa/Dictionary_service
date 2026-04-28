package server

import (
	"dictionary-service/models"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v4"
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

	if req.SourceLang == "" || req.TargetLang == "" || req.Word == "" {
		s.writeError(w, "MISSING_PARAMS", "Поля source_lang, target_lang и word обязательны", http.StatusBadRequest)
		return
	}

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
	err := s.db.QueryRow(r.Context(), query, req.Word).Scan(&translation)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.writeError(w, "WORD_NOT_FOUND", "Слово не найдено в словаре", http.StatusNotFound)
		} else {
			s.logger.Warn("Ошибка запроса перевода",
				slog.String("word", req.Word),
				slog.Any("error", err),
			)
			s.writeError(w, "INTERNAL_ERROR", "Ошибка получения перевода", http.StatusInternalServerError)
		}
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

	rows, err := s.db.Query(r.Context(), `SELECT DISTINCT category FROM dictionary.dictionary_table ORDER BY category`)
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

	if err := rows.Err(); err != nil {
		s.logger.Error("Ошибка итерации тем", slog.Any("error", err))
		s.writeError(w, "INTERNAL_ERROR", "Ошибка получения тем", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.TopicsResponse{Topics: topics})
}

// handleTopicWords возвращает слова по теме для указанных языков
// POST /api/v1/topics/words
// Принимает: {"topic": "animals", "languages": ["ru", "en"]}
// Возвращает: {"topic": "animals", "words": [{"translations": {"ru": "Собака", "en": "Dog"}}]}
func (s *Server) handleTopicWords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только POST метод", http.StatusMethodNotAllowed)
		return
	}

	var req models.TopicWordsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("Ошибка декодирования запроса", slog.Any("error", err))
		s.writeError(w, "INVALID_JSON", "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if req.Topic == "" || len(req.Languages) == 0 {
		s.writeError(w, "MISSING_PARAMS", "Поля topic и languages обязательны", http.StatusBadRequest)
		return
	}

	for _, lang := range req.Languages {
		if !isValidLang(lang) {
			s.writeError(w, "UNSUPPORTED_LANG", "Поддерживаются языки: ru, en, zh", http.StatusBadRequest)
			return
		}
	}

	s.logger.Debug("Запрос слов по теме",
		slog.String("topic", req.Topic),
		slog.Any("languages", req.Languages),
	)

	// Формируем список колонок из whitelist — SQL-инъекция невозможна
	cols := make([]string, len(req.Languages))
	for i, lang := range req.Languages {
		cols[i] = langToColumn(lang)
	}

	query := `SELECT ` + strings.Join(cols, ", ") +
		` FROM dictionary.dictionary_table WHERE LOWER(category) = LOWER($1) ORDER BY ` + cols[0]

	rows, err := s.db.Query(r.Context(), query, req.Topic)
	if err != nil {
		s.logger.Error("Ошибка запроса слов по теме", slog.Any("error", err))
		s.writeError(w, "INTERNAL_ERROR", "Ошибка получения слов по теме", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var words []models.WordEntry
	for rows.Next() {
		vals := make([]any, len(req.Languages))
		ptrs := make([]any, len(req.Languages))
		for i := range ptrs {
			ptrs[i] = &vals[i]
		}

		if err := rows.Scan(ptrs...); err != nil {
			s.logger.Error("Ошибка сканирования строки", slog.Any("error", err))
			continue
		}

		translations := make(map[string]string, len(req.Languages))
		for i, lang := range req.Languages {
			if s, ok := vals[i].(string); ok {
				translations[lang] = s
			}
		}

		words = append(words, models.WordEntry{Translations: translations})
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("Ошибка итерации слов", slog.Any("error", err))
		s.writeError(w, "INTERNAL_ERROR", "Ошибка получения слов по теме", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.TopicWordsResponse{
		Topic: req.Topic,
		Words: words,
	})
}

// handleCheckTranslation проверяет правильность перевода пользователя
// POST /api/v1/check-translation
// Принимает: {"original": "Собака", "translation": "Dog", "source_lang": "ru"}
// Возвращает: {"translations": {"en": "Dog", "zh": "狗"}}
func (s *Server) handleCheckTranslation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только POST метод", http.StatusMethodNotAllowed)
		return
	}

	var req models.CheckTranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("Ошибка декодирования запроса", slog.Any("error", err))
		s.writeError(w, "INVALID_JSON", "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if req.Original == "" || req.SourceLang == "" {
		s.writeError(w, "MISSING_PARAMS", "Поля original и source_lang обязательны", http.StatusBadRequest)
		return
	}

	if !isValidLang(req.SourceLang) {
		s.writeError(w, "UNSUPPORTED_LANG", "Поддерживаются языки: ru, en, zh", http.StatusBadRequest)
		return
	}

	s.logger.Debug("Запрос на проверку перевода",
		slog.String("original", req.Original),
		slog.String("source_lang", req.SourceLang),
	)

	sourceCol := langToColumn(req.SourceLang)

	// Выбираем переводы на все остальные языки (колонки из whitelist — безопасно)
	allLangs := []string{"ru", "en", "zh"}
	var targetLangs []string
	var targetCols []string
	for _, lang := range allLangs {
		if lang != req.SourceLang {
			targetLangs = append(targetLangs, lang)
			targetCols = append(targetCols, langToColumn(lang))
		}
	}

	query := `SELECT ` + strings.Join(targetCols, ", ") +
		` FROM dictionary.dictionary_table WHERE LOWER(` + sourceCol + `) = LOWER($1)`

	vals := make([]any, len(targetLangs))
	ptrs := make([]any, len(targetLangs))
	for i := range ptrs {
		ptrs[i] = &vals[i]
	}

	err := s.db.QueryRow(r.Context(), query, req.Original).Scan(ptrs...)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.writeError(w, "WORD_NOT_FOUND", "Слово не найдено в словаре", http.StatusNotFound)
		} else {
			s.logger.Warn("Ошибка запроса проверки перевода",
				slog.String("word", req.Original),
				slog.Any("error", err),
			)
			s.writeError(w, "INTERNAL_ERROR", "Ошибка проверки перевода", http.StatusInternalServerError)
		}
		return
	}

	translations := make(map[string]string, len(targetLangs))
	for i, lang := range targetLangs {
		if s, ok := vals[i].(string); ok {
			translations[lang] = s
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.CheckTranslationResponse{
		Translations: translations,
	})
}

// handleHealth проверяет состояние сервиса и соединения с БД
// GET /api/v1/health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только GET метод", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Запрос проверки здоровья")

	if err := s.db.Ping(r.Context()); err != nil {
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
