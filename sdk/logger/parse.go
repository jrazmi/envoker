package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

func parseOutput(o string) io.Writer {
	switch strings.ToUpper(o) {
	case "STDOUT":
		return os.Stdout
	default:
		return os.Stdout
	}
}

// Helper to parse level strings
func parseLevel(s string) slog.Level {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
