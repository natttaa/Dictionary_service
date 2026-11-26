package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pashagolub/pgxmock"
)

func TestSQLMetricsIncludeQueryResults(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery("SELECT success").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow("word"))
	mock.ExpectQuery("SELECT missing").WillReturnError(pgx.ErrNoRows)
	mock.ExpectQuery("SELECT broken").WillReturnError(&pgconn.PgError{Code: "42P01"})
	mock.ExpectQuery("SELECT topics").WillReturnRows(pgxmock.NewRows([]string{"value"}).AddRow("animals"))

	s := newTestServer(t, mock)
	var value string
	if err := s.queryRow(context.Background(), "translate", "SELECT success").Scan(&value); err != nil {
		t.Fatal(err)
	}
	if err := s.queryRow(context.Background(), "translate", "SELECT missing").Scan(&value); err == nil {
		t.Fatal("expected no-rows error")
	}
	if err := s.queryRow(context.Background(), "translate", "SELECT broken").Scan(&value); err == nil {
		t.Fatal("expected SQL error")
	}
	rows, err := s.queryRows(context.Background(), "list_topics", "SELECT topics")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		if err := rows.Scan(&value); err != nil {
			t.Fatal(err)
		}
	}
	rows.Close() // A second close must not count the query twice.

	recorder := httptest.NewRecorder()
	s.metrics.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	for _, expected := range []string{
		`dictionary_sql_queries_total{operation="translate",code="OK"} 1`,
		`dictionary_sql_queries_total{operation="translate",code="NO_ROWS"} 1`,
		`dictionary_sql_queries_total{operation="translate",code="42P01"} 1`,
		`dictionary_sql_queries_total{operation="list_topics",code="OK"} 1`,
		`dictionary_sql_query_duration_seconds_count{operation="list_topics",code="OK"} 1`,
	} {
		if !strings.Contains(recorder.Body.String(), expected) {
			t.Errorf("scrape is missing %q", expected)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
