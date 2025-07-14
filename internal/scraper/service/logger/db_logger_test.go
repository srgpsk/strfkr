package logger

import (
	"context"
	"database/sql"
	"testing"

	"app/internal/scraper/db"
)

// MockQueries implements LoggerQueries interface for testing
type MockQueries struct {
	lastLogMessage db.LogMessageParams
	shouldError    bool
}

func (m *MockQueries) LogMessage(ctx context.Context, params db.LogMessageParams) error {
	if m.shouldError {
		return sql.ErrConnDone
	}
	m.lastLogMessage = params
	return nil
}

func TestDBLogger_Log(t *testing.T) {
	mockQueries := &MockQueries{}
	logger := NewDBLogger(mockQueries)

	ctx := context.Background()
	targetID := int64(123)

	logger.Log(ctx, LevelInfo, &targetID, "http://example.com", "test message", map[string]string{"key": "value"})

	// Verify the mock received the correct parameters
	if mockQueries.lastLogMessage.LogType != "info" {
		t.Errorf("Expected LogType 'info', got %q", mockQueries.lastLogMessage.LogType)
	}

	if !mockQueries.lastLogMessage.TargetID.Valid || mockQueries.lastLogMessage.TargetID.Int64 != 123 {
		t.Error("TargetID not set correctly")
	}

	if !mockQueries.lastLogMessage.Url.Valid || mockQueries.lastLogMessage.Url.String != "http://example.com" {
		t.Error("URL not set correctly")
	}

	if mockQueries.lastLogMessage.Message != "test message" {
		t.Errorf("Expected message 'test message', got %q", mockQueries.lastLogMessage.Message)
	}

	if !mockQueries.lastLogMessage.Details.Valid {
		t.Error("Details should be valid when provided")
	}
}

func TestDBLogger_LogLevels(t *testing.T) {
	mockQueries := &MockQueries{}
	logger := NewDBLogger(mockQueries)

	ctx := context.Background()
	targetID := int64(123)

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name:     "info level",
			logFunc:  func() { logger.Info(ctx, &targetID, "", "info message") },
			expected: "info",
		},
		{
			name:     "warn level",
			logFunc:  func() { logger.Warn(ctx, &targetID, "", "warn message") },
			expected: "warn",
		},
		{
			name:     "error level",
			logFunc:  func() { logger.Error(ctx, &targetID, "", "error message") },
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.logFunc()
			if mockQueries.lastLogMessage.LogType != tt.expected {
				t.Errorf("Expected LogType %q, got %q", tt.expected, mockQueries.lastLogMessage.LogType)
			}
		})
	}
}

func TestDBLogger_ErrorHandling(t *testing.T) {
	mockQueries := &MockQueries{shouldError: true}
	logger := NewDBLogger(mockQueries)

	ctx := context.Background()

	// Should not panic even if database logging fails
	logger.Info(ctx, nil, "", "this should not panic")
}

func TestDBLogger_NilDetails(t *testing.T) {
	mockQueries := &MockQueries{}
	logger := NewDBLogger(mockQueries)

	ctx := context.Background()

	// Test with nil details
	logger.Info(ctx, nil, "", "no details")

	if mockQueries.lastLogMessage.Details.Valid {
		t.Error("Details should be invalid when nil")
	}
}

func TestDBLogger_EmptyURL(t *testing.T) {
	mockQueries := &MockQueries{}
	logger := NewDBLogger(mockQueries)

	ctx := context.Background()

	// Test with empty URL
	logger.Info(ctx, nil, "", "empty url test")

	if mockQueries.lastLogMessage.Url.Valid {
		t.Error("URL should be invalid when empty")
	}
}
