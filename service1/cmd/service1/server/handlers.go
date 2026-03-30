package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"service1/models"
)

// handleTranslate обрабатывает запросы на перевод
func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	// Проверка метода
	if r.Method != http.MethodPost {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только POST метод", http.StatusMethodNotAllowed)
		return
	}

	// Декодирование запроса
	var req models.TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("Ошибка декодирования запроса", slog.Any("error", err))
		s.writeError(w, "INVALID_JSON", "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	// Валидация запроса
	if err := s.validateTranslateRequest(&req); err != nil {
		s.logger.Warn("Ошибка валидации запроса", slog.Any("error", err))
		s.writeError(w, "VALIDATION_FAILED", err.Error(), http.StatusBadRequest)
		return
	}

	s.logger.Debug("Запрос на перевод",
		slog.String("source_lang", req.SourceLang),
		slog.String("target_lang", req.TargetLang),
		slog.String("word", req.Word),
	)

	// Перенаправление запроса к словарному сервису
	resp, err := s.forwardToDictionary(r)
	if err != nil {
		s.logger.Error("Ошибка при обращении к словарному сервису", slog.Any("error", err))
		s.writeError(w, "SERVICE_UNAVAILABLE", "Словарный сервис недоступен", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Чтение тела ответа
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Ошибка чтения ответа", slog.Any("error", err))
		s.writeError(w, "INTERNAL_ERROR", "Ошибка чтения ответа", http.StatusInternalServerError)
		return
	}

	// Копирование заголовков ответа
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	// Запись тела ответа
	if _, err := w.Write(bodyBytes); err != nil {
		s.logger.Error("Ошибка записи ответа", slog.Any("error", err))
	}
}

// handleLanguages обрабатывает запросы на получение списка языков
func (s *Server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	// Проверка метода
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только GET метод", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Запрос списка языков")

	// Попытка получить данные от словарного сервиса
	resp, err := s.httpClient.Get(s.config.DictionaryServiceURL + "/api/v1/languages")
	if err != nil {
		// Возвращаем мок-данные, если словарный сервис недоступен
		s.logger.Warn("Словарный сервис недоступен, возвращаем мок-данные", slog.Any("error", err))
		response := models.LanguagesResponse{
			Languages: []models.LanguageInfo{
				{Code: "en", Name: "English"},
				{Code: "ru", Name: "Russian"},
				{Code: "zh", Name: "Chinese"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.logger.Error("Ошибка кодирования мок-ответа", slog.Any("error", err))
		}
		return
	}
	defer resp.Body.Close()

	// Перенаправление ответа от словарного сервиса
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		s.logger.Error("Ошибка копирования ответа", slog.Any("error", err))
	}
}

// handleTopics обрабатывает запросы на получение списка тем
func (s *Server) handleTopics(w http.ResponseWriter, r *http.Request) {
	// Проверка метода
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только GET метод", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Запрос списка тем")

	// Попытка получить данные от словарного сервиса
	resp, err := s.httpClient.Get(s.config.DictionaryServiceURL + "/api/v1/topics")
	if err != nil {
		// Возвращаем мок-данные, если словарный сервис недоступен
		s.logger.Warn("Словарный сервис недоступен, возвращаем мок-темы", slog.Any("error", err))
		response := models.TopicsResponse{
			Topics: []string{"animals", "food", "greetings", "family", "colors"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.logger.Error("Ошибка кодирования мок-ответа", slog.Any("error", err))
		}
		return
	}
	defer resp.Body.Close()

	// Перенаправление ответа от словарного сервиса
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		s.logger.Error("Ошибка копирования ответа", slog.Any("error", err))
	}
}

// handleHealth обрабатывает запросы проверки здоровья
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Проверка метода
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Разрешен только GET метод", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Debug("Запрос проверки здоровья")

	// Проверка доступности словарного сервиса
	resp, err := s.httpClient.Get(s.config.DictionaryServiceURL + "/api/v1/health")
	if err != nil {
		s.logger.Warn("Проверка здоровья не удалась: словарный сервис недоступен", slog.Any("error", err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		response := models.HealthResponse{
			Status:   "unhealthy",
			Service2: "unavailable",
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			s.logger.Error("Ошибка кодирования ответа", slog.Any("error", err))
		}
		return
	}
	defer resp.Body.Close()

	// Декодирование ответа от словарного сервиса
	var dictHealth models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&dictHealth); err != nil {
		s.logger.Error("Ошибка декодирования ответа словарного сервиса", slog.Any("error", err))
		dictHealth.Status = "unknown"
	}

	// Формирование ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := models.HealthResponse{
		Status:   "healthy",
		Service2: dictHealth.Status,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("Ошибка кодирования ответа", slog.Any("error", err))
	}
}

// forwardToDictionary перенаправляет запрос к словарному сервису
func (s *Server) forwardToDictionary(r *http.Request) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", s.config.DictionaryServiceURL, r.URL.Path)

	// Чтение тела оригинального запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать тело запроса: %w", err)
	}

	// Создание нового запроса с тем же телом
	req, err := http.NewRequest(r.Method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("не удалось создать запрос: %w", err)
	}

	// Копирование заголовков
	req.Header = r.Header.Clone()

	// Выполнение запроса
	return s.httpClient.Do(req)
}

// validateTranslateRequest валидирует запрос на перевод
func (s *Server) validateTranslateRequest(req *models.TranslateRequest) error {
	// Проверка обязательных полей
	if req.Word == "" {
		return fmt.Errorf("слово не может быть пустым")
	}
	if req.SourceLang == "" {
		return fmt.Errorf("исходный язык не может быть пустым")
	}
	if req.TargetLang == "" {
		return fmt.Errorf("целевой язык не может быть пустым")
	}

	// Проверка поддерживаемых языков
	supportedLangs := map[string]bool{"zh": true, "ru": true, "en": true}
	if !supportedLangs[req.SourceLang] {
		return fmt.Errorf("неподдерживаемый исходный язык: %s", req.SourceLang)
	}
	if !supportedLangs[req.TargetLang] {
		return fmt.Errorf("неподдерживаемый целевой язык: %s", req.TargetLang)
	}

	return nil
}

// writeError записывает ошибку в ответ
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
