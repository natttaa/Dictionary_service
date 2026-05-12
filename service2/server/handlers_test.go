package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"

	"github.com/pashagolub/pgxmock"
)

func newTestServer(t *testing.T, mockPool pgxmock.PgxPoolIface) *Server {
	t.Helper()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return &Server{logger: logger, db: mockPool}
}

func testHandler(t *testing.T, method, path string, body interface{}, srv *Server, expectedStatus int, expectedBody interface{}) {
	var reqBody io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reqBody)
	w := httptest.NewRecorder()
	// Use mux to route
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/translate", srv.handleTranslate)
	mux.HandleFunc("/api/v1/languages", srv.handleLanguages)
	mux.HandleFunc("/api/v1/topics", srv.handleTopics)
	mux.HandleFunc("/api/v1/topics/words", srv.handleTopicWords)
	mux.HandleFunc("/api/v1/check-translation", srv.handleCheckTranslation)
	mux.HandleFunc("/api/v1/health", srv.handleHealth)
	mux.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != expectedStatus {
		t.Fatalf("expected status %d got %d", expectedStatus, resp.StatusCode)
	}
	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if expectedBody != nil {
		if !compareJSON(respBody, expectedBody) {
			t.Fatalf("expected body %v got %v", expectedBody, respBody)
		}
	}
}

func compareJSON(a, b interface{}) bool {
	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)
	return string(aBytes) == string(bBytes)
}

// ---------- Translate Tests ----------

func TestHandleTranslate_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("could not create mock: %v", err)
	}
	defer mock.Close()
	mock.ExpectQuery(`SELECT english FROM dictionary\.dictionary_table WHERE LOWER\(russian\) = LOWER\(\$1\)`).WithArgs("cat").WillReturnRows(pgxmock.NewRows([]string{"english"}).AddRow("cat"))

	srv := newTestServer(t, mock)

	body := map[string]string{"source_lang": "ru", "target_lang": "en", "word": "cat"}
	expected := map[string]interface{}{"translation": "cat"}
	testHandler(t, http.MethodPost, "/api/v1/translate", body, srv, http.StatusOK, expected)
}

// ---------- Languages Test ----------
func TestHandleLanguages(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	srv := newTestServer(t, mock)
	expected := map[string]interface{}{"languages": []interface{}{map[string]string{"code": "ru", "name": "Русский"}, map[string]string{"code": "en", "name": "English"}, map[string]string{"code": "zh", "name": "中文 (Chinese)"}}}
	testHandler(t, http.MethodGet, "/api/v1/languages", nil, srv, http.StatusOK, expected)
}

// ---------- Topics Test ----------
func TestHandleTopics(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	rows := pgxmock.NewRows([]string{"category"}).AddRow("animals").AddRow("food")
	mock.ExpectQuery(`SELECT DISTINCT category FROM dictionary.dictionary_table ORDER BY category`).WillReturnRows(rows)
	defer mock.Close()
	srv := newTestServer(t, mock)
	expected := map[string]interface{}{"topics": []interface{}{"animals", "food"}}
	testHandler(t, http.MethodGet, "/api/v1/topics", nil, srv, http.StatusOK, expected)
}

// ---------- TopicWords Test ----------
func TestHandleTopicWords(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	rows := pgxmock.NewRows([]string{"english", "russian", "chinese"}).AddRow("cat", "кот", "猫")
	mock.ExpectQuery(`SELECT english, russian, chinese FROM dictionary\.dictionary_table WHERE LOWER\(category\) = LOWER\(\$1\) ORDER BY english`).WithArgs("animals").WillReturnRows(rows)
	defer mock.Close()
	srv := newTestServer(t, mock)
	body := map[string]interface{}{"topic": "animals", "languages": []string{"en", "ru", "zh"}}
	expected := map[string]interface{}{"topic": "animals", "words": []interface{}{map[string]interface{}{"translations": map[string]string{"en": "cat", "ru": "кот", "zh": "猫"}}}}
	testHandler(t, http.MethodPost, "/api/v1/topics/words", body, srv, http.StatusOK, expected)
}
