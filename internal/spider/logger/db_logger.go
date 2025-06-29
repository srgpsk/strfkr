package logger

import (
	"context"
	"database/sql"
	"encoding/json"

	"strfkr/internal/spider/db"
)

// LoggerQueries defines the interface needed for logging
type LoggerQueries interface {
	LogMessage(ctx context.Context, params db.LogMessageParams) error
}

// Level represents log levels
type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// DBLogger logs to the spider_logs table
type DBLogger struct {
	queries LoggerQueries
}

// NewDBLogger creates a new database logger
func NewDBLogger(queries LoggerQueries) *DBLogger {
	return &DBLogger{queries: queries}
}

// Log writes a log entry to the database
func (l *DBLogger) Log(ctx context.Context, level Level, targetID *int64, url, message string, details interface{}) {
	var detailsJSON sql.NullString
	if details != nil {
		if jsonBytes, err := json.Marshal(details); err == nil {
			detailsJSON = sql.NullString{String: string(jsonBytes), Valid: true}
		}
	}

	var targetIDNull sql.NullInt64
	if targetID != nil {
		targetIDNull = sql.NullInt64{Int64: *targetID, Valid: true}
	}

	var urlNull sql.NullString
	if url != "" {
		urlNull = sql.NullString{String: url, Valid: true}
	}

	// Log to database - don't block on errors
	_ = l.queries.LogMessage(ctx, db.LogMessageParams{
		LogType:  string(level),
		TargetID: targetIDNull,
		Url:      urlNull,
		Message:  message,
		Details:  detailsJSON,
	})
}

// Info logs an info message
func (l *DBLogger) Info(ctx context.Context, targetID *int64, url, message string, details ...interface{}) {
	var det interface{}
	if len(details) > 0 {
		det = details[0]
	}
	l.Log(ctx, LevelInfo, targetID, url, message, det)
}

// Warn logs a warning message
func (l *DBLogger) Warn(ctx context.Context, targetID *int64, url, message string, details ...interface{}) {
	var det interface{}
	if len(details) > 0 {
		det = details[0]
	}
	l.Log(ctx, LevelWarn, targetID, url, message, det)
}

// Error logs an error message
func (l *DBLogger) Error(ctx context.Context, targetID *int64, url, message string, details ...interface{}) {
	var det interface{}
	if len(details) > 0 {
		det = details[0]
	}
	l.Log(ctx, LevelError, targetID, url, message, det)
}
