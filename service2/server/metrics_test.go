package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestMetricsScrape(t *testing.T) {
	m := newRequestMetrics()
	m.probe = func(context.Context) bool { return true }
	m.observe(http.MethodGet, "/api/v1/health", http.StatusOK, 50*time.Millisecond)
	m.observe(http.MethodGet, "/unknown/123", http.StatusNotFound, 10*time.Millisecond)

	recorder := httptest.NewRecorder()
	m.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("scrape status = %d", recorder.Code)
	}
	for _, expected := range []string{
		`dictionary_http_requests_total{method="GET",route="/api/v1/health",status="200"} 1`,
		`dictionary_http_requests_total{method="GET",route="other",status="404"} 1`,
		`dictionary_http_request_duration_seconds_count{method="GET",route="/api/v1/health",status="200"} 1`,
		`dictionary_dependency_up{dependency="database"} 1`,
	} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("scrape is missing %q", expected)
		}
	}
}

func TestDependencyDownMetric(t *testing.T) {
	m := newRequestMetrics()
	m.probe = func(context.Context) bool { return false }
	w := httptest.NewRecorder()
	m.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(w.Body.String(), `dictionary_dependency_up{dependency="database"} 0`) {
		t.Fatal("missing unhealthy database metric")
	}
}
