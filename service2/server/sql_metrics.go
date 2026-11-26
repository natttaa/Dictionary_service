package server

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v4"
)

// queryRow records a query once Scan has completed, including the no-rows case.
func (s *Server) queryRow(ctx context.Context, operation, query string, args ...any) pgx.Row {
	started := time.Now()
	return &observedRow{
		Row:       s.db.QueryRow(ctx, query, args...),
		metrics:   s.metrics,
		operation: operation,
		started:   started,
	}
}

type observedRow struct {
	pgx.Row
	metrics   *requestMetrics
	operation string
	started   time.Time
}

func (r *observedRow) Scan(dest ...any) error {
	err := r.Row.Scan(dest...)
	r.metrics.observeSQL(r.operation, err, time.Since(r.started))
	return err
}

// queryRows records the complete query, including reading and scanning its rows.
func (s *Server) queryRows(ctx context.Context, operation, query string, args ...any) (pgx.Rows, error) {
	started := time.Now()
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		s.metrics.observeSQL(operation, err, time.Since(started))
		return nil, err
	}
	return &observedRows{
		Rows:      rows,
		metrics:   s.metrics,
		operation: operation,
		started:   started,
	}, nil
}

type observedRows struct {
	pgx.Rows
	metrics   *requestMetrics
	operation string
	started   time.Time
	firstErr  error
	once      sync.Once
}

func (r *observedRows) Next() bool {
	if r.Rows.Next() {
		return true
	}
	r.finish()
	return false
}

func (r *observedRows) Scan(dest ...any) error {
	err := r.Rows.Scan(dest...)
	if err != nil && r.firstErr == nil {
		r.firstErr = err
	}
	return err
}

func (r *observedRows) Close() {
	r.Rows.Close()
	r.finish()
}

func (r *observedRows) finish() {
	r.once.Do(func() {
		err := r.firstErr
		if err == nil {
			err = r.Rows.Err()
		}
		r.metrics.observeSQL(r.operation, err, time.Since(r.started))
	})
}
