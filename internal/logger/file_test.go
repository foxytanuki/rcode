package logger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewFileWriter(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	fw, err := NewFileWriter(logFile, &FileWriterConfig{
		MaxSize:    1,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   false,
	})
	if err != nil {
		t.Fatalf("NewFileWriter() error = %v", err)
	}
	defer func() { _ = fw.Close() }()

	// Write some data
	data := []byte("test log entry\n")
	n, err := fw.Write(data)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}

	// Check file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestFileWriterRotation(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	fw, err := NewFileWriter(logFile, &FileWriterConfig{
		MaxSize:    1, // 1MB - small for testing
		MaxBackups: 2,
		MaxAge:     7,
		Compress:   false,
	})
	if err != nil {
		t.Fatalf("NewFileWriter() error = %v", err)
	}
	defer func() { _ = fw.Close() }()

	// Write enough data to trigger rotation
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = 'A'
	}

	_, err = fw.Write(largeData)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	// Give rotation time to happen
	time.Sleep(100 * time.Millisecond)

	// Check for rotated files
	files, err := filepath.Glob(filepath.Join(tempDir, "test.log*"))
	if err != nil {
		t.Errorf("Glob() error = %v", err)
	}
	if len(files) < 2 {
		t.Errorf("Expected at least 2 files after rotation, got %d", len(files))
	}
}

func TestFileWriterWithNilConfig(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	fw, err := NewFileWriter(logFile, nil)
	if err != nil {
		t.Fatalf("NewFileWriter() with nil config error = %v", err)
	}
	defer func() { _ = fw.Close() }()

	// Should use default config
	if fw.config.MaxSize != 10 {
		t.Errorf("Default MaxSize = %d, want 10", fw.config.MaxSize)
	}
	if fw.config.MaxBackups != 5 {
		t.Errorf("Default MaxBackups = %d, want 5", fw.config.MaxBackups)
	}
}

func TestGetTraceID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{
			name: "with trace ID",
			ctx:  ContextWithTraceID(context.Background(), "test-123"),
			want: "test-123",
		},
		{
			name: "without trace ID",
			ctx:  context.Background(),
			want: "", // Will generate new one
		},
		{
			name: "nil context",
			ctx:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTraceID(tt.ctx)
			if tt.want != "" {
				if got != tt.want {
					t.Errorf("GetTraceID() = %v, want %v", got, tt.want)
				}
			} else {
				// For generated IDs, just check it's not empty
				if tt.ctx != nil && got == "" {
					t.Error("GetTraceID() returned empty for non-nil context")
				}
			}
		})
	}
}

func TestContextWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "test-trace-456"

	newCtx := ContextWithTraceID(ctx, traceID)

	// Extract and verify
	if val := newCtx.Value(traceIDKey); val != traceID {
		t.Errorf("ContextWithTraceID() trace_id = %v, want %v", val, traceID)
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "plain text",
			want:  "plain text",
		},
		{
			input: "\033[31mred text\033[0m",
			want:  "red text",
		},
		{
			input: "\033[1;32mgreen bold\033[0m and normal",
			want:  "green bold and normal",
		},
		{
			input: "mixed \033[33myellow\033[0m text",
			want:  "mixed yellow text",
		},
	}

	for _, tt := range tests {
		got := StripANSI(tt.input)
		if got != tt.want {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLogFileSort(t *testing.T) {
	now := time.Now()
	files := []logFile{
		{path: "log.3", modTime: now.Add(-3 * time.Hour)},
		{path: "log.1", modTime: now.Add(-1 * time.Hour)},
		{path: "log.2", modTime: now.Add(-2 * time.Hour)},
	}

	sorted := byModTime(files)
	sorted.Swap(0, 2) // Test swap

	if sorted[0].path != "log.2" {
		t.Errorf("After swap, first file = %s, want log.2", sorted[0].path)
	}

	if !sorted.Less(0, 1) {
		t.Error("Less() comparison failed")
	}

	if sorted.Len() != 3 {
		t.Errorf("Len() = %d, want 3", sorted.Len())
	}
}

func TestGenerateTraceID(t *testing.T) {
	id1 := generateTraceID()
	id2 := generateTraceID()

	if id1 == "" {
		t.Error("generateTraceID() returned empty string")
	}

	if id1 == id2 {
		t.Error("generateTraceID() returned duplicate IDs")
	}

	// Check format (should contain hyphen)
	if !strings.Contains(id1, "-") {
		t.Error("generateTraceID() format incorrect, missing hyphen")
	}
}
