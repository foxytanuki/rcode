package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// TextHandler is a custom text formatter for slog
type TextHandler struct {
	opts   *TextHandlerOptions
	writer io.Writer
	mu     sync.Mutex
	attrs  []slog.Attr
	group  string
}

// TextHandlerOptions are options for the TextHandler
type TextHandlerOptions struct {
	Level       slog.Level
	TimeFormat  string
	ColorOutput bool
}

// NewTextHandler creates a new text handler
func NewTextHandler(w io.Writer, opts *TextHandlerOptions) *TextHandler {
	if opts == nil {
		opts = &TextHandlerOptions{
			Level:      slog.LevelInfo,
			TimeFormat: time.RFC3339,
		}
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.RFC3339
	}
	return &TextHandler{
		opts:   opts,
		writer: w,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *TextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level
}

// Handle formats and writes the log record
//
//nolint:gocritic // slog.Handler interface requires value receiver
func (h *TextHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var sb strings.Builder

	// Time
	if h.opts.ColorOutput {
		sb.WriteString("\033[90m") // gray
	}
	sb.WriteString(r.Time.Format(h.opts.TimeFormat))
	if h.opts.ColorOutput {
		sb.WriteString("\033[0m")
	}
	sb.WriteString(" ")

	// Level
	levelStr := formatLevel(r.Level)
	if h.opts.ColorOutput {
		sb.WriteString(levelColor(r.Level))
	}
	sb.WriteString(levelStr)
	if h.opts.ColorOutput {
		sb.WriteString("\033[0m")
	}
	sb.WriteString(" ")

	// Message
	sb.WriteString(r.Message)

	// Attributes from WithAttrs
	for _, attr := range h.attrs {
		sb.WriteString(" ")
		if h.group != "" {
			sb.WriteString(h.group)
			sb.WriteString(".")
		}
		formatAttr(&sb, attr, h.opts.ColorOutput)
	}

	// Attributes from the record
	r.Attrs(func(a slog.Attr) bool {
		sb.WriteString(" ")
		if h.group != "" {
			sb.WriteString(h.group)
			sb.WriteString(".")
		}
		formatAttr(&sb, a, h.opts.ColorOutput)
		return true
	})

	sb.WriteString("\n")

	_, err := h.writer.Write([]byte(sb.String()))
	return err
}

// WithAttrs returns a new Handler with the given attributes added
func (h *TextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TextHandler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  append(h.attrs, attrs...),
		group:  h.group,
	}
}

// WithGroup returns a new Handler with the given group name
func (h *TextHandler) WithGroup(name string) slog.Handler {
	return &TextHandler{
		opts:   h.opts,
		writer: h.writer,
		attrs:  h.attrs,
		group:  name,
	}
}

// formatLevel formats the log level
func formatLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "DEBUG"
	case slog.LevelInfo:
		return "INFO "
	case slog.LevelWarn:
		return "WARN "
	case slog.LevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("%-5s", level.String())
	}
}

// levelColor returns ANSI color code for the level
func levelColor(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return "\033[36m" // cyan
	case slog.LevelInfo:
		return "\033[32m" // green
	case slog.LevelWarn:
		return "\033[33m" // yellow
	case slog.LevelError:
		return "\033[31m" // red
	default:
		return "\033[0m" // reset
	}
}

// formatAttr formats a single attribute
func formatAttr(sb *strings.Builder, attr slog.Attr, color bool) {
	if color {
		sb.WriteString("\033[96m") // light cyan
	}
	sb.WriteString(attr.Key)
	if color {
		sb.WriteString("\033[0m")
	}
	sb.WriteString("=")
	formatValue(sb, attr.Value)
}

// formatValue formats an attribute value
func formatValue(sb *strings.Builder, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		// Quote strings if they contain spaces or special characters
		s := v.String()
		if strings.ContainsAny(s, " \t\n\r\"") {
			fmt.Fprintf(sb, "%q", s)
		} else {
			sb.WriteString(s)
		}
	case slog.KindTime:
		sb.WriteString(v.Time().Format(time.RFC3339))
	case slog.KindGroup:
		sb.WriteString("{")
		first := true
		for _, attr := range v.Group() {
			if !first {
				sb.WriteString(" ")
			}
			formatAttr(sb, attr, false)
			first = false
		}
		sb.WriteString("}")
	default:
		fmt.Fprint(sb, v.Any())
	}
}

// JSONFormatter provides JSON formatting options
type JSONFormatter struct {
	Pretty bool
	Indent string
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter(pretty bool) *JSONFormatter {
	return &JSONFormatter{
		Pretty: pretty,
		Indent: "  ",
	}
}

// Format formats a log record as JSON
//
//nolint:gocritic // slog.Record is part of standard library interface
func (f *JSONFormatter) Format(_ slog.Record) string {
	// This would be used if we need custom JSON formatting
	// For now, we'll use the built-in slog.JSONHandler
	return ""
}
