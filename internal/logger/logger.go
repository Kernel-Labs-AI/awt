package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	// LevelDebug is for detailed debugging information
	LevelDebug LogLevel = iota
	// LevelInfo is for general informational messages
	LevelInfo
	// LevelWarn is for warning messages
	LevelWarn
	// LevelError is for error messages
	LevelError
)

// String returns the string representation of a log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging
type Logger struct {
	level  LogLevel
	writer io.Writer
	silent bool
}

// New creates a new logger
func New(level LogLevel, writer io.Writer) *Logger {
	if writer == nil {
		writer = os.Stderr
	}

	return &Logger{
		level:  level,
		writer: writer,
		silent: false,
	}
}

// Default returns a logger with default settings (INFO level, stderr)
func Default() *Logger {
	return New(LevelInfo, os.Stderr)
}

// Silent returns a logger that doesn't output anything
func Silent() *Logger {
	return &Logger{
		level:  LevelError,
		writer: io.Discard,
		silent: true,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetWriter sets the output writer
func (l *Logger) SetWriter(writer io.Writer) {
	l.writer = writer
}

// SetSilent enables or disables silent mode
func (l *Logger) SetSilent(silent bool) {
	l.silent = silent
	if silent {
		l.writer = io.Discard
	}
}

// log writes a log message if the level is high enough
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if l.silent || level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] %s: %s\n", timestamp, level.String(), message)

	_, _ = l.writer.Write([]byte(logLine))
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

// WithField returns a new logger with a field attached
func (l *Logger) WithField(key, value string) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: map[string]string{key: value},
	}
}

// WithFields returns a new logger with multiple fields attached
func (l *Logger) WithFields(fields map[string]string) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: fields,
	}
}

// FieldLogger is a logger with attached fields
type FieldLogger struct {
	logger *Logger
	fields map[string]string
}

// log writes a log message with fields
func (fl *FieldLogger) log(level LogLevel, format string, args ...interface{}) {
	if fl.logger.silent || level < fl.logger.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)

	// Format fields
	fieldsStr := ""
	if len(fl.fields) > 0 {
		fieldsStr = " ["
		first := true
		for k, v := range fl.fields {
			if !first {
				fieldsStr += ", "
			}
			fieldsStr += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
		fieldsStr += "]"
	}

	logLine := fmt.Sprintf("[%s] %s%s: %s\n", timestamp, level.String(), fieldsStr, message)
	_, _ = fl.logger.writer.Write([]byte(logLine))
}

// Debug logs a debug message
func (fl *FieldLogger) Debug(format string, args ...interface{}) {
	fl.log(LevelDebug, format, args...)
}

// Info logs an info message
func (fl *FieldLogger) Info(format string, args ...interface{}) {
	fl.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (fl *FieldLogger) Warn(format string, args ...interface{}) {
	fl.log(LevelWarn, format, args...)
}

// Error logs an error message
func (fl *FieldLogger) Error(format string, args ...interface{}) {
	fl.log(LevelError, format, args...)
}

// WithField adds a field to the logger
func (fl *FieldLogger) WithField(key, value string) *FieldLogger {
	newFields := make(map[string]string, len(fl.fields)+1)
	for k, v := range fl.fields {
		newFields[k] = v
	}
	newFields[key] = value
	return &FieldLogger{
		logger: fl.logger,
		fields: newFields,
	}
}

// Global logger instance
var globalLogger = Default()

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return globalLogger
}

// Debug logs a debug message using the global logger
func Debug(format string, args ...interface{}) {
	globalLogger.Debug(format, args...)
}

// Info logs an info message using the global logger
func Info(format string, args ...interface{}) {
	globalLogger.Info(format, args...)
}

// Warn logs a warning message using the global logger
func Warn(format string, args ...interface{}) {
	globalLogger.Warn(format, args...)
}

// Error logs an error message using the global logger
func Error(format string, args ...interface{}) {
	globalLogger.Error(format, args...)
}

// WithField returns a field logger using the global logger
func WithField(key, value string) *FieldLogger {
	return globalLogger.WithField(key, value)
}

// WithFields returns a field logger using the global logger
func WithFields(fields map[string]string) *FieldLogger {
	return globalLogger.WithFields(fields)
}
