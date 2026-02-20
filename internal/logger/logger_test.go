package logger

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"error level", "error"},
		{"invalid level", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := New(tt.level)
			if log == nil {
				t.Error("New() returned nil")
			}
		})
	}
}

func TestLoggerLevels(t *testing.T) {
	ctx := context.Background()
	log := New("info")

	// These should not panic
	log.Debug(ctx, "debug message")
	log.Info(ctx, "info message")
	log.Warn(ctx, "warn message")
	log.Error(ctx, "error message")

	// Test with formatting
	log.Info(ctx, "formatted message: %s %d", "test", 123)
}

func TestShouldLog(t *testing.T) {
	tests := []struct {
		name        string
		configLevel string
		logLevel    string
		shouldLog   bool
	}{
		{"debug logs at debug level", "debug", "debug", true},
		{"info logs at debug level", "debug", "info", true},
		{"debug doesn't log at info level", "info", "debug", false},
		{"info logs at info level", "info", "info", true},
		{"error always logs", "debug", "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := New(tt.configLevel).(*implLogger)
			result := log.shouldLog(tt.logLevel)
			if result != tt.shouldLog {
				t.Errorf("shouldLog() = %v, want %v", result, tt.shouldLog)
			}
		})
	}
}
