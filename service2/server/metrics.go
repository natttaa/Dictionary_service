package server

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// requestMetrics exposes a small, dependency-free Prometheus text endpoint.
// Route names are bounded so arbitrary URL paths cannot create unlimited series.
type requestMetrics struct {
	mu     sync.Mutex
	series map[requestLabels]*requestTotals
}

type requestLabels struct {
	method string
	route  string
	status int
}

type requestTotals struct {
	count   uint64
	seconds float64
}

func newRequestMetrics() *requestMetrics {
	return &requestMetrics{series: make(map[requestLabels]*requestTotals)}
}

func metricRoute(path string) string {
	switch path {
	case "/api/v1/translate", "/api/v1/health", "/api/v1/languages",
		"/api/v1/topics", "/api/v1/topics/words", "/api/v1/check-translation":
		return path
	default:
		return "other"
	}
}

func (m *requestMetrics) observe(method, path string, status int, elapsed time.Duration) {
	labels := requestLabels{method: method, route: metricRoute(path), status: status}
	m.mu.Lock()
	defer m.mu.Unlock()
	totals := m.series[labels]
	if totals == nil {
		totals = &requestTotals{}
		m.series[labels] = totals
	}
	totals.count++
	totals.seconds += elapsed.Seconds()
}

func (m *requestMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	m.mu.Lock()
	copyOfSeries := make(map[requestLabels]requestTotals, len(m.series))
	for labels, totals := range m.series {
		copyOfSeries[labels] = *totals
	}
	m.mu.Unlock()

	fmt.Fprintln(w, "# HELP dictionary_http_requests_total Total HTTP requests handled by this service.")
	fmt.Fprintln(w, "# TYPE dictionary_http_requests_total counter")
	for labels, totals := range copyOfSeries {
		fmt.Fprintf(w, "dictionary_http_requests_total{%s} %d\n", metricLabels(labels), totals.count)
	}
	fmt.Fprintln(w, "# HELP dictionary_http_request_duration_seconds Total HTTP request duration in seconds.")
	fmt.Fprintln(w, "# TYPE dictionary_http_request_duration_seconds summary")
	for labels, totals := range copyOfSeries {
		fields := metricLabels(labels)
		fmt.Fprintf(w, "dictionary_http_request_duration_seconds_sum{%s} %g\n", fields, totals.seconds)
		fmt.Fprintf(w, "dictionary_http_request_duration_seconds_count{%s} %d\n", fields, totals.count)
	}
}

func metricLabels(labels requestLabels) string {
	return fmt.Sprintf("method=%s,route=%s,status=%s",
		strconv.Quote(labels.method), strconv.Quote(labels.route), strconv.Quote(strconv.Itoa(labels.status)))
}
