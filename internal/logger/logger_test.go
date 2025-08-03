package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
	}{
		{
			name: "default config",
			config: &Config{
				Level:   "info",
				Console: true,
				Format:  "text",
			},
		},
		{
			name: "json format",
			config: &Config{
				Level:   "debug",
				Console: true,
				Format:  "json",
			},
		},
		{
			name:   "nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			if logger == nil {
				t.Error("New() returned nil")
			}
			if logger.Logger == nil {
				t.Error("New() returned logger with nil slog.Logger")
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"invalid", slog.LevelInfo}, // default
		{"", slog.LevelInfo},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseLevel(tt.input)
			if got != tt.want {
				t.Errorf("parseLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
		config: &Config{},
	}

	// Test various log methods
	logger.Debug("debug message")
	if !strings.Contains(buf.String(), "debug message") {
		t.Error("Debug message not found in output")
	}
	buf.Reset()

	logger.Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info message not found in output")
	}
	buf.Reset()

	logger.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Error("Warn message not found in output")
	}
	buf.Reset()

	logger.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error message not found in output")
	}
	buf.Reset()

	// Test formatted methods
	logger.Debugf("debug %s", "formatted")
	if !strings.Contains(buf.String(), "debug formatted") {
		t.Error("Formatted debug message not found in output")
	}
	buf.Reset()

	logger.Infof("info %d", 123)
	if !strings.Contains(buf.String(), "info 123") {
		t.Error("Formatted info message not found in output")
	}
	buf.Reset()

	logger.Warnf("warn %v", true)
	if !strings.Contains(buf.String(), "warn true") {
		t.Error("Formatted warn message not found in output")
	}
	buf.Reset()

	logger.Errorf("error %.2f", 3.14)
	if !strings.Contains(buf.String(), "error 3.14") {
		t.Error("Formatted error message not found in output")
	}
}

func TestWithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
		config: &Config{},
	}

	fields := map[string]interface{}{
		"user":   "test",
		"action": "login",
		"id":     123,
	}

	logger.WithFields(fields).Info("test message")
	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Error("Message not found in output")
	}
	if !strings.Contains(output, "user=test") {
		t.Error("Field 'user' not found in output")
	}
	if !strings.Contains(output, "action=login") {
		t.Error("Field 'action' not found in output")
	}
	if !strings.Contains(output, "id=123") {
		t.Error("Field 'id' not found in output")
	}
}

func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
		config: &Config{},
	}

	testErr := &testError{msg: "test error"}
	logger.WithError(testErr).Error("operation failed")
	output := buf.String()

	if !strings.Contains(output, "operation failed") {
		t.Error("Message not found in output")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Error message not found in output")
	}
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})),
		config: &Config{},
	}

	ctx := context.WithValue(context.Background(), "trace_id", "test-trace-123")
	logger.WithContext(ctx).Info("with context")
	output := buf.String()

	if !strings.Contains(output, "with context") {
		t.Error("Message not found in output")
	}
	if !strings.Contains(output, "trace_id") {
		t.Error("Trace ID not found in output")
	}
}

func TestDefault(t *testing.T) {
	logger1 := Default()
	logger2 := Default()

	if logger1 == nil {
		t.Error("Default() returned nil")
	}
	if logger1 != logger2 {
		t.Error("Default() should return the same instance")
	}
}

func TestMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	handler1 := slog.NewTextHandler(&buf1, &slog.HandlerOptions{Level: slog.LevelInfo})
	handler2 := slog.NewTextHandler(&buf2, &slog.HandlerOptions{Level: slog.LevelInfo})

	multiHandler := NewMultiHandler(handler1, handler2)
	logger := slog.New(multiHandler)

	logger.Info("test message")

	if !strings.Contains(buf1.String(), "test message") {
		t.Error("Message not found in first handler output")
	}
	if !strings.Contains(buf2.String(), "test message") {
		t.Error("Message not found in second handler output")
	}
}

func TestLogWriter(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{
		Logger: slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
		config: &Config{},
	}

	writer := logger.Writer(slog.LevelInfo)
	writer.Write([]byte("test from writer"))

	if !strings.Contains(buf.String(), "test from writer") {
		t.Error("Writer output not found in logs")
	}
}

func TestGetConfig(t *testing.T) {
	config := &Config{
		Level:   "debug",
		Console: true,
		Format:  "json",
	}

	logger := New(config)
	retrieved := logger.GetConfig()

	if retrieved.Level != config.Level {
		t.Errorf("GetConfig().Level = %s, want %s", retrieved.Level, config.Level)
	}
	if retrieved.Console != config.Console {
		t.Errorf("GetConfig().Console = %v, want %v", retrieved.Console, config.Console)
	}
	if retrieved.Format != config.Format {
		t.Errorf("GetConfig().Format = %s, want %s", retrieved.Format, config.Format)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
