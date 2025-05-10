package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/umesh/dgla/config"
)

// Logger provides structured logging capabilities
type Logger struct {
	level      Level
	format     Format
	output     io.Writer
	fields     map[string]interface{}
}

// Level represents log severity level
type Level int

// Format represents log output format
type Format int

// Log levels
const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// Log formats
const (
	JSONFormat Format = iota
	TextFormat
)

// New creates a new logger with the provided configuration
func New(cfg config.LogConfig) (*Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	format, err := parseFormat(cfg.Format)
	if err != nil {
		return nil, err
	}

	output, err := getOutput(cfg.OutputPath)
	if err != nil {
		return nil, err
	}

	return &Logger{
		level:  level,
		format: format,
		output: output,
		fields: make(map[string]interface{}),
	}, nil
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		format: l.format,
		output: l.output,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		format: l.format,
		output: l.output,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// Debug logs a message at debug level
func (l *Logger) Debug(msg string) {
	if l.level <= DebugLevel {
		l.log("debug", msg)
	}
}

// Info logs a message at info level
func (l *Logger) Info(msg string) {
	if l.level <= InfoLevel {
		l.log("info", msg)
	}
}

// Warn logs a message at warn level
func (l *Logger) Warn(msg string) {
	if l.level <= WarnLevel {
		l.log("warn", msg)
	}
}

// Error logs a message at error level
func (l *Logger) Error(msg string) {
	if l.level <= ErrorLevel {
		l.log("error", msg)
	}
}

// log implements the actual logging logic
func (l *Logger) log(level, msg string) {
	now := time.Now().Format(time.RFC3339)

	var output string
	if l.format == JSONFormat {
		// Simple JSON format for demonstration
		fields := ""
		for k, v := range l.fields {
			fields += fmt.Sprintf(", \"%s\": \"%v\"", k, v)
		}
		output = fmt.Sprintf("{\"time\": \"%s\", \"level\": \"%s\", \"message\": \"%s\"%s}\n", now, level, msg, fields)
	} else {
		// Simple text format for demonstration
		fields := ""
		for k, v := range l.fields {
			fields += fmt.Sprintf(" %s=%v", k, v)
		}
		output = fmt.Sprintf("[%s] %s: %s%s\n", now, strings.ToUpper(level), msg, fields)
	}

	fmt.Fprint(l.output, output)
}

// parseLevel converts a string level to Level
func parseLevel(level string) (Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel, nil
	case "info":
		return InfoLevel, nil
	case "warn":
		return WarnLevel, nil
	case "error":
		return ErrorLevel, nil
	default:
		return InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// parseFormat converts a string format to Format
func parseFormat(format string) (Format, error) {
	switch strings.ToLower(format) {
	case "json":
		return JSONFormat, nil
	case "text":
		return TextFormat, nil
	default:
		return JSONFormat, fmt.Errorf("unknown log format: %s", format)
	}
}

// getOutput returns the appropriate writer based on config
func getOutput(path string) (io.Writer, error) {
	switch strings.ToLower(path) {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		// For a file path
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("error creating log directory: %w", err)
		}
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("error opening log file: %w", err)
		}
		return f, nil
	}
}
