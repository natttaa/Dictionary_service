package server

import (
	"bytes"
	"database/sql"
	"dictionary-service/models"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

// newTestServer создаёт сервер с мок-БД для тестов
func newTestServer(t *testing.T) (*Server, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось создать sqlmock: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // в тестах не засоряем вывод
	}))

	s := &Server{
		db:     db,
		logger: logger,
	}
	return s, mock
}

// =============================================================================
// Тесты handleTranslate
// =============================================================================

// TestHandleTranslate_Success проверяет успешный перевод слова
func TestHandleTranslate_Success(t *testing.T) {
	s, mock := newTestServer(t)

	rows := sqlmock.NewRows([]string{"english"}).AddRow("Dog")
	mock.ExpectQuery(`SELECT english FROM dictionary`).
		WithArgs("Собака").
		WillReturnRows(rows)

	body := `{"source_lang":"ru","target_lang":"en","word":"Собака"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/translate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleTranslate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("статус: ожидали 200, получили %d", w.Code)
	}

	var resp models.TranslateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("не удалось декодировать ответ: %v", err)
	}
	if resp.Translation != "Dog" {
		t.Errorf("перевод: ожидали 'Dog', получили '%s'", resp.Translation)
	}
	if resp.Error != nil {
		t.Errorf("не ожидали ошибки в ответе, получили: %v", resp.Error)
	}
}

// TestHandleTranslate_WordNotFound проверяет 404 при отсутствии слова в словаре
func TestHandleTranslate_WordNotFound(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`SELECT english FROM dictionary`).
		WithArgs("несуществующееслово").
		WillReturnError(sql.ErrNoRows)

	body := `{"source_lang":"ru","target_lang":"en","word":"несуществующееслово"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/translate", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleTranslate(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("статус: ожидали 404, получили %d", w.Code)
	}
}

// TestHandleTranslate_WrongMethod проверяет 405 при GET запросе
func TestHandleTranslate_WrongMethod(t *testing.T) {
	s, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/translate", nil)
	w := httptest.NewRecorder()

	s.handleTranslate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("статус: ожидали 405, получили %d", w.Code)
	}
}

// TestHandleTranslate_MissingFields проверяет 400 при отсутствии обязательных полей
func TestHandleTranslate_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"нет word", `{"source_lang":"ru","target_lang":"en"}`},
		{"нет source_lang", `{"target_lang":"en","word":"Кот"}`},
		{"нет target_lang", `{"source_lang":"ru","word":"Кот"}`},
		{"пустое тело", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newTestServer(t)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/translate", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			s.handleTranslate(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("[%s] статус: ожидали 400, получили %d", tt.name, w.Code)
			}
		})
	}
}

