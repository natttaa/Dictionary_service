package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

// requestMetrics exposes a small, dependency-free Prometheus text endpoint.
// Route names are bounded so arbitrary URL paths cannot create unlimited series.
type requestMetrics struct {
	mu     sync.Mutex
	series map[requestLabels]*requestTotals
	sql    map[sqlLabels]*requestTotals
	probe  func(context.Context) bool
}

type sqlLabels struct {
	operation string
	code      string
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
	return &requestMetrics{
		series: make(map[requestLabels]*requestTotals),
		sql:    make(map[sqlLabels]*requestTotals),
	}
}

func sqlResultCode(err error) string {
	if err == nil {
		return "OK"
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return "NO_ROWS"
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code != "" {
		return pgErr.Code
	}
	return "OTHER"
}

func (m *requestMetrics) observeSQL(operation string, err error, elapsed time.Duration) {
	labels := sqlLabels{operation: operation, code: sqlResultCode(err)}
	m.mu.Lock()
	defer m.mu.Unlock()
	totals := m.sql[labels]
	if totals == nil {
		totals = &requestTotals{}
		m.sql[labels] = totals
	}
	totals.count++
	totals.seconds += elapsed.Seconds()
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
	copyOfSQL := make(map[sqlLabels]requestTotals, len(m.sql))
	for labels, totals := range m.sql {
		copyOfSQL[labels] = *totals
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
	fmt.Fprintln(w, "# HELP dictionary_sql_queries_total Total SQL queries completed by this service.")
	fmt.Fprintln(w, "# TYPE dictionary_sql_queries_total counter")
	for labels, totals := range copyOfSQL {
		fmt.Fprintf(w, "dictionary_sql_queries_total{operation=%s,code=%s} %d\n",
			strconv.Quote(labels.operation), strconv.Quote(labels.code), totals.count)
	}
	fmt.Fprintln(w, "# HELP dictionary_sql_query_duration_seconds Total SQL query duration in seconds, including row reads.")
	fmt.Fprintln(w, "# TYPE dictionary_sql_query_duration_seconds summary")
	for labels, totals := range copyOfSQL {
		fields := fmt.Sprintf("operation=%s,code=%s", strconv.Quote(labels.operation), strconv.Quote(labels.code))
		fmt.Fprintf(w, "dictionary_sql_query_duration_seconds_sum{%s} %g\n", fields, totals.seconds)
		fmt.Fprintf(w, "dictionary_sql_query_duration_seconds_count{%s} %d\n", fields, totals.count)
	}
	if m.probe != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		value := 0
		if m.probe(ctx) {
			value = 1
		}
		fmt.Fprintln(w, "# HELP dictionary_dependency_up Whether the downstream dependency is healthy.")
		fmt.Fprintln(w, "# TYPE dictionary_dependency_up gauge")
		fmt.Fprintf(w, "dictionary_dependency_up{dependency=\"database\"} %d\n", value)
	}
}

func metricLabels(labels requestLabels) string {
	return fmt.Sprintf("method=%s,route=%s,status=%s",
		strconv.Quote(labels.method), strconv.Quote(labels.route), strconv.Quote(strconv.Itoa(labels.status)))
}
