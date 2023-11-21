package core

import (
	"context"
	"database/sql"
	"time"
)

// Queryable provides a common interface between sql.DB and sql.Tx to make methods compatible with both.
type SQLClient interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	Prepare(query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}

// timeToSQL converts a time struct to a string representation compatible with SQLite.
func timeToSQL(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	dateStr := date.Format(time.RFC3339Nano)
	return dateStr
}

// timeFromSQL parses a string representation of a time to a time struct.
func timeFromSQL(dateStr string) time.Time {
	date, err := time.Parse(time.RFC3339Nano, dateStr)
	if err != nil {
		return time.Time{}
	}
	return date
}

// timeFromNullableSQL parses a string representation of a time to a time struct.
func timeFromNullableSQL(dateStr sql.NullString) time.Time {
	if !dateStr.Valid {
		return time.Time{} // zero date
	}
	return timeFromSQL(dateStr.String)
}
