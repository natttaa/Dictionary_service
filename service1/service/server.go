package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"service1/models"
	"time"
)

// Server represents the main application server
type Server struct {
	config     *Config
	httpClient *http.Client
	logger     *log.Logger
}

// NewServer creates a new server instance
func NewServer(config *Config) *Server {
	return &Server{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: log.New(os.Stdout, "[SERVICE1] ", log.LstdFlags),
	}
}

// handleTopics handles topics list request
func (s *Server) handleTopics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Printf("Handling topics request")

	// Try to get from dictionary service
	resp, err := s.httpClient.Get(s.config.DictionaryServiceURL + "/api/v1/topics")
	if err != nil {
		// Return mock data if service2 is unavailable
		s.logger.Printf("Service2 unavailable, returning mock topics: %v", err)
		response := models.TopicsResponse{
			Topics: []string{"animals", "food", "greetings", "family", "colors"},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer resp.Body.Close()

	// Forward response from dictionary service
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/api/v1/translate", s.handleTranslate)
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/languages", s.handleLanguages)
	mux.HandleFunc("/api/v1/topics", s.handleTopics)

	addr := fmt.Sprintf(":%d", s.config.Port)
	s.logger.Printf("Server starting on %s", addr)
	s.logger.Printf("Connected to dictionary service: %s", s.config.DictionaryServiceURL)

	return http.ListenAndServe(addr, s.loggerMiddleware(mux))
}

// loggerMiddleware logs all incoming requests
func (s *Server) loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		s.logger.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		s.logger.Printf("Completed in %v", time.Since(start))
	})
}

// handleTranslate handles translation requests
func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.TranslateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "INVALID_JSON", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := s.validateTranslateRequest(&req); err != nil {
		s.writeError(w, "VALIDATION_FAILED", err.Error(), http.StatusBadRequest)
		return
	}

	// Forward request to dictionary service
	resp, err := s.forwardToDictionary(r)
	if err != nil {
		s.logger.Printf("Error forwarding to dictionary service: %v", err)
		s.writeError(w, "SERVICE_UNAVAILABLE", "Dictionary service is unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Printf("Error reading response body: %v", err)
		s.writeError(w, "INTERNAL_ERROR", "Failed to read response", http.StatusInternalServerError)
		return
	}

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	// Write response body
	if _, err := w.Write(bodyBytes); err != nil {
		s.logger.Printf("Error writing response: %v", err)
	}
}

// forwardToDictionary forwards the request to dictionary service
func (s *Server) forwardToDictionary(r *http.Request) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", s.config.DictionaryServiceURL, r.URL.Path)

	// Read original request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	// Create new request with the same body
	req, err := http.NewRequest(r.Method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	req.Header = r.Header.Clone()

	// Execute request
	return s.httpClient.Do(req)
}

// validateTranslateRequest validates translation request
func (s *Server) validateTranslateRequest(req *models.TranslateRequest) error {
	if req.Word == "" {
		return fmt.Errorf("word cannot be empty")
	}
	if req.SourceLang == "" {
		return fmt.Errorf("source_lang cannot be empty")
	}
	if req.TargetLang == "" {
		return fmt.Errorf("target_lang cannot be empty")
	}

	// Validate supported languages
	supportedLangs := map[string]bool{"zh": true, "ru": true, "en": true}
	if !supportedLangs[req.SourceLang] {
		return fmt.Errorf("unsupported source language: %s", req.SourceLang)
	}
	if !supportedLangs[req.TargetLang] {
		return fmt.Errorf("unsupported target language: %s", req.TargetLang)
	}

	return nil
}

// writeError writes error response
func (s *Server) writeError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := models.TranslateResponse{
		Error: &models.Error{
			Code:    code,
			Message: message,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check dictionary service health
	resp, err := s.httpClient.Get(s.config.DictionaryServiceURL + "/api/v1/health")
	if err != nil {
		s.logger.Printf("Health check failed: dictionary service unavailable: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		response := models.HealthResponse{
			Status:   "unhealthy",
			Service2: "unavailable",
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	defer resp.Body.Close()

	var dictHealth models.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&dictHealth); err != nil {
		s.logger.Printf("Error decoding dictionary health response: %v", err)
		dictHealth.Status = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := models.HealthResponse{
		Status:   "healthy",
		Service2: dictHealth.Status,
	}
	json.NewEncoder(w).Encode(response)
}

// handleLanguages handles languages list request
func (s *Server) handleLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, "METHOD_NOT_ALLOWED", "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Printf("Handling languages request")

	// Try to get from dictionary service
	resp, err := s.httpClient.Get(s.config.DictionaryServiceURL + "/api/v1/languages")
	if err != nil {
		// Return mock data if service2 is unavailable
		s.logger.Printf("Service2 unavailable, returning mock data: %v", err)
		response := models.LanguagesResponse{
			Languages: []models.LanguageInfo{
				{Code: "en", Name: "English"},
				{Code: "ru", Name: "Russian"},
				{Code: "zh", Name: "Chinese"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer resp.Body.Close()

	// Forward response from dictionary service
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
