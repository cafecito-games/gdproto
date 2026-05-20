package applog_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/cafecito-games/gdproto/internal/applog"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"DEBUG": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	for input, want := range cases {
		got, err := applog.ParseLevel(input)
		if err != nil {
			t.Errorf("ParseLevel(%q) error: %v", input, err)
			continue
		}
		if got != want {
			t.Errorf("ParseLevel(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestParseLevelInvalid(t *testing.T) {
	if _, err := applog.ParseLevel("loud"); err == nil {
		t.Fatal("expected error for invalid level, got nil")
	}
}

func TestForAddsComponentAttribute(t *testing.T) {
	var buf bytes.Buffer
	parent := applog.New(&buf, slog.LevelDebug)
	logger := applog.For(parent, "lexer")
	logger.Info("hello")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("log line not JSON: %v; raw=%q", err, buf.String())
	}
	if got := record["component"]; got != "lexer" {
		t.Fatalf("component attr = %v, want %q", got, "lexer")
	}
	if got := record["msg"]; got != "hello" {
		t.Fatalf("msg = %v, want %q", got, "hello")
	}
}

func TestDiscardLoggerSwallowsOutput(t *testing.T) {
	logger := applog.Discard()
	logger.Info("nothing")
	if logger == nil {
		t.Fatal("Discard() returned nil")
	}
}

func TestNewWritesAtLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := applog.New(&buf, slog.LevelWarn)
	logger.Debug("noisy")
	logger.Warn("important")

	out := buf.String()
	if strings.Contains(out, "noisy") {
		t.Errorf("debug message leaked at warn level: %q", out)
	}
	if !strings.Contains(out, "important") {
		t.Errorf("warn message missing: %q", out)
	}
}

// Compile-time check we did not accidentally export the wrong types.
var _ io.Writer = (*bytes.Buffer)(nil)