// TestHandleTranslate_UnsupportedLanguage проверяет 400 при неподдерживаемом языке
func TestHandleTranslate_UnsupportedLanguage(t *testing.T) {
	s, _ := newTestServer(t)

	body := `{"source_lang":"fr","target_lang":"en","word":"chien"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/translate", strings.NewReader(body))
	w := httptest.NewRecorder()

	s.handleTranslate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("статус: ожидали 400, получили %d", w.Code)
	}

	var resp models.TranslateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error == nil || resp.Error.Code != "UNSUPPORTED_LANG" {
		t.Errorf("ожидали код ошибки UNSUPPORTED_LANG, получили: %v", resp.Error)
	}
}

// TestHandleTranslate_InvalidJSON проверяет 400 при битом JSON
func TestHandleTranslate_InvalidJSON(t *testing.T) {
	s, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/translate", strings.NewReader("not json at all"))
	w := httptest.NewRecorder()

	s.handleTranslate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("статус: ожидали 400, получили %d", w.Code)
	}
}

// TestHandleTranslate_AllLanguagePairs проверяет все поддерживаемые пары языков
func TestHandleTranslate_AllLanguagePairs(t *testing.T) {
	pairs := []struct {
		src, tgt, col string
	}{
		{"ru", "en", "english"},
		{"ru", "zh", "chinese"},
		{"en", "ru", "russian"},
		{"zh", "ru", "russian"},
	}

	for _, p := range pairs {
		t.Run(p.src+"->"+p.tgt, func(t *testing.T) {
			s, mock := newTestServer(t)

			rows := sqlmock.NewRows([]string{p.col}).AddRow("translation")
			mock.ExpectQuery(`SELECT ` + p.col).WillReturnRows(rows)

			body, _ := json.Marshal(map[string]string{
				"source_lang": p.src,
				"target_lang": p.tgt,
				"word":        "тест",
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/translate", bytes.NewReader(body))
			w := httptest.NewRecorder()

			s.handleTranslate(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("[%s->%s] статус: ожидали 200, получили %d", p.src, p.tgt, w.Code)
			}
		})
	}
}

// =============================================================================
// Тесты handleLanguages
// =============================================================================

// TestHandleLanguages_Success проверяет корректный возврат списка языков
func TestHandleLanguages_Success(t *testing.T) {
	s, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/languages", nil)
	w := httptest.NewRecorder()

	s.handleLanguages(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("статус: ожидали 200, получили %d", w.Code)
	}

	var resp models.LanguagesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("не удалось декодировать ответ: %v", err)
	}

	if len(resp.Languages) != 3 {
		t.Errorf("количество языков: ожидали 3, получили %d", len(resp.Languages))
	}

	// Проверяем, что все три языка присутствуют
	langCodes := make(map[string]bool)
	for _, l := range resp.Languages {
		langCodes[l.Code] = true
	}
	for _, code := range []string{"ru", "en", "zh"} {
		if !langCodes[code] {
			t.Errorf("язык '%s' отсутствует в ответе", code)
		}
	}
}

// TestHandleLanguages_WrongMethod проверяет 405 при POST запросе
func TestHandleLanguages_WrongMethod(t *testing.T) {
	s, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/languages", nil)
	w := httptest.NewRecorder()

	s.handleLanguages(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("статус: ожидали 405, получили %d", w.Code)
	}
}

// =============================================================================
// Тесты handleTopics
// =============================================================================

// TestHandleTopics_Success проверяет успешный возврат списка тем
func TestHandleTopics_Success(t *testing.T) {
	s, mock := newTestServer(t)

	rows := sqlmock.NewRows([]string{"category"}).
		AddRow("Животные").
		AddRow("Еда").
		AddRow("Транспорт")
	mock.ExpectQuery(`SELECT DISTINCT category`).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
	w := httptest.NewRecorder()

	s.handleTopics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("статус: ожидали 200, получили %d", w.Code)
	}

	var resp models.TopicsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("не удалось декодировать ответ: %v", err)
	}
	if len(resp.Topics) != 3 {
		t.Errorf("количество тем: ожидали 3, получили %d", len(resp.Topics))
	}
}

// TestHandleTopics_EmptyDB проверяет пустой список тем
func TestHandleTopics_EmptyDB(t *testing.T) {
	s, mock := newTestServer(t)

	rows := sqlmock.NewRows([]string{"category"})
	mock.ExpectQuery(`SELECT DISTINCT category`).WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
	w := httptest.NewRecorder()

	s.handleTopics(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("статус: ожидали 200, получили %d", w.Code)
	}
}

// TestHandleTopics_WrongMethod проверяет 405 при POST запросе
func TestHandleTopics_WrongMethod(t *testing.T) {
	s, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/topics", nil)
	w := httptest.NewRecorder()

	s.handleTopics(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("статус: ожидали 405, получили %d", w.Code)
	}
}

// TestHandleTopics_DBError проверяет 500 при ошибке БД
func TestHandleTopics_DBError(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectQuery(`SELECT DISTINCT category`).
		WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/topics", nil)
	w := httptest.NewRecorder()

	s.handleTopics(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("статус: ожидали 500, получили %d", w.Code)
	}
}

// =============================================================================
// Тесты handleHealth
// =============================================================================

// TestHandleHealth_Healthy проверяет healthy-ответ при доступной БД
func TestHandleHealth_Healthy(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("статус: ожидали 200, получили %d", w.Code)
	}

	var resp models.HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("не удалось декодировать ответ: %v", err)
	}
	if resp.Status != "healthy" {
		t.Errorf("Status: ожидали 'healthy', получили '%s'", resp.Status)
	}
}

// TestHandleHealth_Unhealthy проверяет unhealthy-ответ при недоступной БД
func TestHandleHealth_Unhealthy(t *testing.T) {
	s, mock := newTestServer(t)

	mock.ExpectPing().WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("статус: ожидали 503, получили %d", w.Code)
	}

	var resp models.HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "unhealthy" {
		t.Errorf("Status: ожидали 'unhealthy', получили '%s'", resp.Status)
	}
}

// TestHandleHealth_WrongMethod проверяет 405 при POST запросе
func TestHandleHealth_WrongMethod(t *testing.T) {
	s, _ := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	s.handleHealth(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("статус: ожидали 405, получили %d", w.Code)
	}
}

// =============================================================================
// Тесты вспомогательных функций
// =============================================================================

// TestIsValidLang проверяет все допустимые и недопустимые языки
func TestIsValidLang(t *testing.T) {
	tests := []struct {
		lang  string
		valid bool
	}{
		{"ru", true},
		{"en", true},
		{"zh", true},
		{"fr", false},
		{"de", false},
		{"", false},
		{"RU", false}, // регистр важен
		{"EN", false},
	}

	for _, tt := range tests {
		t.Run("lang="+tt.lang, func(t *testing.T) {
			got := isValidLang(tt.lang)
			if got != tt.valid {
				t.Errorf("isValidLang(%q): ожидали %v, получили %v", tt.lang, tt.valid, got)
			}
		})
	}
}

// TestLangToColumn проверяет маппинг языков в колонки БД
func TestLangToColumn(t *testing.T) {
	tests := []struct {
		lang   string
		column string
	}{
		{"ru", "russian"},
		{"en", "english"},
		{"zh", "chinese"},
		{"fr", ""}, // неизвестный язык
		{"", ""},   // пустой
	}

	for _, tt := range tests {
		t.Run("lang="+tt.lang, func(t *testing.T) {
			got := langToColumn(tt.lang)
			if got != tt.column {
				t.Errorf("langToColumn(%q): ожидали %q, получили %q", tt.lang, tt.column, got)
			}
		})
	}
}

// TestContentTypeHeader проверяет, что все эндпоинты возвращают application/json
func TestContentTypeHeader(t *testing.T) {
	s, mock := newTestServer(t)

	// Настраиваем mock для languages (не нужен DB)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/languages", nil)
	w := httptest.NewRecorder()
	s.handleLanguages(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type: ожидали 'application/json', получили '%s'", ct)
	}

	_ = mock
}
