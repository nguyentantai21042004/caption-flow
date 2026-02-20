package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
)

type implLogger struct {
	logger *log.Logger
	level  string
}

// New creates a new Logger instance
func New(level string) Logger {
	return &implLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
		level:  strings.ToLower(level),
	}
}

func (l *implLogger) shouldLog(level string) bool {
	levels := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}

	currentLevel, ok := levels[l.level]
	if !ok {
		currentLevel = 1 // default to info
	}

	targetLevel, ok := levels[level]
	if !ok {
		return true
	}

	return targetLevel >= currentLevel
}

func (l *implLogger) Debug(ctx context.Context, msg string, args ...interface{}) {
	if l.shouldLog("debug") {
		l.logger.Printf("[DEBUG] "+msg, args...)
	}
}

func (l *implLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.shouldLog("info") {
		l.logger.Printf("[INFO] "+msg, args...)
	}
}

func (l *implLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.shouldLog("warn") {
		l.logger.Printf("[WARN] "+msg, args...)
	}
}

func (l *implLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.shouldLog("error") {
		l.logger.Printf("[ERROR] "+msg, args...)
	}
}

// Helper to format error messages
func FormatError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
