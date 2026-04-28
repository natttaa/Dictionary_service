package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"service1/cmd/cli/config"
	"service1/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureOutput перехватывает stdout во время выполнения теста
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestNewCLIClient(t *testing.T) {
	logger := config.DefaultConfig().SetupLogger()
	client := NewCLIClient("http://localhost:8081", 30*time.Second, logger)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8081", client.serverURL)
	assert.NotNil(t, client.client)
	assert.NotNil(t, client.logger)
	assert.Equal(t, 30*time.Second, client.client.Timeout)
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name: "healthy service",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/health", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.HealthResponse{
					Status:   "healthy",
					Service2: "healthy",
				})
			},
			expectError: false,
		},
		{
			name: "unhealthy service",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(models.HealthResponse{
					Status:   "unhealthy",
					Service2: "unavailable",
				})
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := config.DefaultConfig().SetupLogger()
			client := NewCLIClient(server.URL, 5*time.Second, logger)

			// Перехватываем вывод чтобы не засорять консоль тестов
			output := captureOutput(func() {
				err := client.Health()
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})

			// Проверяем что вывод содержит ожидаемые строки
			if !tt.expectError {
				assert.Contains(t, output, "Состояние сервиса")
			}
		})
	}
}

func TestListLanguages(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name: "successful languages list",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/languages", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.LanguagesResponse{
					Languages: []models.LanguageInfo{
						{Code: "en", Name: "English"},
						{Code: "ru", Name: "Russian"},
					},
				})
			},
			expectError: false,
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.LanguagesResponse{
					Error: &models.Error{
						Code:    "INTERNAL_ERROR",
						Message: "Internal server error",
					},
				})
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := config.DefaultConfig().SetupLogger()
			client := NewCLIClient(server.URL, 5*time.Second, logger)

			captureOutput(func() {
				err := client.ListLanguages()
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestListTopics(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name: "successful topics list",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/topics", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.TopicsResponse{
					Topics: []string{"animals", "food", "colors"},
				})
			},
			expectError: false,
		},
		{
			name: "server error",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(models.TopicsResponse{
					Error: &models.Error{
						Code:    "INTERNAL_ERROR",
						Message: "Internal server error",
					},
				})
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := config.DefaultConfig().SetupLogger()
			client := NewCLIClient(server.URL, 5*time.Second, logger)

			captureOutput(func() {
				err := client.ListTopics()
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestTranslate(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		target         string
		word           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name:   "successful translation",
			source: "en",
			target: "ru",
			word:   "hello",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/translate", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				var req models.TranslateRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, "en", req.SourceLang)
				assert.Equal(t, "ru", req.TargetLang)
				assert.Equal(t, "hello", req.Word)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.TranslateResponse{
					Translation: "привет",
				})
			},
			expectError: false,
		},
		{
			name:   "word not found",
			source: "en",
			target: "ru",
			word:   "nonexistent",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(models.TranslateResponse{
					Error: &models.Error{
						Code:    "WORD_NOT_FOUND",
						Message: "Слово не найдено",
					},
				})
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := config.DefaultConfig().SetupLogger()
			client := NewCLIClient(server.URL, 5*time.Second, logger)

			captureOutput(func() {
				err := client.Translate(tt.source, tt.target, tt.word)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestGetTopicWords(t *testing.T) {
	tests := []struct {
		name           string
		topic          string
		languages      []string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name:      "single language",
			topic:     "animals",
			languages: []string{"ru"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/topics/words", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				var req models.TopicWordsRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, "animals", req.Topic)
				assert.Equal(t, []string{"ru"}, req.Languages)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.TopicWordsResponse{
					Topic: "animals",
					Words: []models.WordEntry{
						{Translations: map[string]string{"ru": "собака"}},
						{Translations: map[string]string{"ru": "кошка"}},
					},
				})
			},
			expectError: false,
		},
		{
			name:      "multiple languages",
			topic:     "animals",
			languages: []string{"ru", "en"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				var req models.TopicWordsRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, "animals", req.Topic)
				assert.Equal(t, []string{"ru", "en"}, req.Languages)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.TopicWordsResponse{
					Topic: "animals",
					Words: []models.WordEntry{
						{Translations: map[string]string{"ru": "собака", "en": "dog"}},
						{Translations: map[string]string{"ru": "кошка", "en": "cat"}},
					},
				})
			},
			expectError: false,
		},
		{
			name:      "topic not found",
			topic:     "nonexistent",
			languages: []string{"ru"},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(models.TopicWordsResponse{
					Error: &models.Error{
						Code:    "TOPIC_NOT_FOUND",
						Message: "Тема не найдена",
					},
				})
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := config.DefaultConfig().SetupLogger()
			client := NewCLIClient(server.URL, 5*time.Second, logger)

			captureOutput(func() {
				err := client.GetTopicWords(tt.topic, tt.languages)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestCheckTranslation(t *testing.T) {
	tests := []struct {
		name           string
		word           string
		translation    string
		sourceLang     string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectError    bool
	}{
		{
			name:        "correct translation",
			word:        "собака",
			translation: "dog",
			sourceLang:  "ru",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/check-translation", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)

				var req models.CheckTranslationRequest
				err := json.NewDecoder(r.Body).Decode(&req)
				require.NoError(t, err)
				assert.Equal(t, "собака", req.Word)
				assert.Equal(t, "dog", req.Translation)
				assert.Equal(t, "ru", req.SourceLang)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.CheckTranslationResponse{
					CorrectTranslation: "dog",
				})
			},
			expectError: false,
		},
		{
			name:        "incorrect translation",
			word:        "собака",
			translation: "cat",
			sourceLang:  "ru",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(models.CheckTranslationResponse{
					CorrectTranslation: "dog",
				})
			},
			expectError: false,
		},
		{
			name:        "word not found",
			word:        "несуществующее",
			translation: "something",
			sourceLang:  "ru",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(models.CheckTranslationResponse{
					Error: &models.Error{
						Code:    "WORD_NOT_FOUND",
						Message: "Слово не найдено",
					},
				})
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			logger := config.DefaultConfig().SetupLogger()
			client := NewCLIClient(server.URL, 5*time.Second, logger)

			captureOutput(func() {
				err := client.CheckTranslation(tt.word, tt.translation, tt.sourceLang)
				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestGetLanguageName(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"zh", "Китайский"},
		{"ru", "Русский"},
		{"en", "Английский"},
		{"fr", "fr"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := getLanguageName(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}
