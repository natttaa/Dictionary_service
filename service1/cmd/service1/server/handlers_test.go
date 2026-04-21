package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"service1/cmd/service1/config"
	"service1/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleHealth(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		dictionaryURL  string
		dictionaryMock func() *httptest.Server
		expectedStatus int
		expectedBody   models.HealthResponse
	}{
		{
			name:   "successful health check",
			method: http.MethodGet,
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v1/health", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(models.HealthResponse{Status: "healthy"})
				}))
			},
			expectedStatus: http.StatusOK,
			expectedBody: models.HealthResponse{
				Status:   "healthy",
				Service2: "healthy",
			},
		},
		{
			name:   "dictionary service unavailable",
			method: http.MethodGet,
			dictionaryMock: func() *httptest.Server {
				return nil
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedBody: models.HealthResponse{
				Status:   "unhealthy",
				Service2: "unavailable",
			},
		},
		{
			name:   "wrong method",
			method: http.MethodPost,
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   models.HealthResponse{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dictServer *httptest.Server
			if tt.dictionaryMock != nil {
				dictServer = tt.dictionaryMock()
				if dictServer != nil {
					defer dictServer.Close()
				}
			}

			cfg := config.DefaultConfig()
			if dictServer != nil {
				cfg.DictionaryServiceURL = dictServer.URL
			} else {
				cfg.DictionaryServiceURL = "http://unavailable:8083"
			}
			cfg.Timeout = 1 * time.Second

			s := NewServer(cfg)

			req := httptest.NewRequest(tt.method, "/api/v1/health", nil)
			w := httptest.NewRecorder()

			require.NotPanics(t, func() {
				s.handleHealth(w, req)
			})

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK || tt.expectedStatus == http.StatusServiceUnavailable {
				var response models.HealthResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody.Status, response.Status)
				if tt.expectedBody.Service2 != "" {
					assert.Equal(t, tt.expectedBody.Service2, response.Service2)
				}
			}
		})
	}
}

func TestHandleTranslate(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		dictionaryMock func() *httptest.Server
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "successful translation",
			method: http.MethodPost,
			requestBody: models.TranslateRequest{
				SourceLang: "en",
				TargetLang: "ru",
				Word:       "hello",
			},
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v1/translate", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(models.TranslateResponse{
						Translation: "привет",
					})
				}))
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:        "invalid json - empty body",
			method:      http.MethodPost,
			requestBody: "", // Пустая строка вызовет ошибку парсинга JSON
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_JSON", // Теперь ожидаем INVALID_JSON
		},
		{
			name:        "invalid json - malformed",
			method:      http.MethodPost,
			requestBody: "not a json",
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_JSON",
		},
		{
			name:   "validation failed - empty word",
			method: http.MethodPost,
			requestBody: models.TranslateRequest{
				SourceLang: "en",
				TargetLang: "ru",
				Word:       "",
			},
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_FAILED",
		},
		{
			name:   "validation failed - empty source language",
			method: http.MethodPost,
			requestBody: models.TranslateRequest{
				SourceLang: "",
				TargetLang: "ru",
				Word:       "hello",
			},
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_FAILED",
		},
		{
			name:   "validation failed - unsupported source language",
			method: http.MethodPost,
			requestBody: models.TranslateRequest{
				SourceLang: "fr",
				TargetLang: "ru",
				Word:       "hello",
			},
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_FAILED",
		},
		{
			name:   "wrong method",
			method: http.MethodGet,
			requestBody: models.TranslateRequest{
				SourceLang: "en",
				TargetLang: "ru",
				Word:       "hello",
			},
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  "METHOD_NOT_ALLOWED",
		},
		{
			name:   "dictionary service unavailable",
			method: http.MethodPost,
			requestBody: models.TranslateRequest{
				SourceLang: "en",
				TargetLang: "ru",
				Word:       "hello",
			},
			dictionaryMock: func() *httptest.Server {
				return nil
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedError:  "SERVICE_UNAVAILABLE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dictServer *httptest.Server
			if tt.dictionaryMock != nil {
				dictServer = tt.dictionaryMock()
				if dictServer != nil {
					defer dictServer.Close()
				}
			}

			cfg := config.DefaultConfig()
			if dictServer != nil {
				cfg.DictionaryServiceURL = dictServer.URL
			} else {
				cfg.DictionaryServiceURL = "http://unavailable:8083"
			}
			cfg.Timeout = 1 * time.Second

			s := NewServer(cfg)

			var bodyBytes []byte
			var err error

			// Обрабатываем разные типы requestBody
			switch v := tt.requestBody.(type) {
			case string:
				bodyBytes = []byte(v)
			case models.TranslateRequest:
				bodyBytes, err = json.Marshal(v)
				require.NoError(t, err)
			default:
				bodyBytes, err = json.Marshal(v)
				require.NoError(t, err)
			}

			req := httptest.NewRequest(tt.method, "/api/v1/translate", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			require.NotPanics(t, func() {
				s.handleTranslate(w, req)
			})

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response models.TranslateResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.NotNil(t, response.Error)
				assert.Equal(t, tt.expectedError, response.Error.Code)
			}
		})
	}
}

