package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// Setup initializes a new slog.Logger based on the environment and log file path.
// logFilePath: absolute path to the log file
func Setup(env, logFilePath string) *slog.Logger {
	var level slog.Level
	if env == "dev" {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	// Create log file directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		// Fallback to stderr if file creation fails
		slog.Error("Failed to create log directory", "error", err)
	}

	// Open log file
	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("Failed to open log file", "error", err)
	}

	// Multi-writer: write to both stdout and file
	var writers []io.Writer
	writers = append(writers, os.Stdout)
	if file != nil {
		writers = append(writers, file)
	}
	w := io.MultiWriter(writers...)

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if env == "dev" {
		handler = slog.NewTextHandler(w, opts)
	} else {
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}
