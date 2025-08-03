package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestTextHandler(t *testing.T) {
	tests := []struct {
		name        string
		opts        *TextHandlerOptions
		record      slog.Record
		wantContain []string
	}{
		{
			name: "basic message",
			opts: &TextHandlerOptions{
				Level:      slog.LevelInfo,
				TimeFormat: time.RFC3339,
			},
			record:      makeRecord(slog.LevelInfo, "test message"),
			wantContain: []string{"INFO", "test message"},
		},
		{
			name: "with attributes",
			opts: &TextHandlerOptions{
				Level:      slog.LevelInfo,
				TimeFormat: time.RFC3339,
			},
			record: makeRecordWithAttrs(slog.LevelInfo, "test",
				slog.String("key", "value"),
				slog.Int("count", 42),
			),
			wantContain: []string{"INFO", "test", "key=value", "count=42"},
		},
		{
			name: "debug level",
			opts: &TextHandlerOptions{
				Level:      slog.LevelDebug,
				TimeFormat: time.RFC3339,
			},
			record:      makeRecord(slog.LevelDebug, "debug message"),
			wantContain: []string{"DEBUG", "debug message"},
		},
		{
			name: "error level",
			opts: &TextHandlerOptions{
				Level:      slog.LevelInfo,
				TimeFormat: time.RFC3339,
			},
			record:      makeRecord(slog.LevelError, "error message"),
			wantContain: []string{"ERROR", "error message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			handler := NewTextHandler(&buf, tt.opts)

			err := handler.Handle(context.Background(), tt.record)
			if err != nil {
				t.Errorf("Handle() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing %q\nGot: %s", want, output)
				}
			}
		})
	}
}

func TestTextHandlerEnabled(t *testing.T) {
	handler := NewTextHandler(nil, &TextHandlerOptions{
		Level: slog.LevelInfo,
	})

	tests := []struct {
		level   slog.Level
		enabled bool
	}{
		{slog.LevelDebug, false},
		{slog.LevelInfo, true},
		{slog.LevelWarn, true},
		{slog.LevelError, true},
	}

	for _, tt := range tests {
		got := handler.Enabled(context.Background(), tt.level)
		if got != tt.enabled {
			t.Errorf("Enabled(%v) = %v, want %v", tt.level, got, tt.enabled)
		}
	}
}

func TestTextHandlerWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := NewTextHandler(&buf, &TextHandlerOptions{
		Level: slog.LevelInfo,
	})

	attrs := []slog.Attr{
		slog.String("app", "test"),
		slog.String("version", "1.0"),
	}

	newHandler := handler.WithAttrs(attrs)
	record := makeRecord(slog.LevelInfo, "test message")

	err := newHandler.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "app=test") {
		t.Error("WithAttrs attribute 'app' not found in output")
	}
	if !strings.Contains(output, "version=1.0") {
		t.Error("WithAttrs attribute 'version' not found in output")
	}
}

func TestTextHandlerWithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := NewTextHandler(&buf, &TextHandlerOptions{
		Level: slog.LevelInfo,
	})

	groupHandler := handler.WithGroup("request")
	record := makeRecordWithAttrs(slog.LevelInfo, "test",
		slog.String("method", "GET"),
		slog.String("path", "/api"),
	)

	err := groupHandler.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "request.method=GET") {
		t.Error("Group prefix not found in output")
	}
	if !strings.Contains(output, "request.path=/api") {
		t.Error("Group prefix not found in output")
	}
}

func TestFormatLevel(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO "},
		{slog.LevelWarn, "WARN "},
		{slog.LevelError, "ERROR"},
	}

	for _, tt := range tests {
		got := formatLevel(tt.level)
		if got != tt.want {
			t.Errorf("formatLevel(%v) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestLevelColor(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, "\033[36m"},
		{slog.LevelInfo, "\033[32m"},
		{slog.LevelWarn, "\033[33m"},
		{slog.LevelError, "\033[31m"},
	}

	for _, tt := range tests {
		got := levelColor(tt.level)
		if got != tt.want {
			t.Errorf("levelColor(%v) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		value slog.Value
		want  string
	}{
		{
			name:  "simple string",
			value: slog.StringValue("hello"),
			want:  "hello",
		},
		{
			name:  "string with spaces",
			value: slog.StringValue("hello world"),
			want:  `"hello world"`,
		},
		{
			name:  "integer",
			value: slog.IntValue(42),
			want:  "42",
		},
		{
			name:  "boolean",
			value: slog.BoolValue(true),
			want:  "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			formatValue(&sb, tt.value)
			got := sb.String()
			if got != tt.want {
				t.Errorf("formatValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper functions for creating test records
func makeRecord(level slog.Level, msg string) slog.Record {
	return slog.NewRecord(time.Now(), level, msg, 0)
}

func makeRecordWithAttrs(level slog.Level, msg string, attrs ...slog.Attr) slog.Record {
	r := slog.NewRecord(time.Now(), level, msg, 0)
	r.AddAttrs(attrs...)
	return r
}