func TestHandleLanguages(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		dictionaryMock func() *httptest.Server
		expectedStatus int
		useMockData    bool
	}{
		{
			name:   "successful languages fetch",
			method: http.MethodGet,
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v1/languages", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(models.LanguagesResponse{
						Languages: []models.LanguageInfo{
							{Code: "en", Name: "English"},
							{Code: "es", Name: "Spanish"},
						},
					})
				}))
			},
			expectedStatus: http.StatusOK,
			useMockData:    false,
		},
		{
			name:   "dictionary service unavailable - return mock data",
			method: http.MethodGet,
			dictionaryMock: func() *httptest.Server {
				return nil
			},
			expectedStatus: http.StatusOK,
			useMockData:    true,
		},
		{
			name:   "wrong method",
			method: http.MethodPost,
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusMethodNotAllowed,
			useMockData:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dictServer *httptest.Server
			if tt.dictionaryMock != nil {
				dictServer = tt.dictionaryMock()
				if dictServer != nil {
					defer dictServer.Close()
				}
			}

			cfg := config.DefaultConfig()
			if dictServer != nil {
				cfg.DictionaryServiceURL = dictServer.URL
			} else {
				cfg.DictionaryServiceURL = "http://unavailable:8083"
			}
			cfg.Timeout = 1 * time.Second

			s := NewServer(cfg)

			req := httptest.NewRequest(tt.method, "/api/v1/languages", nil)
			w := httptest.NewRecorder()

			require.NotPanics(t, func() {
				s.handleLanguages(w, req)
			})

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.LanguagesResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				if tt.useMockData {
					// Проверяем, что вернулись мок-данные (3 языка)
					assert.GreaterOrEqual(t, len(response.Languages), 3)
				}
			}
		})
	}
}

func TestHandleTopics(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		dictionaryMock func() *httptest.Server
		expectedStatus int
		useMockData    bool
	}{
		{
			name:   "successful topics fetch",
			method: http.MethodGet,
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/api/v1/topics", r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(models.TopicsResponse{
						Topics: []string{"animals", "food"},
					})
				}))
			},
			expectedStatus: http.StatusOK,
			useMockData:    false,
		},
		{
			name:   "dictionary service unavailable - return mock data",
			method: http.MethodGet,
			dictionaryMock: func() *httptest.Server {
				return nil
			},
			expectedStatus: http.StatusOK,
			useMockData:    true,
		},
		{
			name:   "wrong method",
			method: http.MethodPost,
			dictionaryMock: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			},
			expectedStatus: http.StatusMethodNotAllowed,
			useMockData:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dictServer *httptest.Server
			if tt.dictionaryMock != nil {
				dictServer = tt.dictionaryMock()
				if dictServer != nil {
					defer dictServer.Close()
				}
			}

			cfg := config.DefaultConfig()
			if dictServer != nil {
				cfg.DictionaryServiceURL = dictServer.URL
			} else {
				cfg.DictionaryServiceURL = "http://unavailable:8083"
			}
			cfg.Timeout = 1 * time.Second

			s := NewServer(cfg)

			req := httptest.NewRequest(tt.method, "/api/v1/topics", nil)
			w := httptest.NewRecorder()

			require.NotPanics(t, func() {
				s.handleTopics(w, req)
			})

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var response models.TopicsResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				if tt.useMockData {
					// Проверяем, что вернулись мок-данные (5 тем)
					assert.GreaterOrEqual(t, len(response.Topics), 5)
				}
			}
		})
	}
}

func TestValidateTranslateRequest(t *testing.T) {
	s := &Server{}

	tests := []struct {
		name    string
		req     *models.TranslateRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: &models.TranslateRequest{
				SourceLang: "en",
				TargetLang: "ru",
				Word:       "hello",
			},
			wantErr: false,
		},
		{
			name: "empty word",
			req: &models.TranslateRequest{
				SourceLang: "en",
				TargetLang: "ru",
				Word:       "",
			},
			wantErr: true,
			errMsg:  "слово не может быть пустым",
		},
		{
			name: "empty source language",
			req: &models.TranslateRequest{
				SourceLang: "",
				TargetLang: "ru",
				Word:       "hello",
			},
			wantErr: true,
			errMsg:  "исходный язык не может быть пустым",
		},
		{
			name: "empty target language",
			req: &models.TranslateRequest{
				SourceLang: "en",
				TargetLang: "",
				Word:       "hello",
			},
			wantErr: true,
			errMsg:  "целевой язык не может быть пустым",
		},
		{
			name: "unsupported source language",
			req: &models.TranslateRequest{
				SourceLang: "fr",
				TargetLang: "ru",
				Word:       "hello",
			},
			wantErr: true,
			errMsg:  "неподдерживаемый исходный язык: fr",
		},
		{
			name: "unsupported target language",
			req: &models.TranslateRequest{
				SourceLang: "en",
				TargetLang: "de",
				Word:       "hello",
			},
			wantErr: true,
			errMsg:  "неподдерживаемый целевой язык: de",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := s.validateTranslateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
