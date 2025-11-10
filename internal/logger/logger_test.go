package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.expected {
				t.Errorf("LogLevel.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestLogger_SetLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelInfo, &buf)

	// Debug should not log at INFO level
	logger.Debug("debug message")
	if buf.Len() > 0 {
		t.Error("Debug message logged when level is INFO")
	}

	// Info should log
	logger.Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Error("Info message not logged")
	}

	buf.Reset()

	// Set level to ERROR
	logger.SetLevel(LevelError)

	// Warn should not log at ERROR level
	logger.Warn("warn message")
	if buf.Len() > 0 {
		t.Error("Warn message logged when level is ERROR")
	}

	// Error should log
	logger.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Error("Error message not logged")
	}
}

func TestLogger_Silent(t *testing.T) {
	logger := Silent()

	// Nothing should log
	var buf bytes.Buffer
	logger.SetWriter(&buf)

	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	if buf.Len() > 0 {
		t.Error("Silent logger logged output")
	}
}

func TestLogger_WithField(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelInfo, &buf)

	fieldLogger := logger.WithField("key", "value")
	fieldLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "key=value") {
		t.Error("Field not included in log output")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Message not included in log output")
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelInfo, &buf)

	fields := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	fieldLogger := logger.WithFields(fields)
	fieldLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Error("Field key1 not included in log output")
	}
	if !strings.Contains(output, "key2=value2") {
		t.Error("Field key2 not included in log output")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Message not included in log output")
	}
}

func TestFieldLogger_WithField(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelInfo, &buf)

	fieldLogger := logger.WithField("key1", "value1").WithField("key2", "value2")
	fieldLogger.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Error("Field key1 not included in log output")
	}
	if !strings.Contains(output, "key2=value2") {
		t.Error("Field key2 not included in log output")
	}
}

func TestGlobalLogger(t *testing.T) {
	// Save original global logger
	orig := GetGlobalLogger()
	defer SetGlobalLogger(orig)

	var buf bytes.Buffer
	newLogger := New(LevelInfo, &buf)
	SetGlobalLogger(newLogger)

	Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Error("Global logger did not log message")
	}
	if !strings.Contains(output, "INFO") {
		t.Error("Log level not included in output")
	}
}

func TestLogFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelInfo, &buf)

	logger.Info("test message")

	output := buf.String()

	// Check for timestamp format [YYYY-MM-DD HH:MM:SS]
	if !strings.Contains(output, "[202") {
		t.Error("Timestamp not found in log output")
	}

	// Check for level
	if !strings.Contains(output, "INFO") {
		t.Error("Log level not found in log output")
	}

	// Check for message
	if !strings.Contains(output, "test message") {
		t.Error("Message not found in log output")
	}

	// Check format contains brackets and colon
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") || !strings.Contains(output, ":") {
		t.Error("Log format incorrect")
	}
}

func TestLogger_MultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	logger := New(LevelDebug, &buf)

	logger.Debug("debug 1")
	logger.Info("info 1")
	logger.Warn("warn 1")
	logger.Error("error 1")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 4 {
		t.Errorf("Expected 4 log lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "DEBUG") {
		t.Error("First line should be DEBUG")
	}
	if !strings.Contains(lines[1], "INFO") {
		t.Error("Second line should be INFO")
	}
	if !strings.Contains(lines[2], "WARN") {
		t.Error("Third line should be WARN")
	}
	if !strings.Contains(lines[3], "ERROR") {
		t.Error("Fourth line should be ERROR")
	}
}
