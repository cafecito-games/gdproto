package applog

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// ParseLevel converts a level name (case-insensitive) into a slog.Level.
// Accepted: debug, info, warn, warning, error.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q (want debug|info|warn|error)", s)
	}
}

// New constructs a JSON-formatted slog.Logger writing to w at the given level.
func New(w io.Writer, level slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

// For returns a child logger tagged with the given component name.
// Components should call this once at construction time.
func For(parent *slog.Logger, component string) *slog.Logger {
	if parent == nil {
		return Discard()
	}
	return parent.With("component", component)
}

// Discard returns a logger that writes nothing. Use in tests.
func Discard() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}
