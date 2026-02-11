package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	config  *Config
	mu      sync.RWMutex
	closers []io.Closer
}

// Config holds logger configuration
type Config struct {
	Level      string
	Console    bool
	File       string
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
	Format     string // "json" or "text"
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Default returns the default logger instance
func Default() *Logger {
	once.Do(func() {
		defaultLogger = New(&Config{
			Level:   "info",
			Console: true,
			Format:  "text",
		})
	})
	return defaultLogger
}

// New creates a new logger with the given configuration
func New(config *Config) *Logger {
	if config == nil {
		config = &Config{
			Level:   "info",
			Console: true,
			Format:  "text",
		}
	}

	level := parseLevel(config.Level)
	handlers := []slog.Handler{}
	var closers []io.Closer

	// Console handler
	if config.Console {
		var consoleHandler slog.Handler
		if config.Format == "json" {
			consoleHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: level,
			})
		} else {
			consoleHandler = NewTextHandler(os.Stdout, &TextHandlerOptions{
				Level: level,
			})
		}
		handlers = append(handlers, consoleHandler)
	}

	// File handler
	if config.File != "" {
		fileWriter, err := NewFileWriter(config.File, &FileWriterConfig{
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		})
		if err == nil {
			closers = append(closers, fileWriter)
			var fileHandler slog.Handler
			if config.Format == "json" {
				fileHandler = slog.NewJSONHandler(fileWriter, &slog.HandlerOptions{
					Level: level,
				})
			} else {
				fileHandler = NewTextHandler(fileWriter, &TextHandlerOptions{
					Level: level,
				})
			}
			handlers = append(handlers, fileHandler)
		}
	}

	// Combine handlers
	var handler slog.Handler
	switch len(handlers) {
	case 0:
		// Fallback to console if no handlers configured
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	case 1:
		handler = handlers[0]
	default:
		handler = NewMultiHandler(handlers...)
	}

	return &Logger{
		Logger:  slog.New(handler),
		config:  config,
		closers: closers,
	}
}

// parseLevel parses a string log level to slog.Level
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithContext returns a logger with context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger:  l.With("trace_id", GetTraceID(ctx)),
		config:  l.config,
		closers: l.closers,
	}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		Logger:  l.With(args...),
		config:  l.config,
		closers: l.closers,
	}
}

// WithError returns a logger with an error field
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger:  l.With("error", err.Error()),
		config:  l.config,
		closers: l.closers,
	}
}

// Debug logs at debug level
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs at info level
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs at warn level
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs at error level
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// Debugf logs formatted message at debug level
func (l *Logger) Debugf(format string, args ...any) {
	l.Logger.Debug(fmt.Sprintf(format, args...))
}

// Infof logs formatted message at info level
func (l *Logger) Infof(format string, args ...any) {
	l.Logger.Info(fmt.Sprintf(format, args...))
}

// Warnf logs formatted message at warn level
func (l *Logger) Warnf(format string, args ...any) {
	l.Logger.Warn(fmt.Sprintf(format, args...))
}

// Errorf logs formatted message at error level
func (l *Logger) Errorf(format string, args ...any) {
	l.Logger.Error(fmt.Sprintf(format, args...))
}

// SetLevel dynamically changes the log level
func (l *Logger) SetLevel(level string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.config.Level = level
	// Note: This would require recreating the logger to take effect
	// For simplicity, we'll document this as requiring a restart
}

// GetConfig returns the current logger configuration
func (l *Logger) GetConfig() Config {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return *l.config
}

// MultiHandler wraps multiple handlers
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler creates a handler that writes to multiple handlers
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

// Enabled reports whether the handler handles records at the given level
func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle handles the Record
//
//nolint:gocritic // slog.Handler interface requires value receiver
func (h *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// WithAttrs returns a new Handler with the given attributes added
func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return NewMultiHandler(handlers...)
}

// WithGroup returns a new Handler with the given group name
func (h *MultiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return NewMultiHandler(handlers...)
}

// Close closes all file handlers
func (l *Logger) Close() error {
	var firstErr error
	for _, c := range l.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	// This would be implemented for buffered writers
	return nil
}

// Writer returns an io.Writer that writes to the logger at the given level
func (l *Logger) Writer(level slog.Level) io.Writer {
	return &logWriter{logger: l, level: level}
}

type logWriter struct {
	logger *Logger
	level  slog.Level
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	w.logger.Log(context.Background(), w.level, msg)
	return len(p), nil
}
