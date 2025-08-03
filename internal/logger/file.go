// Package logger provides structured logging capabilities.
package logger

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileWriter implements io.Writer with rotation support
type FileWriter struct {
	config    *FileWriterConfig
	file      *os.File
	size      int64
	mu        sync.Mutex
	millCh    chan struct{}
	startMill sync.Once
}

// FileWriterConfig holds configuration for file writer
type FileWriterConfig struct {
	MaxSize    int  // Maximum size in MB before rotation
	MaxBackups int  // Maximum number of old log files to keep
	MaxAge     int  // Maximum age in days
	Compress   bool // Whether to compress rotated files
}

// NewFileWriter creates a new file writer with rotation support
func NewFileWriter(filename string, config *FileWriterConfig) (*FileWriter, error) {
	if config == nil {
		config = &FileWriterConfig{
			MaxSize:    10,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		}
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	fw := &FileWriter{
		config: config,
		millCh: make(chan struct{}, 1),
	}

	// Open the file
	if err := fw.openFile(filename); err != nil {
		return nil, err
	}

	// Start the mill goroutine
	fw.startMill.Do(func() {
		go fw.millLoop(filename)
	})

	return fw, nil
}

// Write implements io.Writer
func (fw *FileWriter) Write(p []byte) (int, error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.file == nil {
		return 0, fmt.Errorf("file writer is closed")
	}

	n, err := fw.file.Write(p)
	fw.size += int64(n)

	// Check if rotation is needed
	if fw.config.MaxSize > 0 && fw.size >= int64(fw.config.MaxSize)*1024*1024 {
		select {
		case fw.millCh <- struct{}{}:
		default:
		}
	}

	return n, err
}

// Close closes the file writer
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.file == nil {
		return nil
	}

	err := fw.file.Close()
	fw.file = nil
	close(fw.millCh)
	return err
}

// openFile opens the log file
func (fw *FileWriter) openFile(filename string) error {
	// Path is internally managed
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	fw.file = file
	fw.size = info.Size()
	return nil
}

// rotate performs log rotation
func (fw *FileWriter) rotate(filename string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Close current file
	if fw.file != nil {
		_ = fw.file.Close()
		fw.file = nil
	}

	// Generate backup filename with timestamp
	now := time.Now()
	backupName := fmt.Sprintf("%s.%s", filename, now.Format("20060102-150405"))

	// Rename current file
	if err := os.Rename(filename, backupName); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Compress if configured
	if fw.config.Compress {
		go fw.compressFile(backupName)
	}

	// Open new file
	if err := fw.openFile(filename); err != nil {
		return err
	}

	// Clean old files
	go fw.cleanOldFiles(filename)

	return nil
}

// compressFile compresses a rotated log file
func (fw *FileWriter) compressFile(filename string) {
	// Path is internally managed
	src, err := os.Open(filename) // #nosec G304
	if err != nil {
		return
	}
	defer func() { _ = src.Close() }()

	// Path is internally managed
	dst, err := os.Create(filename + ".gz") // #nosec G304
	if err != nil {
		return
	}
	defer func() { _ = dst.Close() }()

	gz := gzip.NewWriter(dst)
	defer func() { _ = gz.Close() }()

	if _, err := io.Copy(gz, src); err != nil {
		return
	}

	// Remove original file after successful compression
	_ = os.Remove(filename)
}

// cleanOldFiles removes old log files based on MaxBackups and MaxAge
func (fw *FileWriter) cleanOldFiles(filename string) {
	if fw.config.MaxBackups <= 0 && fw.config.MaxAge <= 0 {
		return
	}

	dir := filepath.Dir(filename)
	base := filepath.Base(filename)

	// Find all backup files
	matches, err := filepath.Glob(filepath.Join(dir, base+".*"))
	if err != nil {
		return
	}

	backups := make([]logFile, 0, len(matches))
	cutoff := time.Now().Add(-24 * time.Hour * time.Duration(fw.config.MaxAge))

	for _, match := range matches {
		// Skip the current log file
		if match == filename {
			continue
		}

		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		// Check age if MaxAge is set
		if fw.config.MaxAge > 0 && info.ModTime().Before(cutoff) {
			_ = os.Remove(match)
			continue
		}

		backups = append(backups, logFile{
			path:    match,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Sort(sort.Reverse(byModTime(backups)))

	// Remove excess backups
	if fw.config.MaxBackups > 0 && len(backups) > fw.config.MaxBackups {
		for _, backup := range backups[fw.config.MaxBackups:] {
			_ = os.Remove(backup.path)
		}
	}
}

// millLoop runs the rotation loop
func (fw *FileWriter) millLoop(filename string) {
	for range fw.millCh {
		_ = fw.rotate(filename)
	}
}

type logFile struct {
	path    string
	modTime time.Time
}

type byModTime []logFile

func (b byModTime) Len() int           { return len(b) }
func (b byModTime) Less(i, j int) bool { return b[i].modTime.Before(b[j].modTime) }
func (b byModTime) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Check for trace ID in context
	if traceID := ctx.Value(traceIDKey); traceID != nil {
		if s, ok := traceID.(string); ok {
			return s
		}
	}

	// Generate a new trace ID if not found
	return generateTraceID()
}

// generateTraceID generates a new trace ID
func generateTraceID() string {
	// Simple implementation - in production, use a proper UUID library
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}

// contextKey is a type for context keys
type contextKey string

// traceIDKey is the context key for trace ID
const traceIDKey contextKey = "trace_id"

// ContextWithTraceID adds a trace ID to the context
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// StripANSI removes ANSI color codes from strings
func StripANSI(s string) string {
	// Simple implementation - removes common ANSI escape sequences
	var result strings.Builder
	inEscape := false

	for _, ch := range s {
		switch {
		case ch == '\033':
			inEscape = true
		case inEscape:
			if ch == 'm' {
				inEscape = false
			}
		default:
			result.WriteRune(ch)
		}
	}

	return result.String()
}
